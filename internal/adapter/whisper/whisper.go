// Package whisper 实现语音识别适配器
// 支持 whisper.cpp 命令行 和 ffmpeg 内置 whisper 滤镜两种模式
package whisper

import (
	"context"
	"fmt"
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
	return &Runner{bin: bin, model: model}
}

// Transcribe 执行语音识别
func (r *Runner) Transcribe(ctx context.Context, audioPath, outputSRT, lang string) error {
	if lang == "" || lang == "auto" {
		lang = "en"
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

	cmd := exec.CommandContext(ctx, r.bin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("whisper.cpp 执行失败: %s\noutput: %s", err, string(output))
	}
	return nil
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
