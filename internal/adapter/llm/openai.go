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
	"time"
)

// Client OpenAI 兼容 API 客户端
type Client struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewClient 创建 LLM 客户端
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
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
func (c *Client) Translate(ctx context.Context, sentences []string, targetLang, systemPrompt string) ([]string, error) {
	numbered := make([]string, len(sentences))
	for i, s := range sentences {
		numbered[i] = fmt.Sprintf("%d. %s", i+1, s)
	}

	system := systemPrompt
	if system == "" {
		system = fmt.Sprintf(`You are a professional subtitle translator. Translate the following numbered sentences to %s.
Rules:
- You MUST output exactly %d lines, one translation per input line
- Each translated line should start with its number (e.g., "1. 翻译内容")
- Keep translations concise: each translated line should be short enough to read naturally at normal speaking speed within the original subtitle's display duration
- Maintain the meaning and tone of the original
- Use natural, fluent, colloquial expressions suitable for subtitles
- For technical terms, keep the English original in parentheses when first mentioned
- Avoid overly literal or stiff translations`, targetLang, len(sentences))
	} else {
		system = strings.NewReplacer(
			"{{target_lang}}", targetLang,
			"{{count}}", fmt.Sprintf("%d", len(sentences)),
		).Replace(system)
	}

	input := strings.Join(numbered, "\n")
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

	// 重试耗尽仍数量不符，做防御性对齐
	for len(lastTranslated) < expected {
		lastTranslated = append(lastTranslated, sentences[len(lastTranslated)])
	}
	if len(lastTranslated) > expected {
		lastTranslated = lastTranslated[:expected]
	}
	return lastTranslated, nil
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
