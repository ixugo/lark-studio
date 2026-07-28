package whisper

import (
	"errors"
	"testing"
)

// TestResolveRunnerBinary 验证 Homebrew 的 whisper-cli 能接管旧的 whisper-cpp 配置名。
func TestResolveRunnerBinary(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "whisper-cli" {
			return "/opt/homebrew/bin/whisper-cli", nil
		}
		return "", errors.New("not found")
	}

	if got := resolveRunnerBinary("whisper-cpp", lookPath); got != "/opt/homebrew/bin/whisper-cli" {
		t.Fatalf("resolveRunnerBinary() = %q, want whisper-cli path", got)
	}
	if got := resolveRunnerBinary("/custom/whisper", lookPath); got != "/custom/whisper" {
		t.Fatalf("resolveRunnerBinary() = %q, want configured path", got)
	}
}

// TestShouldRetryWithoutGPU 只允许 Metal 缓冲初始化失败触发 CPU 降级。
func TestShouldRetryWithoutGPU(t *testing.T) {
	if !shouldRetryWithoutGPU([]byte("ggml_metal_buffer_init: error: failed to allocate buffer")) {
		t.Fatal("shouldRetryWithoutGPU() = false, want true")
	}
	if shouldRetryWithoutGPU([]byte("model file not found")) {
		t.Fatal("shouldRetryWithoutGPU() = true for a model error")
	}
}

// TestFirstOutputLine 验证版本探测只向界面返回第一条有效信息。
func TestFirstOutputLine(t *testing.T) {
	if got := firstOutputLine("\nwhisper.cpp 1.9.1\nbuild info"); got != "whisper.cpp 1.9.1" {
		t.Fatalf("firstOutputLine() = %q", got)
	}
}

// TestFirstOutputLinePrefersVersion 验证动态后端日志不会冒充版本号。
func TestFirstOutputLinePrefersVersion(t *testing.T) {
	output := "load_backend: loaded Metal\nwhisper.cpp version: 1.9.1\n"
	if got := firstOutputLine(output); got != "whisper.cpp version: 1.9.1" {
		t.Fatalf("firstOutputLine() = %q", got)
	}
}

// TestBundledLibraryDir 验证仅应用包内命令会注入动态库搜索目录。
func TestBundledLibraryDir(t *testing.T) {
	got := bundledLibraryDir("/Applications/VDub.app/Contents/Resources/whisper/bin/whisper-cli")
	want := "/Applications/VDub.app/Contents/Resources/whisper/lib"
	if got != want {
		t.Fatalf("bundledLibraryDir() = %q, want %q", got, want)
	}
	if got := bundledLibraryDir("/opt/homebrew/bin/whisper-cli"); got != "" {
		t.Fatalf("system binary library dir = %q, want empty", got)
	}
}

// TestNotifyWhisperOutputClampsProgress 验证异常百分比不会进入任务进度和日志。
func TestNotifyWhisperOutputClampsProgress(t *testing.T) {
	progress := -1
	logLine := ""
	notifyWhisperOutput(
		"whisper_print_progress_callback: progress = 150%",
		func(value int) { progress = value },
		func(line string) { logLine = line },
	)
	if progress != 100 || logLine != "听写进度 100%" {
		t.Fatalf("progress=%d log=%q", progress, logLine)
	}
}
