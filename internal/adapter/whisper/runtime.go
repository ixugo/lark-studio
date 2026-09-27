package whisper

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ixugo/vdub/internal/conf"
)

const runtimeDownloadTimeout = 10 * time.Minute

// RuntimeInstallOptions 指定缺失运行时的下载位置和目标目录。
type RuntimeInstallOptions struct {
	ReleaseTag  string
	DownloadURL string
	DataDir     string
}

// RuntimeInfo 描述当前实际可用的 whisper.cpp 命令行环境。
type RuntimeInfo struct {
	Installed    bool   `json:"installed"`
	Binary       string `json:"binary"`
	Version      string `json:"version"`
	Acceleration string `json:"acceleration"`
}

// ResolveBinary 优先使用用户或系统已有的命令，再使用应用内嵌或已下载的运行时。
func ResolveBinary(configured string) string {
	if strings.ContainsRune(configured, filepath.Separator) {
		return configured
	}
	if binary := systemBinaryPath(configured); binary != "" {
		return binary
	}
	if bundled := bundledBinaryPath(); bundled != "" {
		return bundled
	}
	if installed := installedBinaryPath(conf.DataDir()); installed != "" {
		return installed
	}
	return runtimeBinaryName()
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
	version := firstOutputLine(string(output))
	if err != nil {
		output, err = exec.CommandContext(
			ctx, path, "-m", "__vdub_runtime_probe_missing__.bin", "-f", runtimeProbeInputPath(),
		).CombinedOutput()
		if !strings.Contains(string(output), "failed to open") {
			return RuntimeInfo{Binary: path}
		}
		version = "whisper.cpp"
	}
	return RuntimeInfo{
		Installed:    true,
		Binary:       path,
		Version:      version,
		Acceleration: runtimeAcceleration(),
	}
}

// runtimeProbeInputPath 提供一个无需读取的输入路径，使 CLI 先验证缺失模型后退出。
func runtimeProbeInputPath() string {
	if runtime.GOOS == "windows" {
		return "NUL"
	}
	return "/dev/null"
}

// InstallRuntime 下载与当前发布版本匹配的官方 whisper.cpp 运行时。
func InstallRuntime(ctx context.Context, options RuntimeInstallOptions, logFn func(string)) error {
	if options.DataDir == "" {
		options.DataDir = conf.DataDir()
	}
	if options.DownloadURL == "" {
		if options.ReleaseTag == "" || options.ReleaseTag == "dev" {
			return fmt.Errorf("开发版本未配置运行时下载地址")
		}
		options.DownloadURL = runtimeAssetURL(options.ReleaseTag)
	}
	if logFn != nil {
		logFn("正在下载 whisper.cpp 运行时")
	}
	archive, err := downloadRuntimeArchive(ctx, options.DownloadURL, logFn)
	if err != nil {
		return err
	}
	defer os.Remove(archive)
	if err := installRuntimeArchive(archive, runtimeInstallDir(options.DataDir)); err != nil {
		return err
	}
	if logFn != nil {
		logFn("whisper.cpp 运行时安装完成")
	}
	return nil
}

// bundledBinaryPath 查找与 Go 引擎一起打包的 whisper-cli。
func bundledBinaryPath() string {
	executable, err := os.Executable()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(filepath.Dir(executable), "whisper", "bin", runtimeBinaryName())
	info, err := os.Stat(candidate)
	if err == nil && !info.IsDir() {
		return candidate
	}
	return ""
}

// systemBinaryPath 查找系统环境已有的兼容命令，macOS 26 未提供时会自然回退到应用运行时。
func systemBinaryPath(configured string) string {
	for _, candidate := range []string{configured, "whisper-cli", "whisper-cpp"} {
		if candidate == "" {
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return ""
}

// installedBinaryPath 返回安装到应用统一资源目录 ~/.lark-studio 的运行时命令。
func installedBinaryPath(dataDir string) string {
	candidate := filepath.Join(conf.StudioDir(), "runtime", "whisper", runtime.GOOS, runtime.GOARCH, "bin", runtimeBinaryName())
	info, err := os.Stat(candidate)
	if err == nil && !info.IsDir() {
		return candidate
	}
	// 兼容旧路径
	oldCandidate := filepath.Join(runtimeInstallDir(dataDir), "bin", runtimeBinaryName())
	if info, err := os.Stat(oldCandidate); err == nil && !info.IsDir() {
		return oldCandidate
	}
	return ""
}

// runtimeInstallDir 为当前平台提供独立的运行时目录，默认在 ~/.lark-studio/runtime 下。
func runtimeInstallDir(dataDir string) string {
	return filepath.Join(conf.StudioDir(), "runtime", "whisper", runtime.GOOS, runtime.GOARCH)
}

// runtimeBinaryName 返回当前系统使用的 whisper.cpp 命令名。
func runtimeBinaryName() string {
	if runtime.GOOS == "windows" {
		return "whisper-cli.exe"
	}
	return "whisper-cli"
}

// runtimeAssetCandidateURLs 构造国内镜像加速节点与 GitHub 原生地址的候选列表。
func runtimeAssetCandidateURLs(releaseTag string) []string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	asset := fmt.Sprintf("vdub-whisper-%s-%s.%s", runtime.GOOS, runtime.GOARCH, ext)
	official := fmt.Sprintf("https://github.com/ixugo/vdub/releases/download/%s/%s", releaseTag, asset)
	return []string{
		"https://ghfast.top/" + official,
		"https://ghproxy.net/" + official,
		official,
	}
}

// runtimeAssetURL 探测并选出最低延迟连通的运行时下载地址。
func runtimeAssetURL(releaseTag string) string {
	candidates := runtimeAssetCandidateURLs(releaseTag)
	return ProbeFastestURL(context.Background(), candidates)
}

// downloadRuntimeArchive 下载运行时归档并把进度写入界面日志。
func downloadRuntimeArchive(ctx context.Context, url string, logFn func(string)) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: runtimeDownloadTimeout}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("下载运行时失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载运行时失败: HTTP %d", response.StatusCode)
	}
	file, err := os.CreateTemp("", "vdub-whisper-*.tar.gz")
	if err != nil {
		return "", err
	}
	if err := copyRuntimeArchive(file, response.Body, response.ContentLength, logFn); err != nil {
		file.Close()
		os.Remove(file.Name())
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

// copyRuntimeArchive 流式保存下载内容，避免大文件占用内存。
func copyRuntimeArchive(file *os.File, source io.Reader, total int64, logFn func(string)) error {
	buffer := make([]byte, 128*1024)
	var written int64
	lastProgress := -1
	for {
		n, err := source.Read(buffer)
		if n > 0 {
			if _, writeErr := file.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
			written += int64(n)
			lastProgress = notifyRuntimeDownloadProgress(written, total, lastProgress, logFn)
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// notifyRuntimeDownloadProgress 按 5% 节流输出进度，避免淹没实时日志。
func notifyRuntimeDownloadProgress(written, total int64, previous int, logFn func(string)) int {
	if total <= 0 {
		return previous
	}
	progress := int(written * 100 / total)
	if progress == previous || progress%5 != 0 || logFn == nil {
		return previous
	}
	logFn(fmt.Sprintf("运行时下载 %d%%", progress))
	return progress
}

// installRuntimeArchive 将可信归档解压到临时目录，再原子替换运行时目录。
func installRuntimeArchive(archivePath, destination string) error {
	temporary := destination + ".tmp"
	if err := os.RemoveAll(temporary); err != nil {
		return err
	}
	if err := extractRuntimeArchive(archivePath, temporary); err != nil {
		os.RemoveAll(temporary)
		return err
	}
	if err := verifyRuntimeBinary(temporary); err != nil {
		os.RemoveAll(temporary)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		os.RemoveAll(temporary)
		return err
	}
	if err := os.RemoveAll(destination); err != nil {
		os.RemoveAll(temporary)
		return err
	}
	return os.Rename(temporary, destination)
}

// extractRuntimeArchive 支持 .zip 与 .tar.gz 两种跨平台归档。
func extractRuntimeArchive(archivePath, destination string) error {
	if strings.HasSuffix(strings.ToLower(archivePath), ".zip") {
		return extractZipArchive(archivePath, destination)
	}

	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	reader := tar.NewReader(gzipReader)
	for {
		header, readErr := reader.Next()
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
		if err := extractRuntimeEntry(reader, header, destination); err != nil {
			return err
		}
	}
}

// extractZipArchive 解压 Windows 平台 .zip 归档。
func extractZipArchive(archivePath, destination string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target, err := safeRuntimePath(destination, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		dstFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(dstFile, rc)
		rc.Close()
		dstFile.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// extractRuntimeEntry 写入单个常规文件或目录，不接受链接和越界路径。
func extractRuntimeEntry(reader *tar.Reader, header *tar.Header, destination string) error {
	target, err := safeRuntimePath(destination, header.Name)
	if err != nil {
		return err
	}
	switch header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(target, 0o755)
	case tar.TypeReg:
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, header.FileInfo().Mode())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(file, reader)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	default:
		return fmt.Errorf("运行时归档含不支持条目: %s", header.Name)
	}
}

// safeRuntimePath 约束归档路径始终落在指定目录内。
func safeRuntimePath(destination, name string) (string, error) {
	clean := filepath.Clean(name)
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("运行时归档路径非法: %s", name)
	}
	return filepath.Join(destination, clean), nil
}

// verifyRuntimeBinary 确认归档确实包含当前平台可以执行的命令。
func verifyRuntimeBinary(directory string) error {
	info, err := os.Stat(filepath.Join(directory, "bin", runtimeBinaryName()))
	if err != nil || info.IsDir() {
		return fmt.Errorf("运行时归档缺少 %s", runtimeBinaryName())
	}
	return nil
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
