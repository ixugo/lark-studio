package tts

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

// TestEdgeRate 验证语速倍率转换为 edge-tts 百分比。
func TestEdgeRate(t *testing.T) {
	tests := map[float64]string{
		0:    "+0%",
		0.8:  "-20%",
		1:    "+0%",
		1.25: "+25%",
	}
	for speed, want := range tests {
		if got := edgeRate(speed); got != want {
			t.Fatalf("edgeRate(%v) = %q，期望 %q", speed, got, want)
		}
	}
}

func TestEdgeTTSWithDesktopPATH(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("仅验证 macOS 桌面环境")
	}
	if _, err := os.Stat("/opt/homebrew/bin/edge-tts"); err != nil {
		t.Skip("本机未安装 Homebrew edge-tts")
	}
	t.Setenv("PATH", "/usr/bin:/bin")
	binary, err := edgeTTSBinary()
	if err != nil {
		t.Fatal(err)
	}
	if binary != "/opt/homebrew/bin/edge-tts" {
		t.Fatalf("binary = %q", binary)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if output, err := exec.CommandContext(ctx, binary, "--version").CombinedOutput(); err != nil {
		t.Fatalf("命令不可执行: %v %s", err, output)
	}
}
