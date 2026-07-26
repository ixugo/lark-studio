package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runWhisper 执行语音识别步骤
// 若同名 .srt 已存在则跳过 whisper，直接使用已有字幕
func (c *Core) runWhisper(ctx context.Context, job Job) error {
	inputDir := filepath.Dir(job.InputPath)
	baseName := strings.TrimSuffix(filepath.Base(job.InputPath), filepath.Ext(job.InputPath))

	// 检查同名 SRT 是否已存在（用户手动提供的字幕）
	existingSRT := filepath.Join(inputDir, baseName+".srt")
	if _, err := os.Stat(existingSRT); err == nil {
		c.notifier.OnLog(job.TaskID, fmt.Sprintf("发现已有字幕: %s，跳过 whisper", existingSRT))
		// 复制到工作目录
		workSRT := filepath.Join(job.OutputDir, "src.srt")
		data, err := os.ReadFile(existingSRT)
		if err != nil {
			return fmt.Errorf("读取已有字幕失败: %w", err)
		}
		return os.WriteFile(workSRT, data, 0o644)
	}

	// 提取音频
	audioPath := filepath.Join(job.OutputDir, "raw.mp3")
	if err := c.extractAudio(ctx, job.InputPath, audioPath); err != nil {
		return fmt.Errorf("提取音频失败: %w", err)
	}

	// whisper 转写
	outputSRT := filepath.Join(job.OutputDir, "src.srt")
	c.notifier.OnProgress(job.TaskID, StepWhisper, 30)

	if err := c.whisper.Transcribe(ctx, audioPath, outputSRT, "auto"); err != nil {
		return fmt.Errorf("whisper 转写失败: %w", err)
	}

	c.notifier.OnProgress(job.TaskID, StepWhisper, 100)
	return nil
}

// extractAudio 用 ffmpeg 从视频提取音频
func (c *Core) extractAudio(ctx context.Context, videoPath, audioPath string) error {
	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	cmd := exec.CommandContext(ctx, ffmpeg,
		"-y", "-i", videoPath,
		"-vn", "-ar", "16000", "-ac", "1",
		"-b:a", "128k", audioPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg 失败: %s, output: %s", err, string(output))
	}
	return nil
}
