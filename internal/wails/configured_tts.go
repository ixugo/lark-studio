package wails

import (
	"context"
	"encoding/base64"
	"fmt"
	ttsadapter "github.com/ixugo/vdub/internal/adapter/tts"
	"github.com/ixugo/vdub/internal/conf"
	"strings"
	"time"
)

func ttsSpeechOptions(config conf.TTS) ttsadapter.SpeechOptions {
	return ttsadapter.SpeechOptions{Protocol: config.Protocol, Language: config.Language, Instructions: config.Instructions}
}

// TestConfiguredTTS 让试听与任务使用相同的协议、语言及情绪参数。
func (s *AppService) TestConfiguredTTS(config conf.TTS, text string) (string, error) {
	if strings.TrimSpace(text) == "" || len(text) > 500 {
		return "", fmt.Errorf("试听文本为空或超过 500 字节")
	}

	if len(config.Voice) > 200 || strings.TrimSpace(config.Voice) == "" {
		return "", fmt.Errorf("请选择有效的试听音色")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := ttsadapter.ValidateSpeechSelection(ctx, config.BaseURL, config.APIKey, config.Model, config.Voice, ttsSpeechOptions(config)); err != nil {
		return "", err
	}
	client := ttsadapter.NewOpenAITTS(config.BaseURL, config.APIKey, config.Model, config.Voice).WithSpeechOptions(ttsSpeechOptions(config))
	data, media, err := client.SynthesizeBytes(ctx, text, config.Voice, 1)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("语音服务没有返回音频")
	}
	if !strings.HasPrefix(media, "audio/") {
		media = "audio/wav"
	}
	return "data:" + media + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
