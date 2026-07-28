package app

import (
	"context"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"github.com/ixugo/vdub/internal/core/task/store/taskdb"
	"gorm.io/gorm"
)

// TestDBNotifierProgressNeverRegresses 验证并发翻译和配音不会让总进度或当前步骤倒退。
func TestDBNotifierProgressNeverRegresses(t *testing.T) {
	core := newNotifierTestCore(t)
	created, err := core.CreateTask(context.Background(), &task.CreateTaskInput{
		InputPath: "video.mp4",
		OutputDir: "output",
		Mode:      pipeline.ModeDub,
	})
	if err != nil {
		t.Fatal(err)
	}
	notifier := newDBNotifier(core, nil, &conf.Bootstrap{})

	notifier.OnStepStart(created.ID, pipeline.StepTranslate)
	notifier.OnProgress(created.ID, pipeline.StepTranslate, 80)
	notifier.OnStepStart(created.ID, pipeline.StepTTS)
	notifier.OnProgress(created.ID, pipeline.StepTTS, 20)
	before := getNotifierTask(t, core, created.ID)
	notifier.OnProgress(created.ID, pipeline.StepTranslate, 100)
	notifier.OnProgress(created.ID, pipeline.StepTTS, 10)
	after := getNotifierTask(t, core, created.ID)

	if after.Progress < before.Progress {
		t.Fatalf("总进度倒退：%d → %d", before.Progress, after.Progress)
	}
	if after.CurrentStep != pipeline.StepTTS || after.StepProgress != 20 {
		t.Fatalf("当前步骤倒退：step=%s progress=%d", after.CurrentStep, after.StepProgress)
	}
}

// newNotifierTestCore 创建互不共享的 SQLite 测试领域。
func newNotifierTestCore(t *testing.T) task.Core {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn))
	if err != nil {
		t.Fatal(err)
	}
	return task.NewCore(taskdb.NewDB(db).AutoMigrate(true))
}

// getNotifierTask 读取测试任务，失败即终止用例。
func getNotifierTask(t *testing.T, core task.Core, id string) *task.Task {
	t.Helper()
	item, err := core.GetTask(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return item
}
