package tts

import (
	"context"
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
	if voice == "" {
		voice = t.voice
	}

	body := fmt.Sprintf(`{"model":"%s","input":"%s","voice":"%s"}`,
		t.model,
		strings.ReplaceAll(text, `"`, `\"`),
		voice,
	)

	req, err := http.NewRequestWithContext(ctx, "POST",
		t.baseURL+"/audio/speech",
		strings.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("TTS API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("TTS API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("写入音频失败: %w", err)
	}

	return nil
}
