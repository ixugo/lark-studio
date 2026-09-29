// Package remoteapi 查询 OpenAI 兼容服务公开的模型列表。
package remoteapi

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"
)

const (
	// 模型目录请求只用于配置页，避免不可达服务长时间阻塞交互。
	modelRequestTimeout = 15 * time.Second
	maxResponseBytes    = 1 << 20
	maxErrorBytes       = 1024
	maxAddressBytes     = 4096
	maxAPIKeyBytes      = 8192
	maxModelIDBytes     = 512
)

// Model 是服务返回的模型标识，不推断模型的语音或翻译能力。
type Model struct {
	ID string `json:"id"`
}

// HTTPError 保留服务状态码，调用方可以识别尚未支持的能力接口。
type HTTPError struct {
	StatusCode int
	Detail     string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("查询服务失败: HTTP %d: %s", e.StatusCode, e.Detail)
}

func modelHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       modelRequestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// ListModels 查询服务的模型目录，拒绝空列表并按标识去重排序。
func ListModels(ctx context.Context, baseURL, apiKey string) ([]Model, error) {
	return listModels(ctx, modelHTTPClient(), baseURL, apiKey)
}

func listModels(ctx context.Context, client *http.Client, baseURL, apiKey string) ([]Model, error) {
	endpoint, err := modelsEndpoint(baseURL)
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []Model `json:"data"`
	}
	if err := getJSON(ctx, client, endpoint, apiKey, &response); err != nil {
		return nil, err
	}
	return validateModels(response.Data)
}

// ValidateSelection 重新查询服务，确保提交的模型仍在远程目录中。
func ValidateSelection(ctx context.Context, baseURL, apiKey, model string) error {
	if model == "" || len(model) > maxModelIDBytes || strings.TrimSpace(model) != model || strings.ContainsFunc(model, unicode.IsControl) {
		return fmt.Errorf("请从远程模型列表中选择模型")
	}
	models, err := ListModels(ctx, baseURL, apiKey)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(models, func(candidate Model) bool { return candidate.ID == model }) {
		return fmt.Errorf("所选模型不在远程模型列表中，请重新获取并选择")
	}
	return nil
}

// GetJSON 查询明确的完整接口地址，支持音色目录等带查询参数的接口。
func GetJSON(ctx context.Context, endpoint, apiKey string, target any) error {
	return getJSON(ctx, modelHTTPClient(), endpoint, apiKey, target)
}

func getJSON(ctx context.Context, client *http.Client, endpoint, apiKey string, target any) error {
	address, err := url.Parse(endpoint)
	if err != nil || len(endpoint) > maxAddressBytes || (address.Scheme != "http" && address.Scheme != "https") || address.Hostname() == "" || address.User != nil || address.Fragment != "" {
		return fmt.Errorf("服务接口地址无效")
	}
	apiKey = strings.TrimSpace(apiKey)
	if len(apiKey) > maxAPIKeyBytes || strings.ContainsAny(apiKey, "\r\n") {
		return fmt.Errorf("API 密钥格式无效")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("创建查询请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("查询服务失败: %w", err)
	}
	defer resp.Body.Close()
	return decodeJSONResponse(resp, apiKey, target)
}

func decodeJSONResponse(resp *http.Response, apiKey string, target any) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("读取服务响应失败: %w", err)
	}
	if len(body) > maxResponseBytes {
		return fmt.Errorf("服务响应过大，最多允许 %d 字节", maxResponseBytes)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{StatusCode: resp.StatusCode, Detail: responseDetail(body, apiKey)}
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("服务响应格式无效: %s", responseDetail([]byte(err.Error()), apiKey))
	}
	return nil
}

func modelsEndpoint(baseURL string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if len(baseURL) == 0 || len(baseURL) > maxAddressBytes {
		return "", fmt.Errorf("请填写有效的服务地址")
	}
	address, err := url.Parse(baseURL)
	if err != nil || (address.Scheme != "http" && address.Scheme != "https") || address.Hostname() == "" || address.User != nil || address.RawQuery != "" || address.Fragment != "" {
		return "", fmt.Errorf("服务地址必须是 HTTP 或 HTTPS 地址，不可包含账号、查询参数或片段")
	}
	// 明确支持常见的完整操作地址，保留服务的自定义路径前缀。
	path := strings.TrimRight(address.Path, "/")
	for _, suffix := range []string{"/chat/completions", "/audio/transcriptions", "/audio/speech", "/models"} {
		if prefix, ok := strings.CutSuffix(path, suffix); ok {
			path = prefix
			break
		}
	}
	address.Path = path + "/models"
	address.RawPath = ""
	return address.String(), nil
}

func validateModels(models []Model) ([]Model, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("服务没有返回可用模型，请检查服务地址与权限")
	}
	for _, model := range models {
		if len(model.ID) == 0 || len(model.ID) > maxModelIDBytes || strings.TrimSpace(model.ID) != model.ID || strings.ContainsFunc(model.ID, unicode.IsControl) {
			return nil, fmt.Errorf("服务返回了无效的模型标识")
		}
	}
	slices.SortFunc(models, func(a, b Model) int { return cmp.Compare(a.ID, b.ID) })
	return slices.Compact(models), nil
}

func responseDetail(body []byte, apiKey string) string {
	detail := strings.TrimSpace(string(body))
	if apiKey != "" {
		detail = strings.ReplaceAll(detail, apiKey, "[已隐藏密钥]")
	}
	if len(detail) > maxErrorBytes {
		detail = detail[:maxErrorBytes] + "…"
	}
	if detail == "" {
		return "服务未返回错误详情"
	}
	return detail
}
