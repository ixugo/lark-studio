package api

import (
	"bytes"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/ixugo/vdub/pkg/ws"
)

type modelDownloadTransport func(*http.Request) (*http.Response, error)

func (fn modelDownloadTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type modelDownloadHub struct {
	ws.Huber
	messages []ws.Message
}

func (h *modelDownloadHub) Broadcast(message ws.Message) {
	h.messages = append(h.messages, message)
}

// TestModelDownloadImmediatelyTransfersBody 下载应直接请求文件，即使很快结束也要广播进度。
func TestModelDownloadImmediatelyTransfersBody(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "直接下载", true: "首选源失败后换源"}[fallback], func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			content := bytes.Repeat([]byte("model data"), 1024)
			var mu sync.Mutex
			var methods []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				methods = append(methods, r.Method+" "+r.URL.Path)
				mu.Unlock()
				if fallback && r.URL.Path == "/mirror" {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				_, _ = w.Write(content)
			}))
			defer server.Close()
			transport := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = transport })
			http.DefaultTransport = modelDownloadTransport(func(req *http.Request) (*http.Response, error) {
				copy := req.Clone(req.Context())
				url := server.URL + "/origin"
				if strings.Contains(req.URL.Host, "hf-mirror") {
					url = server.URL + "/mirror"
				}
				local, err := http.NewRequestWithContext(copy.Context(), copy.Method, url, nil)
				if err != nil {
					return nil, err
				}
				local.Header = copy.Header.Clone()
				return transport.RoundTrip(local)
			})
			hub := &modelDownloadHub{}
			err := downloadModel("tiny", hub)
			if err != nil {
				t.Fatalf("下载失败：%v", err)
			}
			data, err := os.ReadFile(modelPath("tiny"))
			if err != nil || !bytes.Equal(data, content) {
				t.Fatalf("模型内容不完整：%d %v", len(data), err)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(methods) == 0 || methods[0] != "GET /mirror" {
				t.Errorf("第一条请求必须直接下载而非测速：%v", methods)
			}
			wantRequests := 1
			if fallback {
				wantRequests = 2
			}
			if len(methods) != wantRequests {
				t.Errorf("不应额外测速或重复下载：%v", methods)
			}
			var finalProgress bool
			for _, message := range hub.messages {
				if message.Type() != "model_download_progress" {
					continue
				}
				var payload struct {
					Downloaded int64 `json:"downloaded"`
				}
				if err := json.Unmarshal(message.Data(), &payload); err != nil {
					t.Fatal(err)
				}
				finalProgress = finalProgress || payload.Downloaded == int64(len(content))
			}
			if !finalProgress {
				t.Fatal("下载完整文件后未广播实际下载量，快速下载也不能遗漏进度")
			}
			if len(hub.messages) == 0 || hub.messages[len(hub.messages)-1].Type() != "model_download_done" {
				t.Fatal("完成状态必须在进度之后广播")
			}
		})
	}
}

// TestAlreadyDownloadedModelNotifiesCompletion 防止列表未刷新时再次点击下载后界面一直等待。
func TestAlreadyDownloadedModelNotifiesCompletion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path := modelPath("tiny")
	if err := os.WriteFile(path, []byte("already downloaded"), 0o600); err != nil {
		t.Fatal(err)
	}
	hub := &modelDownloadHub{}
	if _, err := startDownload("tiny", hub); err != nil {
		t.Fatal(err)
	}
	if len(hub.messages) != 1 || hub.messages[0].Type() != "model_download_done" {
		t.Fatal("已经下载时也应通知前端完成，不能一直显示下载中")
	}
}
