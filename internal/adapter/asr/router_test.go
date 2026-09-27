package asr

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRouterUsesSavedOpenAIEngine 验证保存的路由配置会调用兼容接口并生成时间轴字幕。
func TestRouterUsesSavedOpenAIEngine(t *testing.T) {
	wantAudio := []byte("sample audio")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("请求路径或认证头错误: %s %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		assertTranscriptionForm(t, r, wantAudio)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "1\n00:00:01,000 --> 00:00:02,000\nhello\n")
	}))
	defer server.Close()

	router := NewRouter(Config{Engine: "whisper-cpp", WhisperBin: "must-not-run"})
	router.SetConfig(Config{Engine: "openai", BaseURL: server.URL + "/v1", APIKey: "test-key", Model: "whisper-1"})
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	if err := os.WriteFile(audioPath, wantAudio, 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "result.srt")
	if err := router.Transcribe(t.Context(), audioPath, output, "en", nil, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "00:00:01,000 --> 00:00:02,000") {
		t.Fatalf("SRT 时间轴缺失: %q", got)
	}
}

// assertTranscriptionForm 检查接口收到的模型参数与原始音频文件。
func assertTranscriptionForm(t *testing.T, r *http.Request, wantAudio []byte) {
	t.Helper()
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	if r.FormValue("model") != "whisper-1" || r.FormValue("response_format") != "srt" || r.FormValue("language") != "en" {
		t.Errorf("转录参数错误: %#v", r.MultipartForm.Value)
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(wantAudio) {
		t.Errorf("上传音频 = %q，期望 %q", got, wantAudio)
	}
}

// TestValidateConfigRejectsInvalidEndpoint 阻止无效地址或过长外部配置进入请求。
func TestValidateConfigRejectsInvalidEndpoint(t *testing.T) {
	for _, config := range []Config{
		{Engine: "openai", BaseURL: "file:///tmp", Model: "whisper-1"},
		{Engine: "openai", BaseURL: "http://localhost/v1", Model: strings.Repeat("m", 101)},
		{Engine: "openai", BaseURL: "http://localhost/v1", Model: " "},
	} {
		if err := ValidateConfig(config); err == nil {
			t.Fatalf("无效配置应被拒绝: %+v", config)
		}
	}
}
