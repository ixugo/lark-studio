package wails

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	asradapter "github.com/ixugo/vdub/internal/adapter/asr"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// TestOpenAITTSReturnsPlayableAudio 保证试听接口返回浏览器可播放的数据地址。
func TestOpenAITTSReturnsPlayableAudio(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" {
			t.Errorf("试听请求路径错误: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		if _, err := w.Write([]byte("sample-audio")); err != nil {
			t.Errorf("写入测试音频失败: %v", err)
		}
	}))
	defer server.Close()

	got, err := (&AppService{}).TestOpenAITTS(server.URL+"/v1", "", "custom-model", "custom-voice", "hello")
	if err != nil {
		t.Fatal(err)
	}
	want := "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString([]byte("sample-audio"))
	if got != want {
		t.Fatalf("试听数据地址错误: got=%q want=%q", got, want)
	}
}

// TestOpenAITTSRejectsMissingConfiguration 避免空配置发送无效试听请求。
func TestOpenAITTSRejectsMissingConfiguration(t *testing.T) {
	_, err := (&AppService{}).TestOpenAITTS("", "", "", "", "hello")
	if err == nil || !strings.Contains(err.Error(), "均不能为空") {
		t.Fatalf("空配置应返回校验错误，实际为 %v", err)
	}
}

// TestWhisperModelConfigRoundTrip 保证模型路径经前端配置、TOML 持久化与重载后保持一致。
func TestWhisperModelConfigRoundTrip(t *testing.T) {
	server := newRemoteModelTestServer(t, "whisper-1")
	path := filepath.Join(t.TempDir(), "config.toml")
	want := filepath.Join(t.TempDir(), "ggml-large-v3-turbo.bin")
	bc := conf.Bootstrap{Runtime: conf.Runtime{ConfigPath: path}}
	svc := &AppService{bc: &bc}
	updates := map[string]any{"pipeline": map[string]any{
		"whisper_model": want,
		"whisper_mode":  "openai",
		"asr_base_url":  server.URL + "/v1",
		"asr_api_key":   "test-key",
		"asr_model":     "whisper-1",
	}}
	if err := svc.UpdateConfig(updates); err != nil {
		t.Fatalf("保存配置：%v", err)
	}

	var loaded conf.Bootstrap
	if err := conf.SetupConfig(&loaded, path); err != nil {
		t.Fatalf("重读配置：%v", err)
	}
	if loaded.Pipeline.WhisperModel != want {
		t.Fatalf("TOML 模型路径 = %q，期望 %q", loaded.Pipeline.WhisperModel, want)
	}
	if loaded.Pipeline.WhisperMode != "openai" || loaded.Pipeline.ASRBaseURL != server.URL+"/v1" || loaded.Pipeline.ASRAPIKey != "test-key" || loaded.Pipeline.ASRModel != "whisper-1" {
		t.Fatalf("ASR 配置未完整持久化: %+v", loaded.Pipeline)
	}

	payload, err := json.Marshal(svc.GetConfig())
	if err != nil {
		t.Fatalf("序列化配置：%v", err)
	}
	var result ConfigDTO
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("解析配置 JSON：%v", err)
	}
	if result.Pipeline.WhisperModel != want {
		t.Fatalf("前端模型路径 = %q，期望 %q", result.Pipeline.WhisperModel, want)
	}
	if result.Pipeline.WhisperMode != "openai" || result.Pipeline.ASRModel != "whisper-1" {
		t.Fatalf("前端默认 ASR 配置 = %+v", result.Pipeline)
	}
}

// TestUpdateConfigSwitchesSharedASRRouter 确保设置页保存的默认引擎用于后续转录。
func TestUpdateConfigSwitchesSharedASRRouter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			if _, err := io.WriteString(w, `{"data":[{"id":"whisper-1"}]}`); err != nil {
				t.Errorf("写入模型列表失败: %v", err)
			}
			return
		}
		if r.URL.Path != "/v1/audio/transcriptions" {
			t.Errorf("ASR 请求路径 = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, "1\n00:00:00,000 --> 00:00:01,000\nhello\n")
	}))
	defer server.Close()
	router := asradapter.NewRouter(asradapter.Config{Engine: "whisper-cpp", WhisperBin: "must-not-run"})
	path := filepath.Join(t.TempDir(), "config.toml")
	bc := conf.Bootstrap{Runtime: conf.Runtime{ConfigPath: path}, Pipeline: conf.Pipeline{WhisperMode: "whisper-cpp"}}
	svc := &AppService{bc: &bc, asrRouter: router}
	updates := map[string]any{"pipeline": map[string]any{
		"whisper_mode": "openai",
		"asr_base_url": server.URL + "/v1",
		"asr_model":    "whisper-1",
	}}
	if err := svc.UpdateConfig(updates); err != nil {
		t.Fatal(err)
	}
	audio := filepath.Join(t.TempDir(), "audio.wav")
	if err := os.WriteFile(audio, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := router.Transcribe(t.Context(), audio, filepath.Join(t.TempDir(), "output.srt"), "auto", nil, nil); err != nil {
		t.Fatalf("保存默认 OpenAI 引擎后转录失败: %v", err)
	}
}

func TestPrepareTaskInput(t *testing.T) {
	// 创建临时测试文件
	tmpDir := t.TempDir()
	testVideo := filepath.Join(tmpDir, "sample.mp4")
	if err := os.WriteFile(testVideo, []byte("fake video content"), 0o644); err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}

	bc := conf.DefaultConfig()
	bc.Pipeline.WhisperMode = "openai"
	modelServer := newRemoteModelTestServer(t, "test-model", bc.LLM.Model)
	bc.Pipeline.ASRBaseURL = modelServer.URL + "/v1"
	bc.LLM.BaseURL = modelServer.URL + "/v1"
	bc.Pipeline.DefaultOutputDir = filepath.Join(tmpDir, "results")
	bc.Pipeline.ASRModel = "test-model"
	bc.LLM.Provider = "bing"
	svc := &AppService{bc: &bc}

	t.Run("自建 local 引擎自动纠正为 openai", func(t *testing.T) {
		in := task.CreateTaskInput{
			InputPath:  testVideo,
			Translator: "local",
			Mode:       pipeline.ModeTranslate,
		}
		if err := svc.prepareTaskInput(&in); err != nil {
			t.Fatalf("prepareTaskInput 失败: %v", err)
		}
		if in.Translator != "openai" {
			t.Errorf("期望 Translator 自动映射为 openai，实际为: %s", in.Translator)
		}
		if in.OutputDir == "" {
			t.Errorf("期望生成默认输出目录，实际为空")
		}
	})

	t.Run("不存在的视频文件直接报错", func(t *testing.T) {
		in := task.CreateTaskInput{
			InputPath: filepath.Join(tmpDir, "not_exists.mp4"),
		}
		if err := svc.prepareTaskInput(&in); err == nil {
			t.Errorf("期望返回视频文件不存在错误，实际未报错")
		}
	})

	t.Run("源文件安全拷贝与 source_meta 记录", func(t *testing.T) {
		trickyVideo := filepath.Join(tmpDir, "TEDxTalks (1080p, h264, youtube).mp4")
		if err := os.WriteFile(trickyVideo, []byte("fake video tricky"), 0o644); err != nil {
			t.Fatalf("创建临时文件失败: %v", err)
		}
		in := task.CreateTaskInput{
			InputPath: trickyVideo,
		}
		if err := svc.prepareTaskInput(&in); err != nil {
			t.Fatalf("prepareTaskInput 失败: %v", err)
		}
		if in.InputPath == trickyVideo {
			t.Errorf("期望 InputPath 改为安全的暂存副本，实际仍为原路径: %s", in.InputPath)
		}
		metaFile := filepath.Join(in.OutputDir, "source_meta.json")
		if _, err := os.Stat(metaFile); err != nil {
			t.Errorf("期望生成 source_meta.json，但未找到: %v", err)
		}
		if _, err := os.Stat(in.InputPath); err != nil {
			t.Errorf("期望暂存副本存在，但未找到: %v", err)
		}
	})
}

func TestWailsServiceBindings(t *testing.T) {
	_ = events.Common.WindowFilesDropped
}

func TestSetActiveWhisperModelRejectsInvalidSelection(t *testing.T) {
	for _, model := range []string{"", "/missing/ggml-tiny.bin", "/models/ggml-silero-v6.2.0.bin"} {
		t.Run(model, func(t *testing.T) {
			bc := conf.Bootstrap{Pipeline: conf.Pipeline{WhisperModel: "previous"}, Runtime: conf.Runtime{ConfigPath: filepath.Join(t.TempDir(), "config.toml")}}
			err := (&AppService{bc: &bc}).SetActiveWhisperModel(model)
			if err == nil || bc.Pipeline.WhisperModel != "previous" {
				t.Fatalf("无效模型被接受或污染配置: err=%v model=%s", err, bc.Pipeline.WhisperModel)
			}
		})
	}
}

func TestSetActiveWhisperModelRollsBackFailedSave(t *testing.T) {
	model := filepath.Join(t.TempDir(), "custom.bin")
	if err := os.WriteFile(model, []byte("model fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	bc := conf.Bootstrap{Pipeline: conf.Pipeline{WhisperModel: "previous"}, Runtime: conf.Runtime{ConfigPath: filepath.Join(t.TempDir(), "missing", "config.toml")}}
	if err := (&AppService{bc: &bc}).SetActiveWhisperModel(model); err == nil || bc.Pipeline.WhisperModel != "previous" {
		t.Fatalf("保存失败时未回滚: err=%v model=%s", err, bc.Pipeline.WhisperModel)
	}
}
