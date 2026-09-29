package tts

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscoverTTSCapabilities(t *testing.T) {
	tests := []struct {
		name, model, schema, voices string
		status                      int
		source                      string
		instruct                    bool
		wantErr                     string
	}{
		{"local_custom_06", "mlx-community/Qwen3-TTS-12Hz-0.6B-CustomVoice-8bit", `{"instruct":{},"lang_code":{},"ref_audio":{},"ref_text":{}}`, `{"data":[]}`, 200, "qwen_builtin", false, ""},
		{"local_custom_17", "Qwen3-TTS-12Hz-1.7B-CustomVoice", `{"instruct":{},"lang_code":{},"ref_audio":{},"ref_text":{}}`, `{"data":[]}`, 200, "qwen_builtin", true, ""},
		{"remote_voice", "Qwen3-TTS-12Hz-1.7B-CustomVoice", `{"instruct":{},"lang_code":{}}`, `{"data":[{"id":"my-voice","name":"自定义音色"}]}`, 200, "remote", true, ""},
		{"qwen_without_mlx_schema", "Qwen3-TTS-12Hz-1.7B-CustomVoice", `{}`, `{"data":[]}`, 200, "manual", false, ""},
		{"generic_unknown", "unknown", `{}`, ``, 404, "manual", false, ""},
		{"standard_openai", "gpt-4o-mini-tts", `{}`, ``, 404, "manual", true, ""},
		{"auth_fail", "unknown", `{}`, `secret-key`, 401, "", false, "HTTP 401"},
		{"server_fail", "unknown", `{}`, `busy`, 503, "", false, "HTTP 503"},
		{"bad_json", "unknown", `{}`, `oops`, 200, "", false, "格式"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer secret-key" {
					t.Error("missing authentication")
				}
				switch r.URL.Path {
				case "/openapi.json":
					if tc.schema == `{}` {
						w.WriteHeader(404)
						return
					}
					_, err := w.Write([]byte(`{"info":{"title":"MLX Audio API"},"components":{"schemas":{"SpeechRequest":{"properties":` + tc.schema + `}}}}`))
					if err != nil {
						t.Error(err)
					}
				case "/v1/audio/voices":
					if r.URL.Query().Get("model") != tc.model {
						t.Error("missing model filter")
					}
					w.WriteHeader(tc.status)
					if _, err := w.Write([]byte(tc.voices)); err != nil {
						t.Error(err)
					}
				default:
					w.WriteHeader(404)
				}
			}))
			defer srv.Close()
			got, err := DiscoverCapabilities(t.Context(), srv.URL+"/v1", "secret-key", tc.model)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error=%v", err)
				}
				if strings.Contains(err.Error(), "secret-key") {
					t.Fatal("API key leaked")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.VoiceSource != tc.source || got.Instructions != tc.instruct {
				t.Fatalf("capabilities=%+v", got)
			}
			if got.VoiceSource == "qwen_builtin" && len(got.Voices) != 9 {
				t.Fatalf("voices=%v", got.Voices)
			}

		})
	}
}

func TestCapabilitiesRejectInvalidInput(t *testing.T) {
	for _, url := range []string{"", "file:///tmp/a", "http://user:password@localhost/v1", "http://localhost/v1?key=x"} {
		if _, err := DiscoverCapabilities(t.Context(), url, "", "model"); err == nil {
			t.Fatalf("accepted %s", url)
		}
	}
}

func TestCapabilitiesOpenAPIAuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte("invalid secret-key")); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()
	_, err := DiscoverCapabilities(t.Context(), srv.URL+"/v1", "secret-key", "tts-1")
	if err == nil || !strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("err=%v", err)
	}
}

func TestCapabilitiesPreservePathPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/service/openapi.json":
			w.WriteHeader(404)
		case "/service/v1/audio/voices":
			if _, err := w.Write([]byte(`{"data":[{"id":"fixed"}]}`)); err != nil {
				t.Error(err)
			}
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	got, err := DiscoverCapabilities(t.Context(), srv.URL+"/service/v1/audio/speech", "", "tts-1")
	if err != nil || got.VoiceSource != "remote" || len(got.Voices) != 1 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestCapabilitiesRejectUnboundedVoiceAndModel(t *testing.T) {
	for _, model := range []string{"", " bad ", "bad\nmodel", strings.Repeat("a", capabilitiesMaxModelBytes+1)} {
		if _, err := capabilitiesBaseURL("http://localhost/v1", model); err == nil {
			t.Fatalf("accepted invalid model %q", model)
		}
	}
	for _, voices := range [][]Voice{{{ID: ""}}, {{ID: "injected\nvoice"}}, {{ID: "v", Name: strings.Repeat("a", capabilitiesMaxModelBytes+1)}}, make([]Voice, capabilitiesMaxVoices+1)} {
		if err := validateVoices(voices); err == nil {
			t.Fatal("accepted invalid voice response")
		}
	}
}
