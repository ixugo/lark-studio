package taskdb

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ixugo/vdub/internal/core/task"
	"gorm.io/gorm"
)

// deletionFixture 使用独立数据库和文件目录，确保测试不会触及用户任务。
func deletionFixture(t *testing.T) (task.Core, *task.Task) {
	t.Helper()
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "tasks.db")))
	if err != nil {
		t.Fatal(err)
	}
	core := task.NewCore(NewDB(db).AutoMigrate(true))
	input, output := filepath.Join(dir, "source.mp4"), filepath.Join(dir, "output")
	if err := os.WriteFile(input, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(output, 0o755); err != nil {
		t.Fatal(err)
	}
	item, err := core.CreateTask(t.Context(), &task.CreateTaskInput{InputPath: input, OutputDir: output})
	if err != nil {
		t.Fatal(err)
	}
	return core, item
}

// TestDeleteTaskRemovesGeneratedFiles 验证实际数据库记录和中间、最终产物一起删除，原素材保留。
func TestDeleteTaskRemovesGeneratedFiles(t *testing.T) {
	core, item := deletionFixture(t)
	files := []string{"src.srt", "trans.srt", "trans.txt", "raw.mp3", "dub.mp3", "concat_list.txt", "silence_0.wav", "source.final.mp4", "source.trans.mp4", "source.sub.mp4", "lipsync.mp4", "task.log"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(item.OutputDir, name), []byte("generated"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(item.OutputDir, "audio_segs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(item.OutputDir, "audio_segs", "0.wav"), []byte("speech"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := core.DeleteTask(t.Context(), item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := core.GetTask(t.Context(), item.ID); err == nil {
		t.Error("任务记录仍存在")
	}
	if _, err := os.Stat(item.OutputDir); !os.IsNotExist(err) {
		t.Fatalf("任务产物目录仍存在：%v", err)
	}
	if data, err := os.ReadFile(item.InputPath); err != nil || string(data) != "original" {
		t.Fatalf("原素材受损：%v", err)
	}
	if _, err := core.DeleteTask(t.Context(), item.ID); err != nil {
		t.Fatalf("重复删除应成功：%v", err)
	}
}

// TestDeleteTaskRemovesStagedCopy 检查桌面任务生成的源视频副本也被删除，真正原文件不动。
func TestDeleteTaskRemovesStagedCopy(t *testing.T) {
	core, item := deletionFixture(t)
	original := item.InputPath
	staged := filepath.Join(item.OutputDir, "staged.mp4")
	if err := os.WriteFile(staged, []byte("copy"), 0o644); err != nil {
		t.Fatal(err)
	}
	meta, err := json.Marshal(map[string]string{"original_path": original, "staged_path": staged, "staged_name": "staged.mp4"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(item.OutputDir, "source_meta.json"), meta, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := core.SetTaskStatus(t.Context(), item.ID, func(value *task.Task) { value.InputPath = staged }); err != nil {
		t.Fatal(err)
	}
	if _, err := core.DeleteTask(t.Context(), item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(item.OutputDir); !os.IsNotExist(err) {
		t.Fatalf("工作副本未清除：%v", err)
	}
	if data, err := os.ReadFile(original); err != nil || string(data) != "original" {
		t.Fatalf("原素材受损：%v", err)
	}
}

// TestDeleteTaskProtectsInputInArtifactDirectory 拒绝把原文件或包含原文件的目录当作产物删除。
func TestDeleteTaskProtectsInputInArtifactDirectory(t *testing.T) {
	core, item := deletionFixture(t)
	input := filepath.Join(item.OutputDir, "src.srt")
	if err := os.WriteFile(input, []byte("original subtitles"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := core.SetTaskStatus(t.Context(), item.ID, func(value *task.Task) { value.InputPath = input }); err != nil {
		t.Fatal(err)
	}
	if _, err := core.DeleteTask(t.Context(), item.ID); err == nil {
		t.Fatal("应拒绝删除原字幕")
	}
	if data, err := os.ReadFile(input); err != nil || string(data) != "original subtitles" {
		t.Fatalf("原字幕受损：%v", err)
	}
	if _, err := core.GetTask(t.Context(), item.ID); err != nil {
		t.Fatal("失败时应保留任务")
	}
}

// TestDeleteTaskPreservesUnrelatedFiles 验证自选输出目录里的其他文件不会被误删。
func TestDeleteTaskPreservesUnrelatedFiles(t *testing.T) {
	core, item := deletionFixture(t)
	for _, name := range []string{"src.srt", "my-notes.txt", "other.final.mp4"} {
		if err := os.WriteFile(filepath.Join(item.OutputDir, name), []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := core.DeleteTask(t.Context(), item.ID); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"my-notes.txt", "other.final.mp4"} {
		if _, err := os.Stat(filepath.Join(item.OutputDir, name)); err != nil {
			t.Fatalf("无关文件被删除：%s %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(item.OutputDir, "src.srt")); !os.IsNotExist(err) {
		t.Error("任务字幕仍存在")
	}
}

// TestDeleteTaskRetainsRecordOnCleanupFailure 保留失败任务记录，让用户能够重试清理。
func TestDeleteTaskRetainsRecordOnCleanupFailure(t *testing.T) {
	core, item := deletionFixture(t)
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := core.SetTaskStatus(t.Context(), item.ID, func(value *task.Task) { value.OutputDir = file }); err != nil {
		t.Fatal(err)
	}
	if _, err := core.DeleteTask(t.Context(), item.ID); err == nil {
		t.Fatal("产物清理失败不应返回成功")
	}
	if _, err := core.GetTask(t.Context(), item.ID); err != nil {
		t.Fatalf("清理失败却删除了记录：%v", err)
	}
}

// TestDeleteTaskRejectsSharedOutput 防止删掉另一个任务仍在使用的字幕和配音。
func TestDeleteTaskRejectsSharedOutput(t *testing.T) {
	core, item := deletionFixture(t)
	if _, err := core.CreateTask(t.Context(), &task.CreateTaskInput{InputPath: item.InputPath, OutputDir: item.OutputDir}); err != nil {
		t.Fatal(err)
	}
	if _, err := core.DeleteTask(t.Context(), item.ID); err == nil {
		t.Fatal("共享产物目录应拒绝联删")
	}
	if _, err := core.GetTask(t.Context(), item.ID); err != nil {
		t.Fatal("联删失败应保留记录")
	}
}

// TestDeleteTaskWithMissingSharedOutput 目录已经被手动删除时，只删除所选任务记录。
func TestDeleteTaskWithMissingSharedOutput(t *testing.T) {
	core, item := deletionFixture(t)
	other, err := core.CreateTask(t.Context(), &task.CreateTaskInput{InputPath: item.InputPath, OutputDir: item.OutputDir})
	if err != nil {
		t.Fatal(err)
	}
	// 删除独立测试目录，复现用户先在文件管理器中删除产物的操作。
	if err := os.Remove(item.OutputDir); err != nil {
		t.Fatal(err)
	}
	if _, err := core.DeleteTask(t.Context(), item.ID); err != nil {
		t.Fatalf("产物目录已不存在，应允许删除任务记录：%v", err)
	}
	if _, err := core.GetTask(t.Context(), item.ID); err == nil {
		t.Fatal("所选任务记录仍存在")
	}
	if _, err := core.GetTask(t.Context(), other.ID); err != nil {
		t.Fatalf("误删了共用目录的另一任务记录：%v", err)
	}
	if data, err := os.ReadFile(item.InputPath); err != nil || string(data) != "original" {
		t.Fatalf("原素材受损：%v", err)
	}
	if _, err := core.DeleteTask(t.Context(), other.ID); err != nil {
		t.Fatalf("目录不存在时也应允许删除剩余任务：%v", err)
	}
}
