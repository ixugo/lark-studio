package pipeline

import (
	"math"
	"testing"
)

func TestParseSRTTime(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"00:00:00,000", 0},
		{"00:00:01,000", 1},
		{"00:01:00,000", 60},
		{"01:00:00,000", 3600},
		{"00:01:23,456", 83.456},
		{"01:02:03,500", 3723.5},
	}

	for _, tt := range tests {
		got := parseSRTTime(tt.input)
		if math.Abs(got-tt.want) > 0.001 {
			t.Errorf("parseSRTTime(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestParseSRTTime_Invalid(t *testing.T) {
	tests := []string{"", "invalid", "00:00", "abc:def:ghi,jkl"}
	for _, input := range tests {
		got := parseSRTTime(input)
		if got != 0 {
			t.Errorf("parseSRTTime(%q) = %f, want 0", input, got)
		}
	}
}

func TestParseSRT(t *testing.T) {
	content := `1
00:00:01,000 --> 00:00:05,000
Hello, world!

2
00:00:06,500 --> 00:00:10,200
This is a test subtitle.

3
00:00:11,000 --> 00:00:15,500
Multiple lines
in one entry
`

	entries := parseSRT(content)
	if len(entries) != 3 {
		t.Fatalf("parseSRT got %d entries, want 3", len(entries))
	}

	// 第一条
	if entries[0].Index != 1 {
		t.Errorf("entry[0].Index = %d, want 1", entries[0].Index)
	}
	if entries[0].Start != "00:00:01,000" {
		t.Errorf("entry[0].Start = %q, want %q", entries[0].Start, "00:00:01,000")
	}
	if entries[0].End != "00:00:05,000" {
		t.Errorf("entry[0].End = %q, want %q", entries[0].End, "00:00:05,000")
	}
	if entries[0].Text != "Hello, world!" {
		t.Errorf("entry[0].Text = %q, want %q", entries[0].Text, "Hello, world!")
	}
	if math.Abs(entries[0].StartSec-1.0) > 0.001 {
		t.Errorf("entry[0].StartSec = %f, want 1.0", entries[0].StartSec)
	}

	// 第三条：多行文本合并为单行（空格连接）
	if entries[2].Text != "Multiple lines in one entry" {
		t.Errorf("entry[2].Text = %q, want %q", entries[2].Text, "Multiple lines in one entry")
	}
}

func TestParseSRT_Empty(t *testing.T) {
	entries := parseSRT("")
	if len(entries) != 0 {
		t.Errorf("parseSRT empty got %d entries, want 0", len(entries))
	}
}

func TestParseSRT_WindowsLineEndings(t *testing.T) {
	content := "1\r\n00:00:01,000 --> 00:00:05,000\r\nHello\r\n\r\n2\r\n00:00:06,000 --> 00:00:10,000\r\nWorld\r\n"
	entries := parseSRT(content)
	if len(entries) != 2 {
		t.Fatalf("parseSRT CRLF got %d entries, want 2", len(entries))
	}
	if entries[0].Text != "Hello" {
		t.Errorf("entry[0].Text = %q, want %q", entries[0].Text, "Hello")
	}
}

func TestSplitSRTBlocks(t *testing.T) {
	content := "block1\n\nblock2\n\n\nblock3"
	blocks := splitSRTBlocks(content)
	if len(blocks) != 3 {
		t.Errorf("splitSRTBlocks got %d blocks, want 3", len(blocks))
	}
}
