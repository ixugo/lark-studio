package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// runSplit 分句步骤：将 whisper 输出的 SRT 做进一步语义分割
// 1. 规则分句（连接词分割）
// 2. LLM 语义精调
func (c *Core) runSplit(ctx context.Context, job Job) error {
	srcSRT := filepath.Join(job.OutputDir, "src.srt")
	data, err := os.ReadFile(srcSRT)
	if err != nil {
		return fmt.Errorf("读取字幕失败: %w", err)
	}

	// 解析 SRT 提取纯文本
	entries := parseSRT(string(data))
	if len(entries) == 0 {
		return fmt.Errorf("字幕文件为空")
	}

	// 合并为完整文本用于分句
	var fullText strings.Builder
	for _, e := range entries {
		fullText.WriteString(e.Text)
		fullText.WriteString(" ")
	}

	// 规则分句
	sentences := splitByConnectors(fullText.String())
	c.notifier.OnProgress(job.TaskID, StepSplit, 40)

	// LLM 语义分割（获取锁）
	c.llmMu.Lock()
	defer c.llmMu.Unlock()

	refined, err := c.llm.SplitSentences(ctx, strings.Join(sentences, "\n"), "en")
	if err != nil {
		c.notifier.OnLog(job.TaskID, fmt.Sprintf("LLM 分句失败，使用规则分句结果: %v", err))
		refined = sentences
	}

	// 写入分句结果
	splitFile := filepath.Join(job.OutputDir, "split.txt")
	if err := os.WriteFile(splitFile, []byte(strings.Join(refined, "\n")), 0o644); err != nil {
		return fmt.Errorf("写入分句结果失败: %w", err)
	}

	c.notifier.OnProgress(job.TaskID, StepSplit, 100)
	return nil
}

// splitByConnectors 基于连接词的规则分句
func splitByConnectors(text string) []string {
	connectors := []string{" and ", " but ", " when ", " where ", " that ", " because ", " so ", " or ", " if ", " while "}

	words := strings.Fields(text)
	var sentences []string
	var current strings.Builder

	for i, word := range words {
		current.WriteString(word)
		if i < len(words)-1 {
			current.WriteString(" ")
		}

		// 句末标点
		if endsWithPunct(word) && current.Len() > 20 {
			sentences = append(sentences, strings.TrimSpace(current.String()))
			current.Reset()
			continue
		}

		// 连接词分割（当前句子已有一定长度）
		if current.Len() > 60 {
			rest := strings.Join(words[i+1:], " ")
			for _, conn := range connectors {
				if strings.HasPrefix(" "+strings.ToLower(rest), conn) {
					sentences = append(sentences, strings.TrimSpace(current.String()))
					current.Reset()
					break
				}
			}
		}
	}

	if current.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(current.String()))
	}

	return sentences
}

func endsWithPunct(s string) bool {
	if len(s) == 0 {
		return false
	}
	runes := []rune(s)
	r := runes[len(runes)-1]
	return r == '.' || r == '!' || r == '?' || unicode.Is(unicode.Han, r)
}
