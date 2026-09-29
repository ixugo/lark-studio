package llm

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip 让测试直接返回内存响应，避免依赖本地监听端口。
func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// TestTranslateBing 验证匿名令牌与批量译文能按输入顺序返回。
func TestTranslateBing(t *testing.T) {
	client := NewRoutingClient("", "", "", "bing", "")
	client.bingAuth = "https://test.local/auth"
	client.bingAPI = "https://test.local/translate"
	client.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := "test-token"
		if r.URL.Path == "/auth" {
			return testResponse(body), nil
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.URL.Query().Get("to"); got != "zh-Hans" {
			t.Fatalf("to = %q", got)
		}
		body = `[{"translations":[{"text":"你好"}]},{"translations":[{"text":"世界"}]}]`
		return testResponse(body), nil
	})}

	got, err := client.Translate(context.Background(), []string{"Hello", "world"}, "zh-CN", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "你好" || got[1] != "世界" {
		t.Fatalf("Translate() = %#v", got)
	}
}

// TestTranslateBingDoesNotRetryRateLimit 验证限流不会额外请求令牌或重复提交同一批字幕。
func TestTranslateBingDoesNotRetryRateLimit(t *testing.T) {
	authCalls := 0
	translateCalls := 0
	client := NewRoutingClient("", "", "", "bing", "")
	client.bingAuth = "https://test.local/auth"
	client.bingAPI = "https://test.local/translate"
	client.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/auth" {
			authCalls++
			return testResponse("test-token"), nil
		}
		translateCalls++
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":429001}}`)),
			Header:     make(http.Header),
		}, nil
	})}

	_, err := client.Translate(context.Background(), []string{"Hello"}, "zh-CN", "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("限流错误 = %v", err)
	}
	if authCalls != 1 || translateCalls != 1 {
		t.Fatalf("限流请求次数 auth=%d translate=%d", authCalls, translateCalls)
	}
}

// TestTranslateBingRefreshesExpiredToken 验证仅令牌失效时才重新授权并重试。
func TestTranslateBingRefreshesExpiredToken(t *testing.T) {
	authCalls := 0
	translateCalls := 0
	client := NewRoutingClient("", "", "", "bing", "")
	client.bingAuth = "https://test.local/auth"
	client.bingAPI = "https://test.local/translate"
	client.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/auth" {
			authCalls++
			return testResponse("token-" + string(rune('0'+authCalls))), nil
		}
		translateCalls++
		if translateCalls == 1 {
			return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader("expired")), Header: make(http.Header)}, nil
		}
		return testResponse(`[{"translations":[{"text":"你好"}]}]`), nil
	})}

	got, err := client.Translate(context.Background(), []string{"Hello"}, "zh-CN", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "你好" || authCalls != 2 || translateCalls != 2 {
		t.Fatalf("重授权结果=%v auth=%d translate=%d", got, authCalls, translateCalls)
	}
}

// TestTranslateDeepLX 验证 DeepLX 的主译文与候选译文均可读取。
func TestTranslateDeepLX(t *testing.T) {
	requests := 0
	client := NewRoutingClient("", "", "", "deeplx", "https://test.local/deeplx")
	client.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		data, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body := string(data)
		if !strings.Contains(body, `"source_lang":"AUTO"`) || !strings.Contains(body, `"target_lang":"ZH"`) {
			t.Fatalf("语言参数 = %s", body)
		}
		response := `{"alternatives":["第二句"]}`
		if requests == 1 {
			response = `{"data":"第一句"}`
		}
		return testResponse(response), nil
	})}

	got, err := client.Translate(context.Background(), []string{"one", "two"}, "zh-CN", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "第一句" || got[1] != "第二句" {
		t.Fatalf("Translate() = %#v", got)
	}
}

// testResponse 构造成功的内存 HTTP 响应。
func testResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestGoogleEncodesSubtitleText(t *testing.T) {
	texts := []string{"Hello world", "A & B + C? #中文\nNext"}
	for _, text := range texts {
		t.Run(text, func(t *testing.T) {
			client := NewRoutingClient("", "", "", "google", "")
			client.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if got := r.URL.Query().Get("q"); got != text {
					t.Errorf("q = %q, want %q", got, text)
				}
				if strings.ContainsAny(r.URL.RawQuery, " \n") || r.URL.Fragment != "" {
					t.Errorf("请求包含未编码文本: %s", r.URL)
				}
				return testResponse(`[[["你好"]]]`), nil
			})}
			got, err := client.Translate(t.Context(), []string{text}, "zh-CN", "", nil, nil)
			if err != nil || len(got) != 1 || got[0] != "你好" {
				t.Fatalf("结果=%v 错误=%v", got, err)
			}
		})
	}
}

func TestBingFallbackPreservesBothHTTPErrors(t *testing.T) {
	client := NewRoutingClient("", "", "", "bing", "")
	client.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		response := testResponse(`{"error":{"message":"invalid target language"}}`)
		response.StatusCode = http.StatusBadRequest
		if r.URL.Host == "edge.microsoft.com" {
			response.StatusCode = http.StatusNotFound
			response.Body = io.NopCloser(strings.NewReader("auth endpoint unavailable"))
		}
		return response, nil
	})}
	_, err := client.Translate(t.Context(), []string{"Hello world"}, "zh-CN", "", nil, nil)
	if err == nil {
		t.Fatal("应返回翻译错误")
	}
	for _, detail := range []string{"404", "auth endpoint unavailable", "400", "invalid target language"} {
		if !strings.Contains(err.Error(), detail) {
			t.Errorf("错误缺少 %q: %v", detail, err)
		}
	}
}
