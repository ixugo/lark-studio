package pipeline

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
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

const (
	maxCJKPerLine   = 20 // CJK 文本每行最大字符数
	maxLatinPerLine = 45 // 拉丁文本每行最大字符数
)

// semanticBreak 按语义对长字幕文本断行
// 优先在标点、连词处换行；CJK 文本以 20 字为上限，拉丁文本以 45 字为上限
func semanticBreak(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}

	limit := maxLatinPerLine
	if hasCJK(runes) {
		limit = maxCJKPerLine
	}

	if len(runes) <= limit {
		return text
	}

	best := findBreakPoint(runes, limit)
	if best <= 0 || best >= len(runes)-1 {
		return text
	}

	line1 := strings.TrimSpace(string(runes[:best]))
	line2 := strings.TrimSpace(string(runes[best:]))
	if line1 == "" || line2 == "" {
		return text
	}
	return line1 + "\n" + line2
}

// findBreakPoint 在 runes 中 limit 附近找最佳语义断点
func findBreakPoint(runes []rune, limit int) int {
	if limit >= len(runes) {
		return -1
	}

	type candidate struct {
		pos      int
		priority int // 数字越小优先级越高
	}

	half := len(runes) / 2
	searchFrom := half - limit/2
	if searchFrom < 1 {
		searchFrom = 1
	}
	searchTo := half + limit/2
	if searchTo >= len(runes) {
		searchTo = len(runes) - 1
	}

	var best candidate
	best.pos = -1
	best.priority = 100

	for i := searchFrom; i < searchTo; i++ {
		r := runes[i]
		var pri int
		switch {
		case strings.ContainsRune("。！？!?", r):
			pri = 1
		case strings.ContainsRune("，,、；;：:", r):
			pri = 2
		case r == ' ' && i > 0 && isConjunction(runes, i):
			pri = 3
		case r == ' ':
			pri = 4
		default:
			continue
		}

		dist := abs(i - half)
		bestDist := abs(best.pos - half)
		if pri < best.priority || (pri == best.priority && dist < bestDist) {
			best = candidate{pos: i + 1, priority: pri}
		}
	}

	if best.pos > 0 {
		return best.pos
	}
	return half
}

// hasCJK 检测 rune 切片中是否包含 CJK 字符
func hasCJK(runes []rune) bool {
	for _, r := range runes {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hiragana, r) {
			return true
		}
	}
	return false
}

// isConjunction 判断空格前的单词是否为英文连词/介词，适合在此处断行
func isConjunction(runes []rune, spaceIdx int) bool {
	end := spaceIdx
	start := end - 1
	for start >= 0 && runes[start] != ' ' {
		start--
	}
	start++
	word := strings.ToLower(string(runes[start:end]))
	switch word {
	case "and", "or", "but", "that", "which", "when", "where", "because", "so", "if", "while", "for", "with", "from", "into", "about":
		return true
	}
	return false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
