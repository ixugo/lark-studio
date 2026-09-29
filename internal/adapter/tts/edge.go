// Package tts 实现 TTS 适配器
// 支持 edge-tts 和 OpenAI 兼容协议
package tts

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const edgeTTSMaxRetries = 3

// EdgeTTS 通过 edge-tts 命令行工具合成语音
// 需要系统安装 edge-tts: pip install edge-tts
type EdgeTTS struct {
	voice string
	run   func(context.Context, string, ...string) ([]byte, error)
	wait  func(context.Context, time.Duration) error
}

// NewEdgeTTS 创建 edge-tts 适配器
func NewEdgeTTS(voice string) *EdgeTTS {
	if voice == "" {
		voice = "zh-CN-XiaoxiaoNeural"
	}
	return &EdgeTTS{voice: voice}
}

// Synthesize 合成语音，失败时指数退避重试
func (e *EdgeTTS) Synthesize(ctx context.Context, text, outputPath, voice string) error {
	return e.SynthesizeWithSpeed(ctx, text, outputPath, voice, 1)
}

// SynthesizeWithSpeed 使用任务指定的音色与语速合成语音。
func (e *EdgeTTS) SynthesizeWithSpeed(
	ctx context.Context,
	text string,
	outputPath string,
	voice string,
	speed float64,
) error {
	if voice == "" {
		voice = e.voice
	}

	binary, err := edgeTTSBinary()
	if err != nil {
		return err
	}

	run := e.run
	if run == nil {
		run = func(ctx context.Context, binary string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, binary, args...).CombinedOutput()
		}
	}
	wait := e.wait
	if wait == nil {
		wait = waitEdgeRetry
	}
	delays := [...]time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}
	var lastErr error
	for attempt := 0; attempt <= edgeTTSMaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		// 未完整合成的文件不进入缓存，重试之间也不复用部分音频。
		tmp, err := os.CreateTemp(filepath.Dir(outputPath), ".edge-*"+filepath.Ext(outputPath))
		if err != nil {
			return err
		}
		temporary := tmp.Name()
		if err := tmp.Close(); err != nil {
			return errors.Join(err, os.Remove(temporary))
		}
		args := []string{"--voice", voice, "--text", text, "--write-media", temporary}
		if rate := edgeRate(speed); rate != "+0%" {
			args = append(args, "--rate", rate)
		}
		output, err := run(ctx, binary, args...)
		if err == nil {
			info, statErr := os.Stat(temporary)
			if statErr != nil {
				err = statErr
			} else if !info.Mode().IsRegular() || info.Size() == 0 {
				err = fmt.Errorf("合成音频为空")
			}
		}
		if err == nil {
			err = os.Rename(temporary, outputPath)
			if err != nil {
				return errors.Join(err, os.Remove(temporary))
			}
			return nil
		}
		if cleanupErr := os.Remove(temporary); cleanupErr != nil && !os.IsNotExist(cleanupErr) {
			return errors.Join(err, cleanupErr)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = fmt.Errorf("edge-tts 失败（第 %d 次请求）: %w, output: %s", attempt+1, err, output)
		if attempt < edgeTTSMaxRetries {
			if err := wait(ctx, delays[attempt]); err != nil {
				return err
			}
		}
	}
	return &RetryExhaustedError{Err: lastErr}
}

// RetryExhaustedError 标记单文件重试预算已耗尽，防止外层流水线重新开始整组重试。
type RetryExhaustedError struct{ Err error }

func (e *RetryExhaustedError) Error() string {
	return fmt.Sprintf("Edge TTS 已重试 3 次仍失败: %v", e.Err)
}
func (e *RetryExhaustedError) Unwrap() error        { return e.Err }
func (e *RetryExhaustedError) RetryExhausted() bool { return true }

func waitEdgeRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// edgeRate 把倍率转换为 edge-tts 使用的百分比参数。
func edgeRate(speed float64) string {
	if speed <= 0 {
		speed = 1
	}
	percent := int(math.Round((speed - 1) * 100))
	return fmt.Sprintf("%+d%%", percent)
}

// edgeTTSBinary 沿用 Whisper 的 macOS 查找顺序，补足 Finder 启动时缺失的 Homebrew PATH。
func edgeTTSBinary() (string, error) {
	path, err := exec.LookPath("edge-tts")
	if err == nil {
		return path, nil
	}
	if runtime.GOOS == "darwin" {
		for _, candidate := range []string{"/opt/homebrew/bin/edge-tts", "/usr/local/bin/edge-tts"} {
			if path, lookupErr := exec.LookPath(candidate); lookupErr == nil {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("未找到 Edge TTS 命令，请安装 edge-tts: %w", err)
}
