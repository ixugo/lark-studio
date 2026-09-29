package wails

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ixugo/vdub/internal/conf"
)

func newRemoteModelTestServer(t *testing.T, models ...string) *httptest.Server {
	t.Helper()
	ids := make([]map[string]string, 0, len(models))
	for _, id := range models {
		ids = append(ids, map[string]string{"id": id})
	}
	body, err := json.Marshal(map[string]any{"data": ids})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("模型路径 = %s", r.URL.Path)
		}
		if _, err := w.Write(body); err != nil {
			t.Errorf("写模型列表: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestValidateRemoteConfigOnlyChecksChangedOpenAISections(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, `{"data":[{"id":"allowed"}]}`)
	}))
	defer server.Close()
	next := conf.Bootstrap{Pipeline: conf.Pipeline{WhisperMode: "openai", ASRBaseURL: server.URL + "/v1", ASRModel: "allowed"}, LLM: conf.LLM{Provider: "openai", BaseURL: server.URL + "/v1", Model: "allowed"}}
	for _, tc := range []struct {
		name         string
		updates      map[string]any
		wantRequests int32
	}{
		{"无关语音合成", map[string]any{"tts": map[string]any{"voice": "new"}}, 0},
		{"无关流水线字段", map[string]any{"pipeline": map[string]any{"tts_workers": 3}}, 0},
		{"识别模型", map[string]any{"pipeline": map[string]any{"asr_model": "allowed"}}, 1},
		{"识别引擎", map[string]any{"pipeline": map[string]any{"whisper_mode": "openai"}}, 1},
		{"翻译密钥", map[string]any{"llm": map[string]any{"api_key": "new"}}, 1},
		{"两种引擎", map[string]any{"pipeline": map[string]any{"asr_model": "allowed"}, "llm": map[string]any{"model": "allowed"}}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests.Store(0)
			if err := (&AppService{}).validateRemoteConfig(tc.updates, &next); err != nil {
				t.Fatal(err)
			}
			if got := requests.Load(); got != tc.wantRequests {
				t.Fatalf("查询次数 = %d, 期望 %d", got, tc.wantRequests)
			}
		})
	}
	next.Pipeline.WhisperMode = "whisper-cpp"
	next.LLM.Provider = "bing"
	requests.Store(0)
	if err := (&AppService{}).validateRemoteConfig(map[string]any{"pipeline": map[string]any{"whisper_mode": "whisper-cpp"}, "llm": map[string]any{"provider": "bing"}}, &next); err != nil || requests.Load() != 0 {
		t.Fatalf("本地或必应不应联网: %v", err)
	}
}

func TestUpdateRemoteConfigFailureKeepsMemoryAndDisk(t *testing.T) {
	for _, kind := range []string{"missing-model", "unauthorized", "offline"} {
		t.Run(kind, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if kind == "unauthorized" {
					w.WriteHeader(http.StatusUnauthorized)
					fmt.Fprint(w, `{"error":"bad secret-key permission"}`)
					return
				}
				fmt.Fprint(w, `{"data":[{"id":"allowed"}]}`)
			}))
			defer server.Close()
			if kind == "offline" {
				server.Close()
			}
			path := filepath.Join(t.TempDir(), "config.toml")
			bc := conf.Bootstrap{Runtime: conf.Runtime{ConfigPath: path}, Pipeline: conf.Pipeline{WhisperMode: "whisper-cpp"}, TTS: conf.TTS{Type: "edge", Voice: "before"}, LLM: conf.LLM{Provider: "bing"}}
			if err := conf.WriteConfig(&bc, path); err != nil {
				t.Fatal(err)
			}
			before := bc
			rawBefore, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			updates := map[string]any{"tts": map[string]any{"voice": "after"}, "llm": map[string]any{"provider": "openai", "base_url": server.URL + "/v1", "api_key": "secret-key", "model": "invented"}}
			err = (&AppService{bc: &bc}).UpdateConfig(updates)
			if err == nil {
				t.Fatal("应拒绝远程列表之外模型或不可查询的服务")
			}
			if strings.Contains(err.Error(), "secret-key") {
				t.Fatalf("错误泄露密钥: %v", err)
			}
			rawAfter, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !reflect.DeepEqual(before, bc) || string(rawBefore) != string(rawAfter) {
				t.Fatal("校验失败改变了内存或持久化配置")
			}
		})
	}
}

func TestSavingTTSDoesNotQueryExistingRemoteConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	bc := conf.Bootstrap{Runtime: conf.Runtime{ConfigPath: path}, Pipeline: conf.Pipeline{WhisperMode: "openai", ASRBaseURL: "http://127.0.0.1:1/v1", ASRModel: "old-asr"}, LLM: conf.LLM{Provider: "openai", BaseURL: "http://127.0.0.1:1/v1", Model: "old-translation"}, TTS: conf.TTS{Type: "edge", Voice: "before"}}
	if err := (&AppService{bc: &bc}).UpdateConfig(map[string]any{"tts": map[string]any{"voice": "after"}}); err != nil {
		t.Fatal(err)
	}
	if bc.TTS.Voice != "after" {
		t.Fatal("无关配置未保存")
	}
}

func TestUpdateASRRemoteModelRejectsUnknownBeforePersisting(t *testing.T) {
	server := newRemoteModelTestServer(t, "allowed-asr")
	path := filepath.Join(t.TempDir(), "config.toml")
	bc := conf.Bootstrap{Runtime: conf.Runtime{ConfigPath: path}, Pipeline: conf.Pipeline{WhisperMode: "whisper-cpp"}, TTS: conf.TTS{Type: "edge", Voice: "before"}}
	before := bc
	updates := map[string]any{"pipeline": map[string]any{"whisper_mode": "openai", "asr_base_url": server.URL + "/v1", "asr_model": "invented-asr"}, "tts": map[string]any{"voice": "after"}}
	err := (&AppService{bc: &bc}).UpdateConfig(updates)
	if err == nil || !strings.Contains(err.Error(), "语音识别") {
		t.Fatalf("应拒绝列表外识别模型: %v", err)
	}
	if !reflect.DeepEqual(before, bc) {
		t.Fatal("拒绝识别模型后配置已改变")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("校验失败却已写配置: %v", err)
	}
}
