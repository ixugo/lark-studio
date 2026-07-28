package api

import (
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

// TestPrepareTaskInputSnapshotsConfig 验证创建任务时会展开全局配置。
func TestPrepareTaskInputSnapshotsConfig(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "sample.mp4")
	if err := os.WriteFile(inputPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := conf.DefaultConfig()
	cfg.LLM.Provider = "deeplx"
	cfg.TTS.Type = "openai"
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
		Translator:     "google",
		OutputContent:  "bilingual",
		TTSEngine:      "edge",
		SpeechRate:     1,
		SubtitleOutput: "burn",
	}
	if err := validateTaskParameters(input); err == nil {
		t.Fatal("无效翻译引擎应返回错误")
	}
}
