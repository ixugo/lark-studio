// Package tts 实现 TTS 适配器
// 支持 edge-tts 和 OpenAI 兼容协议
package tts

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// EdgeTTS 通过 edge-tts 命令行工具合成语音
// 需要系统安装 edge-tts: pip install edge-tts
type EdgeTTS struct {
	voice      string
	retryDelay time.Duration
	maxRetries int
}

// NewEdgeTTS 创建 edge-tts 适配器
func NewEdgeTTS(voice string) *EdgeTTS {
	if voice == "" {
		voice = "zh-CN-YunjianNeural"
	}
	return &EdgeTTS{
		voice:      voice,
		retryDelay: 30 * time.Second,
		maxRetries: 3,
	}
}

// Synthesize 合成语音
func (e *EdgeTTS) Synthesize(ctx context.Context, text, outputPath, voice string) error {
	if voice == "" {
		voice = e.voice
	}

	var lastErr error
	for attempt := 0; attempt <= e.maxRetries; attempt++ {
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

		// 限流重试
		if attempt < e.maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(e.retryDelay):
			}
		}
	}

	return lastErr
}
