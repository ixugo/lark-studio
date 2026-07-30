package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// StepName 步骤名称常量
const (
	StepWhisper   = "whisper"
	StepSplit     = "split"
	StepTranslate = "translate"
	StepTTS       = "tts"
	StepMerge     = "merge"
	StepLipSync   = "lipsync"
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
	WhisperBin         string  // whisper.cpp 可执行文件路径
	WhisperModel       string  // whisper 模型路径
	FFmpegBin          string  // ffmpeg 路径，空则使用 PATH 中的
	WorkDir            string  // 临时工作目录
	TranslatePrompt    string  // 自定义翻译提示词，空串则使用内置默认
	MaxSpeedFactor     float64 // TTS 调速上限，≤1 时不调速（推荐 1.2~1.3）
	TranslateChunkSize int     // 每次发给 LLM 的句子数（默认 10）
	TTSWorkers         int     // TTS 并发协程数（默认 2）
	CleanIntermediate  bool    // 成功后删除中间产物
	SubtitleOutput     string  // "burn" 烧录到视频 / "file" 仅输出字幕文件
	LipSyncEnabled     bool    // 是否启用对口型（仅 ModeDub 生效）
}

// Core 流水线调度核心
type Core struct {
	cfg        Config
	whisper    WhisperRunner
	llm        LLMClient
	tts        TTSClient
	lipSync    LipSyncClient
	notifier   Notifier
	termLister TermLister

	// 资源互斥锁：翻译和 TTS 同一时间只能一个 worker 使用
	llmMu sync.Mutex
	ttsMu sync.Mutex
}

// Option 配置选项
type Option func(*Core)

func WithNotifier(n Notifier) Option {
	return func(c *Core) { c.notifier = n }
}

// WithTermLister 注入术语列表查询，翻译时自动将匹配术语写入 prompt
func WithTermLister(tl TermLister) Option {
	return func(c *Core) { c.termLister = tl }
}

// WithLipSync 注入对口型客户端，ModeDub 且 LipSyncEnabled 时在 merge 后执行
func WithLipSync(ls LipSyncClient) Option {
	return func(c *Core) { c.lipSync = ls }
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
	TaskID         string
	InputPath      string  // 视频文件路径
	OutputDir      string  // 输出目录
	Mode           int     // 处理模式
	SourceLang     string  // 源语言
	TargetLang     string  // 目标语言
	Translator     string  // 翻译引擎
	OutputContent  string  // source / translated / bilingual
	TTSEngine      string  // edge / openai
	TTSVoice       string  // 音色名称
	SpeechRate     float64 // 语速倍率
	SubtitleOutput string  // burn / file / none
	ResumeFrom     string  // 断点恢复：从此步骤开始（空=从头）
}

const (
	stepMaxRetries = 3
	stepBaseDelay  = 3 * time.Second
)

// Run 执行单个任务的流水线，每个步骤失败时自动指数退避重试
// 同时将执行时间线写入 {outputDir}/task.log 供事后排查
func (c *Core) Run(ctx context.Context, job Job) error {
	slog.InfoContext(ctx, "pipeline.Run", "task_id", job.TaskID, "mode", job.Mode, "resume", job.ResumeFrom)
	c.logEvent(job.TaskID, "info", "", "任务开始，模式：%s", modeTitle(job.Mode))

	tl := openTaskLog(job.OutputDir)
	defer tl.Close()
	tl.Write("pipeline started: task=%s mode=%d resume=%q", job.TaskID, job.Mode, job.ResumeFrom)

	steps := c.buildSteps(job)
	pipeStart := time.Now()
	if err := c.runSteps(ctx, job, steps, resumeStepIndex(steps, job.ResumeFrom), tl); err != nil {
		return err
	}
	c.finishTask(job, pipeStart, tl)
	return nil
}

// runSteps 依序执行任务步骤，并统一记录开始、失败和完成事件。
func (c *Core) runSteps(ctx context.Context, job Job, steps []string, startIndex int, tl *taskLog) error {
	for index := startIndex; index < len(steps); index++ {
		step := steps[index]
		if err := ctx.Err(); err != nil {
			tl.Write("pipeline cancelled: %v", err)
			c.logEvent(job.TaskID, "warn", step, "任务已暂停")
			return fmt.Errorf("任务被取消: %w", err)
		}

		c.startStep(job.TaskID, step)
		c.logEvent(job.TaskID, "info", step, "%s开始", stepTitle(step))
		tl.Write("step started: %s", step)

		stepFn := c.stepFunc(step)
		t0 := time.Now()
		err := c.runStepWithRetry(ctx, job, step, stepFn)
		elapsed := time.Since(t0)

		if err != nil {
			tl.Write("step failed: %s (%v): %v", step, elapsed, err)
			c.logEvent(job.TaskID, "error", step, "%s失败，耗时 %s：%v", stepTitle(step), formatDuration(elapsed), err)
			c.failStep(job.TaskID, step, elapsed, err)
			return fmt.Errorf("步骤 %s 失败: %w", step, err)
		}

		tl.Write("step done: %s (%v)", step, elapsed)
		c.notifier.OnProgress(job.TaskID, step, 100)
		c.logEvent(job.TaskID, "success", step, "%s完成，耗时 %s", stepTitle(step), formatDuration(elapsed))
		c.notifier.OnStepDone(job.TaskID, step)
	}
	return nil
}

// finishTask 记录总耗时并按配置清理中间产物。
func (c *Core) finishTask(job Job, pipeStart time.Time, tl *taskLog) {
	totalElapsed := time.Since(pipeStart)
	tl.Write("pipeline completed (%v)", totalElapsed)
	c.logEvent(job.TaskID, "success", "", "任务完成，总耗时 %s", formatDuration(totalElapsed))
	c.notifier.OnTaskDone(job.TaskID)

	if c.cfg.CleanIntermediate {
		cleaned := cleanIntermediate(job.OutputDir)
		if cleaned > 0 {
			tl.Write("cleaned %d intermediate files", cleaned)
			c.logEvent(job.TaskID, "info", "", "已清理 %d 个中间产物", cleaned)
		}
	}
}

// resumeStepIndex 查找断点步骤，未指定或不存在时从头执行。
func resumeStepIndex(steps []string, resumeFrom string) int {
	for index, step := range steps {
		if step == resumeFrom {
			return index
		}
	}
	return 0
}

// taskLog 任务级日志写入器，追加写入 {outputDir}/task.log
type taskLog struct {
	file *os.File
}

func openTaskLog(outputDir string) *taskLog {
	f, err := os.OpenFile(
		filepath.Join(outputDir, "task.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644,
	)
	if err != nil {
		slog.Warn("无法创建 task.log", "dir", outputDir, "err", err)
		return &taskLog{}
	}
	return &taskLog{file: f}
}

func (tl *taskLog) Write(format string, args ...any) {
	if tl.file == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(tl.file, "[%s] %s\n", time.Now().Format(time.DateTime), msg)
}

func (tl *taskLog) Close() {
	if tl.file != nil {
		tl.file.Close()
	}
}

// stepFunc 根据步骤名返回对应的执行函数
func (c *Core) stepFunc(step string) func(context.Context, Job) error {
	switch step {
	case StepWhisper:
		return c.runWhisper
	case StepSplit:
		return c.runSplit
	case StepTranslate:
		return c.runTranslate
	case StepTTS:
		return c.runTTS
	case StepMerge:
		return c.runMerge
	case StepLipSync:
		return c.runLipSync
	case StepBurn:
		return c.runBurn
	default:
		return func(context.Context, Job) error { return nil }
	}
}

// runStepWithRetry 带指数退避重试的步骤执行
// context 取消视为主动中止，不重试
func (c *Core) runStepWithRetry(ctx context.Context, job Job, step string, fn func(context.Context, Job) error) error {
	var lastErr error
	for attempt := range stepMaxRetries {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn(ctx, job)
		if lastErr == nil {
			return nil
		}

		if ctx.Err() != nil {
			return lastErr
		}

		if attempt < stepMaxRetries-1 {
			delay := stepBaseDelay * time.Duration(1<<uint(attempt))
			c.logEvent(
				job.TaskID,
				"warn",
				step,
				"%s失败（第 %d 次）：%v，%s 后重试",
				stepTitle(step),
				attempt+1,
				lastErr,
				formatDuration(delay),
			)
			select {
			case <-ctx.Done():
				return lastErr
			case <-time.After(delay):
			}
		}
	}
	return lastErr
}

// formatDuration 将处理耗时压缩为适合任务日志的中文短格式。
func formatDuration(duration time.Duration) string {
	duration = duration.Round(time.Second)
	if duration < time.Minute {
		return fmt.Sprintf("%d 秒", max(0, int(duration.Seconds())))
	}
	return fmt.Sprintf("%d 分 %02d 秒", int(duration.Minutes()), int(duration.Seconds())%60)
}

// buildSteps 根据任务模式和字幕方式构建步骤列表。
func (c *Core) buildSteps(job Job) []string {
	mode := job.Mode
	var steps []string
	switch mode {
	case ModeSubtitle:
		steps = []string{StepWhisper, StepBurn}
	case ModeTranslate:
		steps = translationSteps(job.Translator, StepWhisper, StepTranslate, StepBurn)
	case ModeDub:
		if c.cfg.LipSyncEnabled && c.lipSync != nil {
			steps = translationSteps(
				job.Translator, StepWhisper, StepTranslate, StepTTS, StepMerge, StepLipSync, StepBurn,
			)
		} else {
			steps = translationSteps(job.Translator, StepWhisper, StepTranslate, StepTTS, StepMerge, StepBurn)
		}
	default:
		steps = []string{StepWhisper, StepBurn}
	}
	subtitleOutput := job.SubtitleOutput
	if subtitleOutput == "" {
		subtitleOutput = c.cfg.SubtitleOutput
		if subtitleOutput == "" {
			subtitleOutput = "burn"
		}
	}
	if mode != ModeDub && subtitleOutput != "burn" {
		filtered := steps[:0]
		for _, s := range steps {
			if s != StepBurn {
				filtered = append(filtered, s)
			}
		}
		return filtered
	}
	return steps
}

// translationSteps 仅在 OpenAI 翻译时插入语义分句，必应与 DeepLX 直接保留 Whisper 时间轴。
func translationSteps(translator string, steps ...string) []string {
	if !RequiresSemanticSplit(translator) {
		return steps
	}
	return append([]string{steps[0], StepSplit}, steps[1:]...)
}

// RequiresSemanticSplit 表示翻译引擎是否需要大模型参与语义分句与上下文翻译。
func RequiresSemanticSplit(translator string) bool {
	return strings.EqualFold(translator, "openai")
}

// noopNotifier 空通知实现
type noopNotifier struct{}

func (*noopNotifier) OnProgress(string, string, int) {}
func (*noopNotifier) OnStepDone(string, string)      {}
func (*noopNotifier) OnTaskDone(string)              {}
func (*noopNotifier) OnTaskFailed(string, error)     {}
func (*noopNotifier) OnLog(string, string)           {}

var ffmpegTimeRe = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})\.(\d{2})`)

// runFFmpegWithProgress 运行 ffmpeg 命令并通过解析 stderr 中的 time= 字段实时回报进度
// totalDuration 为总时长（秒），用于计算百分比；progressFn 在每次解析到进度时被调用
func runFFmpegWithProgress(
	ctx context.Context,
	totalDuration float64,
	progressFn func(int),
	logFn func(string),
	args ...string,
) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取 stderr 管道失败: %w", err)
	}
	cmd.Stdout = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 ffmpeg 失败: %w", err)
	}

	scanner := bufio.NewScanner(stderr)
	scanner.Split(scanFFmpegOutput)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" && logFn != nil {
			logFn(line)
		}
		reportFFmpegProgress(line, totalDuration, progressFn)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取 ffmpeg 输出失败: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

// reportFFmpegProgress 从单行输出提取时间并换算为百分比。
func reportFFmpegProgress(line string, totalDuration float64, progressFn func(int)) {
	if totalDuration <= 0 || progressFn == nil {
		return
	}
	matches := ffmpegTimeRe.FindStringSubmatch(line)
	if len(matches) < 5 {
		return
	}
	hours := parseIntSafe(matches[1])
	minutes := parseIntSafe(matches[2])
	seconds := parseIntSafe(matches[3])
	centiseconds := parseIntSafe(matches[4])
	current := float64(hours)*3600 + float64(minutes)*60 +
		float64(seconds) + float64(centiseconds)/100
	progressFn(min(99, int(current/totalDuration*100)))
}

// scanFFmpegOutput 自定义 bufio.SplitFunc，按 \r 或 \n 分割 ffmpeg 输出
func scanFFmpegOutput(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i, b := range data {
		if b == '\r' || b == '\n' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func parseIntSafe(s string) int {
	var v int
	fmt.Sscanf(s, "%d", &v)
	return v
}

// probeMediaDuration 用 ffmpeg -i 获取媒体文件时长（秒）
// 不依赖 ffprobe，减少一个外部二进制
func probeMediaDuration(ffmpegBin, mediaPath string) float64 {
	bin := "ffmpeg"
	if ffmpegBin != "" {
		bin = ffmpegBin
	}
	cmd := exec.Command(bin, "-i", mediaPath)
	output, _ := cmd.CombinedOutput()
	// 从 stderr 中解析 "Duration: HH:MM:SS.xx"
	s := string(output)
	idx := strings.Index(s, "Duration: ")
	if idx < 0 {
		return 0
	}
	s = s[idx+len("Duration: "):]
	end := strings.IndexByte(s, ',')
	if end < 0 {
		return 0
	}
	parts := strings.Split(s[:end], ":")
	if len(parts) != 3 {
		return 0
	}
	var h, m int
	var sec float64
	fmt.Sscanf(parts[0], "%d", &h)
	fmt.Sscanf(parts[1], "%d", &m)
	fmt.Sscanf(parts[2], "%f", &sec)
	return float64(h)*3600 + float64(m)*60 + sec
}
