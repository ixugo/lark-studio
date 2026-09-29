package wails

import (
	"context"

	"github.com/ixugo/vdub/internal/adapter/remoteapi"
)

// ListRemoteModels 返回配置页可选择的远程模型，不使用默认模型或本地兜底列表。
func (s *AppService) ListRemoteModels(baseURL, apiKey string) ([]remoteapi.Model, error) {
	return remoteapi.ListModels(context.Background(), baseURL, apiKey)
}
