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
// 优先级：输出目录 src.srt > 输入目录同名 .srt > 执行 whisper
func (c *Core) runWhisper(ctx context.Context, job Job) error {
	workSRT := filepath.Join(job.OutputDir, "src.srt")

	// 输出目录已有 src.srt（断点恢复 / 重跑场景）
	if info, err := os.Stat(workSRT); err == nil && info.Size() > 0 {
		c.notifier.OnLog(job.TaskID, "输出目录已有 src.srt，跳过 whisper")
		c.notifier.OnProgress(job.TaskID, StepWhisper, 100)
		return nil
	}

	// 输入目录同名 SRT（用户手动提供）
	inputDir := filepath.Dir(job.InputPath)
	baseName := strings.TrimSuffix(filepath.Base(job.InputPath), filepath.Ext(job.InputPath))
	existingSRT := filepath.Join(inputDir, baseName+".srt")
	if _, err := os.Stat(existingSRT); err == nil {
		c.notifier.OnLog(job.TaskID, fmt.Sprintf("发现已有字幕: %s，跳过 whisper", existingSRT))
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
	c.notifier.OnProgress(job.TaskID, StepWhisper, 30)
	if err := c.whisper.Transcribe(ctx, audioPath, workSRT, "auto"); err != nil {
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
