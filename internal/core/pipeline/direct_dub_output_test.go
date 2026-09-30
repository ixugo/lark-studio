package pipeline

import (
	"slices"
	"testing"
)

func TestDirectDubAlwaysProducesVideo(t *testing.T) {
	for _, output := range []string{"soft", "none", "file", "burn"} {
		core := Core{cfg: Config{SubtitleOutput: output}}
		if steps := core.buildSteps(Job{Mode: ModeDirectDub}); !slices.Contains(steps, StepBurn) {
			t.Fatalf("%s 跳过了原文配音成片: %v", output, steps)
		}
	}
}
