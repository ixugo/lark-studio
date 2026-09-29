package tts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestMLXSpeechOptionsKeepVoiceAndSendExtensions(t *testing.T) {
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		requests = append(requests, body)
		w.Header().Set("Content-Type", "audio/wav")
		if _, err := w.Write([]byte("audio")); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := NewOpenAITTS(server.URL+"/v1", "", "Qwen3-TTS-1.7B-CustomVoice", "Vivian")
	client.options = SpeechOptions{Protocol: "mlx", Language: "Chinese", Instructions: "热情"}
	for _, text := range []string{"你好", "这是第二句"} {
		if _, _, err := client.SynthesizeBytes(t.Context(), text, "Vivian", 1); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range requests {
		if r["voice"] != "Vivian" || r["lang_code"] != "Chinese" || r["instruct"] != "热情" || r["response_format"] != "wav" {
			t.Fatalf("缺少稳定音色与扩展参数: %v", r)
		}
		if _, ok := r["instructions"]; ok {
			t.Fatal("MLX 不能误用标准指令字段")
		}
	}
}

func TestStandardSpeechNeverSendsMLXFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body["instructions"] != "calm" {
			t.Error("缺少标准 instructions")
		}
		for _, field := range []string{"instruct", "lang_code", "ref_audio", "ref_text"} {
			if _, ok := body[field]; ok {
				t.Errorf("标准服务收到扩展字段 %s", field)
			}
		}
		if _, err := w.Write([]byte("audio")); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := NewOpenAITTS(server.URL, "", "gpt-4o-mini-tts", "alloy")
	client.options = SpeechOptions{Instructions: "calm"}
	if _, _, err := client.SynthesizeBytes(t.Context(), "hi", "", 1); err != nil {
		t.Fatal(err)
	}
}

func TestSpeechErrorNeverLeaksAPIKeyAndRejectsEmptyAudio(t *testing.T) {
	for _, status := range []int{200, 400} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			if status == 400 {
				if _, err := w.Write([]byte("api key fixture-secret was rejected")); err != nil {
					t.Error(err)
				}
			}
		}))
		_, _, err := NewOpenAITTS(server.URL, "fixture-secret", "custom", "Vivian").SynthesizeBytes(t.Context(), "hi", "", 1)
		server.Close()
		if err == nil || strings.Contains(err.Error(), "fixture-secret") {
			t.Fatalf("空音频或泄露的接口错误: %v", err)
		}
	}
}

func TestTaskSnapshotRetainsVoiceAfterConfigurationReload(t *testing.T) {
	var voices []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.Write([]byte(`{"data":[{"id":"old"},{"id":"new"}]}`))
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		voices = append(voices, body["voice"].(string))
		if _, err := w.Write([]byte("audio")); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	router := NewRouter("openai", "Vivian", server.URL+"/v1", "", "old")
	taskCtx, err := router.SnapshotTask(t.Context(), "openai", "zh-CN", "")
	if err != nil {
		t.Fatalf("task snapshot: %v", err)
	}
	before := router.SpeechIdentity(taskCtx, "", 1)
	router.SetTTSConfig("openai", "Serena", server.URL+"/v1", "", "new")
	for _, text := range []string{"一句", "二句"} {
		if err := router.SynthesizeWithOptions(taskCtx, text, filepath.Join(t.TempDir(), "audio.wav"), "openai", "", 1); err != nil {
			t.Fatal(err)
		}
	}
	if len(voices) != 2 || voices[0] != "Vivian" || voices[1] != "Vivian" {
		t.Fatalf("任务中途变声: %v", voices)
	}
	newCtx, err := router.SnapshotTask(t.Context(), "openai", "zh-CN", "")
	if err != nil {
		t.Fatal(err)
	}
	if before == router.SpeechIdentity(newCtx, "", 1) {
		t.Fatal("模型与音色改变后缓存身份未变化")
	}
}
