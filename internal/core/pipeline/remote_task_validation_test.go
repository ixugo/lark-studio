package pipeline

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ixugo/vdub/internal/conf"
)

func TestRemoteTaskRejectsUnknownModelsAndDraftVoice(t *testing.T) {
	const qwenModel = "mlx-community/Qwen3-TTS-12Hz-1.7B-CustomVoice-4bit"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			fmt.Fprintf(w, `{"data":[{"id":"known-model"},{"id":%q}]}`, qwenModel)
		case "/openapi.json":
			fmt.Fprint(w, `{"components":{"schemas":{"SpeechRequest":{"properties":{"instruct":{},"lang_code":{}}}}}}`)
		case "/v1/audio/voices":
			if r.URL.Query().Get("model") == qwenModel {
				http.NotFound(w, r)
				return
			}
			fmt.Fprint(w, `{"data":[{"id":"Vivian","name":"Vivian"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	for _, tc := range []struct {
		name string
		job  Job
		bc   conf.Bootstrap
		want string
	}{
		{"旧识别模型", Job{InputPath: "video.mp4", Mode: ModeSubtitle}, conf.Bootstrap{Pipeline: conf.Pipeline{WhisperMode: "openai", ASRBaseURL: server.URL + "/v1", ASRModel: "unknown-asr"}}, "语音识别"},
		{"旧翻译模型", Job{InputPath: "text.txt", Mode: ModeTextTranslate, Translator: "local"}, conf.Bootstrap{LLM: conf.LLM{Provider: "bing", BaseURL: server.URL + "/v1", Model: "unknown-llm"}}, "翻译"},
		{"旧配音模型", Job{InputPath: "text.txt", Mode: ModeDubOnly, TTSEngine: "openai"}, conf.Bootstrap{TTS: conf.TTS{Type: "edge", BaseURL: server.URL + "/v1", Model: "unknown-tts", OpenAIVoice: "Vivian"}}, "语音合成"},
		{"旧草稿未知音色", Job{InputPath: "text.txt", Mode: ModeDubOnly, TTSEngine: "openai", TTSVoice: "unknown-voice"}, conf.Bootstrap{TTS: conf.TTS{BaseURL: server.URL + "/v1", Model: "known-model", OpenAIVoice: "Vivian"}}, "unknown-voice"},
		{"旧草稿未知 Qwen 音色", Job{InputPath: "text.txt", Mode: ModeDubOnly, TTSEngine: "openai", TTSVoice: "alloy"}, conf.Bootstrap{TTS: conf.TTS{BaseURL: server.URL + "/v1", Model: qwenModel, OpenAIVoice: "Vivian"}}, "alloy"},
		{"Edge 历史音色不能冒充 OpenAI", Job{InputPath: "text.txt", Mode: ModeDubOnly, TTSEngine: "openai"}, conf.Bootstrap{TTS: conf.TTS{Type: "edge", BaseURL: server.URL + "/v1", Model: "known-model", Voice: "zh-CN-XiaoxiaoNeural"}}, "请选择有效音色"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRemoteTask(t.Context(), tc.job, &tc.bc)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("应拒绝 %s: %v", tc.name, err)
			}
		})
	}
	bc := conf.Bootstrap{TTS: conf.TTS{Type: "openai", BaseURL: server.URL + "/v1", Model: "known-model", OpenAIVoice: "Vivian", Voice: "legacy-invalid"}}
	job := Job{InputPath: "text.txt", Mode: ModeDubOnly}
	if err := ValidateRemoteTask(t.Context(), job, &bc); err != nil {
		t.Fatalf("独立 OpenAI 默认音色未生效: %v", err)
	}
	bc.TTS.OpenAIVoice = ""
	bc.TTS.Voice = "Vivian"
	if err := ValidateRemoteTask(t.Context(), job, &bc); err != nil {
		t.Fatalf("旧版音色回退未生效: %v", err)
	}
	bc.TTS.Instructions = "unsupported emotion"
	if err := ValidateRemoteTask(t.Context(), job, &bc); err == nil || !strings.Contains(err.Error(), "情绪") {
		t.Fatalf("未拒绝不支持的情绪选项: %v", err)
	}
}

func TestRemoteTaskOnlyQueriesNeededNodes(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, `{"data":[{"id":"known-model"}]}`)
	}))
	defer server.Close()
	dir := t.TempDir()
	video := filepath.Join(dir, "video.mp4")
	sourceSubtitle := filepath.Join(dir, "video.srt")
	if err := os.WriteFile(sourceSubtitle, []byte("subtitle"), 0600); err != nil {
		t.Fatal(err)
	}
	bc := conf.Bootstrap{Pipeline: conf.Pipeline{WhisperMode: "openai", ASRBaseURL: server.URL + "/v1", ASRModel: "known-model"}, LLM: conf.LLM{Provider: "openai", BaseURL: server.URL + "/v1", Model: "known-model"}, TTS: conf.TTS{Type: "openai", BaseURL: "http://127.0.0.1:1/v1", Model: "offline", Voice: "offline"}}
	for _, tc := range []struct {
		name         string
		job          Job
		wantRequests int32
	}{
		{"仅字幕且有输入旁字幕", Job{InputPath: video, Mode: ModeSubtitle}, 0},
		{"错误断点不能跳过识别", Job{InputPath: "other.mp4", Mode: ModeSubtitle, ResumeFrom: StepTTS}, 1},
		{"纯文本翻译", Job{InputPath: "text.txt", Mode: ModeTextTranslate}, 1},
		{"必应字幕翻译", Job{InputPath: video, Mode: ModeTranslate, Translator: "bing"}, 0},
		{"仅合成重跑", Job{InputPath: video, Mode: ModeDub, ResumeFrom: StepMerge}, 0},
		{"配音重跑不查翻译", Job{InputPath: video, Mode: ModeDub, ResumeFrom: StepTTS, TTSEngine: "edge"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests.Store(0)
			if err := ValidateRemoteTask(t.Context(), tc.job, &bc); err != nil {
				t.Fatal(err)
			}
			if got := requests.Load(); got != tc.wantRequests {
				t.Fatalf("查询次数=%d,期望%d", got, tc.wantRequests)
			}
		})
	}
	// 输出目录现有字幕也应跳过识别，但空字幕不能作为已完成转录。
	out := t.TempDir()
	srt := filepath.Join(out, "src.srt")
	for _, size := range []int{1, 0} {
		if err := os.WriteFile(srt, []byte(strings.Repeat("x", size)), 0600); err != nil {
			t.Fatal(err)
		}
		requests.Store(0)
		if err := ValidateRemoteTask(t.Context(), Job{InputPath: "other.mp4", OutputDir: out, Mode: ModeSubtitle}, &bc); err != nil {
			t.Fatal(err)
		}
		want := int32(0)
		if size == 0 {
			want = 1
		}
		if requests.Load() != want {
			t.Fatalf("字幕大小%d,查询%d,期望%d", size, requests.Load(), want)
		}
	}
}

func TestRemoteTaskBoundary(t *testing.T) {
	for _, tc := range []struct {
		name string
		job  Job
		bc   *conf.Bootstrap
	}{
		{"空配置", Job{InputPath: "video.mp4", Mode: ModeSubtitle}, nil},
		{"空输入", Job{Mode: ModeSubtitle}, &conf.Bootstrap{}},
		{"非法配方", Job{InputPath: "video.mp4", Mode: 99}, &conf.Bootstrap{}},
		{"视频跳过听写", Job{InputPath: "video.mp4", Mode: ModeTextTranslate}, &conf.Bootstrap{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateRemoteTask(t.Context(), tc.job, tc.bc); err == nil {
				t.Fatal("边界输入应拒绝")
			}
		})
	}
}

func TestRemoteTaskRejectsUnavailableModelDirectory(t *testing.T) {
	for _, body := range []string{`{}`, `{"data":[]}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		bc := conf.Bootstrap{LLM: conf.LLM{Provider: "openai", BaseURL: server.URL + "/v1", Model: "previous-model"}}
		err := ValidateRemoteTask(t.Context(), Job{InputPath: "text.txt", Mode: ModeTextTranslate}, &bc)
		server.Close()
		if err == nil || !strings.Contains(err.Error(), "模型") {
			t.Fatalf("无模型列表应拒绝: %v", err)
		}
	}
}
