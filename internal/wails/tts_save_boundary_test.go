package wails

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ixugo/vdub/internal/conf"
)

func TestIndependentOpenAISettingsValidateWithDefaultEdge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer denied" {
			http.Error(w, "permission denied", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/models":
			fmt.Fprint(w, `{"data":[{"id":"tts-known"}]}`)
		case "/v1/audio/voices":
			fmt.Fprint(w, `{"data":[{"id":"Vivian","name":"Vivian"},{"id":"Serena","name":"Serena"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	for _, tc := range []struct {
		name      string
		update    map[string]any
		wantError bool
	}{
		{"默认 Edge 也拒绝未知 OpenAI 音色", map[string]any{"openai_voice": "unknown"}, true},
		{"默认 Edge 也拒绝未知 OpenAI 模型", map[string]any{"model": "unknown"}, true},
		{"默认 Edge 也验证 OpenAI 地址", map[string]any{"base_url": "http://127.0.0.1:1/v1"}, true},
		{"默认 Edge 也验证 OpenAI 密钥", map[string]any{"api_key": "denied"}, true},
		{"默认 Edge 也验证 OpenAI 协议", map[string]any{"protocol": "invalid"}, true},
		{"默认 Edge 也验证 OpenAI 语言", map[string]any{"language": strings.Repeat("a", 65)}, true},
		{"默认 Edge 也验证 OpenAI 情绪", map[string]any{"instructions": "unsupported"}, true},
		{"默认 Edge 可保存独立 OpenAI 音色", map[string]any{"openai_voice": "Serena"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			bc := conf.Bootstrap{Runtime: conf.Runtime{ConfigPath: path}, Pipeline: conf.Pipeline{WhisperMode: "whisper-cpp"}, TTS: conf.TTS{Type: "edge", Voice: "zh-CN-XiaoxiaoNeural", EdgeVoice: "zh-CN-XiaoxiaoNeural", OpenAIVoice: "Vivian", BaseURL: server.URL + "/v1", Model: "tts-known"}}
			before := bc
			err := (&AppService{bc: &bc}).UpdateConfig(map[string]any{"tts": tc.update})
			if (err != nil) != tc.wantError {
				t.Fatalf("错误=%v,应失败=%v", err, tc.wantError)
			}
			if tc.wantError {
				if !reflect.DeepEqual(before, bc) {
					t.Fatal("失败改变了配置")
				}
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("无效配置写入了文件")
				}
			} else if bc.TTS.Type != "edge" || bc.TTS.Voice != before.TTS.Voice || bc.TTS.OpenAIVoice != "Serena" {
				t.Fatalf("独立音色影响默认引擎: %+v", bc.TTS)
			}
		})
	}
}

func TestEdgeVoiceSaveDoesNotQueryUnchangedOpenAI(t *testing.T) {
	for _, engine := range []string{"edge", "openai"} {
		t.Run(engine, func(t *testing.T) {
			bc := conf.Bootstrap{Runtime: conf.Runtime{ConfigPath: filepath.Join(t.TempDir(), "config.toml")}, Pipeline: conf.Pipeline{WhisperMode: "whisper-cpp"}, TTS: conf.TTS{Type: engine, Voice: "Vivian", OpenAIVoice: "Vivian", BaseURL: "http://127.0.0.1:1/v1", Model: "unreachable-old"}}
			if err := (&AppService{bc: &bc}).UpdateConfig(map[string]any{"tts": map[string]any{"edge_voice": "en-US-JennyNeural", "base_url": bc.TTS.BaseURL, "api_key": bc.TTS.APIKey, "model": bc.TTS.Model, "openai_voice": bc.TTS.OpenAIVoice, "protocol": bc.TTS.Protocol, "language": bc.TTS.Language, "instructions": bc.TTS.Instructions}}); err != nil {
				t.Fatalf("单改 Edge 音色不应联网: %v", err)
			}
			if bc.TTS.EdgeVoice != "en-US-JennyNeural" {
				t.Fatal("Edge 音色未保存")
			}
		})
	}
}
