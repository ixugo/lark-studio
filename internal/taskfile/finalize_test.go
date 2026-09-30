package taskfile

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"uuid"
)

func preparedResult(t *testing.T, root string) string {
	t.Helper()
	id := uuid.NewV4().String()
	work := filepath.Join(root, id)
	if err := os.Mkdir(work, 0755); err != nil {
		t.Fatal(err)
	}
	meta := Metadata{LayoutVersion: 1, OriginalName: "a.mp4", StagedName: id + ".mp4", StagedPath: filepath.Join(work, id+".mp4"), ResultName: id + ".mp4"}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "source_meta.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, meta.ResultName), []byte(id), 0600); err != nil {
		t.Fatal(err)
	}
	return work
}

func TestFinalizeResultSkipsExistingNames(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.mp4", "a_1.mp4", "a_2.mp4"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	work := preparedResult(t, root)
	if err := FinalizeResult(work); err != nil {
		t.Fatal(err)
	}
	got, err := ResultVideoPath(work)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.Base(work)+".mp4")); !os.IsNotExist(err) {
		t.Fatal("UUID 渲染名未移除")
	}
	if filepath.Base(got) != "a_3.mp4" {
		t.Fatal(got)
	}
	for _, name := range []string{"a.mp4", "a_1.mp4", "a_2.mp4"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(data) != "keep" {
			t.Fatalf("覆盖旧文件 %s", name)
		}
	}
	if err := FinalizeResult(work); err != nil {
		t.Fatal(err)
	}
	again, err := ResultVideoPath(work)
	if err != nil || again != got {
		t.Fatalf("重复收尾改变路径 %s %v", again, err)
	}
}

func TestFinalizeConcurrentSameNames(t *testing.T) {
	root := t.TempDir()
	works := []string{preparedResult(t, root), preparedResult(t, root), preparedResult(t, root)}
	var wg sync.WaitGroup
	for _, work := range works {
		wg.Go(func() {
			if err := FinalizeResult(work); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	seen := map[string]bool{}
	for _, work := range works {
		path, err := ResultVideoPath(work)
		if err != nil {
			t.Fatal(err)
		}
		if seen[path] {
			t.Fatal("重名覆盖")
		}
		seen[path] = true
		data, err := os.ReadFile(path)
		if err != nil || string(data) != filepath.Base(work) {
			t.Fatalf("成片错配 %s", path)
		}
	}
	for _, name := range []string{"a.mp4", "a_1.mp4", "a_2.mp4"} {
		if !seen[filepath.Join(root, name)] {
			t.Fatalf("缺少成片 %s", name)
		}
	}
}
