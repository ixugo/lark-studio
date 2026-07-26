package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runTTS 文本转语音步骤
func (c *Core) runTTS(ctx context.Context, job Job) error {
	transFile := filepath.Join(job.OutputDir, "trans.txt")
	data, err := os.ReadFile(transFile)
	if err != nil {
		return fmt.Errorf("读取翻译文件失败: %w", err)
	}

	sentences := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(sentences) == 0 {
		return fmt.Errorf("翻译结果为空")
	}

	// 获取 TTS 锁（独占 TTS 资源）
	c.ttsMu.Lock()
	defer c.ttsMu.Unlock()

	audioDir := filepath.Join(job.OutputDir, "audio_segs")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	c.notifier.OnLog(job.TaskID, fmt.Sprintf("TTS 合成 %d 句", len(sentences)))

	for i, text := range sentences {
		if err := ctx.Err(); err != nil {
			return err
		}

		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		outputPath := filepath.Join(audioDir, fmt.Sprintf("%d.wav", i))
		if err := c.tts.Synthesize(ctx, text, outputPath, ""); err != nil {
			return fmt.Errorf("TTS 第 %d 句失败: %w", i+1, err)
		}

		progress := ((i + 1) * 100) / len(sentences)
		c.notifier.OnProgress(job.TaskID, StepTTS, progress)
	}

	c.notifier.OnProgress(job.TaskID, StepTTS, 100)
	return nil
}
