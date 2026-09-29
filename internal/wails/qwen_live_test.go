package wails

import (
	"encoding/base64"
	"github.com/ixugo/vdub/internal/conf"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 仅显式本机验收时运行，不在普通测试中访问用户在线模型。
func TestLiveLocalQwenVoice(t *testing.T) {
	if os.Getenv("LARK_QWEN_LIVE") != "1" {
		t.Skip("显式启用后测试用户本机 MLX Qwen 服务")
	}
	svc := new(AppService)
	const model = "mlx-community/Qwen3-TTS-12Hz-0.6B-CustomVoice-8bit"
	models, err := svc.ListRemoteModels("http://127.0.0.1:8399/v1", "not-needed")
	if err != nil {
		t.Fatal(err)
	}
	if len(models) == 0 || models[0].ID != model {
		t.Fatalf("模型目录变化: %+v", models)
	}
	cap, err := svc.GetTTSCapabilities("http://127.0.0.1:8399/v1", "not-needed", model)
	if err != nil {
		t.Fatal(err)
	}
	if cap.Protocol != "mlx" || cap.VoiceSource != "qwen_builtin" || len(cap.Voices) != 9 || cap.Instructions {
		t.Fatalf("现场能力识别不符合模型: %+v", cap)
	}
	for i, text := range []string{"你好。", "欢迎使用。"} {
		uri, err := svc.TestConfiguredTTS(conf.TTS{Type: "openai", BaseURL: "http://127.0.0.1:8399/v1", APIKey: "not-needed", Model: model, Voice: "Vivian", Protocol: "mlx", Language: "Chinese"}, text)
		if err != nil {
			t.Fatal(err)
		}
		_, payload, ok := strings.Cut(uri, ";base64,")
		if !ok {
			t.Fatal("没有音频")
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) < 44 || string(data[:4]) != "RIFF" {
			t.Fatalf("响应并非可封装的 WAV: %d bytes", len(data))
		}
		name := []string{"qwen-vivian-1.wav", "qwen-vivian-2.wav"}[i]
		path := filepath.Join("../../tmp/remote-voice-capabilities", name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		t.Logf("固定 Vivian / Chinese：%s，%d bytes", name, len(data))
	}
}
