package api

import (
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"os"
	"path/filepath"
	"testing"
)

func TestHTTPTaskRejectsMissingRecognitionConfig(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "video.mp4")
	if err := os.WriteFile(input, []byte("video"), 0600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "out")
	bc := conf.DefaultConfig()
	bc.Pipeline.WhisperModel = ""
	api := TaskAPI{conf: &bc}
	in := task.CreateTaskInput{InputPath: input, OutputDir: output, Mode: pipeline.ModeSubtitle}
	if err := api.prepareTaskInput(&in); err == nil {
		t.Fatal("缺模型仍可提交")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("已准备无效任务: %v", err)
	}
}
