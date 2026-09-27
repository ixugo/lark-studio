package tts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// OpenAITTS 通过 OpenAI 兼容 API 合成语音
// 支持 CosyVoice3 等提供 /v1/audio/speech 接口的服务
type OpenAITTS struct {
	baseURL string
	apiKey  string
	model   string
	voice   string
	client  *http.Client
}

// NewOpenAITTS 创建 OpenAI 兼容 TTS 适配器
func NewOpenAITTS(baseURL, apiKey, model, voice string) *OpenAITTS {
	return &OpenAITTS{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		voice:   voice,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// Synthesize 调用 OpenAI TTS API 合成语音
func (t *OpenAITTS) Synthesize(ctx context.Context, text, outputPath, voice string) error {
	return t.SynthesizeWithSpeed(ctx, text, outputPath, voice, 1)
}

// SynthesizeWithSpeed 使用任务指定的音色与语速调用 OpenAI 兼容接口。
func (t *OpenAITTS) SynthesizeWithSpeed(
	ctx context.Context,
	text string,
	outputPath string,
	voice string,
	speed float64,
) error {
	data, _, err := t.SynthesizeBytes(ctx, text, voice, speed)
	if err != nil {
		return err
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("写入音频失败: %w", err)
	}
	return nil
}

// SynthesizeBytes 返回试听音频与服务端格式，供界面直接播放临时合成结果。
func (t *OpenAITTS) SynthesizeBytes(ctx context.Context, text, voice string, speed float64) ([]byte, string, error) {
	if voice == "" {
		voice = t.voice
	}
	if speed <= 0 {
		speed = 1
	}
	req, err := t.speechRequest(ctx, text, voice, speed)
	if err != nil {
		return nil, "", err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("TTS API 请求失败: %w", err)
	}
	defer resp.Body.Close()
	return readSpeechResponse(resp)
}

// speechRequest 统一构造带模型、音色与语速的兼容接口请求。
func (t *OpenAITTS) speechRequest(ctx context.Context, text, voice string, speed float64) (*http.Request, error) {
	body, err := json.Marshal(map[string]any{
		"model": t.model,
		"input": text,
		"voice": voice,
		"speed": speed,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST",
		t.baseURL+"/audio/speech",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}
	return req, nil
}

// readSpeechResponse 读取兼容服务的音频或错误详情，供试听与任务合成共用。
func readSpeechResponse(resp *http.Response) ([]byte, string, error) {
	if resp.StatusCode != http.StatusOK {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, "", fmt.Errorf("读取 TTS 错误响应失败: %w", readErr)
		}
		return nil, "", fmt.Errorf("TTS API 返回 %d: %s", resp.StatusCode, string(respBody))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("读取 TTS 音频失败: %w", err)
	}
	return data, resp.Header.Get("Content-Type"), nil
}
