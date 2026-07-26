package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 翻译分块大小（每次发给 LLM 的句子数量）
const translateChunkSize = 8

// runTranslate 翻译步骤
func (c *Core) runTranslate(ctx context.Context, job Job) error {
	splitFile := filepath.Join(job.OutputDir, "split.txt")
	data, err := os.ReadFile(splitFile)
	if err != nil {
		return fmt.Errorf("读取分句文件失败: %w", err)
	}

	sentences := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(sentences) == 0 {
		return fmt.Errorf("分句结果为空")
	}

	// 获取翻译锁（独占 LLM 资源）
	c.llmMu.Lock()
	defer c.llmMu.Unlock()

	c.notifier.OnLog(job.TaskID, fmt.Sprintf("翻译 %d 句到 %s", len(sentences), job.TargetLang))

	// 分块翻译
	var translated []string
	for i := 0; i < len(sentences); i += translateChunkSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := i + translateChunkSize
		if end > len(sentences) {
			end = len(sentences)
		}
		chunk := sentences[i:end]

		result, err := c.llm.Translate(ctx, chunk, job.TargetLang)
		if err != nil {
			return fmt.Errorf("翻译第 %d-%d 句失败: %w", i+1, end, err)
		}
		translated = append(translated, result...)

		progress := (end * 100) / len(sentences)
		c.notifier.OnProgress(job.TaskID, StepTranslate, progress)
	}

	// 写入翻译结果
	transFile := filepath.Join(job.OutputDir, "trans.txt")
	if err := os.WriteFile(transFile, []byte(strings.Join(translated, "\n")), 0o644); err != nil {
		return fmt.Errorf("写入翻译结果失败: %w", err)
	}

	// 生成翻译字幕文件 (trans.srt)
	srcSRT := filepath.Join(job.OutputDir, "src.srt")
	transSRT := filepath.Join(job.OutputDir, "trans.srt")
	if err := c.buildTransSRT(srcSRT, translated, transSRT); err != nil {
		return fmt.Errorf("生成翻译字幕失败: %w", err)
	}

	c.notifier.OnProgress(job.TaskID, StepTranslate, 100)
	return nil
}

// buildTransSRT 基于原始字幕的时间轴生成翻译字幕
func (c *Core) buildTransSRT(srcSRTPath string, translated []string, outputPath string) error {
	data, err := os.ReadFile(srcSRTPath)
	if err != nil {
		return err
	}

	entries := parseSRT(string(data))
	if len(entries) == 0 {
		return fmt.Errorf("原始字幕为空")
	}

	// 将翻译结果对应到时间轴
	var sb strings.Builder
	for i, entry := range entries {
		transText := entry.Text
		if i < len(translated) {
			transText = translated[i]
		}
		sb.WriteString(fmt.Sprintf("%d\n", i+1))
		sb.WriteString(fmt.Sprintf("%s --> %s\n", entry.Start, entry.End))
		sb.WriteString(transText + "\n\n")
	}

	return os.WriteFile(outputPath, []byte(sb.String()), 0o644)
}
