package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskLog_WriteAndClose(t *testing.T) {
	dir := t.TempDir()
	tl := openTaskLog(dir)

	tl.Write("hello %s", "world")
	tl.Write("step done: %s (%dms)", "whisper", 123)
	tl.Close()

	data, err := os.ReadFile(filepath.Join(dir, "task.log"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "hello world") {
		t.Error("missing 'hello world' in task.log")
	}
	if !strings.Contains(content, "step done: whisper (123ms)") {
		t.Error("missing step log in task.log")
	}

	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "[") {
			t.Errorf("line missing timestamp prefix: %q", line)
		}
	}
}

func TestTaskLog_AppendMode(t *testing.T) {
	dir := t.TempDir()

	tl1 := openTaskLog(dir)
	tl1.Write("first run")
	tl1.Close()

	tl2 := openTaskLog(dir)
	tl2.Write("second run")
	tl2.Close()

	data, _ := os.ReadFile(filepath.Join(dir, "task.log"))
	content := string(data)
	if !strings.Contains(content, "first run") || !strings.Contains(content, "second run") {
		t.Error("append mode failed: missing entries")
	}
}

func TestTaskLog_NilFile(t *testing.T) {
	tl := &taskLog{}
	tl.Write("should not panic")
	tl.Close()
}
