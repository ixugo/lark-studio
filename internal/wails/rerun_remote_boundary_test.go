package wails

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
)

func TestWhisperRerunValidatesRemoteModelBeforeClearingOldSubtitle(t *testing.T) {
	svc, inputs, _ := batchService(t)
	item, err := svc.CreateTask(task.CreateTaskInput{InputPath: inputs[0], Mode: pipeline.ModeSubtitle})
	if err != nil {
		t.Fatal(err)
	}
	srt := filepath.Join(item.OutputDir, "src.srt")
	const previous = "1\n00:00:00,000 --> 00:00:01,000\nprevious\n"
	if err := os.WriteFile(srt, []byte(previous), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := svc.GetTask(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	svc.bc.Pipeline.ASRModel = "unknown-new-model"
	err = svc.RerunTaskWithRecipe(item.ID, RerunTaskOptions{FromStep: pipeline.StepWhisper})
	if err == nil || !strings.Contains(err.Error(), "语音识别") {
		t.Fatalf("旧输出字幕不应让新听写跳过远程校验: %v", err)
	}
	data, readErr := os.ReadFile(srt)
	if readErr != nil || string(data) != previous {
		t.Fatalf("校验失败清除了旧字幕: %q %v", data, readErr)
	}
	after, err := svc.GetTask(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("校验失败改变任务状态: before=%+v after=%+v", before, after)
	}
}
