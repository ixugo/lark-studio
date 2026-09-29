// Package whisper 实现 whisper.cpp 语音识别适配器。
package whisper

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/ixugo/vdub/internal/conf"
)

var whisperProgressPattern = regexp.MustCompile(`progress\s*=\s*(\d+)%`)

var whisperDiagnosticPrefixes = []string{
	"whisper_vad:",
	"whisper_vad_segments_from_",
	"whisper_print_timings:",
}

// Runner whisper.cpp 命令行调用实现
type Runner struct {
	bin   string // whisper.cpp 主程序路径 (如 whisper-cpp)
	model string // 模型文件路径
}

// NewRunner 创建 whisper runner
// bin: whisper.cpp 可执行文件路径
// model: ggml 模型文件路径
func NewRunner(bin, model string) *Runner {
	return &Runner{bin: ResolveBinary(bin), model: model}
}

// resolveRunnerBinary 兼容旧配置名，优先使用配置指定的 whisper.cpp 二进制。
func resolveRunnerBinary(bin string, lookPath func(string) (string, error)) string {
	if bin != "whisper-cpp" {
		return bin
	}
	if _, err := lookPath(bin); err == nil {
		return bin
	}
	if cli, err := lookPath("whisper-cli"); err == nil {
		return cli
	}
	return bin
}

// Transcribe 执行语音识别
func (r *Runner) Transcribe(
	ctx context.Context,
	audioPath string,
	outputSRT string,
	lang string,
	onProgress func(int),
	onLog func(string),
) error {
	model, err := ResolveModel(r.model)
	if err != nil {
		return err
	}
	if onLog != nil {
		onLog("语音识别模型：" + model)
	}
	if lang == "" {
		lang = "auto"
	}
	vadModel, err := EnsureSpeechModel(filepath.Join(conf.StudioDir(), "models"))
	if err != nil {
		return fmt.Errorf("准备人声检测模型失败: %w", err)
	}

	outputBase := strings.TrimSuffix(outputSRT, filepath.Ext(outputSRT))
	args := []string{
		"-m", model,
		"-f", audioPath,
		"-l", lang,
		"--output-srt",
		"-of", outputBase,
		"--vad", "--vad-model", vadModel,
		"--vad-speech-pad-ms", strconv.Itoa(speechPaddingMillis),
		"--print-progress",
	}

	output, err := r.run(ctx, args, onProgress, onLog)
	if err != nil && shouldRetryWithoutGPU(output) {
		if onLog != nil {
			onLog("Metal 初始化失败，切换 CPU 兼容模式")
		}
		args = append(args, "--no-gpu")
		output, err = r.run(ctx, args, onProgress, onLog)
	}
	if err != nil {
		return fmt.Errorf("whisper.cpp 执行失败: %s\noutput: %s", err, string(output))
	}
	return nil
}

// run 执行一次 whisper.cpp 命令，保留标准输出供失败诊断与降级判断使用。
func (r *Runner) run(
	ctx context.Context,
	args []string,
	onProgress func(int),
	onLog func(string),
) ([]byte, error) {
	cmd := r.command(ctx, args)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var output bytes.Buffer
	var outputMu sync.Mutex
	var wg sync.WaitGroup
	scanErrors := make(chan error, 2)
	wg.Add(2)
	go consumeOutput(stdout, &output, &outputMu, onProgress, onLog, scanErrors, &wg)
	go consumeOutput(stderr, &output, &outputMu, onProgress, onLog, scanErrors, &wg)
	waitErr := cmd.Wait()
	wg.Wait()
	close(scanErrors)
	if waitErr == nil {
		for scanErr := range scanErrors {
			if scanErr != nil {
				waitErr = scanErr
				break
			}
		}
	}
	return output.Bytes(), waitErr
}

// command 创建 Whisper 命令，并为应用包内运行时补齐动态库环境。
func (r *Runner) command(ctx context.Context, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, r.bin, args...)
	if libDir := bundledLibraryDir(r.bin); libDir != "" {
		cmd.Env = append(
			os.Environ(),
			"DYLD_LIBRARY_PATH="+libDir,
			"GGML_BACKEND_PATH="+filepath.Join(libDir, "backends"),
		)
	}
	return cmd
}

// consumeOutput 逐行收集命令输出，同时提取 whisper.cpp 百分比。
func consumeOutput(
	reader interface{ Read([]byte) (int, error) },
	output *bytes.Buffer,
	outputMu *sync.Mutex,
	onProgress func(int),
	onLog func(string),
	errCh chan<- error,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		appendCommandOutput(output, outputMu, line)
		notifyWhisperOutput(line, onProgress, onLog)
	}
	errCh <- scanner.Err()
}

// appendCommandOutput 加锁保存双管道输出，供失败诊断和 GPU 降级判断使用。
func appendCommandOutput(output *bytes.Buffer, outputMu *sync.Mutex, line string) {
	outputMu.Lock()
	defer outputMu.Unlock()
	output.WriteString(line)
	output.WriteByte('\n')
}

// notifyWhisperOutput 推送原始日志，并从进度行解析百分比。
func notifyWhisperOutput(line string, onProgress func(int), onLog func(string)) {
	if isWhisperDiagnostic(line) {
		return
	}
	matches := whisperProgressPattern.FindStringSubmatch(line)
	if len(matches) != 2 {
		if onLog != nil {
			onLog(line)
		}
		return
	}
	progress, err := strconv.Atoi(matches[1])
	if err != nil {
		return
	}
	progress = max(0, min(100, progress))
	if onProgress != nil {
		onProgress(progress)
	}
	if onLog != nil {
		onLog(fmt.Sprintf("听写进度 %d%%", progress))
	}
}

// isWhisperDiagnostic 判断只供调试的库内部统计行，避免挤占用户转录日志。
func isWhisperDiagnostic(line string) bool {
	for _, prefix := range whisperDiagnosticPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

// bundledLibraryDir 返回应用包内动态库目录，系统安装的命令无需修改环境。
func bundledLibraryDir(binary string) string {
	binDir := filepath.Dir(binary)
	if filepath.Base(binDir) != "bin" || filepath.Base(filepath.Dir(binDir)) != "whisper" {
		return ""
	}
	return filepath.Join(filepath.Dir(binDir), "lib")
}

// shouldRetryWithoutGPU 仅在 Metal 初始化失败时改用 CPU，避免掩盖模型和音频错误。
func shouldRetryWithoutGPU(output []byte) bool {
	text := string(output)
	return strings.Contains(text, "ggml_metal_buffer_init") ||
		strings.Contains(text, "failed to allocate buffer")
}
