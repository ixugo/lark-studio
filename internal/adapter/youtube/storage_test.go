package youtube

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorageAndClear(t *testing.T) {
	dir := t.TempDir()
	for name, data := range map[string]string{"video.mp4": "video", "video_1.mp4": "second", "keep.txt": "keep"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	m := NewManager()
	got, err := m.Storage(dir)
	if err != nil || got.Bytes != 11 || got.Files != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	m.status = Status{Phase: "downloading"}
	if err := m.ClearStorage(dir); err == nil {
		t.Fatal("cleared while downloading")
	}
	m.status = Status{Phase: "completed", Path: "old", TaskID: "old"}
	if err := m.ClearStorage(dir); err != nil {
		t.Fatal(err)
	}
	got, err = m.Storage(dir)
	if err != nil || got.Bytes != 0 || got.Files != 0 {
		t.Fatalf("%+v %v", got, err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "keep.txt")); err != nil || string(data) != "keep" {
		t.Fatal("unrelated file changed", err)
	}
	if s := m.Status(); s.Phase != "idle" || s.Path != "" || s.TaskID != "" || s.Video != nil {
		t.Fatal(s)
	}
	if err := m.ClearStorage(dir); err != nil {
		t.Fatal(err)
	}
}

func TestResetRejectsBusyAndClearsAllState(t *testing.T) {
	m := NewManager()
	m.status = Status{Phase: "verifying"}
	if err := m.Reset(); err == nil {
		t.Fatal("reset active operation")
	}
	m.status = Status{Phase: "completed", Path: "old", Error: "old", TaskID: "old"}
	m.info = &Info{Title: "old"}
	if err := m.Reset(); err != nil {
		t.Fatal(err)
	}
	if s := m.Status(); s.Phase != "idle" || s.Path != "" || s.Error != "" || s.TaskID != "" || s.Video != nil || s.Verified {
		t.Fatal(s)
	}
}

func TestStorageDoesNotFollowLinksOrDirectories(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "original.mp4")
	if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link.mp4")); err != nil {
		t.Skip(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "nested.mp4"), 0700); err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	if got, err := m.Storage(dir); err != nil || got.Bytes != 0 || got.Files != 0 {
		t.Fatal(got, err)
	}
	if err := m.ClearStorage(dir); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "keep" {
		t.Fatal("followed link", err)
	}
	link := filepath.Join(t.TempDir(), "Downloads")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Storage(link); err == nil {
		t.Fatal("accepted symlink directory")
	}
	if err := m.ClearStorage(link); err == nil {
		t.Fatal("cleared symlink directory")
	}
}
