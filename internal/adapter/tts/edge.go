// Package tts 实现 TTS 适配器
// 支持 edge-tts 和 OpenAI 兼容协议
package tts

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

const (
	edgeTTSMaxRetries = 3
	edgeTTSBaseDelay  = 3 * time.Second
)

// EdgeTTS 通过 edge-tts 命令行工具合成语音
// 需要系统安装 edge-tts: pip install edge-tts
type EdgeTTS struct {
	voice string
}

// NewEdgeTTS 创建 edge-tts 适配器
func NewEdgeTTS(voice string) *EdgeTTS {
	if voice == "" {
		voice = "zh-CN-YunjianNeural"
	}
	return &EdgeTTS{voice: voice}
}

// Synthesize 合成语音，失败时指数退避重试
func (e *EdgeTTS) Synthesize(ctx context.Context, text, outputPath, voice string) error {
	if voice == "" {
		voice = e.voice
	}

	var lastErr error
	for attempt := range edgeTTSMaxRetries {
		if err := ctx.Err(); err != nil {
			return err
		}

		cmd := exec.CommandContext(ctx, "edge-tts",
			"--voice", voice,
			"--text", text,
			"--write-media", outputPath,
		)
		output, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}

		lastErr = fmt.Errorf("edge-tts 失败 (attempt %d): %s, output: %s", attempt+1, err, string(output))

		if attempt < edgeTTSMaxRetries-1 {
			delay := edgeTTSBaseDelay * time.Duration(1<<uint(attempt))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return lastErr
}
