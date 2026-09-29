package pipeline

import (
	"context"
	"fmt"
	ttsadapter "github.com/ixugo/vdub/internal/adapter/tts"
	"testing"
)

func TestPipelineDoesNotRestartExhaustedEdgeRetries(t *testing.T) {
	core := &Core{}
	calls := 0
	expected := fmt.Errorf("TTS 第 230 句失败: %w", &ttsadapter.RetryExhaustedError{Err: fmt.Errorf("connection timeout")})
	err := core.runStepWithRetry(t.Context(), Job{}, StepTTS, func(context.Context, Job) error { calls++; return expected })
	if calls != 1 || err != expected {
		t.Fatalf("外层重新消耗单文件预算: calls=%d err=%v", calls, err)
	}
}
