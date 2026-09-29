package wails

import (
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"os"
	"path/filepath"
	"testing"
)

func TestTaskRejectsMissingASRBeforeCreatingOutput(t *testing.T) {
	for _, engine := range []string{"whisper-cpp", "openai"} {
		t.Run(engine, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "video.mp4")
			if err := os.WriteFile(input, []byte("video"), 0600); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(dir, "output")
			bc := conf.DefaultConfig()
			bc.Pipeline.WhisperMode = engine
			bc.Pipeline.WhisperModel = ""
			bc.Pipeline.ASRBaseURL = ""
			bc.Pipeline.ASRModel = ""
			service := &AppService{bc: &bc}
			in := task.CreateTaskInput{InputPath: input, OutputDir: output, Mode: pipeline.ModeSubtitle}
			if _, err := service.CreateTask(in); err == nil {
				t.Fatal("没有 ASR 配置的任务被接受")
			}
			if created, err := service.BatchCreateTasks([]string{input, input}, in); err == nil || len(created) != 0 {
				t.Fatalf("缺配置的批量任务被接受: count=%d err=%v", len(created), err)
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("错误任务已创建输出目录: %v", err)
			}
		})
	}
}
