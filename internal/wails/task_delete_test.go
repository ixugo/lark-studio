package wails

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"github.com/ixugo/vdub/internal/core/task/store/taskdb"
	"gorm.io/gorm"
)

// TestDesktopDeleteTaskWithMissingSharedOutput 验证桌面删除入口经过调度停止后能删除失去产物的任务。
func TestDesktopDeleteTaskWithMissingSharedOutput(t *testing.T) {
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "tasks.db")))
	if err != nil {
		t.Fatal(err)
	}
	core := task.NewCore(taskdb.NewDB(db).AutoMigrate(true))
	output := filepath.Join(dir, "already-removed")
	var items []*task.Task
	for range 2 {
		item, err := core.CreateTask(t.Context(), &task.CreateTaskInput{InputPath: filepath.Join(dir, "source.mp4"), OutputDir: output})
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, item)
	}
	svc := &AppService{taskCore: core, scheduler: pipeline.NewScheduler(nil, nil)}
	if err := svc.DeleteTask(items[0].ID); err != nil {
		t.Fatalf("桌面入口不应阻止删除已无产物的任务：%v", err)
	}
	if _, err := core.GetTask(t.Context(), items[0].ID); err == nil {
		t.Fatal("所选任务仍在数据库中")
	}
	if _, err := core.GetTask(t.Context(), items[1].ID); err != nil {
		t.Fatalf("另一任务记录应保留：%v", err)
	}
}
