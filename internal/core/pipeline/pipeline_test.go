package pipeline

import (
	"testing"
)

func TestBuildSteps(t *testing.T) {
	c := &Core{}

	tests := []struct {
		mode int
		want []string
	}{
		{ModeSubtitle, []string{StepWhisper, StepBurn}},
		{ModeTranslate, []string{StepWhisper, StepSplit, StepTranslate, StepBurn}},
		{ModeDub, []string{StepWhisper, StepSplit, StepTranslate, StepTTS, StepMerge, StepBurn}},
		{0, []string{StepWhisper, StepBurn}},  // 未知模式退化为 subtitle
		{99, []string{StepWhisper, StepBurn}}, // 未知模式退化为 subtitle
	}

	for _, tt := range tests {
		got := c.buildSteps(tt.mode)
		if len(got) != len(tt.want) {
			t.Errorf("buildSteps(%d) len = %d, want %d\ngot: %v", tt.mode, len(got), len(tt.want), got)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("buildSteps(%d)[%d] = %q, want %q", tt.mode, i, got[i], tt.want[i])
			}
		}
	}
}

func TestStepConstants(t *testing.T) {
	steps := []string{StepWhisper, StepSplit, StepTranslate, StepTTS, StepMerge, StepBurn}
	seen := make(map[string]bool)
	for _, s := range steps {
		if s == "" {
			t.Error("step constant is empty")
		}
		if seen[s] {
			t.Errorf("duplicate step constant: %q", s)
		}
		seen[s] = true
	}
}

func TestModeConstants(t *testing.T) {
	if ModeSubtitle == ModeTranslate || ModeTranslate == ModeDub || ModeSubtitle == ModeDub {
		t.Error("mode constants must be distinct")
	}
}
