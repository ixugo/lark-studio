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
	c.notifier.OnProgress(job.TaskID, StepMerge, 1)
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
	c.logEvent(job.TaskID, "info", StepMerge, "混音开始：共 %d 段配音", len(audioFiles))

	dubAudio := filepath.Join(job.OutputDir, "dub.mp3")
	// 强制删除旧的 dub.mp3 和残留中间目录，防止历史脏文件导致静音未生效
	_ = os.Remove(dubAudio)
	_ = os.Remove(filepath.Join(job.OutputDir, "concat_list.txt"))
	_ = os.RemoveAll(filepath.Join(job.OutputDir, "intermediate"))

	// 优先采用翻译对齐后的字幕时间轴，若不存在则回退至原始听写字幕
	transSRT := filepath.Join(job.OutputDir, "trans.srt")
	srtPathToUse := transSRT
	srtBytes, srtErr := os.ReadFile(srtPathToUse)
	if srtErr != nil {
		srtPathToUse = srcSRT
		srtBytes, srtErr = os.ReadFile(srtPathToUse)
	}

	if srtErr != nil {
		// 纯文本朗读或小说朗读：无时间轴约束，直接按音频段顺次拼接为完整音频
		c.logEvent(job.TaskID, "info", StepMerge, "无外部时间轴，以段落顺次拼接完整音频")
		c.notifier.OnProgress(job.TaskID, StepMerge, 10)
		if err := c.concatSequential(ctx, audioFiles, dubAudio); err != nil {
			return fmt.Errorf("顺次合并音频失败: %w", err)
		}
		c.notifier.OnProgress(job.TaskID, StepMerge, 100)
		c.logEvent(job.TaskID, "success", StepMerge, "混音完成：顺次合并 %d 段配音", len(audioFiles))
		return nil
	}
	srtEntries := parseSRT(string(srtBytes))

	// 听写阶段负责识别人声绝对时间；混音只使用该时间轴，不猜测或平移字幕。

	if len(srtEntries) > 0 && srtEntries[0].StartSec > 0.05 {
		c.logEvent(
			job.TaskID,
			"info",
			StepMerge,
			"按字幕时间轴在 %.2f 秒放置首句配音，片头保持配音静音",
			srtEntries[0].StartSec,
		)
	}

	trimmed := c.trimTrailingSilence(ctx, audioFiles, job)
	if trimmed > 0 {
		c.logEvent(job.TaskID, "info", StepMerge, "裁掉 %d 段音频尾部静音", trimmed)
	}

	c.adjustAudioSpeeds(ctx, audioFiles, srtEntries, job)

	lastLoggedProgress := 0
	err = c.concatWithTimeline(ctx, audioFiles, srtEntries, dubAudio, func(progress int) {
		c.notifier.OnProgress(job.TaskID, StepMerge, 80+progress*19/100)
		if progress >= lastLoggedProgress+10 {
			c.logEvent(job.TaskID, "info", StepMerge, "混音拼接进度 %d%%", progress)
			lastLoggedProgress = progress
		}
	})
	if err != nil {
		return fmt.Errorf("合并音频失败: %w", err)
	}

	c.notifier.OnProgress(job.TaskID, StepMerge, 100)
	c.logEvent(job.TaskID, "success", StepMerge, "混音完成：已按字幕时间轴输出 dub.mp3")
	return nil
}

// trimTrailingSilence 以 O(n) 串行规范化音频并仅裁掉尾部静音。
func (c *Core) trimTrailingSilence(ctx context.Context, audioFiles []string, job Job) int {
	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	trimmed := 0
	for i, audioPath := range audioFiles {
		if ctx.Err() != nil {
			break
		}
		c.notifier.OnProgress(job.TaskID, StepMerge, 5+(i+1)*25/len(audioFiles))
		segmentNumber := i + 1
		if index, ok := audioSegmentIndex(audioPath); ok && index >= 0 {
			segmentNumber = index + 1
		}
		c.logEvent(job.TaskID, "info", StepMerge, "混音准备进度 %d/%d：整理配音段 %d", i+1, len(audioFiles), segmentNumber)
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

// normalizeAudioSegment 保留句中停顿，裁去首尾静音以贴合字幕起点，并统一为单声道 PCM WAV。
func normalizeAudioSegment(ctx context.Context, ffmpeg, audioPath string) (bool, error) {
	normalizedPath := audioPath + ".normalized.wav"
	before := probeMediaDuration(ffmpeg, audioPath)
	filter := "silenceremove=start_periods=1:start_duration=0.02:start_threshold=-45dB," +
		"areverse,silenceremove=" +
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

	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	adjusted := 0
	for i, af := range audioFiles {
		if ctx.Err() != nil {
			break
		}

		c.notifier.OnProgress(job.TaskID, StepMerge, 30+(i+1)*50/len(audioFiles))
		idx, ok := audioSegmentIndex(af)
		if !ok || idx < 0 || idx >= len(entries) {
			c.logEvent(job.TaskID, "warn", StepMerge, "配音段 %d/%d 无对应字幕时间轴，保持原速 1.00x", i+1, len(audioFiles))
			continue
		}
		segmentLabel := fmt.Sprintf("配音段 %d/%d", idx+1, len(audioFiles))
		audioDur := probeMediaDuration(ffmpeg, af)
		segDur := entries[idx].EndSec - entries[idx].StartSec
		if audioDur <= 0 || segDur <= 0 {
			c.logEvent(job.TaskID, "warn", StepMerge, "%s 无法读取有效时长，保持原速 1.00x", segmentLabel)
			continue
		}
		if maxFactor <= 1 {
			c.logEvent(job.TaskID, "info", StepMerge, "%s 保持原速 1.00x（调速上限 %.2fx）", segmentLabel, maxFactor)
			continue
		}
		if audioDur <= segDur {
			c.logEvent(job.TaskID, "info", StepMerge, "%s 保持原速 1.00x（音频 %.2fs，字幕窗 %.2fs；未做慢速）", segmentLabel, audioDur, segDur)
			continue
		}

		factor := audioDur / segDur
		if factor < 1.02 {
			c.logEvent(job.TaskID, "info", StepMerge, "%s 保持原速 1.00x（差异小于 2%%；未做慢速）", segmentLabel)
			continue
		}
		requiredFactor := factor
		if factor > maxFactor {
			factor = maxFactor
		}

		adjPath := af + ".adj.wav"
		cmd := exec.CommandContext(ctx, ffmpeg,
			"-hide_banner", "-y", "-i", af,
			"-filter:a", fmt.Sprintf("atempo=%.3f", factor),
			adjPath,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			slog.Debug("audio speed adjust failed", "file", af, "err", err, "output", string(out))
			c.logEvent(job.TaskID, "warn", StepMerge, "%s 加速失败，保留原速 1.00x：%v", segmentLabel, err)
			continue
		}

		if err := os.Rename(adjPath, af); err != nil {
			c.logEvent(job.TaskID, "warn", StepMerge, "%s 加速结果替换失败，原音频保持不变：%v", segmentLabel, err)
			continue
		}
		adjusted++
		adjustedDuration := audioDur / factor
		if requiredFactor > maxFactor {
			c.logEvent(job.TaskID, "warn", StepMerge, "%s 已加速 %.2fx（上限），时长 %.2fs → 约 %.2fs，仍超字幕窗 %.2fs", segmentLabel, factor, audioDur, adjustedDuration, segDur)
		} else {
			c.logEvent(job.TaskID, "info", StepMerge, "%s 已加速 %.2fx，时长 %.2fs → 约 %.2fs（字幕窗 %.2fs）", segmentLabel, factor, audioDur, adjustedDuration, segDur)
		}
	}

	if adjusted > 0 {
		c.logEvent(job.TaskID, "info", StepMerge, "调速 %d 段音频（上限 %.1fx）", adjusted, maxFactor)
	}
}

// concatWithTimeline 按字幕时间轴插入静音并拼接配音段。
func (c *Core) concatWithTimeline(ctx context.Context, audioFiles []string, entries []srtEntry, outputPath string, progress ...func(int)) error {
	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	intermediateDir := filepath.Join(filepath.Dir(outputPath), "intermediate")
	_ = os.MkdirAll(intermediateDir, 0o755)
	concatList := filepath.Join(intermediateDir, "concat_list.txt")
	var sb strings.Builder

	// ffmpeg concat 解析相对路径时基于列表文件所在目录，必须用绝对路径避免路径翻倍
	absPath := func(p string) string {
		a, err := filepath.Abs(p)
		if err != nil {
			return p
		}
		return a
	}

	currentTimeSec := 0.0
	for _, af := range audioFiles {
		idx, ok := audioSegmentIndex(af)
		if !ok || idx < 0 || idx >= len(entries) {
			continue
		}

		targetStart := entries[idx].StartSec
		// 若当前实际音频时间线落后于原定句首时间戳，精准填充静音到目标起点
		if targetStart > currentTimeSec {
			gap := targetStart - currentTimeSec
			if gap >= 0.03 {
				silPath := filepath.Join(intermediateDir, fmt.Sprintf("silence_%d.wav", idx))
				if err := c.genSilence(ctx, gap, silPath); err != nil {
					return err
				}
				sb.WriteString(fmt.Sprintf("file '%s'\n", absPath(silPath)))
				currentTimeSec += gap
			}
		}

		// 写入该句配音音频
		sb.WriteString(fmt.Sprintf("file '%s'\n", absPath(af)))
		dur := probeMediaDuration(ffmpeg, af)
		if dur > 0 {
			currentTimeSec += dur
		} else {
			currentTimeSec += entries[idx].EndSec - entries[idx].StartSec
		}
	}

	if err := os.WriteFile(concatList, []byte(sb.String()), 0o644); err != nil {
		return err
	}

	args := []string{
		ffmpeg, "-hide_banner", "-y", "-f", "concat", "-safe", "0",
		"-i", concatList,
		"-c:a", "libmp3lame", "-b:a", "128k",
		outputPath,
	}
	var onProgress func(int)
	if len(progress) > 0 {
		onProgress = progress[0]
	}
	var output strings.Builder
	logOutput := func(line string) {
		output.WriteString(line)
		output.WriteByte('\n')
	}
	if err := runFFmpegWithProgress(ctx, currentTimeSec, onProgress, logOutput, args...); err != nil {
		return fmt.Errorf("ffmpeg concat 失败: %w, output: %s", err, output.String())
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
		"-hide_banner", "-y", "-f", "lavfi",
		"-i", fmt.Sprintf("anullsrc=r=24000:cl=mono:d=%.3f", durationSec),
		"-c:a", "pcm_s16le", "-ar", "24000", "-ac", "1",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("生成静音失败: %s, output: %s", err, string(output))
	}
	return nil
}

// concatSequential 顺次拼接所有音频切片，专用于无时间轴的小说朗读或纯文本配音。
func (c *Core) concatSequential(ctx context.Context, audioFiles []string, outputPath string) error {
	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	intermediateDir := filepath.Join(filepath.Dir(outputPath), "intermediate")
	_ = os.MkdirAll(intermediateDir, 0o755)
	concatList := filepath.Join(intermediateDir, "concat_list.txt")
	var sb strings.Builder
	for _, f := range audioFiles {
		absPath, _ := filepath.Abs(f)
		fmt.Fprintf(&sb, "file '%s'\n", strings.ReplaceAll(absPath, "'", "'\\''"))
	}
	if err := os.WriteFile(concatList, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("写入拼接列表失败: %w", err)
	}

	cmd := exec.CommandContext(ctx, ffmpeg,
		"-hide_banner", "-y",
		"-f", "concat",
		"-safe", "0",
		"-i", concatList,
		"-c:a", "libmp3lame",
		"-q:a", "2",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("顺次拼接音频失败: %w, output: %s", err, string(output))
	}
	return nil
}
