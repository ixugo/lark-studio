package whisper

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestTranscribeRejectsInvalidModelBeforeStartingProcess(t *testing.T) {
	for _, model := range []string{"", "/missing/ggml-tiny.bin", "/models/ggml-silero-v6.2.0.bin", t.TempDir(), strings.Repeat("x", maxModelPathLength+1)} {
		t.Run(model, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "launched")
			binary := filepath.Join(t.TempDir(), "whisper-test")
			if err := os.WriteFile(binary, []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0700); err != nil {
				t.Fatal(err)
			}
			err := (&Runner{bin: binary, model: model}).Transcribe(t.Context(), "audio.wav", "output.srt", "auto", nil, nil)
			if err == nil || strings.Contains(err.Error(), "whisper.cpp 执行失败") {
				t.Fatalf("应在启动前报告模型错误: %v", err)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("无效模型启动了进程: %v", err)
			}
		})
	}
}

func TestTranscribeUsesResolvedCustomModelPath(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "custom-whisper.bin")
	if err := os.WriteFile(model, []byte("model fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "whisper-test")
	script := "#!/bin/sh\n[ \"$1\" = '-m' ] && [ \"$2\" = " + "'" + model + "'" + " ] || exit 2\n"
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	var logs []string
	if err := NewRunner(binary, model).Transcribe(t.Context(), "audio.wav", filepath.Join(dir, "out.srt"), "auto", nil, func(line string) { logs = append(logs, line) }); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(logs, "\n"), model) {
		t.Fatalf("日志缺少实际识别模型路径: %v", logs)
	}
}
