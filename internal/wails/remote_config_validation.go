package wails

import (
	"context"
	"fmt"
	"strings"

	"github.com/ixugo/vdub/internal/adapter/remoteapi"
	"github.com/ixugo/vdub/internal/conf"
)

// validateRemoteConfig 只复核本次保存涉及的远程引擎，避免无关设置依赖旧服务联网。
func (s *AppService) validateRemoteConfig(updates map[string]any, next *conf.Bootstrap) error {
	if strings.EqualFold(strings.TrimSpace(next.Pipeline.WhisperMode), "openai") && hasRemoteConfigUpdate(updates, "pipeline", "whisper_mode", "asr_mode", "asr_model", "asr_base_url", "asr_api_key") {
		if err := remoteapi.ValidateSelection(context.Background(), next.Pipeline.ASRBaseURL, next.Pipeline.ASRAPIKey, next.Pipeline.ASRModel); err != nil {
			return fmt.Errorf("语音识别模型校验失败: %w", err)
		}
	}
	if strings.EqualFold(strings.TrimSpace(next.LLM.Provider), "openai") && hasRemoteConfigUpdate(updates, "llm", "provider", "model", "base_url", "api_key") {
		if err := remoteapi.ValidateSelection(context.Background(), next.LLM.BaseURL, next.LLM.APIKey, next.LLM.Model); err != nil {
			return fmt.Errorf("翻译模型校验失败: %w", err)
		}
	}
	return nil
}

func hasRemoteConfigUpdate(updates map[string]any, section string, fields ...string) bool {
	values, ok := updates[section].(map[string]any)
	if !ok {
		return false
	}
	for _, field := range fields {
		if _, exists := values[field]; exists {
			return true
		}
	}
	return false
}
