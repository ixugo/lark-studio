package wails

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 使用本地替身命令验证真实试音入口，不向在线服务发送测试请求。
func TestEdgePreviewUsesSystemTempAndCleansFiles(t *testing.T) {
	for _, fails := range []bool{false, true} {
		name := "success"
		if fails {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			systemTemp := filepath.Join(dir, "system-temp")
			if err := os.Mkdir(systemTemp, 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("TMPDIR", systemTemp)
			t.Setenv("TMP", systemTemp)
			t.Setenv("PATH", dir)
			capture := filepath.Join(dir, "capture")
			script := "#!/bin/sh\nprintf '%s\\n' \"$6\" >> \"" + capture + "\"\nprintf audio > \"$6\"\n"
			if fails {
				script += "printf ConnectionTimeoutError\nexit 1\n"
			}
			if err := os.WriteFile(filepath.Join(dir, "edge-tts"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			svc := &AppService{}
			audio, err := svc.TestEdgeTTS("zh-CN-XiaoxiaoNeural", "你好")
			if fails {
				if err == nil || audio != "" || !strings.Contains(err.Error(), "ConnectionTimeoutError") {
					t.Fatalf("试听失败未保留错误: %q %v", audio, err)
				}
			} else {
				want := "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString([]byte("audio"))
				if err != nil || audio != want {
					t.Fatalf("试听音频错误: %q %v", audio, err)
				}
			}
			paths, err := os.ReadFile(capture)
			if err != nil {
				t.Fatal(err)
			}
			for path := range strings.SplitSeq(strings.TrimSpace(string(paths)), "\n") {
				if filepath.Dir(path) != os.TempDir() {
					t.Fatalf("试听未使用系统临时目录: %q", path)
				}
			}
			remaining, err := os.ReadDir(systemTemp)
			if err != nil || len(remaining) != 0 {
				t.Fatalf("试听结束仍残留临时音频: %v %v", remaining, err)
			}
		})
	}
}
