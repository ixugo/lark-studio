package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ixugo/vdub/internal/adapter/lipsync"
	"github.com/ixugo/vdub/internal/adapter/llm"
	"github.com/ixugo/vdub/internal/adapter/tts"
	"github.com/ixugo/vdub/internal/adapter/whisper"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"github.com/ixugo/vdub/internal/core/term"
	"github.com/ixugo/vdub/pkg/ws"
)

// NewPipelineCore 根据配置创建完整的流水线核心，组装所有适配器
func NewPipelineCore(bc *conf.Bootstrap, opts ...pipeline.Option) *pipeline.Core {
	cfg := pipeline.Config{
		WhisperBin:         bc.Pipeline.WhisperBin,
		WhisperModel:       bc.Pipeline.WhisperModel,
		FFmpegBin:          bc.Pipeline.FFmpegBin,
		TranslatePrompt:    bc.Pipeline.TranslatePrompt,
		MaxSpeedFactor:     bc.Pipeline.MaxSpeedFactor,
		TranslateChunkSize: bc.Pipeline.TranslateChunkSize,
		TTSWorkers:         bc.Pipeline.TTSWorkers,
		CleanIntermediate:  bc.Pipeline.CleanIntermediate,
		SubtitleOutput:     bc.Pipeline.SubtitleOutput,
		LipSyncEnabled:     bc.LipSync.Enabled,
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

	if bc.LipSync.Enabled && bc.LipSync.BaseURL != "" {
		ls := lipsync.NewMuseTalkClient(bc.LipSync.BaseURL, bc.LipSync.APIKey)
		opts = append(opts, pipeline.WithLipSync(ls))
	}

	return pipeline.NewCore(cfg, wr, lc, tc, opts...)
}

// termAdapter 将 term.Core 适配为 pipeline.TermLister
type termAdapter struct {
	core term.Core
}

func (a *termAdapter) ListMappings(ctx context.Context) ([]pipeline.TermMapping, error) {
	terms, err := a.core.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	mappings := make([]pipeline.TermMapping, len(terms))
	for i, t := range terms {
		mappings[i] = pipeline.TermMapping{Text: t.Text, Translation: t.Translation}
	}
	return mappings, nil
}

// NewPipelineScheduler 创建带 DB 状态回写 + WebSocket 广播的流水线调度器
func NewPipelineScheduler(bc *conf.Bootstrap, taskCore task.Core, termCore term.Core, hub ws.Huber) (*pipeline.Scheduler, func()) {
	notifier := &dbNotifier{taskCore: taskCore, hub: hub}
	pipeCore := NewPipelineCore(bc,
		pipeline.WithNotifier(notifier),
		pipeline.WithTermLister(&termAdapter{core: termCore}),
	)

	var sched *pipeline.Scheduler
	onDone := func(taskID string, err error) {
		ctx := context.Background()
		if sched.WasPaused(taskID) {
			if e := taskCore.SetTaskStatus(ctx, taskID, func(t *task.Task) {
				t.Status = 2
				t.Error = ""
			}); e != nil {
				slog.Error("set paused status failed", "task_id", taskID, "err", e)
			}
			notifier.broadcast("task_paused", map[string]any{"task_id": taskID})
			return
		}

		status := 3
		errMsg := ""
		if err != nil {
			status = 4
			errMsg = err.Error()
			slog.Error("task failed", "task_id", taskID, "err", err)
		}
		if e := taskCore.SetTaskStatus(ctx, taskID, func(t *task.Task) {
			t.Status = status
			t.Error = errMsg
		}); e != nil {
			slog.Error("update task status failed", "task_id", taskID, "err", e)
		}
	}

	sched = pipeline.NewScheduler(pipeCore, onDone)
	sched.Start()
	return sched, sched.Stop
}

// dbNotifier 将流水线进度事件回写到 Task DB 并通过 WebSocket 广播
type dbNotifier struct {
	taskCore task.Core
	hub      ws.Huber
}

func (n *dbNotifier) OnProgress(taskID, step string, progress int) {
	ctx := context.Background()
	if err := n.taskCore.SetTaskStatus(ctx, taskID, func(t *task.Task) {
		t.Status = 1
		t.CurrentStep = step
		t.Progress = progress
	}); err != nil {
		slog.Debug("update progress failed", "task_id", taskID, "err", err)
	}
	n.broadcast("task_progress", map[string]any{
		"task_id": taskID, "step": step, "progress": progress,
	})
}

func (n *dbNotifier) OnStepDone(taskID, step string) {
	slog.Info("step done", "task_id", taskID, "step", step)
	n.broadcast("task_step_done", map[string]any{
		"task_id": taskID, "step": step,
	})
}

func (n *dbNotifier) OnTaskDone(taskID string) {
	slog.Info("task done", "task_id", taskID)
	n.broadcast("task_done", map[string]any{"task_id": taskID})
}

func (n *dbNotifier) OnTaskFailed(taskID string, err error) {
	slog.Error("task failed", "task_id", taskID, "err", err)
	n.broadcast("task_failed", map[string]any{
		"task_id": taskID, "error": err.Error(),
	})
}

func (n *dbNotifier) OnLog(taskID, msg string) {
	slog.Info(fmt.Sprintf("[%s] %s", taskID, msg))
	n.broadcast("task_log", map[string]any{
		"task_id": taskID, "message": msg,
	})
}

func (n *dbNotifier) broadcast(msgType string, data map[string]any) {
	if n.hub == nil {
		return
	}
	n.hub.Broadcast(ws.NewMessage(msgType, data))
}
