package wails

import (
	"context"

	"github.com/ixugo/vdub/internal/adapter/tts"
)

// GetTTSCapabilities 返回所选服务和模型的音色及扩展能力。
func (s *AppService) GetTTSCapabilities(baseURL, apiKey, model string) (tts.Capabilities, error) {
	return tts.DiscoverCapabilities(context.Background(), baseURL, apiKey, model)
}
