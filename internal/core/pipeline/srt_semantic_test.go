package pipeline

import (
	"strings"
	"testing"
)

func TestSemanticBreak_Short(t *testing.T) {
	got := semanticBreak("短句不换行")
	if strings.Contains(got, "\n") {
		t.Errorf("短句不应换行: %q", got)
	}
}

func TestSemanticBreak_CJKAtPunctuation(t *testing.T) {
	input := "这是一个较长的中文句子，它包含了逗号分隔的两个子句"
	got := semanticBreak(input)
	if !strings.Contains(got, "\n") {
		t.Errorf("超限 CJK 应断行: %q", got)
	}
	lines := strings.Split(got, "\n")
	for _, l := range lines {
		if len([]rune(l)) > maxCJKPerLine+5 {
			t.Errorf("断行后仍超限: %q (%d runes)", l, len([]rune(l)))
		}
	}
}

func TestSemanticBreak_Latin(t *testing.T) {
	input := "This is a relatively long English sentence that should be broken at a space or conjunction when it exceeds the limit"
	got := semanticBreak(input)
	if len([]rune(input)) > maxLatinPerLine && !strings.Contains(got, "\n") {
		t.Errorf("超限 Latin 应断行: %q", got)
	}
}

func TestSemanticBreak_EmptyAndSingleChar(t *testing.T) {
	if got := semanticBreak(""); got != "" {
		t.Errorf("空串应返回空: %q", got)
	}
	if got := semanticBreak("A"); got != "A" {
		t.Errorf("单字符不应变化: %q", got)
	}
}

func TestSemanticBreak_NoPunctuation(t *testing.T) {
	input := strings.Repeat("字", maxCJKPerLine+10)
	got := semanticBreak(input)
	if !strings.Contains(got, "\n") {
		t.Errorf("超限无标点仍应在中间断行: %q", got)
	}
}
