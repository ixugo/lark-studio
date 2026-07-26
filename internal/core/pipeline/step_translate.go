package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 翻译分块大小（每次发给 LLM 的句子数量）
const translateChunkSize = 8

// runTranslate 翻译步骤
// 直接读取 src.srt 的每条字幕文本做翻译，保证翻译结果与 SRT 条目 1:1 对应
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

	// 提取每条 SRT 的文本，保持 1:1 对应
	sentences := make([]string, len(entries))
	for i, e := range entries {
		sentences[i] = e.Text
	}

	c.llmMu.Lock()
	defer c.llmMu.Unlock()

	c.notifier.OnLog(job.TaskID, fmt.Sprintf("翻译 %d 句到 %s", len(sentences), job.TargetLang))

	var translated []string
	for i := 0; i < len(sentences); i += translateChunkSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+translateChunkSize, len(sentences))
		chunk := sentences[i:end]

		result, err := c.llm.Translate(ctx, chunk, job.TargetLang)
		if err != nil {
			return fmt.Errorf("翻译第 %d-%d 句失败: %w", i+1, end, err)
		}
		translated = append(translated, result...)

		c.notifier.OnProgress(job.TaskID, StepTranslate, (end*100)/len(sentences))
	}

	// 数量对齐（防御性处理）
	for len(translated) < len(entries) {
		translated = append(translated, entries[len(translated)].Text)
	}
	if len(translated) > len(entries) {
		translated = translated[:len(entries)]
	}

	// 写入翻译文本（TTS 读取用）
	transFile := filepath.Join(job.OutputDir, "trans.txt")
	if err := os.WriteFile(transFile, []byte(strings.Join(translated, "\n")), 0o644); err != nil {
		return fmt.Errorf("写入翻译结果失败: %w", err)
	}

	// 生成翻译字幕，时间轴与原始 SRT 完全一致
	transSRT := filepath.Join(job.OutputDir, "trans.srt")
	if err := buildTransSRT(entries, translated, transSRT); err != nil {
		return fmt.Errorf("生成翻译字幕失败: %w", err)
	}

	c.notifier.OnProgress(job.TaskID, StepTranslate, 100)
	return nil
}

// buildTransSRT 将翻译文本按原始时间轴写入 SRT
func buildTransSRT(entries []srtEntry, translated []string, outputPath string) error {
	var sb strings.Builder
	for i, entry := range entries {
		text := entry.Text
		if i < len(translated) {
			text = translated[i]
		}
		sb.WriteString(fmt.Sprintf("%d\n%s --> %s\n%s\n\n", i+1, entry.Start, entry.End, text))
	}
	return os.WriteFile(outputPath, []byte(sb.String()), 0o644)
}
