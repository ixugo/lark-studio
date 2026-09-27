package api

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWhisperModelsIncludeTiny 验证推荐清单包含适合轻量设备的模型。
func TestWhisperModelsIncludeTiny(t *testing.T) {
	if len(whisperModels) != 6 {
		t.Fatalf("模型数量应严格为 6 个，实际为 %d", len(whisperModels))
	}
	want := map[string]bool{
		"large-v3-turbo": true,
		"tiny":           true,
		"medium":         true,
		"small":          true,
		"medium.en":      true,
		"small.en":       true,
	}
	tinyHasExpectedSize := false
	for _, model := range whisperModels {
		delete(want, model.Name)
		if model.Name == "tiny" && model.Size == "75 MiB" {
			tinyHasExpectedSize = true
		}
	}
	if len(want) != 0 {
		t.Fatalf("缺少预期的核心模型: %v", want)
	}
	if !tinyHasExpectedSize {
		t.Fatal("tiny 多语言模型应标记为 75 MiB")
	}
}

// TestListModelsDiscoversEveryInstalledGGML 确保管理页发现清单外的 GGML 与 Silero 文件。
func TestListModelsDiscoversEveryInstalledGGML(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	modelsDir := filepath.Join(home, ".lark-studio", "models")
	writeModelTestFile(t, filepath.Join(modelsDir, "extra", "ggml-tiny.bin"), "tiny")
	writeModelTestFile(t, filepath.Join(modelsDir, "ggml-silero-v6.2.0.bin"), "vad")

	models, err := listModels(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertDiscoveredModel(t, models, "tiny", "asr", "tiny")
	assertDiscoveredModel(t, models, "silero-v6.2.0", "vad", "vad")
}

// writeModelTestFile 创建隔离的模型发现样本。
func writeModelTestFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

// assertDiscoveredModel 验证额外模型的类别、状态与实际位置。
func assertDiscoveredModel(t *testing.T, models []ModelListOutput, name, kind, content string) {
	t.Helper()
	for _, model := range models {
		if model.Name != name {
			continue
		}
		if !model.Downloaded || model.Kind != kind {
			t.Fatalf("模型 %s 状态错误: %+v", name, model)
		}
		data, err := os.ReadFile(model.Path)
		if err != nil || string(data) != content {
			t.Fatalf("模型 %s 未指向原文件: data=%q err=%v", name, data, err)
		}
		return
	}
	t.Fatalf("模型列表未发现 %s: %+v", name, models)
}
