//go:build integration

package whisper

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// TestWhisperRuntime 验证真实 whisper-cli 能输出进度、日志和 SRT 文件。
func TestWhisperRuntime(t *testing.T) {
	model := os.Getenv("VDUB_WHISPER_MODEL")
	audio := os.Getenv("VDUB_WHISPER_AUDIO")
	if model == "" || audio == "" {
		t.Skip("需要 VDUB_WHISPER_MODEL 和 VDUB_WHISPER_AUDIO")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	output := filepath.Join(t.TempDir(), "result.srt")
	var maxProgress atomic.Int32
	var logCount atomic.Int32

	err := NewRunner("whisper-cli", model).Transcribe(
		ctx,
		audio,
		output,
		"auto",
		func(progress int) {
			for progress > int(maxProgress.Load()) &&
				!maxProgress.CompareAndSwap(maxProgress.Load(), int32(progress)) {
			}
		},
		func(string) {
			logCount.Add(1)
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Size() == 0 {
		t.Fatalf("SRT 未生成：%v", err)
	}
	if maxProgress.Load() != 100 || logCount.Load() == 0 {
		t.Fatalf("无实时事件：progress=%d logs=%d", maxProgress.Load(), logCount.Load())
	}
}
