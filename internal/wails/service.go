package wails

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"github.com/ixugo/vdub/internal/core/term"
)

// AppService 聚合所有暴露给前端界面的 Go 接口方法。
type AppService struct {
	app       *application.App
	bc        *conf.Bootstrap
	taskCore  task.Core
	termCore  term.Core
	scheduler *pipeline.Scheduler
}

// NewAppService 创建应用服务。
func NewAppService(
	app *application.App,
	bc *conf.Bootstrap,
	taskCore task.Core,
	termCore term.Core,
	scheduler *pipeline.Scheduler,
) *AppService {
	return &AppService{
		app:       app,
		bc:        bc,
		taskCore:  taskCore,
		termCore:  termCore,
		scheduler: scheduler,
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

// ResumeTask 恢复已暂停的任务。
func (s *AppService) ResumeTask(id string) error {
	ctx := context.Background()
	item, err := s.taskCore.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if item.Status != 2 {
		return fmt.Errorf("任务不在暂停状态")
	}
	if err := s.taskCore.SetTaskStatus(ctx, id, func(t *task.Task) {
		t.Status = 1
		t.Error = ""
	}); err != nil {
		return err
	}
	return s.scheduler.Submit(taskPipelineJob(item))
}

// DeleteTask 删除任务。
func (s *AppService) DeleteTask(id string) error {
	s.scheduler.Pause(id)
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

// OpenInFileManager 在系统访达/资源管理器中定位文件或目录。
func (s *AppService) OpenInFileManager(targetPath string) error {
	if targetPath == "" {
		return fmt.Errorf("路径为空")
	}
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		absPath = targetPath
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", absPath)
	case "windows":
		cmd = exec.Command("explorer", "/select,", absPath)
	default:
		cmd = exec.Command("xdg-open", filepath.Dir(absPath))
	}
	if err := cmd.Start(); err != nil {
		slog.Error("打开文件管理器失败", "path", absPath, "err", err)
		return err
	}
	return nil
}

// ─── 配置管理 ─────────────────────────────────────────────

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

// GetConfig 获取当前系统所有配置。
func (s *AppService) GetConfig() ConfigDTO {
	c := s.bc
	return ConfigDTO{
		Pipeline: c.Pipeline,
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

// UpdateConfig 更新系统配置并自动持久化写入 config.toml。
func (s *AppService) UpdateConfig(updates map[string]any) error {
	c := s.bc
	if p, ok := updates["pipeline"].(map[string]any); ok {
		if v, ok := p["workers"].(float64); ok {
			c.Pipeline.Workers = int(v)
		}
		if v, ok := p["whisper_model"].(string); ok {
			c.Pipeline.WhisperModel = v
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

	return conf.WriteConfig(c, c.Runtime.ConfigPath)
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

// ─── 内部校验与辅助 ───────────────────────────────────────

func (s *AppService) prepareTaskInput(in *task.CreateTaskInput) error {
	if in.InputPath == "" {
		return fmt.Errorf("视频文件路径不能为空")
	}
	if _, err := os.Stat(in.InputPath); err != nil {
		return fmt.Errorf("视频文件不存在: %s", in.InputPath)
	}
	if in.Mode < pipeline.ModeSubtitle || in.Mode > pipeline.ModeDub {
		in.Mode = pipeline.ModeDub
	}
	if in.OutputDir == "" {
		baseName := strings.TrimSuffix(filepath.Base(in.InputPath), filepath.Ext(in.InputPath))
		in.OutputDir = filepath.Join(filepath.Dir(in.InputPath), baseName+"_vdub")
	}
	if err := os.MkdirAll(in.OutputDir, 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
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
			in.Translator = "bing"
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
			in.SubtitleOutput = "burn"
		}
	}
	return nil
}

func taskModeTitle(mode int) string {
	switch mode {
	case pipeline.ModeSubtitle:
		return "转写字幕"
	case pipeline.ModeTranslate:
		return "双语翻译"
	case pipeline.ModeDub:
		return "翻译配音"
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
