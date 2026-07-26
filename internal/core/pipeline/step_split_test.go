package pipeline

import (
	"testing"
)

func TestEndsWithPunct(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Hello.", true},
		{"Hello!", true},
		{"Hello?", true},
		{"Hello", false},
		{"Hello,", false},
		{"好", true}, // 汉字被视为句末
		{"", false},
	}

	for _, tt := range tests {
		got := endsWithPunct(tt.input)
		if got != tt.want {
			t.Errorf("endsWithPunct(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSplitByConnectors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		minCount int // 至少分出几句
	}{
		{
			name:     "short sentences stay merged under 20 chars",
			input:    "Hello world. Test.",
			minCount: 1,
		},
		{
			name:     "short text stays as one",
			input:    "Hello world",
			minCount: 1,
		},
		{
			name:     "long sentences with periods split",
			input:    "This is a fairly long first sentence here. And this is another long second sentence right here.",
			minCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitByConnectors(tt.input)
			if len(got) < tt.minCount {
				t.Errorf("splitByConnectors(%q) = %d sentences, want >= %d\ngot: %v",
					tt.input, len(got), tt.minCount, got)
			}
			for i, s := range got {
				if s == "" {
					t.Errorf("sentence[%d] is empty", i)
				}
			}
		})
	}
}

func TestSplitByConnectors_NoEmptySentences(t *testing.T) {
	input := "This is the first sentence. And this is the second one. But wait there is more."
	sentences := splitByConnectors(input)
	for i, s := range sentences {
		if s == "" {
			t.Errorf("sentence[%d] is empty", i)
		}
	}
	if len(sentences) == 0 {
		t.Error("splitByConnectors returned no sentences")
	}
}
