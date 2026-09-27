package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runTTS 文本转语音步骤
// ModeDub 模式下，若翻译步骤已通过流水线完成 TTS，则自动跳过
func (c *Core) runTTS(ctx context.Context, job Job) error {
	if job.Mode == ModeDub {
		audioDir := filepath.Join(job.OutputDir, "audio_segs")
		if info, err := os.Stat(audioDir); err == nil && info.IsDir() {
			entries, _ := os.ReadDir(audioDir)
			wavCount := 0
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".wav") {
					wavCount++
				}
			}
			if wavCount > 0 {
				c.logEvent(job.TaskID, "success", StepTTS, "配音已完成：%d 段", wavCount)
				c.notifier.OnProgress(job.TaskID, StepTTS, 100)
				return nil
			}
		}
	}

	transFile := filepath.Join(job.OutputDir, "trans.txt")
	data, err := os.ReadFile(transFile)
	if err != nil {
		// 1. 若经过听写转录(1-3-4流程)，src.srt 已就绪，直接以听写原文作为配音文本源
		srcSRTPath := filepath.Join(job.OutputDir, "src.srt")
		if srcSRTData, srtErr := os.ReadFile(srcSRTPath); srtErr == nil {
			entries := parseSRT(string(srcSRTData))
			var lines []string
			for _, e := range entries {
				t := strings.TrimSpace(e.Text)
				if t != "" {
					lines = append(lines, t)
				}
			}
			if len(lines) > 0 {
				data = []byte(strings.Join(lines, "\n"))
				_ = os.WriteFile(transFile, data, 0o644)
				// 若 trans.srt 亦不存在，同步复制 src.srt 为 trans.srt
				transSRTPath := filepath.Join(job.OutputDir, "trans.srt")
				if _, statErr := os.Stat(transSRTPath); statErr != nil {
					_ = os.WriteFile(transSRTPath, srcSRTData, 0o644)
				}
			}
		}

		// 2. 若仍未获取到，尝试从原始输入文件（如纯文本 .txt 小说章节或输入 .srt 字幕）中自动载入句子
		if len(data) == 0 {
			data, err = c.loadSentencesFromInput(job)
			if err != nil {
				return fmt.Errorf("读取待配音文本源文件失败: %w", err)
			}
			_ = os.WriteFile(transFile, data, 0o644)
		}
	}

	sentences := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(sentences) == 0 {
		return fmt.Errorf("翻译结果为空")
	}

	c.ttsMu.Lock()
	defer c.ttsMu.Unlock()

	audioDir := filepath.Join(job.OutputDir, "audio_segs")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	c.logEvent(job.TaskID, "info", StepTTS, "配音开始：%d 句", len(sentences))

	for i, text := range sentences {
		if err := ctx.Err(); err != nil {
			return err
		}

		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		outputPath := filepath.Join(audioDir, fmt.Sprintf("%d.wav", i))

		// 断点恢复：已存在的音频跳过
		if info, e := os.Stat(outputPath); e == nil && info.Size() > 0 {
			progress := ((i + 1) * 100) / len(sentences)
			c.notifier.OnProgress(job.TaskID, StepTTS, progress)
			continue
		}

		if err := c.synthesizeWithJob(ctx, job, text, outputPath); err != nil {
			return fmt.Errorf("TTS 第 %d 句失败: %w", i+1, err)
		}

		progress := ((i + 1) * 100) / len(sentences)
		c.notifier.OnProgress(job.TaskID, StepTTS, progress)
		c.logEvent(
			job.TaskID,
			"info",
			StepTTS,
			"配音进度 %d%%（%d/%d）",
			progress,
			i+1,
			len(sentences),
		)
	}

	c.notifier.OnProgress(job.TaskID, StepTTS, 100)
	return nil
}

// loadSentencesFromInput 当无翻译文件时，直接从用户提供的输入文件提取待朗读/配音句子。
// 支持纯文本 .txt（按非空行拆分）或 .srt 字幕文件。
func (c *Core) loadSentencesFromInput(job Job) ([]byte, error) {
	if job.InputPath == "" {
		return nil, fmt.Errorf("任务输入文件路径为空")
	}
	content, err := os.ReadFile(job.InputPath)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(job.InputPath))
	if ext == ".srt" {
		entries := parseSRT(string(content))
		if len(entries) == 0 {
			return nil, fmt.Errorf("字幕文件无有效条目: %s", job.InputPath)
		}
		// 同步写出 src.srt 供后续合并时间轴使用
		srcSRT := filepath.Join(job.OutputDir, "src.srt")
		_ = os.WriteFile(srcSRT, content, 0o644)

		var lines []string
		for _, e := range entries {
			t := strings.TrimSpace(e.Text)
			if t != "" {
				lines = append(lines, t)
			}
		}
		return []byte(strings.Join(lines, "\n")), nil
	}

	// 纯文本（如小说 .txt 文件）：按非空行或段落分句
	rawLines := strings.Split(string(content), "\n")
	var validLines []string
	for _, l := range rawLines {
		t := strings.TrimSpace(l)
		if t != "" {
			validLines = append(validLines, t)
		}
	}
	if len(validLines) == 0 {
		return nil, fmt.Errorf("文本文件内容为空: %s", job.InputPath)
	}
	return []byte(strings.Join(validLines, "\n")), nil
}
