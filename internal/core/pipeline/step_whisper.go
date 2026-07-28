package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runWhisper 执行语音识别步骤
// 优先级：输出目录 src.srt > 输入目录同名 .srt > 执行 whisper
func (c *Core) runWhisper(ctx context.Context, job Job) error {
	workSRT := filepath.Join(job.OutputDir, "src.srt")
	reused, err := c.reuseWhisperSubtitle(job, workSRT)
	if err != nil || reused {
		return err
	}

	audioPath := filepath.Join(job.OutputDir, "raw.mp3")
	if err := c.extractAudio(ctx, job, audioPath); err != nil {
		return fmt.Errorf("提取音频失败: %w", err)
	}
	return c.transcribeAudio(ctx, job, audioPath, workSRT)
}

// reuseWhisperSubtitle 复用断点字幕或用户提供的同名字幕。
func (c *Core) reuseWhisperSubtitle(job Job, workSRT string) (bool, error) {
	if info, err := os.Stat(workSRT); err == nil && info.Size() > 0 {
		c.logEvent(job.TaskID, "info", StepWhisper, "已有听写字幕，跳过重复识别")
		c.notifier.OnProgress(job.TaskID, StepWhisper, 100)
		return true, nil
	}
	baseName := strings.TrimSuffix(filepath.Base(job.InputPath), filepath.Ext(job.InputPath))
	existingSRT := filepath.Join(filepath.Dir(job.InputPath), baseName+".srt")
	if _, err := os.Stat(existingSRT); err != nil {
		return false, nil
	}
	c.logEvent(job.TaskID, "info", StepWhisper, "发现已有字幕：%s", existingSRT)
	data, err := os.ReadFile(existingSRT)
	if err != nil {
		return false, fmt.Errorf("读取已有字幕失败: %w", err)
	}
	if err := os.WriteFile(workSRT, data, 0o644); err != nil {
		return false, err
	}
	c.notifier.OnProgress(job.TaskID, StepWhisper, 100)
	return true, nil
}

// transcribeAudio 调用 Whisper 并把原始输出和百分比实时转发。
func (c *Core) transcribeAudio(ctx context.Context, job Job, audioPath, workSRT string) error {
	c.notifier.OnProgress(job.TaskID, StepWhisper, 20)
	err := c.whisper.Transcribe(
		ctx,
		audioPath,
		workSRT,
		"auto",
		func(progress int) {
			c.notifier.OnProgress(job.TaskID, StepWhisper, 20+progress*80/100)
		},
		func(line string) {
			c.logEvent(job.TaskID, "info", StepWhisper, "%s", line)
		},
	)
	if err != nil {
		return fmt.Errorf("whisper 转写失败: %w", err)
	}
	c.notifier.OnProgress(job.TaskID, StepWhisper, 100)
	if data, err := os.ReadFile(workSRT); err == nil {
		c.logEvent(job.TaskID, "success", StepWhisper, "听写完成：%d 条字幕", len(parseSRT(string(data))))
	}
	return nil
}

// extractAudio 用 ffmpeg 从视频提取音频
func (c *Core) extractAudio(ctx context.Context, job Job, audioPath string) error {
	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	totalDuration := probeMediaDuration(ffmpeg, job.InputPath)
	args := []string{ffmpeg,
		"-y", "-i", job.InputPath,
		"-vn", "-ar", "16000", "-ac", "1",
		"-b:a", "128k", audioPath,
	}
	err := runFFmpegWithProgress(
		ctx,
		totalDuration,
		func(progress int) {
			c.notifier.OnProgress(job.TaskID, StepWhisper, progress/5)
		},
		func(line string) {
			c.logEvent(job.TaskID, "info", StepWhisper, "ffmpeg：%s", line)
		},
		args...,
	)
	if err != nil {
		return fmt.Errorf("ffmpeg 提取音频失败: %w", err)
	}
	return nil
}
