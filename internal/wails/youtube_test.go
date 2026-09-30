package wails

import (
	"encoding/base64"
	"fmt"
	youtubeadapter "github.com/ixugo/vdub/internal/adapter/youtube"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/wailsapp/wails/v3/pkg/application"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestYouTubeMessageOrigin(t *testing.T) {
	for _, tc := range []struct {
		origin *application.OriginInfo
		valid  bool
	}{
		{nil, false},
		{&application.OriginInfo{Origin: "https://embed.dlsrv.online/v1/full?videoId=test", IsMainFrame: true}, true},
		{&application.OriginInfo{Origin: "https://embed.dlsrv.online:443/v1/full", IsMainFrame: true}, true},
		{&application.OriginInfo{Origin: "https://embed.dlsrv.online/v1/full", TopOrigin: "https://embed.dlsrv.online/v1/full"}, true},
		{&application.OriginInfo{Origin: "https://embed.dlsrv.online/v1/full", TopOrigin: "https://evil.test"}, false},
		{&application.OriginInfo{Origin: "https://embed.dlsrv.online.evil.test/", IsMainFrame: true}, false},
		{&application.OriginInfo{Origin: "http://embed.dlsrv.online", IsMainFrame: true}, false},
		{&application.OriginInfo{Origin: "https://embed.dlsrv.online:444", IsMainFrame: true}, false},
	} {
		if got := validYouTubeOrigin(tc.origin); got != tc.valid {
			t.Fatalf("origin=%+v got=%v", tc.origin, got)
		}
	}
}

type youtubeTestWindow struct {
	application.Window
	id     uint
	closed int
}

func (w *youtubeTestWindow) ID() uint { return w.id }
func (w *youtubeTestWindow) Close()   { w.closed++ }

func TestYouTubeStaleWindowDoesNotCloseCurrentVerification(t *testing.T) {
	manager := youtubeadapter.NewManager()
	if err := manager.BeginVerify(); err != nil {
		t.Fatal(err)
	}
	current := &youtubeTestWindow{id: 2}
	svc := &AppService{youtube: manager, youtubeVerification: youtubeVerification{window: current, nonce: "new"}}
	svc.closeYouTubeVerificationID(1)
	if current.closed != 0 || manager.Status().Phase != "verifying" {
		t.Fatal("旧超时关闭了新校验窗口")
	}
	svc.CancelYouTubeDownload()
	if current.closed != 1 || svc.youtubeVerification.window != nil {
		t.Fatal("取消未释放校验窗口")
	}
}

func TestYouTubeLateVerificationIsIgnored(t *testing.T) {
	token := "e30." + base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, time.Now().Add(time.Minute).Unix()))) + ".sig"
	message := fmt.Sprintf(`{"kind":"youtube-session","nonce":"old","token":%q,"user_agent":"test","address":"https://yt1s-worker-5.dlsrv.online/tunnel?id=test"}`, token)
	for _, cancel := range []bool{false, true} {
		manager := youtubeadapter.NewManager()
		if err := manager.BeginVerify(); err != nil {
			t.Fatal(err)
		}
		window := &youtubeTestWindow{id: 2}
		svc := &AppService{youtube: manager, youtubeVerification: youtubeVerification{window: window, nonce: "new"}}
		if cancel {
			svc.CancelYouTubeDownload()
		}
		svc.handleYouTubeMessage(window, message, &application.OriginInfo{Origin: youtubeadapter.ServiceOrigin, IsMainFrame: true})
		if manager.Status().Verified {
			t.Fatal("迟到会话被接受")
		}
	}
}

func TestYouTubeDirectoryDoesNotFollowTaskOutputSetting(t *testing.T) {
	bc := conf.DefaultConfig()
	bc.Pipeline.DefaultOutputDir = t.TempDir()
	svc := &AppService{bc: &bc}
	status, err := svc.GetYouTubeDownload()
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if status.Directory != filepath.Join(home, "Documents", "lark-studio", "Downloads") {
		t.Fatalf("错误目录: %s", status.Directory)
	}
}
