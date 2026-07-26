package pipeline

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// runSplit 句级重分段：将 whisper 碎片合并为自然句并重分配时间轴
// 流程：读取 whisper src.srt → 构建字符-时间映射 → LLM 分句 → 时间轴对齐 → 覆写 src.srt
func (c *Core) runSplit(ctx context.Context, job Job) error {
	srcSRT := filepath.Join(job.OutputDir, "src.srt")
	data, err := os.ReadFile(srcSRT)
	if err != nil {
		return fmt.Errorf("读取字幕失败: %w", err)
	}

	entries := parseSRT(string(data))
	if len(entries) == 0 {
		return fmt.Errorf("字幕文件为空")
	}

	fullRunes, charTimes := buildCharTimes(entries)
	fullText := string(fullRunes)

	c.llmMu.Lock()
	sentences, err := c.llm.SplitSentences(ctx, fullText, "en")
	c.llmMu.Unlock()
	if err != nil {
		c.notifier.OnLog(job.TaskID, fmt.Sprintf("LLM 分句失败，保留原始字幕: %v", err))
		c.notifier.OnProgress(job.TaskID, StepSplit, 100)
		return nil
	}

	c.notifier.OnProgress(job.TaskID, StepSplit, 60)

	newEntries := alignSentences(fullRunes, charTimes, sentences)
	if len(newEntries) == 0 {
		c.notifier.OnLog(job.TaskID, "句子对齐失败，保留原始字幕")
		c.notifier.OnProgress(job.TaskID, StepSplit, 100)
		return nil
	}

	// 消除相邻条目之间的小间隙（< 1s），对齐 VideoLingo 做法
	for i := 0; i < len(newEntries)-1; i++ {
		gap := newEntries[i+1].StartSec - newEntries[i].EndSec
		if gap > 0 && gap < 1.0 {
			newEntries[i].EndSec = newEntries[i+1].StartSec
			newEntries[i].End = secToSRTTime(newEntries[i].EndSec)
		}
	}

	var sb strings.Builder
	for _, e := range newEntries {
		sb.WriteString(fmt.Sprintf("%d\n%s --> %s\n%s\n\n", e.Index, e.Start, e.End, e.Text))
	}
	if err := os.WriteFile(srcSRT, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("写入重分段字幕失败: %w", err)
	}

	c.notifier.OnLog(job.TaskID, fmt.Sprintf("重分段完成: %d 条 → %d 句", len(entries), len(newEntries)))
	c.notifier.OnProgress(job.TaskID, StepSplit, 100)
	return nil
}

// buildCharTimes 从 whisper SRT 条目构建字符级时间映射
// 每个字符按在原条目中的比例插值出对应时间点
func buildCharTimes(entries []srtEntry) ([]rune, []float64) {
	var fullRunes []rune
	var times []float64

	for i, e := range entries {
		runes := []rune(e.Text)
		n := len(runes)
		for j, r := range runes {
			fullRunes = append(fullRunes, r)
			t := e.StartSec
			if n > 1 {
				t += (e.EndSec - e.StartSec) * float64(j) / float64(n-1)
			}
			times = append(times, t)
		}

		if i < len(entries)-1 {
			fullRunes = append(fullRunes, ' ')
			times = append(times, e.EndSec)
		}
	}

	return fullRunes, times
}

// alignSentences 将 LLM 分出的自然句映射回原始时间轴
func alignSentences(fullRunes []rune, charTimes []float64, sentences []string) []srtEntry {
	pos := 0
	var result []srtEntry

	for _, sent := range sentences {
		sentRunes := []rune(strings.TrimSpace(sent))
		if len(sentRunes) == 0 {
			continue
		}

		idx := findRuneSubstring(fullRunes, sentRunes, pos)
		if idx < 0 {
			continue
		}

		endIdx := idx + len(sentRunes) - 1
		if endIdx >= len(charTimes) {
			endIdx = len(charTimes) - 1
		}

		result = append(result, srtEntry{
			Index:    len(result) + 1,
			StartSec: charTimes[idx],
			EndSec:   charTimes[endIdx],
			Start:    secToSRTTime(charTimes[idx]),
			End:      secToSRTTime(charTimes[endIdx]),
			Text:     sent,
		})

		pos = idx + len(sentRunes)
	}

	return result
}

// findRuneSubstring 大小写不敏感地在 rune slice 中查找子串
func findRuneSubstring(haystack, needle []rune, from int) int {
	if len(needle) == 0 || from+len(needle) > len(haystack) {
		return -1
	}
	for i := from; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := range needle {
			if unicode.ToLower(haystack[i+j]) != unicode.ToLower(needle[j]) {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// secToSRTTime 秒数转 SRT 时间格式 "00:01:23,456"
func secToSRTTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	h := int(sec / 3600)
	m := int(sec/60) % 60
	s := int(sec) % 60
	ms := int(math.Round((sec - float64(int(sec))) * 1000))
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}
