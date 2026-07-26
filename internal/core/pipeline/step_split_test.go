package pipeline

import (
	"testing"
)

func TestBuildCharTimes(t *testing.T) {
	entries := []srtEntry{
		{StartSec: 0, EndSec: 3, Text: "Hello"},
		{StartSec: 3, EndSec: 6, Text: "World"},
	}

	runes, times := buildCharTimes(entries)
	text := string(runes)

	if text != "Hello World" {
		t.Errorf("fullText = %q, want %q", text, "Hello World")
	}
	if len(times) != len(runes) {
		t.Fatalf("len(times) = %d, len(runes) = %d, must match", len(times), len(runes))
	}

	// 'H' 应在 0 秒附近
	if times[0] < -0.01 || times[0] > 0.01 {
		t.Errorf("times[0] = %f, want ~0", times[0])
	}
	// 空格应在 3 秒附近（第一条目的结束时间）
	spaceIdx := 5
	if times[spaceIdx] < 2.99 || times[spaceIdx] > 3.01 {
		t.Errorf("times[space] = %f, want ~3", times[spaceIdx])
	}
}

func TestAlignSentences(t *testing.T) {
	entries := []srtEntry{
		{StartSec: 0, EndSec: 5, Text: "Hello world this is"},
		{StartSec: 5, EndSec: 10, Text: "a test sentence here"},
	}
	fullRunes, charTimes := buildCharTimes(entries)

	sentences := []string{
		"Hello world",
		"this is a test sentence here",
	}

	result := alignSentences(fullRunes, charTimes, sentences)
	if len(result) != 2 {
		t.Fatalf("got %d entries, want 2", len(result))
	}

	// 第一句应从 0 秒开始
	if result[0].StartSec > 0.5 {
		t.Errorf("first sentence start = %.1f, want ~0", result[0].StartSec)
	}
	// 第二句应在中间某处开始
	if result[1].StartSec < 2 || result[1].StartSec > 6 {
		t.Errorf("second sentence start = %.1f, want between 2~6", result[1].StartSec)
	}
}

func TestFindRuneSubstring(t *testing.T) {
	haystack := []rune("Hello World Test")
	needle := []rune("world")
	idx := findRuneSubstring(haystack, needle, 0)
	if idx != 6 {
		t.Errorf("findRuneSubstring = %d, want 6", idx)
	}

	idx = findRuneSubstring(haystack, needle, 7)
	if idx != -1 {
		t.Errorf("findRuneSubstring from 7 = %d, want -1", idx)
	}
}

func TestSecToSRTTime(t *testing.T) {
	tests := []struct {
		sec  float64
		want string
	}{
		{0, "00:00:00,000"},
		{1.5, "00:00:01,500"},
		{61.123, "00:01:01,123"},
		{3661.5, "01:01:01,500"},
	}

	for _, tt := range tests {
		got := secToSRTTime(tt.sec)
		if got != tt.want {
			t.Errorf("secToSRTTime(%f) = %q, want %q", tt.sec, got, tt.want)
		}
	}
}
