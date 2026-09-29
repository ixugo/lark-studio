package wails

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	asradapter "github.com/ixugo/vdub/internal/adapter/asr"
	llmadapter "github.com/ixugo/vdub/internal/adapter/llm"
	ttsadapter "github.com/ixugo/vdub/internal/adapter/tts"
	whisperadapter "github.com/ixugo/vdub/internal/adapter/whisper"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/recipe"
	"github.com/ixugo/vdub/internal/core/task"
	"github.com/ixugo/vdub/internal/core/term"
	"github.com/ixugo/vdub/internal/web/api"
	"github.com/ixugo/vdub/pkg/ws"
)

// AppService 聚合所有暴露给前端界面的 Go 接口方法。
type AppService struct {
	mu         sync.RWMutex
	app        *application.App
	bc         *conf.Bootstrap
	taskCore   task.Core
	termCore   term.Core
	recipeCore recipe.Core
	scheduler  *pipeline.Scheduler
	hub        ws.Huber
	asrRouter  *asradapter.Router
}

// SetASRRouter 保存流水线共用的引擎路由器，使配置页保存可影响后续任务。
func (s *AppService) SetASRRouter(router *asradapter.Router) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.asrRouter = router
}

// NewAppService 创建应用服务。
func NewAppService(
	app *application.App,
	bc *conf.Bootstrap,
	taskCore task.Core,
	termCore term.Core,
	recipeCore recipe.Core,
	scheduler *pipeline.Scheduler,
	hub ws.Huber,
) *AppService {
	return &AppService{
		app:        app,
		bc:         bc,
		taskCore:   taskCore,
		termCore:   termCore,
		recipeCore: recipeCore,
		scheduler:  scheduler,
		hub:        hub,
	}
}

// ─── 任务管理 ─────────────────────────────────────────────

// ListTasks 获取所有任务列表。
func (s *AppService) ListTasks() ([]*task.Task, error) {
	in := &task.ListTaskInput{}
	in.Size = 200
	tasks, _, err := s.taskCore.ListTasks(context.Background(), in)
	return tasks, err
}

// GetTask 获取指定任务详情。
func (s *AppService) GetTask(id string) (*task.Task, error) {
	return s.taskCore.GetTask(context.Background(), id)
}

// ListTaskLogs 获取任务运行日志。
func (s *AppService) ListTaskLogs(id string) ([]*task.TaskLog, error) {
	return s.taskCore.ListRecentTaskLogs(context.Background(), id)
}

// ListTaskSteps 获取指定任务的所有步骤执行明细及耗时。
func (s *AppService) ListTaskSteps(taskID string) ([]*task.Step, error) {
	in := &task.ListStepInput{
		TaskID: taskID,
	}
	in.Size = 50
	steps, _, err := s.taskCore.ListSteps(context.Background(), in)
	return steps, err
}

// CreateTask 创建单个转写/翻译/配音任务。
func (s *AppService) CreateTask(in task.CreateTaskInput) (*task.Task, error) {
	if err := s.prepareTaskInput(&in); err != nil {
		return nil, err
	}
	ctx := context.Background()
	t, err := s.taskCore.CreateTask(ctx, &in)
	if err != nil {
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}
	_, _ = s.taskCore.AppendTaskLog(ctx, t.ID, "info", "", "任务创建，模式："+taskModeTitle(t.Mode))

	if err := s.scheduler.Submit(taskPipelineJob(t)); err != nil {
		return nil, fmt.Errorf("提交流水线失败: %w", err)
	}
	return t, nil
}

// BatchCreateTasks 批量创建多视频任务。
func (s *AppService) BatchCreateTasks(videos []string, recipe task.CreateTaskInput) ([]*task.Task, error) {
	var created []*task.Task
	for _, video := range videos {
		item := recipe
		item.InputPath = video
		item.OutputDir = ""
		t, err := s.CreateTask(item)
		if err != nil {
			return created, err
		}
		created = append(created, t)
	}
	return created, nil
}

// MergeSubtitleInput 独立字幕合成输入参数
type MergeSubtitleInput struct {
	VideoPath        string `json:"video_path"`
	PrimarySubPath   string `json:"primary_sub_path"`
	SecondarySubPath string `json:"secondary_sub_path"`
	OutputDir        string `json:"output_dir"`
	OutputContent    string `json:"output_content"`
}

// MergeSubtitle 将外部字幕与视频合成为成片视频。
func (s *AppService) MergeSubtitle(in MergeSubtitleInput) (*task.Task, error) {
	if in.VideoPath == "" {
		return nil, fmt.Errorf("请选择视频文件")
	}
	if in.PrimarySubPath == "" {
		return nil, fmt.Errorf("请选择字幕文件")
	}
	if _, err := os.Stat(in.VideoPath); err != nil {
		return nil, fmt.Errorf("视频文件不存在: %s", in.VideoPath)
	}
	if _, err := os.Stat(in.PrimarySubPath); err != nil {
		return nil, fmt.Errorf("字幕文件不存在: %s", in.PrimarySubPath)
	}

	// 字幕合成固定输出到统一任务目录 ~/.lark-studio/tasks。
	baseName := strings.TrimSuffix(filepath.Base(in.VideoPath), filepath.Ext(in.VideoPath))
	in.OutputDir = filepath.Join(conf.TasksDir(), baseName+"_vdub")
	if err := os.MkdirAll(in.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	if in.OutputContent == "bilingual" && in.SecondarySubPath != "" {
		secData, err := os.ReadFile(in.SecondarySubPath)
		if err == nil {
			_ = os.WriteFile(filepath.Join(in.OutputDir, "src.srt"), secData, 0o644)
		}
		priData, err := os.ReadFile(in.PrimarySubPath)
		if err != nil {
			return nil, fmt.Errorf("读取主字幕失败: %w", err)
		}
		_ = os.WriteFile(filepath.Join(in.OutputDir, "trans.srt"), priData, 0o644)
	} else {
		in.OutputContent = "source"
		priData, err := os.ReadFile(in.PrimarySubPath)
		if err != nil {
			return nil, fmt.Errorf("读取字幕失败: %w", err)
		}
		_ = os.WriteFile(filepath.Join(in.OutputDir, "src.srt"), priData, 0o644)
	}

	taskInput := task.CreateTaskInput{
		InputPath:      in.VideoPath,
		OutputDir:      in.OutputDir,
		Mode:           pipeline.ModeSubtitle,
		OutputContent:  in.OutputContent,
		SubtitleOutput: "burn",
	}

	return s.CreateTask(taskInput)
}

// PauseTask 暂停进行中的任务。
func (s *AppService) PauseTask(id string) error {
	ctx := context.Background()
	item, err := s.taskCore.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if item.Status != 1 {
		return fmt.Errorf("任务不在运行中，当前状态: %d", item.Status)
	}
	s.scheduler.Pause(id)
	return nil
}

// ResumeTask 恢复已暂停或已失败的任务。
func (s *AppService) ResumeTask(id string) error {
	ctx := context.Background()
	item, err := s.taskCore.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if item.Status != 2 && item.Status != 4 {
		return fmt.Errorf("只能恢复已暂停或已失败的任务")
	}
	resumeFrom := item.CurrentStep
	if err := s.taskCore.SetTaskStatus(ctx, id, func(t *task.Task) {
		t.Status = 1
		t.Error = ""
	}); err != nil {
		return err
	}
	job := taskPipelineJob(item)
	job.ResumeFrom = resumeFrom
	return s.scheduler.Submit(job)
}

// RerunTaskOptions 重跑任务的可选新配方与覆盖参数
type RerunTaskOptions struct {
	FromStep       string  `json:"from_step"`
	Mode           int     `json:"mode,omitempty"`
	TargetLang     string  `json:"target_lang,omitempty"`
	SourceLang     string  `json:"source_lang,omitempty"`
	Translator     string  `json:"translator,omitempty"`
	OutputContent  string  `json:"output_content,omitempty"`
	TTSEngine      string  `json:"tts_engine,omitempty"`
	TTSVoice       string  `json:"tts_voice,omitempty"`
	SpeechRate     float64 `json:"speech_rate,omitempty"`
	SubtitleOutput string  `json:"subtitle_output,omitempty"`
	RecipeName     string  `json:"recipe_name,omitempty"`
}

// RerunTaskWithRecipe 支持修改配置配方并指定节点重跑
func (s *AppService) RerunTaskWithRecipe(id string, opts RerunTaskOptions) error {
	ctx := context.Background()
	item, err := s.taskCore.GetTask(ctx, id)
	if err != nil {
		return err
	}

	fromStep := opts.FromStep
	if fromStep == "" {
		fromStep = pipeline.StepWhisper
	}

	targetMode := item.Mode
	if opts.Mode > 0 {
		targetMode = opts.Mode
	}
	// 校验失败时保留原任务状态及产物，不能先清理再报错。
	if err := pipeline.ValidateResourceMode(item.InputPath, targetMode); err != nil {
		return err
	}
	if fromStep == pipeline.StepWhisper {
		s.mu.RLock()
		config := asrConfigFromPipeline(s.bc.Pipeline)
		s.mu.RUnlock()
		if err := pipeline.ValidateRecognitionConfig(item.InputPath, "", targetMode, config); err != nil {
			return err
		}
	}

	s.scheduler.Pause(id)
	pipeline.CleanStepAndSubsequent(item.OutputDir, fromStep)

	targetSubOutput := item.SubtitleOutput
	if opts.SubtitleOutput != "" {
		targetSubOutput = opts.SubtitleOutput
	}
	targetTranslator := item.Translator
	if opts.Translator != "" {
		targetTranslator = opts.Translator
	}

	baseProgress := 0
	if resetter, ok := s.scheduler.Notifier().(interface {
		ResetTaskProgress(taskID, fromStep string, mode int, subtitleOutput, translator string) int
	}); ok {
		baseProgress = resetter.ResetTaskProgress(id, fromStep, targetMode, targetSubOutput, targetTranslator)
	}

	// 重置任务状态为处理中，清空错误与当前进度，若传入新配方参数则同步更新持久化
	if err := s.taskCore.SetTaskStatus(ctx, id, func(t *task.Task) {
		t.Status = 1
		t.Error = ""
		t.CurrentStep = fromStep
		t.CurrentDetail = "重跑节点: " + fromStep
		t.Progress = baseProgress
		t.StepProgress = 0
		if opts.Mode > 0 {
			t.Mode = opts.Mode
		}
		if opts.TargetLang != "" {
			t.TargetLang = opts.TargetLang
		}
		if opts.SourceLang != "" {
			t.SourceLang = opts.SourceLang
		}
		if opts.Translator != "" {
			t.Translator = opts.Translator
		}
		if opts.OutputContent != "" {
			t.OutputContent = opts.OutputContent
		}
		if opts.TTSEngine != "" {
			t.TTSEngine = opts.TTSEngine
		}
		if opts.TTSVoice != "" {
			t.TTSVoice = opts.TTSVoice
		}
		if opts.SpeechRate > 0 {
			t.SpeechRate = opts.SpeechRate
		}
		if opts.SubtitleOutput != "" {
			t.SubtitleOutput = opts.SubtitleOutput
		}
		if opts.RecipeName != "" {
			t.RecipeName = opts.RecipeName
		}
	}); err != nil {
		return err
	}

	if s.hub != nil {
		s.hub.Broadcast(ws.NewMessage("task_progress", map[string]any{
			"task_id":        id,
			"step":           fromStep,
			"detail":         "重跑节点: " + fromStep,
			"step_progress":  0,
			"total_progress": baseProgress,
			"progress":       baseProgress,
		}))
	}

	// 重新获取已更新配置的任务并提交流水线
	freshTask, err := s.taskCore.GetTask(ctx, id)
	if err != nil {
		freshTask = item
	}

	job := taskPipelineJob(freshTask)
	job.ResumeFrom = fromStep
	return s.scheduler.Submit(job)
}

// RerunTaskFromStep 用户主动要求从某一个步骤节点重新执行该任务
// 清除该步骤及后续所有产物，并将该步骤作为起始步骤重跑
func (s *AppService) RerunTaskFromStep(id string, fromStep string) error {
	return s.RerunTaskWithRecipe(id, RerunTaskOptions{FromStep: fromStep})
}

// AutoResumeInterruptedTasks 扫描并在应用启动时自动恢复因停机/断电/崩溃而中断的任务
func (s *AppService) AutoResumeInterruptedTasks(ctx context.Context) {
	in := &task.ListTaskInput{}
	in.Size = 200
	tasks, _, err := s.taskCore.ListTasks(ctx, in)
	if err != nil {
		slog.Warn("自动检测中断任务失败", "err", err)
		return
	}
	for _, t := range tasks {
		if t.Status == 1 {
			slog.Info("自动恢复因程序停止中断的任务", "task_id", t.ID, "step", t.CurrentStep)
			job := taskPipelineJob(t)
			job.ResumeFrom = t.CurrentStep
			if err := s.scheduler.Submit(job); err != nil {
				slog.Error("自动恢复中断任务失败", "task_id", t.ID, "err", err)
			}
		}
	}
}

// DeleteTask 等待后台停止写入后，连同任务生成的文件一起删除记录。
func (s *AppService) DeleteTask(id string) error {
	if err := s.scheduler.CancelAndWait(context.Background(), id); err != nil {
		return err
	}
	_, err := s.taskCore.DeleteTask(context.Background(), id)
	return err
}

// ─── 系统文件与目录操作 ───────────────────────────────────

// PickFiles 弹出系统原生文件选择器，选取音视频文件。
func (s *AppService) PickFiles() ([]string, error) {
	if s.app == nil {
		return nil, fmt.Errorf("桌面上下文未初始化")
	}
	dialog := s.app.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false)

	dialog.AddFilter("音视频与字幕", "*.mp4;*.mkv;*.mov;*.avi;*.webm;*.flv;*.mp3;*.wav;*.m4a;*.srt")
	dialog.AddFilter("所有文件", "*.*")

	files, err := dialog.PromptForMultipleSelection()
	if err != nil {
		return nil, err
	}
	return files, nil
}

// OpenInFileManager 在系统访达或文件资源管理器中定位文件，包含路径存在性检查与防注入清洗。
func (s *AppService) OpenInFileManager(targetPath string) error {
	cleanPath := filepath.Clean(strings.TrimSpace(targetPath))
	if cleanPath == "" || cleanPath == "." {
		return fmt.Errorf("路径为空或无效")
	}
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %w", err)
	}
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("目录不存在: %s", absPath)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		if fileInfo.IsDir() {
			// 如果是目录，优先寻找目录下生成的成片 mp4 文件，如果有则直接定位高亮选中视频；否则直接进入并打开该目录
			entries, _ := os.ReadDir(absPath)
			var targetVideo string
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".mp4") {
					targetVideo = filepath.Join(absPath, e.Name())
					if e.Name() == "output.mp4" || strings.Contains(e.Name(), ".final.") {
						break
					}
				}
			}
			if targetVideo != "" {
				cmd = exec.Command("open", "-R", targetVideo)
			} else {
				cmd = exec.Command("open", absPath)
			}
		} else {
			cmd = exec.Command("open", "-R", absPath)
		}
	case "windows":
		if fileInfo.IsDir() {
			cmd = exec.Command("explorer", absPath)
		} else {
			cmd = exec.Command("explorer", "/select,", absPath)
		}
	default:
		if fileInfo.IsDir() {
			cmd = exec.Command("xdg-open", absPath)
		} else {
			cmd = exec.Command("xdg-open", filepath.Dir(absPath))
		}
	}
	if err := cmd.Start(); err != nil {
		slog.Error("打开文件管理器失败", "path", absPath, "err", err)
		return err
	}
	return nil
}

// ─── 配置与系统信息管理 ─────────────────────────────────────────────

// AppInfo 系统运行环境与版本信息。
type AppInfo struct {
	AppName      string `json:"app_name"`
	BuildVersion string `json:"build_version"`
	Platform     string `json:"platform"`
	Arch         string `json:"arch"`
}

// GetAppInfo 获取系统运行环境与真实构建版本信息。
func (s *AppService) GetAppInfo() AppInfo {
	s.mu.RLock()
	ver := s.bc.Runtime.BuildVersion
	s.mu.RUnlock()
	if ver == "" {
		ver = "0.1.0"
	}
	return AppInfo{
		AppName:      "Lark Studio",
		BuildVersion: ver,
		Platform:     runtime.GOOS,
		Arch:         runtime.GOARCH,
	}
}

// ConfigDTO 配置传输对象。
type ConfigDTO struct {
	Pipeline conf.Pipeline `json:"pipeline"`
	LLM      LLMDTO        `json:"llm"`
	TTS      conf.TTS      `json:"tts"`
	LipSync  conf.LipSync  `json:"lip_sync"`
	Runtime  conf.Runtime  `json:"runtime"`
}

type LLMDTO struct {
	Provider  string `json:"provider"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	DeepLXURL string `json:"deeplx_url"`
}

// GetConfig 获取当前系统所有配置，采用读锁保护并发安全。
func (s *AppService) GetConfig() ConfigDTO {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c := s.bc
	pipe := c.Pipeline
	pipe.DefaultOutputDir = conf.TaskOutputDir(pipe.DefaultOutputDir)
	return ConfigDTO{
		Pipeline: pipe,
		LLM: LLMDTO{
			Provider:  c.LLM.Provider,
			BaseURL:   c.LLM.BaseURL,
			APIKey:    c.LLM.APIKey,
			Model:     c.LLM.Model,
			DeepLXURL: c.LLM.DeepLXURL,
		},
		TTS:     c.TTS,
		LipSync: c.LipSync,
		Runtime: c.Runtime,
	}
}

// UpdateConfig 更新系统配置并自动持久化写入 config.toml，采用写锁保护。
func (s *AppService) UpdateConfig(updates map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	c := s.bc
	previousPipeline := c.Pipeline
	previousLLM := c.LLM
	if p, ok := updates["pipeline"].(map[string]any); ok {
		if v, ok := p["whisper_mode"].(string); ok {
			c.Pipeline.WhisperMode = v
		}
		if v, ok := p["workers"].(float64); ok {
			c.Pipeline.Workers = int(v)
		}
		if v, ok := p["whisper_model"].(string); ok {
			c.Pipeline.WhisperModel = v
		}
		if v, ok := p["asr_base_url"].(string); ok {
			c.Pipeline.ASRBaseURL = v
		}
		if v, ok := p["asr_api_key"].(string); ok {
			c.Pipeline.ASRAPIKey = v
		}
		if v, ok := p["asr_model"].(string); ok {
			c.Pipeline.ASRModel = v
		}
		if v, ok := p["ffmpeg_bin"].(string); ok {
			c.Pipeline.FFmpegBin = v
		}
		if v, ok := p["default_target_lang"].(string); ok {
			c.Pipeline.DefaultTargetLang = v
		}
		if v, ok := p["subtitle_output"].(string); ok {
			c.Pipeline.SubtitleOutput = v
		}
	}

	if l, ok := updates["llm"].(map[string]any); ok {
		if v, ok := l["provider"].(string); ok {
			if v == "local" {
				v = "openai"
			}
			c.LLM.Provider = v
		}
		if v, ok := l["base_url"].(string); ok {
			c.LLM.BaseURL = v
		}
		if v, ok := l["api_key"].(string); ok {
			c.LLM.APIKey = v
		}
		if v, ok := l["model"].(string); ok {
			c.LLM.Model = v
		}
		if v, ok := l["deeplx_url"].(string); ok {
			c.LLM.DeepLXURL = v
		}
	}

	if t, ok := updates["tts"].(map[string]any); ok {
		if v, ok := t["type"].(string); ok {
			c.TTS.Type = v
		}
		if v, ok := t["voice"].(string); ok {
			c.TTS.Voice = v
		}
		if v, ok := t["base_url"].(string); ok {
			c.TTS.BaseURL = v
		}
		if v, ok := t["api_key"].(string); ok {
			c.TTS.APIKey = v
		}
		if v, ok := t["model"].(string); ok {
			c.TTS.Model = v
		}
	}

	config := asrConfigFromPipeline(c.Pipeline)
	if err := asradapter.ValidateConfig(config); err != nil {
		c.Pipeline = previousPipeline
		c.LLM = previousLLM
		return err
	}
	if err := conf.WriteConfig(c, c.Runtime.ConfigPath); err != nil {
		c.Pipeline = previousPipeline
		c.LLM = previousLLM
		return err
	}
	if s.asrRouter != nil {
		s.asrRouter.SetConfig(config)
	}
	if s.scheduler != nil {
		client := llmadapter.NewRoutingClient(c.LLM.BaseURL, c.LLM.APIKey, c.LLM.Model, c.LLM.Provider, c.LLM.DeepLXURL)
		ready := strings.EqualFold(c.LLM.Provider, "openai") && strings.TrimSpace(c.LLM.BaseURL) != "" && strings.TrimSpace(c.LLM.Model) != ""
		s.scheduler.SetTranslationClient(client, ready)
	}
	return nil
}

// asrConfigFromPipeline 映射已持久化字段到运行时 ASR 路由配置。
func asrConfigFromPipeline(p conf.Pipeline) asradapter.Config {
	return asradapter.Config{
		Engine: p.WhisperMode, WhisperBin: p.WhisperBin, WhisperModel: p.WhisperModel,
		BaseURL: p.ASRBaseURL, APIKey: p.ASRAPIKey, Model: p.ASRModel,
	}
}

// ─── 术语库 ───────────────────────────────────────────────

// ListTerms 列出所有翻译术语。
func (s *AppService) ListTerms() ([]term.Term, error) {
	return s.termCore.ListAll(context.Background())
}

// SaveTerm 添加或更新术语。
func (s *AppService) SaveTerm(source, target string) error {
	ctx := context.Background()
	glossaries, err := s.termCore.ListGlossaries(ctx)
	if err != nil {
		return err
	}
	var glossaryID int64
	if len(glossaries) > 0 {
		glossaryID = glossaries[0].ID
	} else {
		g, err := s.termCore.CreateGlossary(ctx, "默认词库")
		if err != nil {
			return err
		}
		glossaryID = g.ID
	}
	_, err = s.termCore.Add(ctx, glossaryID, source, target, "")
	return err
}

// DeleteTerm 删除指定术语。
func (s *AppService) DeleteTerm(id int64) error {
	return s.termCore.Remove(context.Background(), id)
}

// ─── 任务配方管理 (SQLite 持久化) ──────────────────────────

// ListRecipes 获取所有保存的自定义工作流配方。
func (s *AppService) ListRecipes() ([]recipe.Recipe, error) {
	return s.recipeCore.List(context.Background())
}

// SaveRecipe 保存或更新工作流配方。
func (s *AppService) SaveRecipe(r recipe.Recipe) (*recipe.Recipe, error) {
	if r.ID == "" {
		r.ID = fmt.Sprintf("rcp_%d", time.Now().UnixMilli())
	}
	r.IsCustom = true
	if err := s.recipeCore.Save(context.Background(), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// DeleteRecipe 删除指定的工作流配方。
func (s *AppService) DeleteRecipe(id string) error {
	return s.recipeCore.Delete(context.Background(), id)
}

// ─── 内部校验与辅助 ───────────────────────────────────────

func (s *AppService) prepareTaskInput(in *task.CreateTaskInput) error {
	if in.InputPath == "" {
		return fmt.Errorf("输入文件路径不能为空")
	}
	if _, err := os.Stat(in.InputPath); err != nil {
		return fmt.Errorf("输入文件不存在: %s", in.InputPath)
	}
	if in.Mode < pipeline.ModeSubtitle || in.Mode > pipeline.ModeTextTranslate {
		in.Mode = pipeline.ModeDub
	}
	s.mu.RLock()
	config := asrConfigFromPipeline(s.bc.Pipeline)
	s.mu.RUnlock()
	if err := pipeline.ValidateRecognitionConfig(in.InputPath, in.OutputDir, in.Mode, config); err != nil {
		return err
	}
	if in.OutputDir == "" {
		baseName := strings.TrimSuffix(filepath.Base(in.InputPath), filepath.Ext(in.InputPath))
		in.OutputDir = filepath.Join(conf.TaskOutputDir(s.bc.Pipeline.DefaultOutputDir), baseName+"_vdub")
	}
	if err := os.MkdirAll(in.OutputDir, 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 保存统一命名的源文件副本，后续步骤只操作任务目录中的文件。
	if err := stageSourceFile(in); err != nil {
		return fmt.Errorf("准备工作源文件失败: %w", err)
	}
	if in.TargetLang == "" {
		in.TargetLang = s.bc.Pipeline.DefaultTargetLang
		if in.TargetLang == "" {
			in.TargetLang = "zh-CN"
		}
	}
	if in.SourceLang == "" {
		in.SourceLang = "auto"
	}
	// 重点修复：自建 local 或空值，合乎规范地归一为 openai 或 bing
	if in.Translator == "" || in.Translator == "local" {
		if in.Translator == "local" || s.bc.LLM.Provider == "openai" {
			in.Translator = "openai"
		} else if s.bc.LLM.Provider != "" {
			in.Translator = s.bc.LLM.Provider
		} else {
			in.Translator = "google"
		}
	}
	if in.OutputContent == "" {
		in.OutputContent = "bilingual"
	}
	if in.TTSEngine == "" {
		in.TTSEngine = s.bc.TTS.Type
		if in.TTSEngine == "" {
			in.TTSEngine = "edge"
		}
	}
	if in.TTSVoice == "" {
		in.TTSVoice = s.bc.TTS.Voice
	}
	if in.SpeechRate == 0 {
		in.SpeechRate = 1
	}
	if in.SubtitleOutput == "" {
		in.SubtitleOutput = s.bc.Pipeline.SubtitleOutput
		if in.SubtitleOutput == "" {
			in.SubtitleOutput = "soft"
		}
	}
	return nil
}

func taskModeTitle(mode int) string {
	switch mode {
	case pipeline.ModeSubtitle:
		return "转写字幕(1)"
	case pipeline.ModeTranslate:
		return "双语翻译(1-2)"
	case pipeline.ModeDub:
		return "视频配音成片(1-2-3-4)"
	case pipeline.ModeDubOnly:
		return "AI朗读配音(3)"
	case pipeline.ModeDirectDub:
		return "原文配音成片(1-3-4)"
	case pipeline.ModeTextTranslate:
		return "文本翻译(2)"
	default:
		return fmt.Sprintf("模式 %d", mode)
	}
}

func taskPipelineJob(t *task.Task) pipeline.Job {
	return pipeline.Job{
		TaskID:         t.ID,
		InputPath:      t.InputPath,
		OutputDir:      t.OutputDir,
		Mode:           t.Mode,
		SourceLang:     t.SourceLang,
		TargetLang:     t.TargetLang,
		Translator:     t.Translator,
		OutputContent:  t.OutputContent,
		TTSEngine:      t.TTSEngine,
		TTSVoice:       t.TTSVoice,
		SpeechRate:     t.SpeechRate,
		SubtitleOutput: t.SubtitleOutput,
		ResumeFrom:     t.CurrentStep,
	}
}

// ─── Whisper 模型与运行时管理 ──────────────────────────────

// ListWhisperModels 获取所有可用的 GGML 模型及下载安装状态。
func (s *AppService) ListWhisperModels() []api.ModelListOutput {
	return api.ListAllWhisperModels()
}

// DownloadWhisperModel 立即启动模型异步下载任务，失败时自动切换下载源。
func (s *AppService) DownloadWhisperModel(name string) error {
	_, err := api.StartWhisperModelDownload(name, s.hub)
	return err
}

// DeleteWhisperModel 从统一 models 目录删除指定模型。
func (s *AppService) DeleteWhisperModel(name string) error {
	return api.DeleteWhisperModelFile(name)
}

// InspectWhisperRuntime 探测当前系统 whisper.cpp 运行时环境。
func (s *AppService) InspectWhisperRuntime() whisperadapter.RuntimeInfo {
	s.mu.RLock()
	bin := s.bc.Pipeline.WhisperBin
	s.mu.RUnlock()
	return whisperadapter.InspectRuntime(bin)
}

// InstallWhisperRuntime 自动下载并解包安装平台对应的 whisper.cpp 运行时。
func (s *AppService) InstallWhisperRuntime() error {
	ctx := context.Background()
	tag := s.bc.Runtime.BuildVersion
	if tag == "" {
		tag = "v1.0.0"
	}
	return whisperadapter.InstallRuntime(ctx, whisperadapter.RuntimeInstallOptions{
		ReleaseTag: tag,
		DataDir:    conf.StudioDir(),
	}, func(line string) {
		if s.hub != nil {
			s.hub.Broadcast(ws.NewMessage("whisper_runtime_log", map[string]any{
				"message": line,
			}))
		}
	})
}

// SetActiveWhisperModel 将模型设为全局默认使用的模型。
func (s *AppService) SetActiveWhisperModel(nameOrPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	targetPath, err := whisperadapter.ResolveModel(nameOrPath)
	if err != nil {
		return err
	}
	previousModel := s.bc.Pipeline.WhisperModel
	s.bc.Pipeline.WhisperModel = targetPath
	if err := conf.WriteConfig(s.bc, s.bc.Runtime.ConfigPath); err != nil {
		s.bc.Pipeline.WhisperModel = previousModel
		return err
	}
	if s.asrRouter != nil {
		s.asrRouter.SetConfig(asrConfigFromPipeline(s.bc.Pipeline))
	}
	return nil
}

// TestOpenAITranslate 测试 OpenAI 兼容端点的连通性与模型响应。
func (s *AppService) TestOpenAITranslate(baseURL, apiKey, model string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return llmadapter.TestOpenAIConnection(ctx, baseURL, apiKey, model)
}

// TestOpenAITTS 验证用户自定义的 OpenAI 兼容模型与音色，并返回可直接播放的音频。
func (s *AppService) TestOpenAITTS(baseURL, apiKey, model, voice, text string) (string, error) {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(model) == "" || strings.TrimSpace(voice) == "" || strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("TTS 地址、模型、音色和试听文本均不能为空")
	}
	if len(model) > 200 || len(voice) > 200 || len(text) > 500 {
		return "", fmt.Errorf("TTS 模型、音色或试听文本超出长度限制")
	}
	if len(baseURL) > 2048 || len(apiKey) > 4096 {
		return "", fmt.Errorf("TTS 地址或密钥超出长度限制")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	audio, contentType, err := ttsadapter.NewOpenAITTS(baseURL, apiKey, model, voice).SynthesizeBytes(ctx, text, voice, 1)
	if err != nil {
		return "", err
	}
	if contentType == "" {
		contentType = "audio/mpeg"
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "audio/") {
		return "", fmt.Errorf("TTS 接口返回的内容不是音频")
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(audio), nil
}

// stageSourceFile 与 HTTP、命令行入口共用相同的源文件暂存规则。
func stageSourceFile(in *task.CreateTaskInput) error {
	return api.StageSourceFile(in)
}
