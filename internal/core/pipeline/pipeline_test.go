package pipeline

import (
	"testing"
)

func TestBuildSteps(t *testing.T) {
	c := &Core{}

	tests := []struct {
		job  Job
		want []string
	}{
		{Job{Mode: ModeSubtitle, SubtitleOutput: "burn"}, []string{StepWhisper, StepBurn}},
		{Job{Mode: ModeTranslate, Translator: "openai", SubtitleOutput: "burn"}, []string{StepWhisper, StepSplit, StepTranslate, StepBurn}},
		{Job{Mode: ModeDub, Translator: "openai", SubtitleOutput: "burn"}, []string{StepWhisper, StepSplit, StepTranslate, StepTTS, StepMerge, StepBurn}},
		{Job{Mode: ModeTranslate, Translator: "openai", SubtitleOutput: "file"}, []string{StepWhisper, StepSplit, StepTranslate}},
		{Job{Mode: ModeDub, Translator: "openai", SubtitleOutput: "file"}, []string{StepWhisper, StepSplit, StepTranslate, StepTTS, StepMerge, StepBurn}},
		{Job{Mode: ModeTranslate, Translator: "bing", SubtitleOutput: "burn"}, []string{StepWhisper, StepTranslate, StepBurn}},
		{Job{Mode: ModeDub, Translator: "bing", SubtitleOutput: "burn"}, []string{StepWhisper, StepTranslate, StepTTS, StepMerge, StepBurn}},
		{Job{Mode: ModeTranslate, Translator: "deeplx", SubtitleOutput: "file"}, []string{StepWhisper, StepTranslate}},
		{Job{Mode: 0, SubtitleOutput: "burn"}, []string{StepWhisper, StepBurn}},
	}

	for _, tt := range tests {
		got := c.buildSteps(tt.job)
		if len(got) != len(tt.want) {
			t.Errorf("buildSteps(%+v) len = %d, want %d\ngot: %v", tt.job, len(got), len(tt.want), got)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("buildSteps(%+v)[%d] = %q, want %q", tt.job, i, got[i], tt.want[i])
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
