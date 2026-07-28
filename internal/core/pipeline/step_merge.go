package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// runMerge 合并音频段到时间轴
// 先对超时段做温和调速（上限 MaxSpeedFactor），再按字幕时间轴拼接
func (c *Core) runMerge(ctx context.Context, job Job) error {
	audioDir := filepath.Join(job.OutputDir, "audio_segs")
	srcSRT := filepath.Join(job.OutputDir, "src.srt")

	entries, err := os.ReadDir(audioDir)
	if err != nil {
		return fmt.Errorf("读取音频目录失败: %w", err)
	}

	var audioFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".wav") {
			audioFiles = append(audioFiles, filepath.Join(audioDir, e.Name()))
		}
	}
	sort.Strings(audioFiles)

	if len(audioFiles) == 0 {
		return fmt.Errorf("无音频段文件")
	}

	srtData, err := os.ReadFile(srcSRT)
	if err != nil {
		return fmt.Errorf("读取字幕时间轴失败: %w", err)
	}
	srtEntries := parseSRT(string(srtData))

	trimmed := c.trimTrailingSilence(ctx, audioFiles, job)
	if trimmed > 0 {
		c.logEvent(job.TaskID, "info", StepMerge, "裁掉 %d 段音频尾部静音", trimmed)
	}

	c.adjustAudioSpeeds(ctx, audioFiles, srtEntries, job)

	dubAudio := filepath.Join(job.OutputDir, "dub.mp3")
	if err := c.concatWithTimeline(ctx, audioFiles, srtEntries, dubAudio); err != nil {
		return fmt.Errorf("合并音频失败: %w", err)
	}

	c.notifier.OnProgress(job.TaskID, StepMerge, 100)
	return nil
}

// trimTrailingSilence 裁掉每段 TTS 音频尾部的无声段
// 避免静音膨胀导致调速计算偏差和字幕间隙异常
func (c *Core) trimTrailingSilence(ctx context.Context, audioFiles []string, job Job) int {
	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	trimmed := 0
	for _, af := range audioFiles {
		if ctx.Err() != nil {
			break
		}

		trimPath := af + ".trim.wav"
		cmd := exec.CommandContext(ctx, ffmpeg,
			"-y", "-i", af,
			"-af", "silenceremove=stop_periods=1:stop_threshold=-40dB:stop_duration=0.05",
			trimPath,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			slog.Debug("trim trailing silence failed", "file", af, "err", err, "output", string(out))
			continue
		}

		trimInfo, _ := os.Stat(trimPath)
		origInfo, _ := os.Stat(af)
		if trimInfo != nil && origInfo != nil && trimInfo.Size() > 0 && trimInfo.Size() < origInfo.Size() {
			os.Remove(af)
			os.Rename(trimPath, af)
			trimmed++
		} else {
			os.Remove(trimPath)
		}
	}
	return trimmed
}

// adjustAudioSpeeds 温和调速：TTS 音频超过原始字幕时长时，用 atempo 适度加速
// maxFactor ≤1 时跳过调速；差异 <2% 时不处理
func (c *Core) adjustAudioSpeeds(ctx context.Context, audioFiles []string, entries []srtEntry, job Job) {
	maxFactor := c.cfg.MaxSpeedFactor
	if maxFactor <= 1 {
		return
	}

	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	adjusted := 0
	for i, af := range audioFiles {
		if i >= len(entries) || ctx.Err() != nil {
			break
		}

		audioDur := probeMediaDuration(ffmpeg, af)
		segDur := entries[i].EndSec - entries[i].StartSec
		if audioDur <= 0 || segDur <= 0 || audioDur <= segDur {
			continue
		}

		factor := audioDur / segDur
		if factor < 1.02 {
			continue
		}
		if factor > maxFactor {
			factor = maxFactor
		}

		adjPath := af + ".adj.wav"
		cmd := exec.CommandContext(ctx, ffmpeg,
			"-y", "-i", af,
			"-filter:a", fmt.Sprintf("atempo=%.3f", factor),
			adjPath,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			slog.Debug("audio speed adjust failed", "file", af, "err", err, "output", string(out))
			continue
		}

		os.Remove(af)
		os.Rename(adjPath, af)
		adjusted++
	}

	if adjusted > 0 {
		c.logEvent(job.TaskID, "info", StepMerge, "调速 %d 段音频（上限 %.1fx）", adjusted, maxFactor)
	}
}

// concatWithTimeline 按字幕时间轴拼接音频段（中间插入静音）
func (c *Core) concatWithTimeline(ctx context.Context, audioFiles []string, entries []srtEntry, outputPath string) error {
	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	concatList := filepath.Join(filepath.Dir(outputPath), "concat_list.txt")
	var sb strings.Builder

	// ffmpeg concat 解析相对路径时基于列表文件所在目录，必须用绝对路径避免路径翻倍
	absPath := func(p string) string {
		a, err := filepath.Abs(p)
		if err != nil {
			return p
		}
		return a
	}

	for i, af := range audioFiles {
		if i >= len(entries) {
			break
		}

		if i == 0 && entries[0].StartSec > 0 {
			silPath := filepath.Join(filepath.Dir(outputPath), "silence_start.wav")
			if err := c.genSilence(ctx, entries[0].StartSec, silPath); err != nil {
				return err
			}
			sb.WriteString(fmt.Sprintf("file '%s'\n", absPath(silPath)))
		}

		sb.WriteString(fmt.Sprintf("file '%s'\n", absPath(af)))

		if i < len(audioFiles)-1 && i+1 < len(entries) {
			gap := entries[i+1].StartSec - entries[i].EndSec
			if gap > 0.1 {
				silPath := filepath.Join(filepath.Dir(outputPath), fmt.Sprintf("silence_%d.wav", i))
				if err := c.genSilence(ctx, gap, silPath); err != nil {
					return err
				}
				sb.WriteString(fmt.Sprintf("file '%s'\n", absPath(silPath)))
			}
		}
	}

	if err := os.WriteFile(concatList, []byte(sb.String()), 0o644); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, ffmpeg,
		"-y", "-f", "concat", "-safe", "0",
		"-i", concatList,
		"-c:a", "libmp3lame", "-b:a", "128k",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg concat 失败: %s, output: %s", err, string(output))
	}
	return nil
}

// genSilence 生成指定时长的静音文件
func (c *Core) genSilence(ctx context.Context, durationSec float64, outputPath string) error {
	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	cmd := exec.CommandContext(ctx, ffmpeg,
		"-y", "-f", "lavfi",
		"-i", fmt.Sprintf("anullsrc=r=16000:cl=mono:d=%.3f", durationSec),
		"-ar", "16000", "-ac", "1",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("生成静音失败: %s, output: %s", err, string(output))
	}
	return nil
}
