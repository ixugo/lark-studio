// Package llm 实现 LLM 翻译/分句适配器
// 支持 OpenAI 兼容协议（含 Ollama）
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// Translate 翻译句子列表
func (c *Client) Translate(ctx context.Context, sentences []string, targetLang string) ([]string, error) {
	numbered := make([]string, len(sentences))
	for i, s := range sentences {
		numbered[i] = fmt.Sprintf("%d. %s", i+1, s)
	}

	system := fmt.Sprintf(`You are a professional translator. Translate the following numbered sentences to %s.
Rules:
- Keep the same number of lines as input
- Each translated line should start with its number (e.g., "1. 翻译内容")
- Maintain the meaning and tone
- Use natural, fluent expressions
- For technical terms, keep the English original in parentheses when first mentioned`, targetLang)

	result, err := c.chat(ctx, system, strings.Join(numbered, "\n"))
	if err != nil {
		return nil, err
	}

	// 解析编号输出
	lines := strings.Split(result, "\n")
	var translated []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 去除编号前缀 "1. "
		if idx := strings.Index(line, ". "); idx > 0 && idx < 5 {
			line = line[idx+2:]
		}
		translated = append(translated, line)
	}

	// 数量对齐
	for len(translated) < len(sentences) {
		translated = append(translated, sentences[len(translated)])
	}
	if len(translated) > len(sentences) {
		translated = translated[:len(sentences)]
	}

	return translated, nil
}
