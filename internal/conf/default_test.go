package conf

import "testing"

func TestDefaultConfigUsesXiaoxiaoVoice(t *testing.T) {
	t.Parallel()

	if got := DefaultConfig().TTS.Voice; got != "zh-CN-XiaoxiaoNeural" {
		t.Fatalf("默认音色应为晓晓，实际为 %q", got)
	}
}
