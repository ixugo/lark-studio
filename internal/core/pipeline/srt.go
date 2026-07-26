package pipeline

import (
	"regexp"
	"strconv"
	"strings"
)

// srtEntry SRT 字幕条目
type srtEntry struct {
	Index    int
	Start    string // "00:00:01,000"
	End      string // "00:00:05,000"
	Text     string
	StartSec float64 // 秒数
	EndSec   float64 // 秒数
}

var srtTimeRe = regexp.MustCompile(`(\d{2}:\d{2}:\d{2},\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2},\d{3})`)

// parseSRT 解析 SRT 字幕文件内容
func parseSRT(content string) []srtEntry {
	var entries []srtEntry
	blocks := splitSRTBlocks(content)

	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 3 {
			continue
		}

		idx, err := strconv.Atoi(strings.TrimSpace(lines[0]))
		if err != nil {
			continue
		}

		matches := srtTimeRe.FindStringSubmatch(lines[1])
		if len(matches) < 3 {
			continue
		}

		text := strings.Join(lines[2:], " ")
		text = strings.TrimSpace(text)

		entries = append(entries, srtEntry{
			Index:    idx,
			Start:    matches[1],
			End:      matches[2],
			Text:     text,
			StartSec: parseSRTTime(matches[1]),
			EndSec:   parseSRTTime(matches[2]),
		})
	}

	return entries
}

// splitSRTBlocks 按空行分割 SRT 块
func splitSRTBlocks(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return regexp.MustCompile(`\n\n+`).Split(content, -1)
}

// parseSRTTime 解析 "00:01:23,456" 为秒数
func parseSRTTime(t string) float64 {
	t = strings.ReplaceAll(t, ",", ".")
	parts := strings.Split(t, ":")
	if len(parts) != 3 {
		return 0
	}

	h, _ := strconv.ParseFloat(parts[0], 64)
	m, _ := strconv.ParseFloat(parts[1], 64)
	s, _ := strconv.ParseFloat(parts[2], 64)
	return h*3600 + m*60 + s
}
