package wails

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	ttsadapter "github.com/ixugo/vdub/internal/adapter/tts"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
)

func TestSavedTTSConfigChangesActualSynthesisRequest(t *testing.T) {
	var got struct {
		Model string `json:"model"`
		Voice string `json:"voice"`
	}
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" {
			t.Errorf("合成路径=%s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer fixture-new" {
			t.Error("合成请求未使用保存的新密钥")
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("test audio"))
	}))
	defer endpoint.Close()
	router := ttsadapter.NewRouter("openai", "old-voice", "http://127.0.0.1:1/v1", "old-key", "old-model")
	core := pipeline.NewCore(pipeline.Config{}, nil, nil, router)
	bc := conf.Bootstrap{Runtime: conf.Runtime{ConfigPath: filepath.Join(t.TempDir(), "config.toml")}}
	svc := &AppService{bc: &bc, scheduler: pipeline.NewScheduler(core, nil)}
	if err := svc.UpdateConfig(map[string]any{"tts": map[string]any{"type": "openai", "voice": "new-voice", "base_url": endpoint.URL + "/v1", "api_key": "fixture-new", "model": "new-model"}}); err != nil {
		t.Fatal(err)
	}
	if err := router.Synthesize(t.Context(), "hello", filepath.Join(t.TempDir(), "speech.mp3"), ""); err != nil {
		t.Fatal(err)
	}
	if got.Model != "new-model" || got.Voice != "new-voice" {
		t.Fatalf("合成仍使用旧配置: %+v", got)
	}
}
