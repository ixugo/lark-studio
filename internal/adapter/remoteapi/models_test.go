package remoteapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestListModelsAuthenticatesAndNormalizesEndpoint(t *testing.T) {
	for _, path := range []string{"/v1", "/v1/", "/v1/models", "/v1/chat/completions", "/v1/audio/transcriptions", "/v1/audio/speech"} {
		t.Run(path, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("请求不符: %s %s %q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
				}
				fmt.Fprint(w, `{"data":[{"id":"z"},{"id":"a"},{"id":"z"}]}`)
			}))
			defer srv.Close()
			got, err := ListModels(t.Context(), srv.URL+path, "test-key")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, []Model{{ID: "a"}, {ID: "z"}}) {
				t.Fatalf("模型列表 = %#v", got)
			}
		})
	}
}

func TestListModelsInvalidResponses(t *testing.T) {
	for _, body := range []string{`{}`, `{"data":[]}`, `{"data":[{"id":""}]}`, `{"data":[{"id":12}]}`, `{"data":[{"id":"bad\nname"}]}`, `{"data":[{"id":"a"}]} {}`, `not json`} {
		t.Run(body, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
			defer srv.Close()
			if _, err := ListModels(t.Context(), srv.URL, ""); err == nil {
				t.Fatalf("应拒绝 %q", body)
			}
		})
	}
}

func TestListModelsRejectsUnsafeInput(t *testing.T) {
	for _, address := range []string{"", "file:///tmp/models", "https://", "https://user:secret@localhost/v1", "https://localhost/v1?api_key=secret", "https://localhost/v1#fragment"} {
		if _, err := ListModels(t.Context(), address, ""); err == nil {
			t.Fatalf("应拒绝地址 %q", address)
		}
	}
	for _, key := range []string{"bad\r\nkey", strings.Repeat("a", 8193)} {
		if _, err := ListModels(t.Context(), "https://localhost/v1", key); err == nil {
			t.Fatal("应拒绝非法密钥")
		}
	}
}

func TestListModelsAuthenticationErrorPreservesDetailWithoutKey(t *testing.T) {
	const key = "secret-key-123"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error":{"message":"invalid API key %s, permission denied"}}`, key)
	}))
	defer srv.Close()
	_, err := ListModels(t.Context(), srv.URL, key)
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), key) {
		t.Fatalf("错误详情或脱敏不符: %v", err)
	}
}

func TestListModelsCancellationAndTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := ListModels(ctx, "http://localhost:1/v1", "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("取消错误 = %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer srv.Close()
	_, err = listModels(t.Context(), &http.Client{Timeout: 30 * time.Millisecond}, srv.URL, "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("超时错误 = %v", err)
	}
}

func TestListModelsResponseLimitAndRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.Repeat("a", maxResponseBytes+1))
	}))
	defer srv.Close()
	if _, err := ListModels(t.Context(), srv.URL, ""); err == nil || !strings.Contains(err.Error(), "过大") {
		t.Fatalf("超限结果 = %v", err)
	}
	called := false
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer destination.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, http.StatusFound) }))
	defer redirect.Close()
	if _, err := ListModels(t.Context(), redirect.URL, "secret"); err == nil || called {
		t.Fatalf("重定向应失败且不转发密钥: %v, called %v", err, called)
	}
}

func TestValidateSelectionUsesRemoteDirectory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"data":[{"id":"allowed"}]}`) }))
	defer srv.Close()
	if err := ValidateSelection(t.Context(), srv.URL, "", "allowed"); err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{"", "invented", " allowed "} {
		if err := ValidateSelection(t.Context(), srv.URL, "", model); err == nil {
			t.Fatalf("应拒绝选择 %q", model)
		}
	}
}

func TestGetJSONSupportsQueriedEndpointAndPreservesHTTPStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/voices" || r.URL.Query().Get("model") != "qwen" {
			t.Errorf("查询地址不符: %s", r.URL)
		}
		fmt.Fprint(w, `{"voices":["Vivian"]}`)
	}))
	defer srv.Close()
	var result struct {
		Voices []string `json:"voices"`
	}
	if err := GetJSON(t.Context(), srv.URL+"/v1/audio/voices?model=qwen", "", &result); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Voices, []string{"Vivian"}) {
		t.Fatalf("音色 = %v", result.Voices)
	}
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "unsupported", http.StatusNotFound) }))
	defer failed.Close()
	err := GetJSON(t.Context(), failed.URL, "", &result)
	httpErr, ok := errors.AsType[*HTTPError](err)
	if !ok || httpErr.StatusCode != http.StatusNotFound {
		t.Fatalf("HTTP 错误类型丢失: %v", err)
	}
}
