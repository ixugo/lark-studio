package update

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func makeArchive(t *testing.T, name string, mode os.FileMode, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "asset.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	h := &zip.FileHeader{Name: name, Method: zip.Store}
	h.SetMode(mode)
	entry, err := w.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = entry.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestArchiveRejectsCorruptEntry(t *testing.T) {
	path := makeArchive(t, "lark-studio/ffmpeg.exe", 0700, "unique-body-to-corrupt")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	position := bytes.Index(data, []byte("unique-body-to-corrupt"))
	if position < 0 {
		t.Fatal("fixture payload not found")
	}
	data[position] = 'X'
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractZip(t.Context(), path, t.TempDir()); err == nil {
		t.Fatal("corrupt archive accepted")
	}
}

func TestArchiveRejectsTraversalAndLinks(t *testing.T) {
	for _, name := range []string{"../escape", "lark-studio/../../escape", "/absolute", `lark-studio\..\escape`, "C:/escape", "lark-studio/file:stream", "lark-studio/CON", "lark-studio/file.", "lark-studio/file "} {
		if err := extractZip(t.Context(), makeArchive(t, name, 0600, "x"), t.TempDir()); err == nil {
			t.Errorf("accepted archive path %q", name)
		}
	}
	if err := extractZip(t.Context(), makeArchive(t, "lark-studio/link", os.ModeSymlink|0700, "../../escape"), t.TempDir()); err == nil {
		t.Error("accepted symlink")
	}
}

func TestArchiveExtractsWholeApplication(t *testing.T) {
	dest := t.TempDir()
	if err := extractZip(t.Context(), makeArchive(t, "lark-studio/ffmpeg.exe", 0600, "ffmpeg"), dest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dest, "lark-studio", "ffmpeg.exe"))
	if err != nil || string(data) != "ffmpeg" {
		t.Fatalf("data=%q,error=%v", data, err)
	}
}
