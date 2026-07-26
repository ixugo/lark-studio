package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// StepName 步骤名称常量
const (
	StepWhisper   = "whisper"
	StepSplit     = "split"
	StepTranslate = "translate"
	StepTTS       = "tts"
	StepMerge     = "merge"
	StepBurn      = "burn"
)

// Mode 处理模式
const (
	ModeSubtitle  = 1 // 仅生成字幕
	ModeTranslate = 2 // 生成+翻译字幕
	ModeDub       = 3 // 生成+翻译+配音
)

// StepStatus 步骤状态
const (
	StepPending = 0
	StepRunning = 1
	StepDone    = 2
	StepFailed  = 3
	StepSkipped = 4
)

// Notifier 通知接口，用于向 UI 推送状态变更
type Notifier interface {
	OnProgress(taskID string, step string, progress int)
	OnStepDone(taskID string, step string)
	OnTaskDone(taskID string)
	OnTaskFailed(taskID string, err error)
	OnLog(taskID string, msg string)
}

// Config 流水线依赖配置
type Config struct {
	WhisperBin   string // whisper.cpp 可执行文件路径
	WhisperModel string // whisper 模型路径
	FFmpegBin    string // ffmpeg 路径，空则使用 PATH 中的
	WorkDir      string // 临时工作目录
}

// Core 流水线调度核心
type Core struct {
	cfg      Config
	whisper  WhisperRunner
	llm      LLMClient
	tts      TTSClient
	notifier Notifier

	// 资源互斥锁：翻译和 TTS 同一时间只能一个 worker 使用
	llmMu sync.Mutex
	ttsMu sync.Mutex
}

// Option 配置选项
type Option func(*Core)

func WithNotifier(n Notifier) Option {
	return func(c *Core) { c.notifier = n }
}

// NewCore 创建流水线核心
func NewCore(cfg Config, whisper WhisperRunner, llm LLMClient, tts TTSClient, opts ...Option) *Core {
	c := &Core{
		cfg:     cfg,
		whisper: whisper,
		llm:     llm,
		tts:     tts,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.notifier == nil {
		c.notifier = &noopNotifier{}
	}
	return c
}

// Job 描述一个待处理的视频任务
type Job struct {
	TaskID     string
	InputPath  string // 视频文件路径
	OutputDir  string // 输出目录
	Mode       int    // 处理模式
	TargetLang string // 目标语言
	ResumeFrom string // 断点恢复：从此步骤开始（空=从头）
}

// Run 执行单个任务的流水线
func (c *Core) Run(ctx context.Context, job Job) error {
	slog.InfoContext(ctx, "pipeline.Run", "task_id", job.TaskID, "mode", job.Mode, "resume", job.ResumeFrom)

	steps := c.buildSteps(job.Mode)
	startIdx := 0
	if job.ResumeFrom != "" {
		for i, s := range steps {
			if s == job.ResumeFrom {
				startIdx = i
				break
			}
		}
	}

	for i := startIdx; i < len(steps); i++ {
		step := steps[i]
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("任务被取消: %w", err)
		}

		c.notifier.OnLog(job.TaskID, fmt.Sprintf("开始步骤: %s", step))

		var err error
		switch step {
		case StepWhisper:
			err = c.runWhisper(ctx, job)
		case StepSplit:
			err = c.runSplit(ctx, job)
		case StepTranslate:
			err = c.runTranslate(ctx, job)
		case StepTTS:
			err = c.runTTS(ctx, job)
		case StepMerge:
			err = c.runMerge(ctx, job)
		case StepBurn:
			err = c.runBurn(ctx, job)
		}

		if err != nil {
			c.notifier.OnTaskFailed(job.TaskID, err)
			return fmt.Errorf("步骤 %s 失败: %w", step, err)
		}

		c.notifier.OnStepDone(job.TaskID, step)
	}

	c.notifier.OnTaskDone(job.TaskID)
	return nil
}

// buildSteps 根据模式构建步骤列表
// split 步骤暂不启用：当前直接翻译 SRT 条目以保证时间轴 1:1 对齐
func (c *Core) buildSteps(mode int) []string {
	switch mode {
	case ModeSubtitle:
		return []string{StepWhisper, StepBurn}
	case ModeTranslate:
		return []string{StepWhisper, StepTranslate, StepBurn}
	case ModeDub:
		return []string{StepWhisper, StepTranslate, StepTTS, StepMerge, StepBurn}
	default:
		return []string{StepWhisper, StepBurn}
	}
}

// noopNotifier 空通知实现
type noopNotifier struct{}

func (*noopNotifier) OnProgress(string, string, int) {}
func (*noopNotifier) OnStepDone(string, string)      {}
func (*noopNotifier) OnTaskDone(string)              {}
func (*noopNotifier) OnTaskFailed(string, error)     {}
func (*noopNotifier) OnLog(string, string)           {}
