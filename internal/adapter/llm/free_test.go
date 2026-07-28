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
