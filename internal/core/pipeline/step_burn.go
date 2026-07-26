package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
func (c *Core) burnSubtitle(ctx context.Context, ffmpeg, videoPath, transSRT, srcSRT, outputPath string) error {
	var filterParts []string

	// 翻译字幕（中文/主语言，在上方，纯白无描边）
	if _, err := os.Stat(transSRT); err == nil {
		filterParts = append(filterParts, fmt.Sprintf(
			"subtitles=%s:force_style='FontSize=18,FontName=Arial Unicode MS,"+
				"PrimaryColour=&HFFFFFF,OutlineWidth=0,"+
				"Alignment=2,MarginV=50,BorderStyle=3'",
			transSRT,
		))
	}

	// 原文字幕（英文，在下方，浅灰无描边）
	if srcSRT != "" {
		if _, err := os.Stat(srcSRT); err == nil {
			filterParts = append(filterParts, fmt.Sprintf(
				"subtitles=%s:force_style='FontSize=14,FontName=Arial Unicode MS,"+
					"PrimaryColour=&HCCCCCC,OutlineWidth=0,"+
					"Alignment=2,MarginV=20,BorderStyle=3'",
				srcSRT,
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

	// 翻译字幕（中文，上方，纯白无描边）
	filterParts = append(filterParts, fmt.Sprintf(
		"subtitles=%s:force_style='FontSize=18,FontName=Arial Unicode MS,"+
			"PrimaryColour=&HFFFFFF,OutlineWidth=0,"+
			"Alignment=2,MarginV=50,BorderStyle=3'",
		transSRT,
	))

	// 原文字幕（英文，下方，浅灰无描边）
	if srcSRT != "" {
		if _, err := os.Stat(srcSRT); err == nil {
			filterParts = append(filterParts, fmt.Sprintf(
				"subtitles=%s:force_style='FontSize=14,FontName=Arial Unicode MS,"+
					"PrimaryColour=&HCCCCCC,OutlineWidth=0,"+
					"Alignment=2,MarginV=20,BorderStyle=3'",
				srcSRT,
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
