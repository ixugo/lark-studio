// Package llm 实现 LLM 翻译/分句适配器
// 支持 OpenAI 兼容协议（含 Ollama）
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client OpenAI 兼容 API 客户端
type Client struct {
	baseURL   string
	apiKey    string
	model     string
	provider  string
	deepLXURL string
	client    *http.Client
	token     string
	tokenAt   time.Time
	tokenMu   sync.Mutex
	bingAuth  string
	bingAPI   string
}

// NewRoutingClient 创建支持必应、DeepLX 与 OpenAI 的翻译客户端。
func NewRoutingClient(baseURL, apiKey, model, provider, deepLXURL string) *Client {
	client := NewClient(baseURL, apiKey, model)
	if strings.TrimSpace(provider) == "" {
		provider = "bing"
	}
	client.provider = provider
	client.deepLXURL = strings.TrimRight(deepLXURL, "/")
	return client
}

// NewClient 创建 LLM 客户端
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		apiKey:   apiKey,
		model:    model,
		client:   &http.Client{Timeout: 120 * time.Second},
		bingAuth: bingAuthURL,
		bingAPI:  bingTranslateURL,
	}
}

// chatRequest OpenAI Chat Completion 请求体
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// chat 发送聊天请求
func (c *Client) chat(ctx context.Context, system, user string) (string, error) {
	req := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.3,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API 错误: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API 返回空响应")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

// SplitSentences 用 LLM 做语义分句
func (c *Client) SplitSentences(ctx context.Context, text string, lang string) ([]string, error) {
	system := `You are a sentence segmentation expert. Split the following text into complete, natural sentences.
Rules:
- Each sentence should be a complete thought
- Don't split in the middle of a clause
- Output one sentence per line
- Don't add numbers or bullet points
- Keep the original text unchanged, just add line breaks at sentence boundaries`

	result, err := c.chat(ctx, system, text)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(result, "\n")
	var sentences []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sentences = append(sentences, line)
		}
	}
	return sentences, nil
}

const (
	translateMaxRetries = 3
	translateBaseDelay  = 2 * time.Second
)

// Translate 翻译句子列表，数量不符时自动重试
// systemPrompt 为空时使用内置默认提示词
// contextBefore/contextAfter 作为上下文帮助 LLM 理解语境，不计入翻译输出
func (c *Client) Translate(ctx context.Context, sentences []string, targetLang, systemPrompt string, contextBefore, contextAfter []string) ([]string, error) {
	return c.TranslateWithProvider(
		ctx, sentences, targetLang, systemPrompt, contextBefore, contextAfter, c.provider,
	)
}

// TranslateWithProvider 使用任务快照指定的翻译引擎。
func (c *Client) TranslateWithProvider(
	ctx context.Context,
	sentences []string,
	targetLang string,
	systemPrompt string,
	contextBefore []string,
	contextAfter []string,
	provider string,
) ([]string, error) {
	switch strings.ToLower(provider) {
	case "bing":
		return c.translateBing(ctx, sentences, targetLang)
	case "deeplx":
		return c.translateDeepLX(ctx, sentences, targetLang)
	}
	numbered := make([]string, len(sentences))
	for i, s := range sentences {
		numbered[i] = fmt.Sprintf("%d. %s", i+1, s)
	}

	hasContext := len(contextBefore) > 0 || len(contextAfter) > 0
	system := systemPrompt
	if system == "" {
		contextRule := ""
		if hasContext {
			contextRule = "\n- Lines marked [CTX] are context for reference only — do NOT translate them"
		}
		system = fmt.Sprintf(`You are a professional subtitle translator. Translate the following numbered sentences to %s.
Rules:
- You MUST output exactly %d lines, one translation per input line
- Each translated line should start with its number (e.g., "1. 翻译内容")%s
- Keep translations concise: each translated line should be short enough to read naturally at normal speaking speed within the original subtitle's display duration
- Maintain the meaning and tone of the original
- Use natural, fluent, colloquial expressions suitable for subtitles
- For technical terms, keep the English original in parentheses when first mentioned
- Avoid overly literal or stiff translations`, targetLang, len(sentences), contextRule)
	} else {
		system = strings.NewReplacer(
			"{{target_lang}}", targetLang,
			"{{count}}", fmt.Sprintf("%d", len(sentences)),
		).Replace(system)
	}

	var inputParts []string
	for _, s := range contextBefore {
		inputParts = append(inputParts, "[CTX] "+s)
	}
	inputParts = append(inputParts, numbered...)
	for _, s := range contextAfter {
		inputParts = append(inputParts, "[CTX] "+s)
	}
	input := strings.Join(inputParts, "\n")
	expected := len(sentences)

	var lastTranslated []string
	for attempt := range translateMaxRetries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		result, err := c.chat(ctx, system, input)
		if err != nil {
			if attempt < translateMaxRetries-1 {
				delay := translateBaseDelay * time.Duration(1<<uint(attempt))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
				}
				continue
			}
			return nil, err
		}

		translated := parseNumberedLines(result)
		lastTranslated = translated

		if len(translated) == expected {
			return translated, nil
		}

		slog.Debug("LLM 翻译数量不符，重试",
			"expected", expected, "got", len(translated), "attempt", attempt+1)
	}

	return requireTranslationCount(lastTranslated, expected)
}

// requireTranslationCount 拒绝缺行译文，避免用原文补位后送入配音。
func requireTranslationCount(translated []string, expected int) ([]string, error) {
	if len(translated) != expected {
		return nil, fmt.Errorf(
			"翻译数量不符: 期望 %d 行，实际 %d 行",
			expected,
			len(translated),
		)
	}
	return translated, nil
}

// parseNumberedLines 解析 LLM 返回的编号行
func parseNumberedLines(text string) []string {
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if idx := strings.Index(line, ". "); idx > 0 && idx < 5 {
			line = line[idx+2:]
		}
		result = append(result, line)
	}
	return result
}
