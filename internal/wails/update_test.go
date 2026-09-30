package wails

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/update"
)

func TestIgnoreUpdatePersistsAndRejectsUnseenRelease(t *testing.T) {
	bc := conf.DefaultConfig()
	bc.Runtime.ConfigDir = t.TempDir()
	svc := &AppService{bc: &bc}
	if err := svc.IgnoreUpdate("v0.1.0"); err == nil {
		t.Fatal("不能忽略尚未查验的发布")
	}
	if err := svc.IgnoreUpdate("../../bad"); err == nil {
		t.Fatal("不能保存非法版本")
	}
	state, err := conf.ReadUpdateState(filepath.Join(bc.Runtime.ConfigDir, "update.toml"))
	if err != nil || state.IgnoredVersion != "" {
		t.Fatalf("失败时不能落盘: %#v %v", state, err)
	}
}

func TestInstallRejectsUnseenReleaseWithoutQuitting(t *testing.T) {
	svc := &AppService{bc: &conf.Bootstrap{}}
	if err := svc.InstallUpdate("v0.1.0"); err == nil {
		t.Fatal("不能安装未检查的版本")
	}
	if svc.GetUpdateStatus().Phase == "restarting" {
		t.Fatal("拒绝安装不能重启")
	}
}

type updateTransport func(*http.Request) (*http.Response, error)

func (f updateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func updateFixtureService(t *testing.T, version string) *AppService {
	t.Helper()
	bc := conf.DefaultConfig()
	bc.Runtime.BuildVersion = "v0.0.1"
	bc.Runtime.ConfigDir = t.TempDir()
	svc := &AppService{bc: &bc}
	return updateFixtureServiceAt(t, svc, version, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
}

func updateFixtureServiceAt(t *testing.T, svc *AppService, version string, updated time.Time) *AppService {
	t.Helper()
	name := fmt.Sprintf("lark-studio_%s_%s", version, map[string]string{"darwin": "macos_arm64.dmg", "windows": "windows_amd64.zip"}[runtime.GOOS])
	svc.updates.client = update.NewClient()
	svc.updates.client.HTTP.Transport = updateTransport(func(r *http.Request) (*http.Response, error) {
		body, err := json.Marshal(map[string]any{"tag_name": version, "updated_at": updated, "body": "真实发布说明结构\n第二行", "assets": []any{map[string]any{"name": name, "state": "uploaded", "size": 3, "browser_download_url": "https://github.com/ixugo/lark-studio/releases/download/" + version + "/" + name}}})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})
	return svc
}

func TestIgnoredReleaseSuppressesOnlyAutomaticChecks(t *testing.T) {
	svc := updateFixtureService(t, "v0.1.0")
	info, err := svc.CheckForUpdates(false)
	if err != nil || !info.Available || info.Notes != "真实发布说明结构\n第二行" {
		t.Fatalf("首次自动检查 %#v, %v", info, err)
	}
	if err := svc.IgnoreUpdate(info.Version); err != nil {
		t.Fatal(err)
	}
	restarted := updateFixtureService(t, "v0.1.0")
	restarted.bc.Runtime.ConfigDir = svc.bc.Runtime.ConfigDir
	for _, manual := range []bool{false, true} {
		info, err := restarted.CheckForUpdates(manual)
		if err != nil || info.Available != manual {
			t.Fatalf("重启后 manual=%v %#v %v", manual, info, err)
		}
	}
	older := updateFixtureService(t, "v0.0.127")
	older.bc.Runtime.ConfigDir = svc.bc.Runtime.ConfigDir
	info, err = older.CheckForUpdates(false)
	if err != nil || info.Available {
		t.Fatalf("低于忽略版本不能自动提示 %#v %v", info, err)
	}
	newer := updateFixtureService(t, "v0.2.0")
	newer.bc.Runtime.ConfigDir = svc.bc.Runtime.ConfigDir
	info, err = newer.CheckForUpdates(false)
	if err != nil || !info.Available {
		t.Fatalf("更新发布必须提示 %#v %v", info, err)
	}
	if err := older.IgnoreUpdate("v0.0.127"); err != nil {
		t.Fatal(err)
	}
	state, err := conf.ReadUpdateState(svc.updateStatePath())
	if err != nil || state.IgnoredVersion != "v0.1.0" {
		t.Fatalf("不能降低忽略阈值 %#v %v", state, err)
	}
}

func TestInstalledReleaseReceiptMustMatchExecutable(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	state := conf.UpdateState{InstalledReleaseVersion: "v0.1.0", InstalledBinarySHA256: hex.EncodeToString(digest[:])}
	if got := installedReleaseVersion("v0.0.127", state); got != "v0.1.0" {
		t.Fatalf("有效收据 = %s", got)
	}
	state.InstalledBinarySHA256 = strings.Repeat("0", 64)
	if got := installedReleaseVersion("v0.0.127", state); got != "v0.0.127" {
		t.Fatalf("替换程序后旧收据失效 = %s", got)
	}
}

func TestUpdateProtectsDataStoredInsideInstallation(t *testing.T) {
	target := t.TempDir()
	for _, kind := range []string{"database", "output", "model"} {
		t.Run(kind, func(t *testing.T) {
			bc := conf.DefaultConfig()
			path := filepath.Join(target, "userdata")
			switch kind {
			case "database":
				bc.Data.Database.Dsn = path
			case "output":
				bc.Pipeline.DefaultOutputDir = path
			case "model":
				bc.Pipeline.WhisperModel = path
			}
			if err := (&AppService{bc: &bc}).validateUpdateDataPaths(target); err == nil {
				t.Fatal("不能把用户数据随旧包移走")
			}
		})
	}
}

func TestUpdateProtectsDataReferencedThroughLinks(t *testing.T) {
	target, external := t.TempDir(), t.TempDir()
	for _, paths := range [][2]string{{target, external}, {external, target}} {
		link := filepath.Join(paths[0], "linked-data")
		if err := os.Symlink(paths[1], link); err != nil {
			t.Fatal(err)
		}
		if err := validateExternalDataPath(target, link); err == nil {
			t.Fatal("包内路径或包外指向包内的链接必须拒绝")
		}
	}
	if err := validateExternalDataPath(target, filepath.Join(external, "data")); err != nil {
		t.Fatal(err)
	}
}

func TestUploadedReleasePromptsImmediatelyForAutomaticAndManualChecks(t *testing.T) {
	for _, manual := range []bool{false, true} {
		svc := updateFixtureService(t, "v0.1.25")
		updateFixtureServiceAt(t, svc, "v0.1.25", time.Now())
		info, err := svc.CheckForUpdates(manual)
		if err != nil || !info.Available || !info.Supported {
			t.Fatalf("刚上传的当前平台安装包必须立即认可: %#v %v", info, err)
		}
	}
}

func TestReleaseWithoutCurrentPlatformPackageDoesNotPrompt(t *testing.T) {
	for _, state := range []string{"missing", "new", "uploaded"} {
		t.Run(state, func(t *testing.T) {
			svc := updateFixtureService(t, "v0.1.25")
			if _, err := svc.CheckForUpdates(true); err != nil {
				t.Fatal(err)
			}
			original := svc.updates.client.HTTP.Transport
			svc.updates.client.HTTP.Transport = updateTransport(func(r *http.Request) (*http.Response, error) {
				response, err := original.RoundTrip(r)
				if err != nil {
					return nil, err
				}
				defer response.Body.Close()
				data, err := io.ReadAll(response.Body)
				if err != nil {
					return nil, err
				}
				var body map[string]any
				if err := json.Unmarshal(data, &body); err != nil {
					return nil, err
				}
				assets := body["assets"].([]any)
				if state == "missing" {
					body["assets"] = []any{}
				} else {
					assets[0].(map[string]any)["state"] = state
				}
				data, err = json.Marshal(body)
				if err != nil {
					return nil, err
				}
				response.Body = io.NopCloser(strings.NewReader(string(data)))
				return response, nil
			})
			for _, manual := range []bool{false, true} {
				info, err := svc.CheckForUpdates(manual)
				if err != nil || info.Available != (state == "uploaded") {
					t.Fatalf("state=%s manual=%v: %#v %v", state, manual, info, err)
				}
				if state != "uploaded" && info.Reason == "" {
					t.Fatal("未完成上传必须明确原因")
				}
			}
			if state != "uploaded" {
				if err := svc.IgnoreUpdate("v0.1.25"); err == nil {
					t.Fatal("未显示的版本不能记录忽略")
				}
			}
		})
	}
}

func TestUpdateStatusExposesProgress(t *testing.T) {
	svc := &AppService{}
	if status := svc.GetUpdateStatus(); status.Phase != "idle" || status.Percent != 0 {
		t.Fatalf("初始状态: %+v", status)
	}
	for _, tt := range []struct {
		phase   string
		percent int
	}{
		{"downloading", 42}, {"preparing", 99}, {"restarting", 100},
	} {
		svc.setUpdateStatus(tt.phase, "正在更新", tt.percent)
		status := svc.GetUpdateStatus()
		if status.Phase != tt.phase || status.Percent != tt.percent {
			t.Fatalf("状态丢失下载进度: %+v", status)
		}
	}
}

func TestCheckCapturedGitHubRelease(t *testing.T) {
	path := os.Getenv("LARK_UPDATE_RELEASE_JSON")
	if path == "" {
		t.Skip("需要捕获的 GitHub Releases API 响应")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	svc := updateFixtureService(t, "v0.1.25")
	svc.bc.Runtime.BuildVersion = "v0.1.23"
	svc.updates.client.HTTP.Transport = updateTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(data))), Request: r}, nil
	})
	for _, manual := range []bool{false, true} {
		info, err := svc.CheckForUpdates(manual)
		if err != nil || !info.Available || !info.Supported {
			t.Fatalf("真实发布响应未识别为可更新: %#v %v", info, err)
		}
		t.Logf("manual=%v version=%s available=%v supported=%v", manual, info.Version, info.Available, info.Supported)
	}
}
