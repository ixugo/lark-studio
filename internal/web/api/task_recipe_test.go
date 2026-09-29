package api

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/task"
	"gorm.io/gorm"
)

// TestNewTaskCoreMigratesLegacySchema 验证旧数据库启动后会补齐任务进度与配方字段。
func TestNewTaskCoreMigratesLegacySchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(
		"CREATE TABLE tasks (id TEXT PRIMARY KEY, input_path TEXT NOT NULL, output_dir TEXT NOT NULL)",
	).Error; err != nil {
		t.Fatal(err)
	}

	NewTaskCore(db)

	for _, column := range []string{"step_progress", "translator", "subtitle_output"} {
		if !db.Migrator().HasColumn(&task.Task{}, column) {
			t.Fatalf("旧任务表未补齐字段 %s", column)
		}
	}
}

// TestPrepareTaskInputDefaultsToBing 验证未显式选择翻译服务时创建任务固定使用必应。
func TestPrepareTaskInputDefaultsToBing(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "sample.mp4")
	if err := os.WriteFile(inputPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := conf.DefaultConfig()
	cfg.Pipeline.WhisperMode = "openai"
	modelServer := newTaskModelCatalogFixture(t, "test-model", cfg.TTS.Model)
	cfg.Pipeline.ASRBaseURL = modelServer.URL + "/v1"
	cfg.Pipeline.DefaultOutputDir = filepath.Join(t.TempDir(), "results")
	cfg.Pipeline.ASRModel = "test-model"
	api := TaskAPI{conf: &cfg}
	input := &task.CreateTaskInput{InputPath: inputPath, Mode: 2}

	if err := api.prepareTaskInput(input); err != nil {
		t.Fatal(err)
	}
	if input.Translator != "bing" {
		t.Fatalf("默认翻译服务 = %q，期望 bing", input.Translator)
	}
}

// TestTaskPipelineJobPreservesRecipe 验证调度器收到的参数与任务配方快照完全一致。
func TestTaskPipelineJobPreservesRecipe(t *testing.T) {
	item := &task.Task{
		ID: "task-1", Mode: 3, SourceLang: "en", TargetLang: "zh-CN",
		Translator: "bing", OutputContent: "translated", TTSEngine: "edge",
		TTSVoice: "zh-CN-XiaoxiaoNeural", SpeechRate: 1.2, SubtitleOutput: "none",
	}
	job := taskPipelineJob(item)
	if job.Translator != "bing" || job.TTSEngine != "edge" || job.SubtitleOutput != "none" {
		t.Fatalf("配方快照丢失：%+v", job)
	}
	if job.SourceLang != "en" || job.TargetLang != "zh-CN" || job.SpeechRate != 1.2 {
		t.Fatalf("语言或语速快照丢失：%+v", job)
	}
}

// TestAppendCreationLog 验证任务从不同入口创建时都会拥有独立首条日志。
func TestAppendCreationLog(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "task.db")))
	if err != nil {
		t.Fatal(err)
	}
	core := NewTaskCore(db)
	item, err := core.CreateTask(context.Background(), &task.CreateTaskInput{Mode: 3})
	if err != nil {
		t.Fatal(err)
	}
	api := TaskAPI{taskCore: core}
	if err := api.appendCreationLog(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	logs, err := core.ListRecentTaskLogs(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Message != "任务创建，模式：配音成片" {
		t.Fatalf("首条任务日志错误：%+v", logs)
	}
}

// TestPrepareTaskInputSnapshotsConfig 验证创建任务时会展开全局配置。
func TestPrepareTaskInputSnapshotsConfig(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "sample.mp4")
	if err := os.WriteFile(inputPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := conf.DefaultConfig()
	cfg.Pipeline.WhisperMode = "openai"
	modelServer := newTaskModelCatalogFixture(t, "test-model", cfg.TTS.Model)
	cfg.Pipeline.ASRBaseURL = modelServer.URL + "/v1"
	cfg.Pipeline.DefaultOutputDir = filepath.Join(t.TempDir(), "results")
	cfg.Pipeline.ASRModel = "test-model"
	cfg.LLM.Provider = "deeplx"
	cfg.TTS.Type = "openai"
	cfg.TTS.BaseURL = modelServer.URL + "/v1"
	cfg.TTS.Voice = "alloy"
	cfg.Pipeline.SubtitleOutput = "file"
	api := TaskAPI{conf: &cfg}
	input := &task.CreateTaskInput{InputPath: inputPath, Mode: 3}

	if err := api.prepareTaskInput(input); err != nil {
		t.Fatal(err)
	}
	if input.SourceLang != "auto" || input.TargetLang != "zh-CN" {
		t.Fatalf("语言快照错误：%+v", input)
	}
	if input.Translator != "deeplx" || input.TTSEngine != "openai" {
		t.Fatalf("引擎快照错误：%+v", input)
	}
	if input.TTSVoice != "alloy" || input.SpeechRate != 1 {
		t.Fatalf("音色快照错误：%+v", input)
	}
	if input.SubtitleOutput != "file" || input.OutputContent != "bilingual" {
		t.Fatalf("输出快照错误：%+v", input)
	}
}

// TestValidateTaskParametersRejectsInvalidRecipe 验证无效配方不会进入调度器。
func TestValidateTaskParametersRejectsInvalidRecipe(t *testing.T) {
	input := &task.CreateTaskInput{
		SourceLang:     "auto",
		TargetLang:     "zh-CN",
		Translator:     "invalid-engine",
		OutputContent:  "bilingual",
		TTSEngine:      "edge",
		SpeechRate:     1,
		SubtitleOutput: "burn",
	}
	if err := validateTaskParameters(input); err == nil {
		t.Fatal("无效翻译引擎应返回错误")
	}
}

// newTaskModelCatalogFixture 保留真实模型标识，只替换测试中不可达的服务地址。
func newTaskModelCatalogFixture(t *testing.T, models ...string) *httptest.Server {
	t.Helper()
	items := make([]map[string]string, 0, len(models))
	for _, model := range models {
		items = append(items, map[string]string{"id": model})
	}
	body, err := json.Marshal(map[string]any{"data": items})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		if _, err := w.Write(body); err != nil {
			t.Errorf("写测试模型目录失败: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}
