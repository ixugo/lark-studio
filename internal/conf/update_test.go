package conf

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUpdateStatePersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "update.toml")
	got, err := ReadUpdateState(path)
	if err != nil || got.IgnoredVersion != "" {
		t.Fatalf("新安装状态 = %#v, %v", got, err)
	}
	want := UpdateState{IgnoredVersion: "v0.1.0"}
	if err := WriteConfig(want, path); err != nil {
		t.Fatal(err)
	}
	got, err = ReadUpdateState(path)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("重启后 = %#v, %v", got, err)
	}
}

func TestUpdateStateReadFailureIsVisible(t *testing.T) {
	path := filepath.Join(t.TempDir(), "update.toml")
	if err := os.WriteFile(path, []byte("invalid [toml"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadUpdateState(path); err == nil {
		t.Fatal("损坏配置不能当作未忽略")
	}
}
