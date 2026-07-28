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

	switch job.Mode {
	case ModeSubtitle:
		outputVideo := filepath.Join(job.OutputDir, baseName+".sub.mp4")
		if err := c.burnSubtitle(ctx, ffmpeg, job.InputPath, srcSRT, "", outputVideo, job.TaskID); err != nil {
			return err
		}

	case ModeTranslate:
		outputVideo := filepath.Join(job.OutputDir, baseName+".trans.mp4")
		if err := c.burnSubtitle(ctx, ffmpeg, job.InputPath, transSRT, srcSRT, outputVideo, job.TaskID); err != nil {
			return err
		}

	case ModeDub:
		outputVideo := filepath.Join(job.OutputDir, baseName+".final.mp4")
		if err := c.burnWithDub(ctx, ffmpeg, videoSrc, transSRT, srcSRT, dubAudio, outputVideo, job.TaskID); err != nil {
			return err
		}
	}

	c.notifier.OnProgress(job.TaskID, StepBurn, 100)
	return nil
}

// burnSubtitle 烧录字幕（双语或单语），带 ffmpeg 进度回调
func (c *Core) burnSubtitle(ctx context.Context, ffmpeg, videoPath, transSRT, srcSRT, outputPath string, taskID string) error {
	var filterParts []string

	if _, err := os.Stat(transSRT); err == nil {
		filterParts = append(filterParts, fmt.Sprintf(
			"subtitles=%s:force_style='FontSize=%d,FontName=%s,"+
				"PrimaryColour=%s,OutlineColour=%s,OutlineWidth=%d,"+
				"Alignment=2,MarginV=%d,BorderStyle=1'",
			escapeFFmpegPath(transSRT),
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
				escapeFFmpegPath(srcSRT),
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

	args := []string{ffmpeg, "-y", "-i", videoPath, "-vf", vf, "-c:a", "copy", outputPath}
	err := runFFmpegWithProgress(ctx, totalDur, func(pct int) {
		c.notifier.OnProgress(taskID, StepBurn, pct)
	}, func(line string) {
		c.logEvent(taskID, "info", StepBurn, "ffmpeg：%s", line)
	}, args...)
	if err != nil {
		return fmt.Errorf("烧录字幕失败: %w", err)
	}
	return nil
}

// escapeFFmpegPath 转义 ffmpeg subtitles 滤镜中路径的特殊字符
func escapeFFmpegPath(path string) string {
	r := strings.NewReplacer(":", "\\:", "'", "\\'", "[", "\\[", "]", "\\]")
	return r.Replace(path)
}

// burnWithDub 烧录字幕 + 混合配音音频，带 ffmpeg 进度回调
func (c *Core) burnWithDub(ctx context.Context, ffmpeg, videoPath, transSRT, srcSRT, dubAudio, outputPath, taskID string) error {
	var filterParts []string

	filterParts = append(filterParts, fmt.Sprintf(
		"subtitles=%s:force_style='FontSize=%d,FontName=%s,"+
			"PrimaryColour=%s,OutlineColour=%s,OutlineWidth=%d,"+
			"Alignment=2,MarginV=%d,BorderStyle=1'",
		escapeFFmpegPath(transSRT),
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
				escapeFFmpegPath(srcSRT),
				subSrcFontSize, subFontName,
				subSrcColor, subOutlineColor, subOutlineWidth,
				subSrcMarginV,
			))
		}
	}

	vf := strings.Join(filterParts, ",")
	totalDur := probeMediaDuration(ffmpeg, videoPath)

	args := []string{
		ffmpeg, "-y", "-i", videoPath, "-i", dubAudio,
		"-filter_complex",
		fmt.Sprintf("[0:v]%s[v];[0:a]volume=0.15[bg];[bg][1:a]amix=inputs=2:duration=first:dropout_transition=3[a]", vf),
		"-map", "[v]", "-map", "[a]",
		"-c:a", "aac", "-b:a", "128k",
		outputPath,
	}
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
