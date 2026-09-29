package update

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// 显式指定真实发行包后，验证下载摘要、挂载、签名、整包暂存和版本探测。
func TestRealMacReleasePrepare(t *testing.T) {
	dmg := os.Getenv("LARK_UPDATE_DMG_PATH")
	releasePath := os.Getenv("LARK_UPDATE_RELEASE_JSON")
	if runtime.GOOS != "darwin" || dmg == "" || releasePath == "" {
		t.Skip("需要指定真实 macOS 发行包和 GitHub 发行信息")
	}
	data, err := os.ReadFile(releasePath)
	if err != nil {
		t.Fatal(err)
	}
	release, err := parseRelease(data, "darwin", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	executable := filepath.Join(root, "Lark Studio.app", "Contents", "MacOS", "lark-studio")
	if err := os.MkdirAll(filepath.Dir(executable), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("#!/bin/sh\necho 'lark-studio v0.0.1'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	c := NewClient()
	c.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != release.asset.URL {
			t.Errorf("unexpected download URL %s", r.URL)
		}
		f, err := os.Open(dmg)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Body: f, Header: make(http.Header), Request: r}, nil
	})
	p, err := c.Prepare(t.Context(), release, executable)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Discard(); err != nil {
			t.Error(err)
		}
	})
	if p.ReleaseVersion != "v0.1.0" || p.BinaryVersion != "v0.0.127" {
		t.Fatalf("release=%s,binary=%s", p.ReleaseVersion, p.BinaryVersion)
	}
	if err := regularExecutable(filepath.Join(p.Staged, "Contents", "MacOS", "ffmpeg")); err != nil {
		t.Fatal(err)
	}
	if err := p.validate(false); err != nil {
		t.Fatal(err)
	}
	old, err := os.ReadFile(executable)
	if err != nil || string(old) != "#!/bin/sh\necho 'lark-studio v0.0.1'\n" {
		t.Fatalf("original application changed: %q,%v", old, err)
	}
	t.Logf("完整暂存通过：release=%s binary=%s SHA256=%s", p.ReleaseVersion, p.BinaryVersion, p.StagedSHA256)
}
