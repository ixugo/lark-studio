package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

	switch job.Mode {
	case ModeSubtitle:
		// 仅烧录原文字幕
		outputVideo := filepath.Join(job.OutputDir, baseName+".sub.mp4")
		if err := c.burnSubtitle(ctx, ffmpeg, job.InputPath, srcSRT, "", outputVideo); err != nil {
			return err
		}

	case ModeTranslate:
		// 烧录双语字幕（中上英下）
		outputVideo := filepath.Join(job.OutputDir, baseName+".trans.mp4")
		if err := c.burnSubtitle(ctx, ffmpeg, job.InputPath, transSRT, srcSRT, outputVideo); err != nil {
			return err
		}

	case ModeDub:
		// 烧录双语字幕 + 混合配音
		outputVideo := filepath.Join(job.OutputDir, baseName+".final.mp4")
		if err := c.burnWithDub(ctx, ffmpeg, job.InputPath, transSRT, srcSRT, dubAudio, outputVideo); err != nil {
			return err
		}
	}

	c.notifier.OnProgress(job.TaskID, StepBurn, 100)
	return nil
}

// burnSubtitle 烧录字幕（双语或单语）
// transSRT: 翻译字幕（中文），srcSRT: 原文字幕（英文），srcSRT 为空则仅烧录 transSRT
// 样式对齐 VideoLingo: BorderStyle=1（描边无色块），OutlineWidth=1（1像素黑色描边）
func (c *Core) burnSubtitle(ctx context.Context, ffmpeg, videoPath, transSRT, srcSRT, outputPath string) error {
	var filterParts []string

	if _, err := os.Stat(transSRT); err == nil {
		filterParts = append(filterParts, fmt.Sprintf(
			"subtitles=%s:force_style='FontSize=%d,FontName=%s,"+
				"PrimaryColour=%s,OutlineColour=%s,OutlineWidth=%d,"+
				"Alignment=2,MarginV=%d,BorderStyle=1'",
			transSRT,
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
				srcSRT,
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
	cmd := exec.CommandContext(ctx, ffmpeg,
		"-y", "-i", videoPath,
		"-vf", vf,
		"-c:a", "copy",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("烧录字幕失败: %s, output: %s", err, string(output))
	}
	return nil
}

// burnWithDub 烧录字幕 + 混合配音音频
func (c *Core) burnWithDub(ctx context.Context, ffmpeg, videoPath, transSRT, srcSRT, dubAudio, outputPath string) error {
	var filterParts []string

	filterParts = append(filterParts, fmt.Sprintf(
		"subtitles=%s:force_style='FontSize=%d,FontName=%s,"+
			"PrimaryColour=%s,OutlineColour=%s,OutlineWidth=%d,"+
			"Alignment=2,MarginV=%d,BorderStyle=1'",
		transSRT,
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
				srcSRT,
				subSrcFontSize, subFontName,
				subSrcColor, subOutlineColor, subOutlineWidth,
				subSrcMarginV,
			))
		}
	}

	vf := strings.Join(filterParts, ",")

	// 混合原视频音频(降到15%) + 配音音频
	cmd := exec.CommandContext(ctx, ffmpeg,
		"-y", "-i", videoPath, "-i", dubAudio,
		"-filter_complex",
		fmt.Sprintf("[0:v]%s[v];[0:a]volume=0.15[bg];[bg][1:a]amix=inputs=2:duration=first:dropout_transition=3[a]", vf),
		"-map", "[v]", "-map", "[a]",
		"-c:a", "aac", "-b:a", "128k",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("合成配音视频失败: %s, output: %s", err, string(output))
	}
	return nil
}
