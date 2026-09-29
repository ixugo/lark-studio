package taskfile

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"
	"uuid"
)

func TestResultPathRejectsOutsideNames(t *testing.T) {
	work := filepath.Join(t.TempDir(), uuid.NewV4().String())
	if err := os.Mkdir(work, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../other.mp4", "/outside.mp4", uuid.NewV4().String() + ".mp4"} {
		meta := Metadata{LayoutVersion: 1, ResultName: name, StagedName: filepath.Base(work) + ".mp4", StagedPath: filepath.Join(work, filepath.Base(work)+".mp4")}
		data, err := json.Marshal(meta)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, "source_meta.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
		if path, err := ResultVideoPath(work); err == nil {
			t.Fatalf("允许删除其他任务或外部成片: %q", path)
		}
	}
}
