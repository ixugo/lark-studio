package conf

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestMigrateStudioDirRenamesLegacyTree 确保旧资源树在新目录缺失时整体迁移。
func TestMigrateStudioDirRenamesLegacyTree(t *testing.T) {
	oldDir := filepath.Join(t.TempDir(), ".vdub_studio")
	newDir := filepath.Join(filepath.Dir(oldDir), ".lark-studio")
	writeStudioTestFile(t, filepath.Join(oldDir, "tasks", "job.mp4"), "media")

	if err := migrateStudioDir(oldDir, newDir); err != nil {
		t.Fatal(err)
	}
	assertStudioTestFile(t, filepath.Join(newDir, "tasks", "job.mp4"), "media")
	assertStudioPathMissing(t, oldDir)
}

// TestMigrateStudioDirMergesWithoutReplacingTarget 确保合并时目标文件优先且旧冲突文件留档。
func TestMigrateStudioDirMergesWithoutReplacingTarget(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, ".vdub_studio")
	newDir := filepath.Join(root, ".lark-studio")
	writeStudioTestFile(t, filepath.Join(oldDir, "models", "ggml-tiny.bin"), "old")
	writeStudioTestFile(t, filepath.Join(oldDir, "models", "ggml-small.bin"), "small")
	writeStudioTestFile(t, filepath.Join(newDir, "models", "ggml-tiny.bin"), "target")

	if err := migrateStudioDir(oldDir, newDir); err != nil {
		t.Fatal(err)
	}
	assertStudioTestFile(t, filepath.Join(newDir, "models", "ggml-tiny.bin"), "target")
	assertStudioTestFile(t, filepath.Join(newDir, "models", "ggml-tiny.bin.from-vdub_studio"), "old")
	assertStudioTestFile(t, filepath.Join(newDir, "models", "ggml-small.bin"), "small")
	assertStudioPathMissing(t, oldDir)
}

// writeStudioTestFile 创建目录迁移用的隔离样本文件。
func writeStudioTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// assertStudioTestFile 比较迁移后的文件内容，避免只凭目标存在判定成功。
func assertStudioTestFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s 内容 = %q，期望 %q", path, got, want)
	}
}

// assertStudioPathMissing 检查旧资源目录已整体迁出。
func assertStudioPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("旧路径仍存在或检查失败: %s, %v", path, err)
	}
}
