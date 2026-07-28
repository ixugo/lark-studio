package whisper

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// RuntimeInfo 描述当前实际可用的 whisper.cpp 命令行环境。
type RuntimeInfo struct {
	Installed    bool   `json:"installed"`
	Binary       string `json:"binary"`
	Version      string `json:"version"`
	Acceleration string `json:"acceleration"`
}

// ResolveBinary 优先返回应用包内引擎，再兼容系统 PATH 与旧命令名。
func ResolveBinary(configured string) string {
	if strings.ContainsRune(configured, filepath.Separator) {
		return configured
	}
	if bundled := bundledBinaryPath(); bundled != "" {
		return bundled
	}
	return resolveRunnerBinary(configured, exec.LookPath)
}

// InspectRuntime 实际执行版本探测，避免界面把未安装引擎显示为可用。
func InspectRuntime(configured string) RuntimeInfo {
	binary := ResolveBinary(configured)
	path, err := exec.LookPath(binary)
	if err != nil {
		return RuntimeInfo{Binary: binary}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, path, "--version").CombinedOutput()
	if err != nil {
		return RuntimeInfo{Binary: path}
	}
	return RuntimeInfo{
		Installed:    true,
		Binary:       path,
		Version:      firstOutputLine(string(output)),
		Acceleration: runtimeAcceleration(),
	}
}

// InstallRuntime 在 macOS 上通过 Homebrew 安装官方 whisper.cpp 命令行工具。
func InstallRuntime(ctx context.Context, logFn func(string)) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("当前系统不支持自动安装")
	}
	brew, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf("未找到 Homebrew，请先安装 Homebrew")
	}
	cmd := exec.CommandContext(ctx, brew, "install", "whisper-cpp")
	return streamCommand(cmd, logFn)
}

// bundledBinaryPath 查找与 Go 引擎一起打包的 whisper-cli。
func bundledBinaryPath() string {
	executable, err := os.Executable()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(filepath.Dir(executable), "whisper", "bin", "whisper-cli")
	info, err := os.Stat(candidate)
	if err == nil && !info.IsDir() {
		return candidate
	}
	return ""
}

// streamCommand 同时转发命令的标准输出与错误输出，安装过程不会静默。
func streamCommand(cmd *exec.Cmd, logFn func(string)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	scanErrors := make(chan error, 2)
	wg.Add(2)
	go scanCommandOutput(stdout, logFn, scanErrors, &wg)
	go scanCommandOutput(stderr, logFn, scanErrors, &wg)
	wg.Wait()
	close(scanErrors)
	for scanErr := range scanErrors {
		if scanErr != nil {
			return scanErr
		}
	}
	return cmd.Wait()
}

// scanCommandOutput 按行转发安装输出，空行不进入界面日志。
func scanCommandOutput(
	reader interface{ Read([]byte) (int, error) },
	logFn func(string),
	errCh chan<- error,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && logFn != nil {
			logFn(line)
		}
	}
	errCh <- scanner.Err()
}

// firstOutputLine 提取适合界面展示的单行版本信息。
func firstOutputLine(output string) string {
	first := ""
	for _, line := range strings.Split(output, "\n") {
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		if first == "" {
			first = text
		}
		if strings.Contains(strings.ToLower(text), "version") {
			return text
		}
	}
	return first
}

// runtimeAcceleration 返回当前平台能够使用的默认加速名称。
func runtimeAcceleration() string {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return "Metal"
	}
	return "CPU"
}
