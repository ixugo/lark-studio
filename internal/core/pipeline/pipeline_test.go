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
		{Job{Mode: ModeTranslate, Translator: "openai", SubtitleOutput: "burn"}, []string{StepWhisper, StepTranslate, StepBurn}},
		{Job{Mode: ModeDub, Translator: "openai", SubtitleOutput: "burn"}, []string{StepWhisper, StepTranslate, StepTTS, StepMerge, StepBurn}},
		{Job{Mode: ModeTranslate, Translator: "openai", SubtitleOutput: "file"}, []string{StepWhisper, StepTranslate}},
		{Job{Mode: ModeDub, Translator: "openai", SubtitleOutput: "file"}, []string{StepWhisper, StepTranslate, StepTTS, StepMerge, StepBurn}},
		{Job{Mode: ModeTranslate, Translator: "bing", SubtitleOutput: "burn"}, []string{StepWhisper, StepTranslate, StepBurn}},
		{Job{Mode: ModeDub, Translator: "bing", SubtitleOutput: "burn"}, []string{StepWhisper, StepTranslate, StepTTS, StepMerge, StepBurn}},
		{Job{Mode: ModeTranslate, Translator: "deeplx", SubtitleOutput: "file"}, []string{StepWhisper, StepTranslate}},
		{Job{Mode: ModeDubOnly}, []string{StepTTS, StepMerge}},
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

// TestBuildStepsWithConfiguredOpenAI 保证端点配置完整时才加入语义分句步骤。
func TestBuildStepsWithConfiguredOpenAI(t *testing.T) {
	c := NewCore(Config{SemanticSplitReady: true}, nil, nil, nil)
	steps := c.buildSteps(Job{Mode: ModeTranslate, Translator: "openai", SubtitleOutput: "burn"})
	want := []string{StepWhisper, StepSplit, StepTranslate, StepBurn}
	if len(steps) != len(want) {
		t.Fatalf("步骤数量 = %d，期望 %d：%v", len(steps), len(want), steps)
	}
	for i := range want {
		if steps[i] != want[i] {
			t.Fatalf("步骤[%d] = %q，期望 %q", i, steps[i], want[i])
		}
	}
}

// TestSetTranslationClientRefreshesSplitAvailability 保证配置保存后新任务读取最新分句开关。
func TestSetTranslationClientRefreshesSplitAvailability(t *testing.T) {
	c := NewCore(Config{}, nil, nil, nil)
	job := Job{Mode: ModeTranslate, Translator: "openai", SubtitleOutput: "burn"}
	c.SetTranslationClient(nil, true)
	if got := c.buildSteps(job); len(got) < 2 || got[1] != StepSplit {
		t.Fatalf("启用语义分句后步骤 = %v", got)
	}
	c.SetTranslationClient(nil, false)
	if got := c.buildSteps(job); len(got) > 1 && got[1] == StepSplit {
		t.Fatalf("关闭语义分句后仍包含分句：%v", got)
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

func TestIsFFmpegProgressLine(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"frame=  123 fps= 24 q=28.0 size=    1024kB time=00:00:05.12 bitrate=1638.4kbits/s speed=1.02x", true},
		{"  frame= 456 fps= 30", true},
		{"size=     512kB time=00:00:02.50 bitrate=1677.7kbits/s", true},
		{"[subtitles @ 0x1234567] Shaper: FriBidi 1.0.12 (SIMPLE)", false},
		{"Error opening input file: No such file or directory", false},
		{"Stream #0:0: Video: h264", false},
	}

	for _, tt := range tests {
		got := isFFmpegProgressLine(tt.line)
		if got != tt.want {
			t.Errorf("isFFmpegProgressLine(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}
