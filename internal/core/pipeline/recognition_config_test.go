package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ixugo/vdub/internal/adapter/asr"
)

func TestRecognitionReadiness(t *testing.T) {
	model := filepath.Join(t.TempDir(), "custom.bin")
	if err := os.WriteFile(model, []byte("model"), 0600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		mode    int
		config  asr.Config
		wantErr bool
	}{
		{"未选择", ModeSubtitle, asr.Config{}, true},
		{"本地缺模型", ModeSubtitle, asr.Config{Engine: "whisper-cpp"}, true},
		{"本地有效模型", ModeSubtitle, asr.Config{Engine: "whisper-cpp", WhisperModel: model}, false},
		{"兼容缺配置", ModeSubtitle, asr.Config{Engine: "openai"}, true},
		{"兼容缺模型名", ModeSubtitle, asr.Config{Engine: "openai", BaseURL: "http://localhost:8000/v1"}, true},
		{"兼容无需本地模型与密钥", ModeSubtitle, asr.Config{Engine: "openai", BaseURL: "http://localhost:8000/v1", Model: "whisper-1"}, false},
		{"纯文本翻译", ModeTextTranslate, asr.Config{}, false},
		{"文本朗读", ModeDubOnly, asr.Config{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inputPath := "video.mp4"
			if tc.mode == ModeTextTranslate || tc.mode == ModeDubOnly {
				inputPath = "input.txt"
			}
			if err := ValidateRecognitionConfig(inputPath, filepath.Join(t.TempDir(), "out"), tc.mode, tc.config); (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestExistingSubtitleDoesNotRequireRecognition(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "src.srt"), []byte("subtitle"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRecognitionConfig("video.mp4", dir, ModeSubtitle, asr.Config{}); err != nil {
		t.Fatal(err)
	}
}

func TestMediaCannotUseTextOnlyRecipe(t *testing.T) {
	for _, input := range []string{"video.mp4", "audio.wav"} {
		for _, mode := range []int{ModeTextTranslate, ModeDubOnly} {
			if err := ValidateRecognitionConfig(input, t.TempDir(), mode, asr.Config{}); err == nil {
				t.Errorf("%s 模式 %d 跳过听写，应拒绝提交", input, mode)
			}
		}
	}
}
