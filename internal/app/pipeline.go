package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ixugo/vdub/internal/adapter/llm"
	"github.com/ixugo/vdub/internal/adapter/tts"
	"github.com/ixugo/vdub/internal/adapter/whisper"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
)

// NewPipelineCore 根据配置创建完整的流水线核心，组装所有适配器
func NewPipelineCore(bc *conf.Bootstrap, opts ...pipeline.Option) *pipeline.Core {
	cfg := pipeline.Config{
		WhisperBin:   bc.Pipeline.WhisperBin,
		WhisperModel: bc.Pipeline.WhisperModel,
		FFmpegBin:    bc.Pipeline.FFmpegBin,
	}

	var wr pipeline.WhisperRunner
	switch bc.Pipeline.WhisperMode {
	case "whisper-cpp":
		wr = whisper.NewRunner(bc.Pipeline.WhisperBin, bc.Pipeline.WhisperModel)
	default:
		wr = whisper.NewFFmpegRunner(bc.Pipeline.FFmpegBin, bc.Pipeline.WhisperModel)
	}

	lc := llm.NewClient(bc.LLM.BaseURL, bc.LLM.APIKey, bc.LLM.Model)

	var tc pipeline.TTSClient
	switch bc.TTS.Type {
	case "openai":
		tc = tts.NewOpenAITTS(bc.TTS.BaseURL, bc.TTS.APIKey, bc.TTS.Model, bc.TTS.Voice)
	default:
		tc = tts.NewEdgeTTS(bc.TTS.Voice)
	}

	return pipeline.NewCore(cfg, wr, lc, tc, opts...)
}

// NewPipelineScheduler 创建带 DB 状态回写的流水线调度器
// 返回 Scheduler 和用于优雅停机的 cleanup 函数
func NewPipelineScheduler(bc *conf.Bootstrap, taskCore task.Core) (*pipeline.Scheduler, func()) {
	notifier := &dbNotifier{taskCore: taskCore}
	pipeCore := NewPipelineCore(bc, pipeline.WithNotifier(notifier))

	onDone := func(taskID string, err error) {
		ctx := context.Background()
		status := 3
		errMsg := ""
		if err != nil {
			status = 4
			errMsg = err.Error()
			slog.Error("task failed", "task_id", taskID, "err", err)
		}
		if _, e := taskCore.UpdateTask(ctx, &task.UpdateTaskInput{
			ID: taskID, Status: status, Error: errMsg,
		}, taskID); e != nil {
			slog.Error("update task status failed", "task_id", taskID, "err", e)
		}
	}

	sched := pipeline.NewScheduler(pipeCore, onDone)
	sched.Start()
	return sched, sched.Stop
}

// dbNotifier 将流水线进度事件回写到 Task DB
type dbNotifier struct {
	taskCore task.Core
}

func (n *dbNotifier) OnProgress(taskID, step string, progress int) {
	ctx := context.Background()
	if _, err := n.taskCore.UpdateTask(ctx, &task.UpdateTaskInput{
		ID: taskID, CurrentStep: step, Progress: progress, Status: 1,
	}, taskID); err != nil {
		slog.Debug("update progress failed", "task_id", taskID, "err", err)
	}
}

func (n *dbNotifier) OnStepDone(taskID, step string) {
	slog.Info("step done", "task_id", taskID, "step", step)
}

func (n *dbNotifier) OnTaskDone(taskID string) {
	slog.Info("task done", "task_id", taskID)
}

func (n *dbNotifier) OnTaskFailed(taskID string, err error) {
	slog.Error("task failed", "task_id", taskID, "err", err)
}

func (n *dbNotifier) OnLog(taskID, msg string) {
	slog.Info(fmt.Sprintf("[%s] %s", taskID, msg))
}
