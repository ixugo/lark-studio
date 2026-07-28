//go:build integration

package app

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
)

// TestPipelineWhisperObservability 验证真实听写会写入单调总进度和结构化历史日志。
func TestPipelineWhisperObservability(t *testing.T) {
	model := os.Getenv("VDUB_WHISPER_MODEL")
	video := os.Getenv("VDUB_WHISPER_VIDEO")
	if model == "" || video == "" {
		t.Skip("需要 VDUB_WHISPER_MODEL 和 VDUB_WHISPER_VIDEO")
	}
	taskCore := newNotifierTestCore(t)
	outputDir := t.TempDir()
	created, err := taskCore.CreateTask(context.Background(), &task.CreateTaskInput{
		InputPath: video, OutputDir: outputDir, Mode: pipeline.ModeSubtitle,
	})
	if err != nil {
		t.Fatal(err)
	}

	cfg := whisperIntegrationConfig(model)
	notifier := newDBNotifier(taskCore, nil, cfg)
	core := NewPipelineCore(cfg, pipeline.WithNotifier(notifier))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := core.Run(ctx, pipeline.Job{
		TaskID: created.ID, InputPath: video, OutputDir: outputDir, Mode: pipeline.ModeSubtitle,
	}); err != nil {
		t.Fatal(err)
	}
	assertWhisperObservability(t, taskCore, created.ID)
}

// whisperIntegrationConfig 创建只输出字幕文件的真实听写配置。
func whisperIntegrationConfig(model string) *conf.Bootstrap {
	cfg := conf.DefaultConfig()
	cfg.Pipeline.WhisperMode = "whisper-cpp"
	cfg.Pipeline.WhisperBin = "whisper-cli"
	cfg.Pipeline.WhisperModel = model
	cfg.Pipeline.SubtitleOutput = "file"
	return &cfg
}

// assertWhisperObservability 检查进度完成且历史日志含真实听写结果。
func assertWhisperObservability(t *testing.T, core task.Core, taskID string) {
	t.Helper()
	item := getNotifierTask(t, core, taskID)
	if item.Progress != 100 || item.StepProgress != 100 {
		t.Fatalf("进度未完成：total=%d step=%d", item.Progress, item.StepProgress)
	}
	logs, err := core.ListRecentTaskLogs(context.Background(), taskID)
	if err != nil {
		t.Fatal(err)
	}
	for _, log := range logs {
		if strings.Contains(log.Message, "听写完成") {
			return
		}
	}
	t.Fatalf("未找到听写完成日志，共 %d 行", len(logs))
}
