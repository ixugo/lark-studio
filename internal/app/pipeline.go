package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	asradapter "github.com/ixugo/vdub/internal/adapter/asr"
	"github.com/ixugo/vdub/internal/adapter/lipsync"
	"github.com/ixugo/vdub/internal/adapter/llm"
	"github.com/ixugo/vdub/internal/adapter/tts"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"github.com/ixugo/vdub/internal/core/term"
	"github.com/ixugo/vdub/pkg/ws"
)

// NewPipelineCore 根据配置创建完整的流水线核心，组装所有适配器
func NewPipelineCore(bc *conf.Bootstrap, opts ...pipeline.Option) *pipeline.Core {
	return NewPipelineCoreWithASR(bc, NewASRRouter(bc), opts...)
}

// NewASRRouter 以当前配置创建可即时更新的 ASR 引擎路由器。
func NewASRRouter(bc *conf.Bootstrap) *asradapter.Router {
	return asradapter.NewRouter(asradapter.Config{
		Engine:       bc.Pipeline.WhisperMode,
		WhisperBin:   bc.Pipeline.WhisperBin,
		WhisperModel: bc.Pipeline.WhisperModel,
		BaseURL:      bc.Pipeline.ASRBaseURL,
		APIKey:       bc.Pipeline.ASRAPIKey,
		Model:        bc.Pipeline.ASRModel,
	})
}

// NewPipelineCoreWithASR 将共享路由器装入流水线，使新任务读取最新默认引擎。
func NewPipelineCoreWithASR(bc *conf.Bootstrap, asrRouter *asradapter.Router, opts ...pipeline.Option) *pipeline.Core {
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
		SemanticSplitReady: strings.EqualFold(bc.LLM.Provider, "openai") && strings.TrimSpace(bc.LLM.BaseURL) != "" && strings.TrimSpace(bc.LLM.Model) != "",
	}

	lc := llm.NewRoutingClient(
		bc.LLM.BaseURL,
		bc.LLM.APIKey,
		bc.LLM.Model,
		bc.LLM.Provider,
		bc.LLM.DeepLXURL,
	)

	tc := tts.NewRouter(
		bc.TTS.Type,
		bc.TTS.Voice,
		bc.TTS.BaseURL,
		bc.TTS.APIKey,
		bc.TTS.Model,
	)

	tc.SetTTSConfig(bc.TTS.Type, bc.TTS.Voice, bc.TTS.BaseURL, bc.TTS.APIKey, bc.TTS.Model, tts.SpeechOptions{Protocol: bc.TTS.Protocol, Language: bc.TTS.Language, Instructions: bc.TTS.Instructions})

	if bc.LipSync.Enabled && bc.LipSync.BaseURL != "" {
		ls := lipsync.NewMuseTalkClient(bc.LipSync.BaseURL, bc.LipSync.APIKey)
		opts = append(opts, pipeline.WithLipSync(ls))
	}

	return pipeline.NewCore(cfg, asrRouter, lc, tc, opts...)
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
	scheduler, _, cleanup := NewPipelineSchedulerWithASR(bc, taskCore, termCore, hub)
	return scheduler, cleanup
}

// NewPipelineSchedulerWithASR 返回调度器与其共享 ASR 路由器，供设置保存时同步更新。
func NewPipelineSchedulerWithASR(bc *conf.Bootstrap, taskCore task.Core, termCore term.Core, hub ws.Huber) (*pipeline.Scheduler, *asradapter.Router, func()) {
	notifier := newDBNotifier(taskCore, hub, bc)
	asrRouter := NewASRRouter(bc)
	pipeCore := NewPipelineCoreWithASR(bc, asrRouter,
		pipeline.WithNotifier(notifier),
		pipeline.WithTermLister(&termAdapter{core: termCore}),
	)

	var sched *pipeline.Scheduler
	onDone := func(taskID string, err error) {
		handlePipelineDone(sched, notifier, taskCore, taskID, err)
	}

	sched = pipeline.NewScheduler(pipeCore, onDone)
	workers := bc.Pipeline.Workers
	if workers == 0 {
		workers = conf.DefaultConfig().Pipeline.Workers
	}
	if err := sched.SetWorkers(workers); err != nil {
		slog.Error("invalid task concurrency, using default", "err", err)
	}
	sched.Start()
	return sched, asrRouter, sched.Stop
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
		notifier.OnDetailedLog(taskID, "error", "", fmt.Sprintf("任务执行失败: %v", runErr))
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
	taskCore           task.Core
	hub                ws.Huber
	details            map[string]string
	lipSync            bool
	semanticSplitReady bool
	mu                 sync.Mutex
	stepProgress       map[string]map[string]int
	stepStartTime      map[string]map[string]time.Time
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
			pipeline.StepMerge:     "音轨混音",
			pipeline.StepLipSync:   "MuseTalk",
			pipeline.StepBurn:      "画面压制",
		},
		lipSync:            bc.LipSync.Enabled,
		semanticSplitReady: strings.EqualFold(bc.LLM.Provider, "openai") && strings.TrimSpace(bc.LLM.BaseURL) != "" && strings.TrimSpace(bc.LLM.Model) != "",
		stepProgress:       make(map[string]map[string]int),
		stepStartTime:      make(map[string]map[string]time.Time),
	}
}

// SetSemanticSplitReady 让总进度权重跟随当前翻译配置。
func (n *dbNotifier) SetSemanticSplitReady(ready bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.semanticSplitReady = ready
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
	detail := n.taskStepDetail(taskID, step)
	if n.stepStartTime[taskID] == nil {
		n.stepStartTime[taskID] = make(map[string]time.Time)
	}
	n.stepStartTime[taskID][step] = now
	if err := n.taskCore.UpsertStep(context.Background(), taskID, step, func(item *task.Step) {
		item.Status = pipeline.StepRunning
		item.Detail = detail
		item.StartedAt = &now
		item.EndedAt = nil
		item.Error = ""
	}); err != nil {
		slog.Error("start step failed", "task_id", taskID, "step", step, "err", err)
	}
	n.broadcast("task_step_started", map[string]any{
		"task_id": taskID, "step": step, "detail": detail,
	})
}

// taskStepDetail 优先显示任务创建时固定的引擎名称。
func (n *dbNotifier) taskStepDetail(taskID, step string) string {
	item, err := n.taskCore.GetTask(context.Background(), taskID)
	if err != nil {
		return n.details[step]
	}
	switch step {
	case pipeline.StepSplit:
		if item.Translator == "bing" || item.Translator == "deeplx" {
			return "原始时间轴"
		}
	case pipeline.StepTranslate:
		if item.Translator != "" {
			return item.Translator
		}
	case pipeline.StepTTS:
		if item.TTSEngine != "" {
			return item.TTSEngine
		}
	}
	return n.details[step]
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
		"task_id": taskID, "step": step, "detail": n.taskStepDetail(taskID, step),
		"step_progress": progress, "total_progress": totalProgress,
		"progress": totalProgress,
	})
}

// ResetTaskProgress 在任务从指定步骤节点重跑时，重置目标步骤及后续步骤在内存中的进度，并返回前置已完成步骤的基准百分比。
func (n *dbNotifier) ResetTaskProgress(taskID, fromStep string, mode int, subtitleOutput, translator string) int {
	n.mu.Lock()
	defer n.mu.Unlock()

	fromRank := stepRank(fromStep)
	if n.stepProgress[taskID] == nil {
		n.stepProgress[taskID] = make(map[string]int)
	}

	for _, s := range []string{
		pipeline.StepWhisper,
		pipeline.StepSplit,
		pipeline.StepTranslate,
		pipeline.StepTTS,
		pipeline.StepMerge,
		pipeline.StepLipSync,
		pipeline.StepBurn,
	} {
		if stepRank(s) >= fromRank {
			n.stepProgress[taskID][s] = 0
			if n.stepStartTime[taskID] != nil {
				delete(n.stepStartTime[taskID], s)
			}
			// 同时清理 DB 中旧步骤的开始结束时间与耗时，防止历史时间戳污染
			_ = n.taskCore.UpsertStep(context.Background(), taskID, s, func(item *task.Step) {
				item.Status = pipeline.StepPending
				item.Progress = 0
				item.StartedAt = nil
				item.EndedAt = nil
				item.Error = ""
			})
		} else {
			n.stepProgress[taskID][s] = 100
		}
	}

	return n.totalProgress(mode, subtitleOutput, translator, n.stepProgress[taskID])
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
	detail := n.taskStepDetail(taskID, step)
	err := n.taskCore.SetTaskStatus(ctx, taskID, func(item *task.Task) {
		totalProgress = n.totalProgress(item.Mode, item.SubtitleOutput, item.Translator, n.stepProgress[taskID])
		item.Status = 1
		if stepRank(step) >= stepRank(item.CurrentStep) {
			item.CurrentStep = step
			item.CurrentDetail = detail
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
	detail := n.taskStepDetail(taskID, step)
	return n.taskCore.UpsertStep(ctx, taskID, step, func(item *task.Step) {
		item.Status = pipeline.StepRunning
		item.Progress = max(item.Progress, progress)
		item.Detail = detail
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
	startTime := n.stepStartTime[taskID][step]
	if startTime.IsZero() {
		startTime = now
	}
	elapsed := now.Sub(startTime)
	if err := n.taskCore.UpsertStep(context.Background(), taskID, step, func(item *task.Step) {
		item.Status = pipeline.StepDone
		item.Progress = 100
		if item.StartedAt == nil || startTime.After(*item.StartedAt) {
			item.StartedAt = &startTime
		}
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
func (n *dbNotifier) totalProgress(mode int, subtitleOutput, translator string, progress map[string]int) int {
	weights := n.progressWeights(mode, subtitleOutput, translator)
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
func (n *dbNotifier) progressWeights(mode int, subtitleOutput, translator string) map[string]int {
	if subtitleOutput == "" {
		subtitleOutput = "burn"
	}
	weights := map[string]int{pipeline.StepWhisper: 25}
	switch mode {
	case pipeline.ModeSubtitle:
		weights[pipeline.StepWhisper] = 65
	case pipeline.ModeTranslate:
		weights[pipeline.StepWhisper] = 35
		if n.semanticSplitReady && pipeline.RequiresSemanticSplit(translator) {
			weights[pipeline.StepSplit] = 5
		}
		weights[pipeline.StepTranslate] = 25
	case pipeline.ModeDub:
		if n.semanticSplitReady && pipeline.RequiresSemanticSplit(translator) {
			weights[pipeline.StepSplit] = 5
		}
		weights[pipeline.StepTranslate] = 20
		weights[pipeline.StepTTS] = 25
		weights[pipeline.StepMerge] = 10
	case pipeline.ModeDirectDub:
		weights[pipeline.StepTTS] = 40
		weights[pipeline.StepMerge] = 10
	case pipeline.ModeDubOnly:
		weights[pipeline.StepTTS] = 80
		weights[pipeline.StepMerge] = 20
	}
	if n.lipSync && mode == pipeline.ModeDub {
		weights[pipeline.StepLipSync] = 10
	}
	if subtitleOutput == "burn" || mode == pipeline.ModeDub || mode == pipeline.ModeDirectDub {
		weights[pipeline.StepBurn] = 35
		if mode == pipeline.ModeDub {
			weights[pipeline.StepBurn] = 15
		} else if mode == pipeline.ModeDirectDub {
			weights[pipeline.StepBurn] = 25
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
