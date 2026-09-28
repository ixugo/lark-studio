package whisper

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestInstallRuntimeArchive 验证完整归档会原子安装并保留可执行文件。
func TestInstallRuntimeArchive(t *testing.T) {
	archive := writeRuntimeArchive(t, "bin/"+runtimeBinaryName())
	destination := filepath.Join(t.TempDir(), "runtime")

	if err := installRuntimeArchive(archive, destination); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "bin", runtimeBinaryName())); err != nil {
		t.Fatalf("运行时命令未安装: %v", err)
	}
}

// TestInstallRuntimeArchiveRejectsTraversal 验证归档路径不能写出运行时目录。
func TestInstallRuntimeArchiveRejectsTraversal(t *testing.T) {
	archive := writeRuntimeArchive(t, "../outside")

	if err := installRuntimeArchive(archive, filepath.Join(t.TempDir(), "runtime")); err == nil {
		t.Fatal("路径穿越归档未被拒绝")
	}
}

// TestRuntimeAssetURL 验证修复运行时始终请求当前系统对应的发布资产。
func TestRuntimeAssetURL(t *testing.T) {
	want := "vdub-whisper-" + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"
	if got := runtimeAssetURL("v1.2.3"); filepath.Base(got) != want {
		t.Fatalf("运行时资产 = %q，期望后缀 %q", got, want)
	}
}

// TestBinaryAtPrefixesFindsWhisperCLI 验证 PATH 缺失时仍能从已知目录发现运行时。
func TestBinaryAtPrefixesFindsWhisperCLI(t *testing.T) {
	prefix := t.TempDir()
	binary := filepath.Join(prefix, "whisper-cli")
	if err := os.WriteFile(binary, []byte("runtime"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := binaryAtPrefixes([]string{"whisper-cpp", "whisper-cli"}, []string{prefix}); got != binary {
		t.Fatalf("binaryAtPrefixes() = %q，期望 %q", got, binary)
	}
}

// TestInspectHomebrewRuntimeWithoutPATH 复现 Finder 启动应用时缺少 Homebrew PATH 的环境。
func TestInspectHomebrewRuntimeWithoutPATH(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Homebrew 路径探测仅适用于 macOS")
	}
	if _, err := os.Stat("/opt/homebrew/bin/whisper-cli"); err != nil {
		t.Skip("当前设备未安装 Apple Silicon Homebrew whisper-cli")
	}
	t.Setenv("PATH", "/usr/bin:/bin")

	info := InspectRuntime("whisper-cpp")
	if !info.Installed || info.Binary != "/opt/homebrew/bin/whisper-cli" {
		t.Fatalf("Homebrew 运行时未通过无 PATH 探测：%+v", info)
	}
}

// TestBundledWhisperCLIIsProbeable 验证构建入 vendor 的运行时能通过真实命令探测。
func TestBundledWhisperCLIIsProbeable(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("当前平台不能执行 macOS ARM64 运行时")
	}
	binary := filepath.Join("..", "..", "..", "vendor", "whisper", "darwin", "bin", "whisper-cli")
	if _, err := os.Stat(binary); err != nil {
		t.Skip("当前平台未构建 vendor whisper runtime")
	}
	info := InspectRuntime(binary)
	if !info.Installed {
		t.Fatalf("vendor whisper runtime 未通过探测：%+v", info)
	}
}

// writeRuntimeArchive 生成最小运行时归档，供安装边界测试使用。
func writeRuntimeArchive(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "runtime.tar.gz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	header := &tar.Header{Name: name, Mode: 0o755, Size: int64(len("runtime"))}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write([]byte("runtime")); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
