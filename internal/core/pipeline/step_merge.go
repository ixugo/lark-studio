package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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
	sortAudioFiles(audioFiles)

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

// trimTrailingSilence 以 O(n) 串行规范化音频并仅裁掉尾部静音。
func (c *Core) trimTrailingSilence(ctx context.Context, audioFiles []string, job Job) int {
	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	trimmed := 0
	for _, audioPath := range audioFiles {
		if ctx.Err() != nil {
			break
		}
		wasTrimmed, err := normalizeAudioSegment(ctx, ffmpeg, audioPath)
		if err != nil {
			slog.Debug("normalize audio failed", "file", audioPath, "err", err)
			continue
		}
		if wasTrimmed {
			trimmed++
		}
	}
	return trimmed
}

// normalizeAudioSegment 保留句中停顿，裁去尾静音并统一为单声道 PCM WAV。
func normalizeAudioSegment(ctx context.Context, ffmpeg, audioPath string) (bool, error) {
	normalizedPath := audioPath + ".normalized.wav"
	before := probeMediaDuration(ffmpeg, audioPath)
	filter := "areverse,silenceremove=" +
		"start_periods=1:start_duration=0.05:start_threshold=-45dB,areverse"
	cmd := exec.CommandContext(
		ctx,
		ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-i", audioPath,
		"-af", filter, "-ar", "24000", "-ac", "1", "-c:a", "pcm_s16le",
		normalizedPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(normalizedPath)
		return false, fmt.Errorf("ffmpeg 规范化失败: %w, output: %s", err, output)
	}
	info, err := os.Stat(normalizedPath)
	if err != nil || info.Size() == 0 {
		_ = os.Remove(normalizedPath)
		return false, fmt.Errorf("规范化音频为空")
	}
	after := probeMediaDuration(ffmpeg, normalizedPath)
	if err := os.Rename(normalizedPath, audioPath); err != nil {
		_ = os.Remove(normalizedPath)
		return false, fmt.Errorf("替换规范化音频失败: %w", err)
	}
	return before-after > 0.03, nil
}

// sortAudioFiles 按字幕数字序号排序，避免 10.wav 排在 2.wav 前。
func sortAudioFiles(audioFiles []string) {
	sort.SliceStable(audioFiles, func(left, right int) bool {
		leftIndex, leftOK := audioSegmentIndex(audioFiles[left])
		rightIndex, rightOK := audioSegmentIndex(audioFiles[right])
		if leftOK && rightOK {
			return leftIndex < rightIndex
		}
		if leftOK != rightOK {
			return leftOK
		}
		return audioFiles[left] < audioFiles[right]
	})
}

// audioSegmentIndex 从配音文件名读取字幕序号。
func audioSegmentIndex(path string) (int, bool) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	index, err := strconv.Atoi(name)
	return index, err == nil
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
