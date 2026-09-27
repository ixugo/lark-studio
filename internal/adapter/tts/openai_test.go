package tts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSynthesizeBytesUsesConfiguredModelAndVoice 保证试听按用户配置请求兼容端点并保留音频格式。
func TestSynthesizeBytesUsesConfiguredModelAndVoice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("请求路由或认证头错误: path=%s auth=%s", r.URL.Path, r.Header.Get("Authorization"))
			return
		}
		var request struct {
			Model string `json:"model"`
			Voice string `json:"voice"`
			Input string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("读取试听请求失败: %v", err)
			return
		}
		if request.Model != "custom-model" || request.Voice != "custom-voice" || request.Input != "hello" {
			t.Fatalf("试听参数未按配置发送: %+v", request)
		}
		w.Header().Set("Content-Type", "audio/wav")
		if _, err := w.Write([]byte("audio-data")); err != nil {
			t.Errorf("写入测试音频失败: %v", err)
		}
	}))
	defer server.Close()

	audio, mediaType, err := NewOpenAITTS(server.URL+"/v1", "secret", "custom-model", "default").SynthesizeBytes(context.Background(), "hello", "custom-voice", 1)
	if err != nil {
		t.Fatal(err)
	}
	if string(audio) != "audio-data" || mediaType != "audio/wav" {
		t.Fatalf("试听响应错误: audio=%q mediaType=%q", audio, mediaType)
	}
}

// TestSynthesizeBytesReturnsEndpointErrors 保证接口错误能反馈到试听提示中。
func TestSynthesizeBytesReturnsEndpointErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("unsupported model")); err != nil {
			t.Errorf("写入测试错误响应失败: %v", err)
		}
	}))
	defer server.Close()

	_, _, err := NewOpenAITTS(server.URL, "", "custom-model", "custom-voice").SynthesizeBytes(context.Background(), "hello", "", 1)
	if err == nil || err.Error() != "TTS API 返回 400: unsupported model" {
		t.Fatalf("应返回兼容端点错误，实际为 %v", err)
	}
}
