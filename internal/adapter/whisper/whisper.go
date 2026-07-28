// Package whisper 实现语音识别适配器
// 支持 whisper.cpp 命令行 和 ffmpeg 内置 whisper 滤镜两种模式
package whisper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
func (r *Runner) Transcribe(ctx context.Context, audioPath, outputSRT, lang string) error {
	if lang == "" {
		lang = "auto"
	}

	outputBase := strings.TrimSuffix(outputSRT, filepath.Ext(outputSRT))
	args := []string{
		"-m", r.model,
		"-f", audioPath,
		"-l", lang,
		"--output-srt",
		"-of", outputBase,
		"--no-timestamps",
	}

	output, err := r.run(ctx, args)
	if err != nil && shouldRetryWithoutGPU(output) {
		args = append(args, "--no-gpu")
		output, err = r.run(ctx, args)
	}
	if err != nil {
		return fmt.Errorf("whisper.cpp 执行失败: %s\noutput: %s", err, string(output))
	}
	return nil
}

// run 执行一次 whisper.cpp 命令，保留标准输出供失败诊断与降级判断使用。
func (r *Runner) run(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, r.bin, args...)
	if libDir := bundledLibraryDir(r.bin); libDir != "" {
		cmd.Env = append(
			os.Environ(),
			"DYLD_LIBRARY_PATH="+libDir,
			"GGML_BACKEND_PATH="+filepath.Join(libDir, "backends"),
		)
	}
	return cmd.CombinedOutput()
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

// FFmpegRunner 使用 ffmpeg 内置 whisper 滤镜做语音识别
// 适用于 ffmpeg 编译时启用了 --enable-whisper 的场景，无需单独安装 whisper.cpp
type FFmpegRunner struct {
	ffmpegBin string // ffmpeg 路径
	model     string // ggml 模型文件路径
}

// NewFFmpegRunner 创建 ffmpeg whisper runner
func NewFFmpegRunner(ffmpegBin, model string) *FFmpegRunner {
	if ffmpegBin == "" {
		ffmpegBin = "ffmpeg"
	}
	return &FFmpegRunner{ffmpegBin: ffmpegBin, model: model}
}

// Transcribe 通过 ffmpeg whisper 滤镜转写音频为 SRT
func (r *FFmpegRunner) Transcribe(ctx context.Context, audioPath, outputSRT, lang string) error {
	if lang == "" {
		lang = "auto"
	}

	filter := fmt.Sprintf("whisper=model=%s:language=%s:format=srt:destination=%s",
		r.model, lang, outputSRT,
	)

	cmd := exec.CommandContext(ctx, r.ffmpegBin,
		"-y", "-i", audioPath,
		"-af", filter,
		"-f", "null", "-",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg whisper 转写失败: %s\noutput: %s", err, string(output))
	}
	return nil
}
