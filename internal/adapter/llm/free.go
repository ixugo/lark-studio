package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	bingAuthURL      = "https://edge.microsoft.com/translate/auth"
	bingTranslateURL = "https://api-edge.cognitive.microsofttranslator.com/translate"
	bingTokenTTL     = 9 * time.Minute
	freeUserAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Edg/131.0.0.0"
)

// bingHTTPError 保留微软接口状态码，以便只在令牌失效时重新授权。
type bingHTTPError struct {
	statusCode int
}

// Error 将服务端状态转换为可展示的翻译错误。
func (e bingHTTPError) Error() string {
	return fmt.Sprintf("必应翻译返回状态 %d", e.statusCode)
}

// translateBing 使用 Edge 浏览器的匿名令牌批量翻译，结果与输入逐项对应。
func (c *Client) translateBing(ctx context.Context, texts []string, targetLang string) ([]string, error) {
	result, err := c.requestBing(ctx, texts, targetLang, false)
	if err == nil {
		return result, nil
	}
	if !needsBingTokenRefresh(err) {
		return nil, err
	}
	c.tokenMu.Lock()
	c.token = ""
	c.tokenMu.Unlock()
	return c.requestBing(ctx, texts, targetLang, true)
}

// requestBing 获取匿名令牌并请求微软翻译接口。
func (c *Client) requestBing(ctx context.Context, texts []string, targetLang string, force bool) ([]string, error) {
	token, err := c.bingToken(ctx, force)
	if err != nil {
		return nil, err
	}
	body := make([]map[string]string, len(texts))
	for i, text := range texts {
		body[i] = map[string]string{"Text": text}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := c.bingAPI + "?api-version=3.0&includeSentenceLength=true&to=" + bingLanguage(targetLang)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", freeUserAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("必应翻译请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, bingHTTPError{statusCode: resp.StatusCode}
	}
	var payload []struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("解析必应翻译失败: %w", err)
	}
	result := make([]string, len(payload))
	for i := range payload {
		if len(payload[i].Translations) == 0 {
			return nil, fmt.Errorf("必应翻译第 %d 项为空", i+1)
		}
		result[i] = payload[i].Translations[0].Text
	}
	return requireTranslationCount(result, len(texts))
}

// needsBingTokenRefresh 仅在授权失效时重新获取令牌，限流与业务错误直接返回。
func needsBingTokenRefresh(err error) bool {
	var responseErr bingHTTPError
	return errors.As(err, &responseErr) &&
		(responseErr.statusCode == http.StatusUnauthorized || responseErr.statusCode == http.StatusForbidden)
}

// bingToken 缓存短期匿名令牌，减少每批字幕的授权请求。
func (c *Client) bingToken(ctx context.Context, force bool) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if !force && c.token != "" && time.Since(c.tokenAt) < bingTokenTTL {
		return c.token, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.bingAuth, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", freeUserAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取必应翻译令牌失败: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取必应翻译令牌失败: 状态 %d", resp.StatusCode)
	}
	c.token = strings.TrimSpace(string(data))
	if c.token == "" {
		return "", fmt.Errorf("获取必应翻译令牌失败: 空令牌")
	}
	c.tokenAt = time.Now()
	return c.token, nil
}

// translateDeepLX 逐句调用用户配置的 DeepLX 服务。
func (c *Client) translateDeepLX(ctx context.Context, texts []string, targetLang string) ([]string, error) {
	if c.deepLXURL == "" {
		return nil, fmt.Errorf("DeepLX 接口地址未配置")
	}
	result := make([]string, len(texts))
	for i, text := range texts {
		payload := map[string]string{"text": text, "source_lang": "AUTO", "target_lang": deepLLanguage(targetLang)}
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.deepLXURL, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("DeepLX 第 %d 句请求失败: %w", i+1, err)
		}
		var output struct {
			Data         string   `json:"data"`
			Alternatives []string `json:"alternatives"`
		}
		decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&output)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || decodeErr != nil {
			return nil, fmt.Errorf("DeepLX 第 %d 句返回无效", i+1)
		}
		result[i] = output.Data
		if result[i] == "" && len(output.Alternatives) > 0 {
			result[i] = output.Alternatives[0]
		}
		if result[i] == "" {
			return nil, fmt.Errorf("DeepLX 第 %d 句译文为空", i+1)
		}
	}
	return result, nil
}

// bingLanguage 转换微软翻译使用的语言代码。
func bingLanguage(lang string) string {
	switch strings.ToLower(lang) {
	case "zh", "zh-cn", "zh-hans":
		return "zh-Hans"
	case "zh-tw", "zh-hant":
		return "zh-Hant"
	default:
		return strings.Split(lang, "-")[0]
	}
}

// deepLLanguage 转换 DeepLX 使用的大写语言代码。
func deepLLanguage(lang string) string {
	if strings.EqualFold(lang, "zh-Hant") {
		return "ZH-HANT"
	}
	return strings.ToUpper(strings.Split(lang, "-")[0])
}
