package taskdb

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ixugo/vdub/internal/core/task"
	"gorm.io/gorm"
)

// TestListStepsByTask 验证步骤查询只返回指定任务的数据。
func TestListStepsByTask(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:step-list?mode=memory&cache=shared"))
	if err != nil {
		t.Fatal(err)
	}
	core := task.NewCore(NewDB(db).AutoMigrate(true))
	ctx := context.Background()

	for _, item := range []task.CreateStepInput{
		{TaskID: "task-a", Name: "whisper"},
		{TaskID: "task-a", Name: "translate"},
		{TaskID: "task-b", Name: "tts"},
	} {
		if err := core.UpsertStep(ctx, item.TaskID, item.Name, func(step *task.Step) {
			step.Status = 2
		}); err != nil {
			t.Fatal(err)
		}
	}

	input := task.ListStepInput{TaskID: "task-a"}
	input.Size = 20
	items, total, err := core.ListSteps(ctx, &input)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("步骤数 = %d/%d，期望 2/2", total, len(items))
	}
	for _, item := range items {
		if item.TaskID != "task-a" {
			t.Fatalf("混入其他任务步骤：%+v", item)
		}
	}
}
