package conf

import "testing"

func TestVoiceForEngineRetainsUntypedLegacyVoice(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config TTS
		engine string
		want   string
	}{
		{"历史无类型 OpenAI 音色", TTS{Voice: "alloy"}, "openai", "alloy"},
		{"历史无类型 Edge 音色", TTS{Voice: "zh-CN-XiaoxiaoNeural"}, "edge", "zh-CN-XiaoxiaoNeural"},
		{"明确 Edge 不传 OpenAI", TTS{Type: "edge", Voice: "zh-CN-XiaoxiaoNeural"}, "openai", ""},
		{"独立 OpenAI 优先", TTS{Voice: "alloy", OpenAIVoice: "Vivian"}, "openai", "Vivian"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := VoiceForEngine(tc.config, tc.engine); got != tc.want {
				t.Fatalf("音色=%q,期望%q", got, tc.want)
			}
		})
	}
}
