package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVersionOutputSupportsPublishedAndHistoricalBrand(t *testing.T) {
	for _, output := range []string{"lark-studio v0.0.127 (main/hash) built date", "vdub v1.2.3 (main/hash) built date"} {
		if _, err := parseVersionOutput(output); err != nil {
			t.Errorf("%q: %v", output, err)
		}
	}
	for _, output := range []string{"", "lark-studio", "other 1.2.3", "lark-studio dev", "lark-studio v1.2.3-beta"} {
		if _, err := parseVersionOutput(output); err == nil {
			t.Errorf("accepted %q", output)
		}
	}
}

func TestMacTargetRejectsDevelopmentBinary(t *testing.T) {
	for _, path := range []string{"/tmp/lark-studio", "/tmp/Other/Contents/MacOS/lark-studio", "/tmp/App.app/Contents/MacOS/other"} {
		if _, _, err := installationTarget(path, "darwin"); err == nil {
			t.Errorf("accepted %s", path)
		}
	}
	if target, relative, err := installationTarget("/tmp/Lark Studio.app/Contents/MacOS/lark-studio", "darwin"); err != nil || target != "/tmp/Lark Studio.app" || relative != filepath.Join("Contents", "MacOS", "lark-studio") {
		t.Fatalf("target=%s,relative=%s,err=%v", target, relative, err)
	}
}

func TestRegularExecutableRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary")
	if err := os.WriteFile(path, []byte("binary"), 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := regularExecutable(link); err == nil {
		t.Fatal("symlink executable accepted")
	}
}
