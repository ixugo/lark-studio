package wails

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"github.com/ixugo/vdub/internal/core/task/store/taskdb"
	"github.com/ixugo/vdub/internal/taskfile"
	"gorm.io/gorm"
)

func batchService(t *testing.T) (*AppService, []string, *gorm.DB) {
	t.Helper()
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "tasks.db")))
	if err != nil {
		t.Fatal(err)
	}
	bc := conf.DefaultConfig()
	bc.Pipeline.DefaultOutputDir = filepath.Join(dir, "results")
	bc.Pipeline.WhisperMode = "openai"
	modelServer := newRemoteModelTestServer(t, "test-model")
	bc.Pipeline.ASRBaseURL = modelServer.URL + "/v1"
	bc.Pipeline.ASRModel = "test-model"
	core := task.NewCore(taskdb.NewDB(db).AutoMigrate(true))
	scheduler := pipeline.NewScheduler(nil, nil)
	t.Cleanup(scheduler.Stop)
	svc := &AppService{bc: &bc, taskCore: core, scheduler: scheduler}
	var inputs []string
	for _, name := range []string{"a", "b"} {
		path := filepath.Join(dir, name, "同名 视频.mp4")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, path)
	}
	return svc, inputs, db
}

func TestBatchLayoutNamesAndDeletion(t *testing.T) {
	svc, inputs, db := batchService(t)
	created, err := svc.BatchCreateTasks(inputs, task.CreateTaskInput{Mode: pipeline.ModeSubtitle})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 || created[0].BatchDir != created[1].BatchDir || created[0].OutputDir == created[1].OutputDir || created[0].ResultPath == created[1].ResultPath {
		t.Fatalf("批次未隔离: %+v", created)
	}
	for i, item := range created {
		if item.OriginalName != filepath.Base(inputs[i]) || filepath.Dir(item.OutputDir) != item.BatchDir || filepath.Dir(item.ResultPath) != item.BatchDir {
			t.Fatalf("标题或成片位置错误: %+v", item)
		}
		if data, err := os.ReadFile(item.InputPath); err != nil || string(data) != []string{"a", "b"}[i] {
			t.Fatalf("同名输入覆盖: %q %v", data, err)
		}
		if err := os.WriteFile(item.ResultPath, []byte("result"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(item.OutputDir, "arbitrary.intermediate"), []byte("work"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := taskfile.FinalizeResult(item.OutputDir); err != nil {
			t.Fatal(err)
		}
		item.ResultPath, err = taskfile.ResultVideoPath(item.OutputDir)
		if err != nil {
			t.Fatal(err)
		}
		expected := "同名 视频.mp4"
		if i == 1 {
			expected = "同名 视频_1.mp4"
		}
		if filepath.Base(item.ResultPath) != expected {
			t.Fatal(item.ResultPath)
		}
		fetched, err := svc.GetTask(item.ID)
		if err != nil || fetched.OriginalName != item.OriginalName || fetched.ResultPath != item.ResultPath {
			t.Fatalf("数据库重读丢失名称: %+v %v", fetched, err)
		}
	}
	listed, err := svc.ListTasks()
	if err != nil || len(listed) != 2 || listed[0].OriginalName != "同名 视频.mp4" {
		t.Fatalf("看板列表原名丢失: %v", err)
	}
	for _, column := range []string{"original_name", "batch_dir", "result_path"} {
		if db.Migrator().HasColumn(&task.Task{}, column) {
			t.Fatalf("展示字段不应新增数据库列: %s", column)
		}
	}
	again, err := svc.BatchCreateTasks(inputs, task.CreateTaskInput{Mode: pipeline.ModeSubtitle})
	if err != nil || again[0].BatchDir == created[0].BatchDir {
		t.Fatalf("再次提交复用批次: %v", err)
	}
	if _, err := svc.taskCore.DeleteTask(t.Context(), created[0].ID); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{created[0].InputPath, created[0].OutputDir, created[0].ResultPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("所选任务产物未清除: %q %v", path, err)
		}
	}
	for _, path := range append(inputs, created[1].InputPath, created[1].ResultPath) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("删除影响同批任务或原文件: %q %v", path, err)
		}
	}
	if _, err := svc.taskCore.DeleteTask(t.Context(), created[1].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(created[0].BatchDir); !os.IsNotExist(err) {
		t.Fatal("空批次目录未清除")
	}
}

func TestSingleSourceLayoutAndDuplicateSubmission(t *testing.T) {
	svc, inputs, _ := batchService(t)
	var first *task.Task
	for range 2 {
		items, err := svc.BatchCreateTasks(inputs[:1], task.CreateTaskInput{Mode: pipeline.ModeSubtitle})
		if err != nil {
			t.Fatal(err)
		}
		item := items[0]
		if filepath.Base(item.ResultPath) != filepath.Base(item.OutputDir)+".mp4" || filepath.Dir(item.ResultPath) != filepath.Dir(item.OutputDir) {
			t.Fatalf("单源成片位置错误: %+v", item)
		}
		if first != nil && first.BatchDir == item.BatchDir {
			t.Fatal("重复提交覆盖旧任务")
		}
		first = item
	}
}

func TestExternalSubtitleStaysInWorkDirectory(t *testing.T) {
	svc, inputs, _ := batchService(t)
	sub := filepath.Join(t.TempDir(), "字幕.srt")
	if err := os.WriteFile(sub, []byte("1\n00:00:00,000 --> 00:00:01,000\nhello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	item, err := svc.MergeSubtitle(MergeSubtitleInput{VideoPath: inputs[0], PrimarySubPath: sub})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(item.OutputDir, "src.srt")); err != nil {
		t.Fatal("合成字幕未放在工作目录")
	}
	if filepath.Base(item.ResultPath) != filepath.Base(item.OutputDir)+".mp4" {
		t.Fatal("字幕合成成片位置错误")
	}
}
