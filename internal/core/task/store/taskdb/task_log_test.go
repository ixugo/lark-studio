package taskdb

import (
	"context"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ixugo/vdub/internal/core/task"
	"gorm.io/gorm"
)

// TestTaskLogRetention 验证日志按任务隔离、最多保留 1000 行且返回时间正序。
func TestTaskLogRetention(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"))
	if err != nil {
		t.Fatal(err)
	}
	core := task.NewCore(NewDB(db).AutoMigrate(true))
	ctx := context.Background()

	for index := 0; index < 1005; index++ {
		message := fmt.Sprintf("日志-%04d", index)
		if _, err := core.AppendTaskLog(ctx, "task-a", "info", "whisper", message); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := core.AppendTaskLog(ctx, "task-b", "error", "tts", "独立任务"); err != nil {
		t.Fatal(err)
	}

	assertTaskALogs(t, core, ctx)
	assertTaskBLogs(t, core, ctx)
}

// assertTaskALogs 检查超限任务仅保留末 1000 行。
func assertTaskALogs(t *testing.T, core task.Core, ctx context.Context) {
	t.Helper()
	items, err := core.ListRecentTaskLogs(ctx, "task-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1000 {
		t.Fatalf("日志数 = %d，期望 1000", len(items))
	}
	if items[0].Message != "日志-0005" || items[999].Message != "日志-1004" {
		t.Fatalf("日志范围 = %q..%q", items[0].Message, items[999].Message)
	}
	for index := 1; index < len(items); index++ {
		if items[index-1].ID >= items[index].ID {
			t.Fatalf("日志顺序错误：%d >= %d", items[index-1].ID, items[index].ID)
		}
	}
}

// assertTaskBLogs 检查另一任务的日志未被混入。
func assertTaskBLogs(t *testing.T, core task.Core, ctx context.Context) {
	t.Helper()
	items, err := core.ListRecentTaskLogs(ctx, "task-b")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Message != "独立任务" {
		t.Fatalf("任务隔离失败：%+v", items)
	}
}
