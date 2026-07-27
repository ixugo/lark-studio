package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanIntermediate(t *testing.T) {
	dir := t.TempDir()

	keepers := []string{"src.srt", "trans.srt", "task.log", "clipped.trans.mp4", "video.final.mp4"}
	removables := []string{"raw.mp3", "trans.txt", "concat_list.txt", "dub.mp3", "silence_0.wav", "silence_start.wav"}

	for _, f := range append(keepers, removables...) {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	audioDir := filepath.Join(dir, "audio_segs")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(audioDir, "0.wav"), []byte("wav"), 0o644)

	removed := cleanIntermediate(dir)

	expectedRemoved := len(removables) + 1 // +1 for audio_segs dir
	if removed != expectedRemoved {
		t.Errorf("cleanIntermediate() removed %d, want %d", removed, expectedRemoved)
	}

	for _, f := range keepers {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("keeper %q was deleted", f)
		}
	}

	for _, f := range removables {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			t.Errorf("removable %q still exists", f)
		}
	}

	if _, err := os.Stat(audioDir); err == nil {
		t.Error("audio_segs directory still exists")
	}
}

func TestCleanIntermediate_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	removed := cleanIntermediate(dir)
	if removed != 0 {
		t.Errorf("empty dir: removed %d, want 0", removed)
	}
}

func TestCleanIntermediate_OnlyKeepers(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"src.srt", "trans.srt", "output.mp4"} {
		os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644)
	}
	removed := cleanIntermediate(dir)
	if removed != 0 {
		t.Errorf("only keepers: removed %d, want 0", removed)
	}
}

func TestCleanIntermediate_NonexistentDir(t *testing.T) {
	removed := cleanIntermediate("/nonexistent/path/12345")
	if removed != 0 {
		t.Errorf("nonexistent dir: removed %d, want 0", removed)
	}
}
