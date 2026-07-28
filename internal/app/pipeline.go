package app

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	notifier := newDBNotifier(taskCore, hub, bc)
	pipeCore := NewPipelineCore(bc,
		pipeline.WithNotifier(notifier),
		pipeline.WithTermLister(&termAdapter{core: termCore}),
	)

	var sched *pipeline.Scheduler
	onDone := func(taskID string, err error) {
		handlePipelineDone(sched, notifier, taskCore, taskID, err)
	}

	sched = pipeline.NewScheduler(pipeCore, onDone)
	sched.Start()
	return sched, sched.Stop
}

// handlePipelineDone 根据暂停、成功或失败结果回写任务终态。
func handlePipelineDone(
	scheduler *pipeline.Scheduler,
	notifier *dbNotifier,
	taskCore task.Core,
	taskID string,
	runErr error,
) {
	ctx := context.Background()
	if scheduler.WasPaused(taskID) {
		if err := taskCore.SetTaskStatus(ctx, taskID, func(item *task.Task) {
			item.Status = 2
			item.Error = ""
		}); err != nil {
			slog.Error("set paused status failed", "task_id", taskID, "err", err)
		}
		notifier.broadcast("task_paused", map[string]any{"task_id": taskID})
		return
	}
	status, errorMessage := 3, ""
	if runErr != nil {
		status, errorMessage = 4, runErr.Error()
		slog.Error("task failed", "task_id", taskID, "err", runErr)
	}
	if err := taskCore.SetTaskStatus(ctx, taskID, func(item *task.Task) {
		item.Status = status
		item.Error = errorMessage
	}); err != nil {
		slog.Error("update task status failed", "task_id", taskID, "err", err)
	}
}

// dbNotifier 将流水线进度事件回写到 Task DB 并通过 WebSocket 广播
type dbNotifier struct {
	taskCore      task.Core
	hub           ws.Huber
	details       map[string]string
	subtitleFile  bool
	lipSync       bool
	mu            sync.Mutex
	stepProgress  map[string]map[string]int
	stepStartTime map[string]map[string]time.Time
}

// newDBNotifier 创建带单调进度状态和步骤模型名称的通知器。
func newDBNotifier(taskCore task.Core, hub ws.Huber, bc *conf.Bootstrap) *dbNotifier {
	return &dbNotifier{
		taskCore: taskCore,
		hub:      hub,
		details: map[string]string{
			pipeline.StepWhisper:   whisperModelName(bc.Pipeline.WhisperModel),
			pipeline.StepSplit:     bc.LLM.Model,
			pipeline.StepTranslate: bc.LLM.Model,
			pipeline.StepTTS:       ttsModelName(bc),
			pipeline.StepMerge:     "ffmpeg",
			pipeline.StepLipSync:   "MuseTalk",
			pipeline.StepBurn:      "ffmpeg",
		},
		subtitleFile:  bc.Pipeline.SubtitleOutput == "file",
		lipSync:       bc.LipSync.Enabled,
		stepProgress:  make(map[string]map[string]int),
		stepStartTime: make(map[string]map[string]time.Time),
	}
}

// whisperModelName 提取适合界面显示的模型名称。
func whisperModelName(path string) string {
	name := filepath.Base(path)
	name = strings.TrimPrefix(name, "ggml-")
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if name == "." || name == "" {
		return "Whisper"
	}
	return name
}

// ttsModelName 返回当前配音引擎名称。
func ttsModelName(bc *conf.Bootstrap) string {
	if bc.TTS.Type == "openai" {
		if bc.TTS.Model != "" {
			return bc.TTS.Model
		}
		return "OpenAI TTS"
	}
	return "edge-tts"
}

// OnStepStart 创建步骤记录，并保存本次尝试的开始时间。
func (n *dbNotifier) OnStepStart(taskID, step string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	now := time.Now()
	if n.stepStartTime[taskID] == nil {
		n.stepStartTime[taskID] = make(map[string]time.Time)
	}
	if _, exists := n.stepStartTime[taskID][step]; !exists {
		n.stepStartTime[taskID][step] = now
	}
	if err := n.taskCore.UpsertStep(context.Background(), taskID, step, func(item *task.Step) {
		item.Status = pipeline.StepRunning
		item.Detail = n.details[step]
		if item.StartedAt == nil {
			item.StartedAt = &now
		}
	}); err != nil {
		slog.Error("start step failed", "task_id", taskID, "step", step, "err", err)
	}
	n.broadcast("task_step_started", map[string]any{
		"task_id": taskID, "step": step, "detail": n.details[step],
	})
}

// OnProgress 保存步骤百分比，并按权重计算只增不减的总进度。
func (n *dbNotifier) OnProgress(taskID, step string, progress int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	progress = max(0, min(100, progress))
	if n.stepProgress[taskID] == nil {
		n.stepProgress[taskID] = make(map[string]int)
	}
	if progress < n.stepProgress[taskID][step] {
		return
	}
	n.stepProgress[taskID][step] = progress
	startedAt := n.ensureStepStartLocked(taskID, step)

	ctx := context.Background()
	totalProgress, err := n.updateTaskProgress(ctx, taskID, step, progress, startedAt)
	if err != nil {
		slog.Error("update progress failed", "task_id", taskID, "err", err)
	}
	if err := n.updateStepProgress(ctx, taskID, step, progress, startedAt); err != nil {
		slog.Error("update step progress failed", "task_id", taskID, "step", step, "err", err)
	}
	n.broadcast("task_progress", map[string]any{
		"task_id": taskID, "step": step, "detail": n.details[step],
		"step_progress": progress, "total_progress": totalProgress,
		"progress": totalProgress,
	})
}

// updateTaskProgress 更新任务总进度和最靠后的当前步骤。
func (n *dbNotifier) updateTaskProgress(
	ctx context.Context,
	taskID string,
	step string,
	progress int,
	startedAt time.Time,
) (int, error) {
	totalProgress := 0
	err := n.taskCore.SetTaskStatus(ctx, taskID, func(item *task.Task) {
		totalProgress = max(n.totalProgress(item.Mode, n.stepProgress[taskID]), item.Progress)
		item.Status = 1
		if stepRank(step) >= stepRank(item.CurrentStep) {
			item.CurrentStep = step
			item.CurrentDetail = n.details[step]
			item.StepStartedAt = &startedAt
			item.StepProgress = progress
		}
		item.Progress = totalProgress
	})
	return totalProgress, err
}

// updateStepProgress 更新单个步骤自己的进度和模型明细。
func (n *dbNotifier) updateStepProgress(
	ctx context.Context,
	taskID string,
	step string,
	progress int,
	startedAt time.Time,
) error {
	return n.taskCore.UpsertStep(ctx, taskID, step, func(item *task.Step) {
		item.Status = pipeline.StepRunning
		item.Progress = max(item.Progress, progress)
		item.Detail = n.details[step]
		if item.StartedAt == nil {
			item.StartedAt = &startedAt
		}
	})
}

// ensureStepStartLocked 为并发提前到达的步骤进度补齐开始时间。
func (n *dbNotifier) ensureStepStartLocked(taskID, step string) time.Time {
	if n.stepStartTime[taskID] == nil {
		n.stepStartTime[taskID] = make(map[string]time.Time)
	}
	if n.stepStartTime[taskID][step].IsZero() {
		n.stepStartTime[taskID][step] = time.Now()
	}
	return n.stepStartTime[taskID][step]
}

// OnStepDone 将步骤标为完成并记录结束时间。
func (n *dbNotifier) OnStepDone(taskID, step string) {
	n.OnProgress(taskID, step, 100)
	n.mu.Lock()
	defer n.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(n.stepStartTime[taskID][step])
	if err := n.taskCore.UpsertStep(context.Background(), taskID, step, func(item *task.Step) {
		item.Status = pipeline.StepDone
		item.Progress = 100
		item.EndedAt = &now
	}); err != nil {
		slog.Error("finish step failed", "task_id", taskID, "step", step, "err", err)
	}
	slog.Info("step done", "task_id", taskID, "step", step)
	n.broadcast("task_step_done", map[string]any{
		"task_id": taskID, "step": step, "elapsed_ms": elapsed.Milliseconds(),
	})
}

// OnStepFailed 将步骤标为失败并保留错误和耗时。
func (n *dbNotifier) OnStepFailed(taskID, step string, _ time.Duration, stepErr error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	now := time.Now()
	if err := n.taskCore.UpsertStep(context.Background(), taskID, step, func(item *task.Step) {
		item.Status = pipeline.StepFailed
		item.Error = stepErr.Error()
		item.EndedAt = &now
	}); err != nil {
		slog.Error("fail step update failed", "task_id", taskID, "step", step, "err", err)
	}
}

// OnTaskDone 标记总进度完成并释放内存状态。
func (n *dbNotifier) OnTaskDone(taskID string) {
	if err := n.taskCore.SetTaskStatus(context.Background(), taskID, func(item *task.Task) {
		item.Progress = 100
		item.StepProgress = 100
	}); err != nil {
		slog.Error("finish task progress failed", "task_id", taskID, "err", err)
	}
	slog.Info("task done", "task_id", taskID)
	n.broadcast("task_done", map[string]any{"task_id": taskID})
	n.clearTask(taskID)
}

// OnTaskFailed 广播任务失败事件，具体日志由结构化日志事件持久化。
func (n *dbNotifier) OnTaskFailed(taskID string, err error) {
	slog.Error("task failed", "task_id", taskID, "err", err)
	n.broadcast("task_failed", map[string]any{
		"task_id": taskID, "error": err.Error(),
	})
}

// OnLog 兼容基础通知接口，将未知步骤日志按普通信息保存。
func (n *dbNotifier) OnLog(taskID, msg string) {
	n.OnDetailedLog(taskID, "info", "", msg)
}

// OnDetailedLog 写入 SQLite 后再广播，确保实时消息和历史记录一致。
func (n *dbNotifier) OnDetailedLog(taskID, level, step, message string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	item, err := n.taskCore.AppendTaskLog(context.Background(), taskID, level, step, message)
	if err != nil {
		slog.Error("persist task log failed", "task_id", taskID, "step", step, "err", err)
		return
	}
	slog.Info("task log", "task_id", taskID, "level", level, "step", step, "message", message)
	n.broadcast("task_log", map[string]any{
		"id": item.ID, "task_id": taskID, "level": level, "step": step,
		"message": message, "created_at": item.CreatedAt,
	})
}

// clearTask 清除完成任务的临时进度状态。
func (n *dbNotifier) clearTask(taskID string) {
	n.mu.Lock()
	delete(n.stepProgress, taskID)
	delete(n.stepStartTime, taskID)
	n.mu.Unlock()
}

// totalProgress 按任务模式权重汇总所有步骤进度。
func (n *dbNotifier) totalProgress(mode int, progress map[string]int) int {
	weights := n.progressWeights(mode)
	totalWeight := 0
	weightedProgress := 0
	for step, weight := range weights {
		totalWeight += weight
		weightedProgress += weight * progress[step]
	}
	if totalWeight == 0 {
		return 0
	}
	return weightedProgress / totalWeight
}

// progressWeights 返回当前模式实际执行步骤的进度权重。
func (n *dbNotifier) progressWeights(mode int) map[string]int {
	weights := map[string]int{pipeline.StepWhisper: 25}
	switch mode {
	case pipeline.ModeSubtitle:
		weights[pipeline.StepWhisper] = 65
	case pipeline.ModeTranslate:
		weights[pipeline.StepWhisper] = 35
		weights[pipeline.StepSplit] = 5
		weights[pipeline.StepTranslate] = 25
	case pipeline.ModeDub:
		weights[pipeline.StepSplit] = 5
		weights[pipeline.StepTranslate] = 20
		weights[pipeline.StepTTS] = 25
		weights[pipeline.StepMerge] = 10
	}
	if n.lipSync && mode == pipeline.ModeDub {
		weights[pipeline.StepLipSync] = 10
	}
	if !n.subtitleFile {
		weights[pipeline.StepBurn] = 35
		if mode == pipeline.ModeDub {
			weights[pipeline.StepBurn] = 15
		}
	}
	return weights
}

// stepRank 返回步骤在界面上的固定顺序，防止并发进度令当前步骤倒退。
func stepRank(step string) int {
	switch step {
	case pipeline.StepWhisper:
		return 1
	case pipeline.StepSplit:
		return 2
	case pipeline.StepTranslate:
		return 3
	case pipeline.StepTTS:
		return 4
	case pipeline.StepMerge:
		return 5
	case pipeline.StepLipSync:
		return 6
	case pipeline.StepBurn:
		return 7
	default:
		return 0
	}
}

// broadcast 向所有已连接界面推送事件。
func (n *dbNotifier) broadcast(msgType string, data map[string]any) {
	if n.hub == nil {
		return
	}
	n.hub.Broadcast(ws.NewMessage(msgType, data))
}
