package update

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ixugo/vdub/internal/conf"
)

func replacementFixture(t *testing.T) *Plan {
	t.Helper()
	dir := t.TempDir()
	p := &Plan{Target: filepath.Join(dir, "target"), Staged: filepath.Join(dir, "stage"), Backup: filepath.Join(dir, "backup"), ExecutableRelative: "lark-studio"}
	for path, body := range map[string]string{p.Target: "old", p.Staged: "new"} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, p.ExecutableRelative), []byte(body), 0700); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func TestReceiptPreservesIgnoreAndBindsInstalledBinary(t *testing.T) {
	p := replacementFixture(t)
	p.StatePath = filepath.Join(t.TempDir(), "update.toml")
	p.ReleaseVersion = "v0.1.0"
	if err := conf.WriteConfig(conf.UpdateState{IgnoredVersion: "v0.2.0"}, p.StatePath); err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(p, func(string, []string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	state, err := conf.ReadUpdateState(p.StatePath)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := fileSHA256(filepath.Join(p.Target, p.ExecutableRelative))
	if err != nil {
		t.Fatal(err)
	}
	if state.IgnoredVersion != "v0.2.0" || state.InstalledReleaseVersion != "v0.1.0" || state.InstalledBinarySHA256 != hash {
		t.Fatalf("receipt=%+v", state)
	}
}

func TestFailedRestartRestoresPreviousReceipt(t *testing.T) {
	p := replacementFixture(t)
	p.StatePath = filepath.Join(t.TempDir(), "update.toml")
	p.ReleaseVersion = "v0.1.0"
	old := []byte("IgnoredVersion = 'v0.2.0'\nInstalledReleaseVersion = 'v0.0.1'\nInstalledBinarySHA256 = 'previous'\n")
	if err := os.WriteFile(p.StatePath, old, 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	if err := applyPlan(p, func(string, []string) error {
		calls++
		if calls == 1 {
			return errors.New("failed")
		}
		return nil
	}); err == nil {
		t.Fatal("restart failure accepted")
	}
	got, err := os.ReadFile(p.StatePath)
	if err != nil || !bytes.Equal(got, old) {
		t.Fatalf("receipt=%q,error=%v", got, err)
	}
}

func TestReceiptWriteFailureRestoresAndRestartsOldApplication(t *testing.T) {
	p := replacementFixture(t)
	p.ReleaseVersion = "v0.1.0"
	p.StatePath = filepath.Join(t.TempDir(), "missing-parent", "update.toml")
	calls := 0
	err := applyPlan(p, func(path string, args []string) error {
		calls++
		data, readErr := os.ReadFile(path)
		if readErr != nil || string(data) != "old" {
			t.Errorf("restarted data=%q,error=%v", data, readErr)
		}
		return nil
	})
	if err == nil || calls != 1 {
		t.Fatalf("error=%v,restarts=%d", err, calls)
	}
	if _, err := os.Stat(p.StatePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("receipt should not exist: %v", err)
	}
}

func TestBackupRenameFailureRestartsUntouchedApplication(t *testing.T) {
	p := replacementFixture(t)
	p.Backup = filepath.Join(t.TempDir(), "missing-parent", "backup")
	calls := 0
	err := applyPlan(p, func(path string, args []string) error {
		calls++
		data, readErr := os.ReadFile(path)
		if readErr != nil || string(data) != "old" {
			t.Errorf("restart data=%q,error=%v", data, readErr)
		}
		return nil
	})
	if err == nil || calls != 1 {
		t.Fatalf("error=%v,restarts=%d", err, calls)
	}
}

func TestPlanRejectsArbitraryLocationAndHelperInvocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(path, []byte(`{"target":"/Applications/Other.app"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RunHelper(path); err == nil {
		t.Fatal("arbitrary update plan accepted")
	}
	for _, args := range [][]string{{"--apply-update", "other"}, {"-version"}, {"--version=true"}, {"\x00"}} {
		if err := validateRestartArgs(args); err == nil {
			t.Errorf("accepted restart args %q", args)
		}
	}
	if err := validateRestartArgs([]string{"-conf", "/tmp/config dir/custom.toml"}); err != nil {
		t.Fatal(err)
	}
}

func TestWindowsTargetRequiresDedicatedDirectory(t *testing.T) {
	for _, dir := range []string{"Desktop", "Downloads", "Applications"} {
		if _, _, err := installationTarget(filepath.Join(t.TempDir(), dir, "lark-studio.exe"), "windows"); err == nil {
			t.Errorf("accepted %s", dir)
		}
	}
	if _, _, err := installationTarget(filepath.Join(t.TempDir(), "lark-studio", "lark-studio.exe"), "windows"); err != nil {
		t.Fatal(err)
	}
}

func TestStateInsideInstallationIsRejected(t *testing.T) {
	target := filepath.Join(t.TempDir(), "lark-studio")
	for _, path := range []string{filepath.Join(target, "update.toml"), filepath.Join(target, "config", "update.toml")} {
		if err := validateStateLocation(target, path); err == nil {
			t.Errorf("accepted state in installation: %s", path)
		}
	}
	if err := validateStateLocation(target, filepath.Join(filepath.Dir(target), "external-config", "update.toml")); err != nil {
		t.Fatal(err)
	}
}

func TestRestartReportsImmediateExit(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LARK_UPDATE_RESTART_EXIT_TEST", "yes")
	if err := restartApplication(executable, []string{"-test.run=^TestRestartExitProcess$"}, t.TempDir()); err == nil {
		t.Fatal("immediate exit accepted as healthy restart")
	}
}

func TestRestartExitProcess(t *testing.T) {
	if os.Getenv("LARK_UPDATE_RESTART_EXIT_TEST") == "yes" {
		os.Exit(0)
	}
}

func TestDiscardRejectsBackupOrUnownedPaths(t *testing.T) {
	p := replacementFixture(t)
	if err := p.Discard(); err == nil {
		t.Fatal("unowned staging directory accepted")
	}
	p.started = true
	if err := p.Discard(); err == nil {
		t.Fatal("started plan discarded")
	}
	data, err := os.ReadFile(filepath.Join(p.Target, p.ExecutableRelative))
	if err != nil || string(data) != "old" {
		t.Fatalf("original application changed: %q,%v", data, err)
	}
}

func TestReplacementRollbackWhenRestartFails(t *testing.T) {
	p := replacementFixture(t)
	calls := 0
	err := applyPlan(p, func(string, []string) error {
		calls++
		if calls == 1 {
			return errors.New("new executable failed")
		}
		return nil
	})
	if err == nil || calls != 2 {
		t.Fatalf("err=%v,restarts=%d", err, calls)
	}
	data, err := os.ReadFile(filepath.Join(p.Target, p.ExecutableRelative))
	if err != nil || string(data) != "old" {
		t.Fatalf("rollback data=%q,error=%v", data, err)
	}
}

func TestReplacementRollbackWhenStageMissing(t *testing.T) {
	p := replacementFixture(t)
	p.Staged = filepath.Join(filepath.Dir(p.Staged), "missing")
	err := applyPlan(p, func(string, []string) error { return nil })
	if err == nil {
		t.Fatal("missing staged app accepted")
	}
	data, err := os.ReadFile(filepath.Join(p.Target, p.ExecutableRelative))
	if err != nil || string(data) != "old" {
		t.Fatalf("rollback data=%q,error=%v", data, err)
	}
}

func TestReplacementRetainsRecoveryBackup(t *testing.T) {
	p := replacementFixture(t)
	if err := applyPlan(p, func(string, []string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{p.Target: "new", p.Backup: "old"} {
		data, err := os.ReadFile(filepath.Join(path, p.ExecutableRelative))
		if err != nil || string(data) != want {
			t.Errorf("%s=%q,error=%v", path, data, err)
		}
	}
}
