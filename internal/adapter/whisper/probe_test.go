package whisper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestProbeFastestURL 验证并发测速探针能够优先择取响应最迅速的镜像源
func TestProbeFastestURL(t *testing.T) {
	// 慢速模拟服务
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	// 快速模拟服务
	fastServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fastServer.Close()

	// 错误模拟服务
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errorServer.Close()

	candidates := []string{
		slowServer.URL,
		errorServer.URL,
		fastServer.URL,
	}

	selected := ProbeFastestURL(context.Background(), candidates)
	if selected != fastServer.URL {
		t.Fatalf("探针未选中最快节点，期望: %s, 实际: %s", fastServer.URL, selected)
	}

	// 验证回退首个选项
	fallback := ProbeFastestURL(context.Background(), []string{"http://127.0.0.1:59999/not-exist"})
	if fallback != "http://127.0.0.1:59999/not-exist" {
		t.Fatalf("单节点或失败未正确回退，实际: %s", fallback)
	}
}
