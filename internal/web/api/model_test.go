package api

import "testing"

// TestWhisperModelsCoverSmartSubCatalog 验证界面提供 SmartSub 的全部 GGML 模型档位。
func TestWhisperModelsCoverSmartSubCatalog(t *testing.T) {
	want := map[string]bool{
		"tiny.en-q8_0": true, "base.en-q8_0": true, "small.en-q8_0": true,
		"medium.en-q8_0": true, "large-v3-turbo-q8_0": true,
		"large-v3-q5_0": true, "large-v2-q8_0": true, "large-v1": true,
	}
	for _, model := range whisperModels {
		delete(want, model.Name)
	}
	if len(want) != 0 {
		t.Fatalf("缺少 SmartSub 模型：%v", want)
	}
}
