package taskdb

import (
	"context"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ixugo/vdub/internal/core/task"
	"gorm.io/gorm"
)

// TestTaskRecipeSnapshotPersistence 验证配方参数能完整写入并读取 SQLite。
func TestTaskRecipeSnapshotPersistence(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn))
	if err != nil {
		t.Fatal(err)
	}
	core := task.NewCore(NewDB(db).AutoMigrate(true))
	created, err := core.CreateTask(context.Background(), &task.CreateTaskInput{
		InputPath:      "sample.mp4",
		OutputDir:      "output",
		Mode:           3,
		TargetLang:     "zh-CN",
		SourceLang:     "en",
		Translator:     "bing",
		OutputContent:  "translated",
		TTSEngine:      "edge",
		TTSVoice:       "zh-CN-XiaoxiaoNeural",
		SpeechRate:     1.15,
		SubtitleOutput: "none",
		RecipeName:     "英文配音",
	})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := core.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.SourceLang != "en" || stored.Translator != "bing" ||
		stored.TTSVoice != "zh-CN-XiaoxiaoNeural" || stored.SpeechRate != 1.15 ||
		stored.SubtitleOutput != "none" || stored.RecipeName != "英文配音" {
		t.Fatalf("任务快照不完整：%+v", stored)
	}
}
