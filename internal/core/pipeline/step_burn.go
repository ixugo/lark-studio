package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// 字幕样式常量（对齐 VideoLingo _7_sub_into_vid.py / _12_dub_to_vid.py）
const (
	subTransFontSize = 18
	subSrcFontSize   = 14
	subTransColor    = "&HFFFFFF"
	subSrcColor      = "&HCCCCCC"
	subOutlineColor  = "&H000000"
	subOutlineWidth  = 1
	subTransMarginV  = 36 // 翻译字幕距底部（缩小间距，VideoLingo 用 50）
	subSrcMarginV    = 12 // 原文字幕距底部（缩小间距，VideoLingo 用 20）
)

// subFontName 根据平台选择字体名（对齐 VideoLingo）
var subFontName = func() string {
	switch runtime.GOOS {
	case "linux":
		return "NotoSansCJK-Regular"
	case "darwin":
		return "Arial Unicode MS"
	default:
		return "Arial"
	}
}()

// resolveSourceVideo 若对口型步骤产出了 lipsync.mp4 则优先使用，否则用原始视频
func resolveSourceVideo(job Job) string {
	lipsync := filepath.Join(job.OutputDir, "lipsync.mp4")
	if _, err := os.Stat(lipsync); err == nil {
		return lipsync
	}
	return job.InputPath
}

// runBurn 烧录字幕到视频 / 合并配音
func (c *Core) runBurn(ctx context.Context, job Job) error {
	ffmpeg := c.cfg.FFmpegBin
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	baseName := strings.TrimSuffix(filepath.Base(job.InputPath), filepath.Ext(job.InputPath))
	transSRT := filepath.Join(job.OutputDir, "trans.srt")
	srcSRT := filepath.Join(job.OutputDir, "src.srt")
	dubAudio := filepath.Join(job.OutputDir, "dub.mp3")
	videoSrc := resolveSourceVideo(job)
	subtitleOutput := job.SubtitleOutput
	if subtitleOutput == "" {
		subtitleOutput = c.cfg.SubtitleOutput
	}
	primarySRT, secondarySRT := selectedSubtitleFiles(job.OutputContent, transSRT, srcSRT)

	switch job.Mode {
	case ModeSubtitle:
		outputVideo := filepath.Join(job.OutputDir, baseName+".sub.mp4")
		if subtitleOutput == "none" {
			if err := c.copyVideoOnly(ctx, ffmpeg, job.InputPath, outputVideo, job.TaskID); err != nil {
				return err
			}
		} else if subtitleOutput == "burn" {
			if err := c.burnHardSubtitle(ctx, ffmpeg, job.InputPath, srcSRT, "", outputVideo, job.TaskID); err != nil {
				return err
			}
		} else {
			// soft / file / 默认：封装软字幕轨，视频原生 copy
			if err := c.burnSubtitle(ctx, ffmpeg, job.InputPath, srcSRT, "", outputVideo, job.TaskID); err != nil {
				return err
			}
		}

	case ModeTranslate:
		outputVideo := filepath.Join(job.OutputDir, baseName+".trans.mp4")
		if subtitleOutput == "none" {
			if err := c.copyVideoOnly(ctx, ffmpeg, job.InputPath, outputVideo, job.TaskID); err != nil {
				return err
			}
		} else if subtitleOutput == "burn" {
			if err := c.burnHardSubtitle(ctx, ffmpeg, job.InputPath, primarySRT, secondarySRT, outputVideo, job.TaskID); err != nil {
				return err
			}
		} else {
			// soft / file / 默认：封装软字幕轨，视频原生 copy
			if err := c.burnSubtitle(ctx, ffmpeg, job.InputPath, primarySRT, secondarySRT, outputVideo, job.TaskID); err != nil {
				return err
			}
		}

	case ModeDub:
		outputVideo := filepath.Join(job.OutputDir, baseName+".final.mp4")
		if subtitleOutput == "burn" {
			// 硬字幕 + 配音混音
			if err := c.burnHardWithDub(ctx, ffmpeg, videoSrc, primarySRT, secondarySRT, dubAudio, outputVideo, job.TaskID); err != nil {
				return err
			}
		} else if subtitleOutput == "none" {
			// 无字幕，纯配音合成 (-c:v copy)
			if err := c.mergeDubVideo(ctx, ffmpeg, videoSrc, dubAudio, outputVideo, job.TaskID); err != nil {
				return err
			}
		} else {
			// soft / 默认：软字幕封装 + 配音混音 (-c:v copy)
			if err := c.burnWithDub(ctx, ffmpeg, videoSrc, primarySRT, secondarySRT, dubAudio, outputVideo, job.TaskID); err != nil {
				return err
			}
		}
	}

	c.notifier.OnProgress(job.TaskID, StepBurn, 100)
	return nil
}

// selectedSubtitleFiles 根据输出内容选择烧录的主副字幕。
func selectedSubtitleFiles(content, transSRT, srcSRT string) (string, string) {
	switch content {
	case "source":
		return srcSRT, ""
	case "translated":
		return transSRT, ""
	default:
		return transSRT, srcSRT
	}
}

// burnSubtitle 将字幕封装进视频容器中（流复制 -c:v copy，保持原画质，不重新对视频编码）
func (c *Core) burnSubtitle(ctx context.Context, ffmpeg, videoPath, transSRT, srcSRT, outputPath string, taskID string) error {
	var srtInputs []string
	if _, err := os.Stat(transSRT); err == nil {
		srtInputs = append(srtInputs, transSRT)
	}
	if srcSRT != "" && srcSRT != transSRT {
		if _, err := os.Stat(srcSRT); err == nil {
			srtInputs = append(srtInputs, srcSRT)
		}
	}

	if len(srtInputs) == 0 {
		return fmt.Errorf("无字幕文件可供封装")
	}

	totalDur := probeMediaDuration(ffmpeg, videoPath)

	subCodec := "mov_text"
	if strings.ToLower(filepath.Ext(outputPath)) == ".mkv" {
		subCodec = "srt"
	}

	args := []string{ffmpeg, "-hide_banner", "-y", "-i", videoPath}
	for _, s := range srtInputs {
		args = append(args, "-i", s)
	}

	// 映射原视频的视频流与音频流
	args = append(args, "-map", "0:v", "-map", "0:a?")

	// 映射字幕流
	for i := range srtInputs {
		args = append(args, "-map", fmt.Sprintf("%d:s", i+1))
	}

	// 绝对保持原有视频编码，直接 copy 流复制
	args = append(args, "-c:v", "copy", "-c:a", "copy", "-c:s", subCodec)

	if len(srtInputs) >= 1 {
		args = append(args, "-metadata:s:s:0", "language=chi")
	}
	if len(srtInputs) >= 2 {
		args = append(args, "-metadata:s:s:1", "language=eng")
	}
	args = append(args, outputPath)

	err := runFFmpegWithProgress(ctx, totalDur, func(pct int) {
		c.notifier.OnProgress(taskID, StepBurn, pct)
	}, func(line string) {
		c.logEvent(taskID, "info", StepBurn, "ffmpeg：%s", line)
	}, args...)
	if err != nil {
		return fmt.Errorf("封装字幕视频失败: %w", err)
	}
	return nil
}

// escapeFFmpegSubtitlesPath 转义 ffmpeg subtitles 滤镜中的文件名。
// 在 FFmpeg filtergraph 语法中，文件名如果包含冒号、逗号、反斜杠、单引号、中括号等，
// 必须做转义处理。最安全且健壮的格式是：
// 将反斜杠、单引号、冒号、中括号与逗号全部做 FFmpeg 特殊字符转义并用单引号包裹。
func escapeFFmpegSubtitlesPath(path string) string {
	clean := strings.ReplaceAll(path, "\\", "/")
	clean = strings.ReplaceAll(clean, "'", `'\''`)
	clean = strings.ReplaceAll(clean, ":", `\:`)
	clean = strings.ReplaceAll(clean, ",", `\,`)
	clean = strings.ReplaceAll(clean, "[", `\[`)
	clean = strings.ReplaceAll(clean, "]", `\]`)
	return "'" + clean + "'"
}

// burnWithDub 合成配音与字幕进视频容器中（视频画面严格使用 -c:v copy 流复制，不重新对视频编码）
func (c *Core) burnWithDub(ctx context.Context, ffmpeg, videoPath, transSRT, srcSRT, dubAudio, outputPath, taskID string) error {
	var srtInputs []string
	if _, err := os.Stat(transSRT); err == nil {
		srtInputs = append(srtInputs, transSRT)
	}
	if srcSRT != "" && srcSRT != transSRT {
		if _, err := os.Stat(srcSRT); err == nil {
			srtInputs = append(srtInputs, srcSRT)
		}
	}

	totalDur := probeMediaDuration(ffmpeg, videoPath)

	subCodec := "mov_text"
	if strings.ToLower(filepath.Ext(outputPath)) == ".mkv" {
		subCodec = "srt"
	}

	args := []string{
		ffmpeg, "-hide_banner", "-y",
		"-i", videoPath,
		"-i", dubAudio,
	}
	for _, s := range srtInputs {
		args = append(args, "-i", s)
	}

	// 混音原音频与新配音，原画面流直接复制
	args = append(args,
		"-filter_complex",
		"[0:a]volume=0.15[bg];[bg][1:a]amix=inputs=2:duration=first:dropout_transition=3[a]",
		"-map", "0:v:0",
		"-map", "[a]",
	)

	// 映射字幕轨
	for i := range srtInputs {
		args = append(args, "-map", fmt.Sprintf("%d:s", i+2))
	}

	// 映射字幕编码器
	if len(srtInputs) > 0 {
		args = append(args, "-c:s", subCodec)
		if len(srtInputs) >= 1 {
			args = append(args, "-metadata:s:s:0", "language=chi")
		}
		if len(srtInputs) >= 2 {
			args = append(args, "-metadata:s:s:1", "language=eng")
		}
	}

	args = append(args, "-c:v", "copy", "-c:a", "aac", "-b:a", "128k", outputPath)

	err := runFFmpegWithProgress(ctx, totalDur, func(pct int) {
		c.notifier.OnProgress(taskID, StepBurn, pct)
	}, func(line string) {
		c.logEvent(taskID, "info", StepBurn, "ffmpeg：%s", line)
	}, args...)
	if err != nil {
		return fmt.Errorf("合成配音视频失败: %w", err)
	}
	return nil
}

// mergeDubVideo 合成配音视频但不把字幕写入画面。
func (c *Core) mergeDubVideo(
	ctx context.Context,
	ffmpeg string,
	videoPath string,
	dubAudio string,
	outputPath string,
	taskID string,
) error {
	totalDuration := probeMediaDuration(ffmpeg, videoPath)
	args := []string{
		ffmpeg, "-hide_banner", "-y", "-i", videoPath, "-i", dubAudio,
		"-filter_complex",
		"[0:a]volume=0.15[bg];[bg][1:a]amix=inputs=2:duration=first:dropout_transition=3[a]",
		"-map", "0:v:0", "-map", "[a]", "-c:v", "copy",
		"-c:a", "aac", "-b:a", "128k", outputPath,
	}
	err := runFFmpegWithProgress(ctx, totalDuration, func(progress int) {
		c.notifier.OnProgress(taskID, StepBurn, progress)
	}, func(line string) {
		c.logEvent(taskID, "info", StepBurn, "ffmpeg：%s", line)
	}, args...)
	if err != nil {
		return fmt.Errorf("合成无字幕配音视频失败: %w", err)
	}
	return nil
}

// copyVideoOnly 无需字幕时，直接以最高性能流复制视频与音频文件
func (c *Core) copyVideoOnly(ctx context.Context, ffmpeg, videoPath, outputPath, taskID string) error {
	totalDuration := probeMediaDuration(ffmpeg, videoPath)
	args := []string{
		ffmpeg, "-hide_banner", "-y", "-i", videoPath,
		"-c", "copy", outputPath,
	}
	err := runFFmpegWithProgress(ctx, totalDuration, func(progress int) {
		c.notifier.OnProgress(taskID, StepBurn, progress)
	}, func(line string) {
		c.logEvent(taskID, "info", StepBurn, "ffmpeg：%s", line)
	}, args...)
	if err != nil {
		return fmt.Errorf("复制视频失败: %w", err)
	}
	return nil
}

// burnHardSubtitle 将字幕像素点阵直接烧录进画面（硬字幕，需重新编码画面）
func (c *Core) burnHardSubtitle(ctx context.Context, ffmpeg, videoPath, transSRT, srcSRT, outputPath string, taskID string) error {
	var filterParts []string

	if _, err := os.Stat(transSRT); err == nil {
		filterParts = append(filterParts, fmt.Sprintf(
			"subtitles=%s:force_style='FontSize=%d,FontName=%s,"+
				"PrimaryColour=%s,OutlineColour=%s,OutlineWidth=%d,"+
				"Alignment=2,MarginV=%d,BorderStyle=1'",
			escapeFFmpegSubtitlesPath(transSRT),
			subTransFontSize, subFontName,
			subTransColor, subOutlineColor, subOutlineWidth,
			subTransMarginV,
		))
	}

	if srcSRT != "" {
		if _, err := os.Stat(srcSRT); err == nil {
			filterParts = append(filterParts, fmt.Sprintf(
				"subtitles=%s:force_style='FontSize=%d,FontName=%s,"+
					"PrimaryColour=%s,OutlineColour=%s,OutlineWidth=%d,"+
					"Alignment=2,MarginV=%d,BorderStyle=1'",
				escapeFFmpegSubtitlesPath(srcSRT),
				subSrcFontSize, subFontName,
				subSrcColor, subOutlineColor, subOutlineWidth,
				subSrcMarginV,
			))
		}
	}

	if len(filterParts) == 0 {
		return fmt.Errorf("无字幕文件可烧录")
	}

	vf := strings.Join(filterParts, ",")
	totalDur := probeMediaDuration(ffmpeg, videoPath)

	args := []string{
		ffmpeg, "-hide_banner", "-y", "-i", videoPath,
		"-vf", vf,
		"-c:a", "copy",
		"-preset", "veryfast",
		outputPath,
	}
	err := runFFmpegWithProgress(ctx, totalDur, func(pct int) {
		c.notifier.OnProgress(taskID, StepBurn, pct)
	}, func(line string) {
		c.logEvent(taskID, "info", StepBurn, "ffmpeg：%s", line)
	}, args...)
	if err != nil {
		return fmt.Errorf("烧录画面字幕失败: %w", err)
	}
	return nil
}

// burnHardWithDub 将字幕直接烧录进画面并混合配音（硬字幕，需重新编码画面）
func (c *Core) burnHardWithDub(ctx context.Context, ffmpeg, videoPath, transSRT, srcSRT, dubAudio, outputPath, taskID string) error {
	var filterParts []string

	filterParts = append(filterParts, fmt.Sprintf(
		"subtitles=%s:force_style='FontSize=%d,FontName=%s,"+
			"PrimaryColour=%s,OutlineColour=%s,OutlineWidth=%d,"+
			"Alignment=2,MarginV=%d,BorderStyle=1'",
		escapeFFmpegSubtitlesPath(transSRT),
		subTransFontSize, subFontName,
		subTransColor, subOutlineColor, subOutlineWidth,
		subTransMarginV,
	))

	if srcSRT != "" {
		if _, err := os.Stat(srcSRT); err == nil {
			filterParts = append(filterParts, fmt.Sprintf(
				"subtitles=%s:force_style='FontSize=%d,FontName=%s,"+
					"PrimaryColour=%s,OutlineColour=%s,OutlineWidth=%d,"+
					"Alignment=2,MarginV=%d,BorderStyle=1'",
				escapeFFmpegSubtitlesPath(srcSRT),
				subSrcFontSize, subFontName,
				subSrcColor, subOutlineColor, subOutlineWidth,
				subSrcMarginV,
			))
		}
	}

	vf := strings.Join(filterParts, ",")
	totalDur := probeMediaDuration(ffmpeg, videoPath)

	args := []string{
		ffmpeg, "-hide_banner", "-y", "-i", videoPath, "-i", dubAudio,
		"-filter_complex",
		fmt.Sprintf("[0:v]%s[v];[0:a]volume=0.15[bg];[bg][1:a]amix=inputs=2:duration=first:dropout_transition=3[a]", vf),
		"-map", "[v]", "-map", "[a]",
		"-c:a", "aac", "-b:a", "128k",
		"-preset", "veryfast",
		outputPath,
	}
	err := runFFmpegWithProgress(ctx, totalDur, func(pct int) {
		c.notifier.OnProgress(taskID, StepBurn, pct)
	}, func(line string) {
		c.logEvent(taskID, "info", StepBurn, "ffmpeg：%s", line)
	}, args...)
	if err != nil {
		return fmt.Errorf("合成画面硬字幕配音视频失败: %w", err)
	}
	return nil
}
