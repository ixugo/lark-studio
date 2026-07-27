package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// 上下文窗口大小：每次翻译时携带前后各 3 句供 LLM 参考语境
const contextWindow = 3

// ttsPair 翻译→TTS 的流水线数据单元
type ttsPair struct {
	index int
	text  string
}

// runTranslate 翻译步骤
// ModeDub 模式下同时启动 TTS goroutine，实现翻译与 TTS 真流水线并行
func (c *Core) runTranslate(ctx context.Context, job Job) error {
	srcSRT := filepath.Join(job.OutputDir, "src.srt")
	data, err := os.ReadFile(srcSRT)
	if err != nil {
		return fmt.Errorf("读取字幕失败: %w", err)
	}

	entries := parseSRT(string(data))
	if len(entries) == 0 {
		return fmt.Errorf("字幕为空")
	}

	sentences := make([]string, len(entries))
	for i, e := range entries {
		sentences[i] = e.Text
	}

	if job.Mode == ModeDub {
		return c.translateAndTTS(ctx, job, entries, sentences)
	}
	return c.translateOnly(ctx, job, entries, sentences)
}

// translateOnly 仅翻译（ModeTranslate 用），不做 TTS
func (c *Core) translateOnly(ctx context.Context, job Job, entries []srtEntry, sentences []string) error {
	c.llmMu.Lock()
	defer c.llmMu.Unlock()

	c.notifier.OnLog(job.TaskID, fmt.Sprintf("翻译 %d 句到 %s", len(sentences), job.TargetLang))

	translated, err := c.translateAllChunks(ctx, job, sentences, nil)
	if err != nil {
		return err
	}

	return c.writeTranslationOutputs(entries, translated, job.OutputDir)
}

// translateAndTTS 翻译与 TTS 真流水线并行（ModeDub 用）
// 翻译每完成一个 chunk，立即将结果推入 channel；N 个 TTS worker 并发消费
// 有序性靠文件名 {index}.wav 保证，不会朗读两遍（已存在的文件自动跳过）
func (c *Core) translateAndTTS(ctx context.Context, job Job, entries []srtEntry, sentences []string) error {
	ttsWorkers := c.cfg.TTSWorkers
	if ttsWorkers <= 0 {
		ttsWorkers = 2
	}
	if ttsWorkers > 4 {
		ttsWorkers = 4
	}

	ttsCh := make(chan ttsPair, 300)
	audioDir := filepath.Join(job.OutputDir, "audio_segs")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	total := len(sentences)
	c.notifier.OnLog(job.TaskID, fmt.Sprintf("翻译+TTS 流水线: %d 句, TTS 并发 %d", total, ttsWorkers))

	var ttsDone atomic.Int32
	var ttsErr error
	var ttsOnce sync.Once
	var ttsWg sync.WaitGroup

	// TTS 消费者——持有 ttsMu 阻止其他任务同时使用 TTS 资源
	c.ttsMu.Lock()
	for w := range ttsWorkers {
		ttsWg.Add(1)
		go func(workerID int) {
			defer ttsWg.Done()
			for pair := range ttsCh {
				if ctx.Err() != nil {
					ttsOnce.Do(func() { ttsErr = ctx.Err() })
					for range ttsCh {
					}
					return
				}

				outputPath := filepath.Join(audioDir, fmt.Sprintf("%d.wav", pair.index))

				if info, e := os.Stat(outputPath); e == nil && info.Size() > 0 {
					done := int(ttsDone.Add(1))
					c.notifier.OnProgress(job.TaskID, StepTTS, (done*100)/total)
					continue
				}

				if err := c.tts.Synthesize(ctx, pair.text, outputPath, ""); err != nil {
					ttsOnce.Do(func() {
						ttsErr = fmt.Errorf("TTS 第 %d 句失败: %w", pair.index+1, err)
					})
					for range ttsCh {
					}
					return
				}
				done := int(ttsDone.Add(1))
				c.notifier.OnProgress(job.TaskID, StepTTS, (done*100)/total)
			}
		}(w)
	}

	// 翻译生产者：逐块翻译，每完成一块立即推入 channel
	c.llmMu.Lock()
	translated, translateErr := c.translateAllChunks(ctx, job, sentences, ttsCh)
	c.llmMu.Unlock()

	close(ttsCh)
	ttsWg.Wait()
	c.ttsMu.Unlock()

	if translateErr != nil {
		return translateErr
	}
	if ttsErr != nil {
		return ttsErr
	}

	return c.writeTranslationOutputs(entries, translated, job.OutputDir)
}

// translateAllChunks 分块翻译全部句子（调用方须持有 llmMu 锁）
// 每个 chunk 额外携带前后各 contextWindow 句作为上下文
// 若 streamTo 非 nil，每完成一个 chunk 立即将 (index, text) 推入 channel
func (c *Core) translateAllChunks(
	ctx context.Context, job Job, sentences []string, streamTo chan<- ttsPair,
) ([]string, error) {
	chunkSize := c.cfg.TranslateChunkSize
	if chunkSize <= 0 {
		chunkSize = 10
	}

	var translated []string
	total := len(sentences)

	for i := 0; i < total; i += chunkSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		end := min(i+chunkSize, total)
		chunk := sentences[i:end]

		ctxStart := max(0, i-contextWindow)
		ctxEnd := min(total, end+contextWindow)
		before := sentences[ctxStart:i]
		after := sentences[end:ctxEnd]

		result, err := c.llm.Translate(ctx, chunk, job.TargetLang, c.cfg.TranslatePrompt, before, after)
		if err != nil {
			return nil, fmt.Errorf("翻译第 %d-%d 句失败: %w", i+1, end, err)
		}
		translated = append(translated, result...)

		if streamTo != nil {
			for j, t := range result {
				streamTo <- ttsPair{index: i + j, text: t}
			}
		}

		c.notifier.OnProgress(job.TaskID, StepTranslate, (end*100)/total)
	}

	for len(translated) < total {
		translated = append(translated, sentences[len(translated)])
	}
	if len(translated) > total {
		translated = translated[:total]
	}

	c.notifier.OnProgress(job.TaskID, StepTranslate, 100)
	return translated, nil
}

// writeTranslationOutputs 写入翻译文本和翻译字幕文件
func (c *Core) writeTranslationOutputs(entries []srtEntry, translated []string, outputDir string) error {
	transFile := filepath.Join(outputDir, "trans.txt")
	if err := os.WriteFile(transFile, []byte(strings.Join(translated, "\n")), 0o644); err != nil {
		return fmt.Errorf("写入翻译结果失败: %w", err)
	}

	transSRT := filepath.Join(outputDir, "trans.srt")
	if err := buildTransSRT(entries, translated, transSRT); err != nil {
		return fmt.Errorf("生成翻译字幕失败: %w", err)
	}
	return nil
}

// buildTransSRT 将翻译文本按原始时间轴写入 SRT，自动对长行做语义断行
func buildTransSRT(entries []srtEntry, translated []string, outputPath string) error {
	var sb strings.Builder
	for i, entry := range entries {
		text := entry.Text
		if i < len(translated) {
			text = semanticBreak(translated[i])
		}
		sb.WriteString(fmt.Sprintf("%d\n%s --> %s\n%s\n\n", i+1, entry.Start, entry.End, text))
	}
	return os.WriteFile(outputPath, []byte(sb.String()), 0o644)
}
