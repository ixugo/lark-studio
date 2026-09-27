package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTestOpenAIConnection(t *testing.T) {
	// 模拟正常的 OpenAI 兼容端点
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"pong"}}]}`))
	}))
	defer server.Close()

	// 1. 成功场景
	msg, err := TestOpenAIConnection(context.Background(), server.URL, "test-key", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("预期成功，实际报错: %v", err)
	}
	if !strings.Contains(msg, "连通成功") {
		t.Fatalf("预期包含'连通成功'，实际: %s", msg)
	}

	// 2. 鉴权失败场景
	_, err = TestOpenAIConnection(context.Background(), server.URL, "wrong-key", "gpt-4o-mini")
	if err == nil {
		t.Fatal("预期鉴权失败报错，实际未报错")
	}

	// 3. 空 URL 校验
	_, err = TestOpenAIConnection(context.Background(), "", "", "")
	if err == nil {
		t.Fatal("预期空 URL 报错，实际未报错")
	}
}
