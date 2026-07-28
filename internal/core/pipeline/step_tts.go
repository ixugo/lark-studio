package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runTTS 文本转语音步骤
// ModeDub 模式下，若翻译步骤已通过流水线完成 TTS，则自动跳过
func (c *Core) runTTS(ctx context.Context, job Job) error {
	if job.Mode == ModeDub {
		audioDir := filepath.Join(job.OutputDir, "audio_segs")
		if info, err := os.Stat(audioDir); err == nil && info.IsDir() {
			entries, _ := os.ReadDir(audioDir)
			wavCount := 0
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".wav") {
					wavCount++
				}
			}
			if wavCount > 0 {
				c.logEvent(job.TaskID, "success", StepTTS, "配音已完成：%d 段", wavCount)
				c.notifier.OnProgress(job.TaskID, StepTTS, 100)
				return nil
			}
		}
	}

	transFile := filepath.Join(job.OutputDir, "trans.txt")
	data, err := os.ReadFile(transFile)
	if err != nil {
		return fmt.Errorf("读取翻译文件失败: %w", err)
	}

	sentences := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(sentences) == 0 {
		return fmt.Errorf("翻译结果为空")
	}

	c.ttsMu.Lock()
	defer c.ttsMu.Unlock()

	audioDir := filepath.Join(job.OutputDir, "audio_segs")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	c.logEvent(job.TaskID, "info", StepTTS, "配音开始：%d 句", len(sentences))

	for i, text := range sentences {
		if err := ctx.Err(); err != nil {
			return err
		}

		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		outputPath := filepath.Join(audioDir, fmt.Sprintf("%d.wav", i))

		// 断点恢复：已存在的音频跳过
		if info, e := os.Stat(outputPath); e == nil && info.Size() > 0 {
			progress := ((i + 1) * 100) / len(sentences)
			c.notifier.OnProgress(job.TaskID, StepTTS, progress)
			continue
		}

		if err := c.tts.Synthesize(ctx, text, outputPath, ""); err != nil {
			return fmt.Errorf("TTS 第 %d 句失败: %w", i+1, err)
		}

		progress := ((i + 1) * 100) / len(sentences)
		c.notifier.OnProgress(job.TaskID, StepTTS, progress)
		c.logEvent(
			job.TaskID,
			"info",
			StepTTS,
			"配音进度 %d%%（%d/%d）",
			progress,
			i+1,
			len(sentences),
		)
	}

	c.notifier.OnProgress(job.TaskID, StepTTS, 100)
	return nil
}
