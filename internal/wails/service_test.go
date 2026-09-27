package wails

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// TestWhisperModelConfigRoundTrip 保证模型路径经前端配置、TOML 持久化与重载后保持一致。
func TestWhisperModelConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	want := filepath.Join(t.TempDir(), "ggml-large-v3-turbo.bin")
	bc := conf.Bootstrap{Runtime: conf.Runtime{ConfigPath: path}}
	svc := &AppService{bc: &bc}
	updates := map[string]any{"pipeline": map[string]any{"whisper_model": want}}
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
}

func TestPrepareTaskInput(t *testing.T) {
	// 创建临时测试文件
	tmpDir := t.TempDir()
	testVideo := filepath.Join(tmpDir, "sample.mp4")
	if err := os.WriteFile(testVideo, []byte("fake video content"), 0o644); err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}

	bc := conf.DefaultConfig()
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

	t.Run("空引擎兜底使用全局配置", func(t *testing.T) {
		in := task.CreateTaskInput{
			InputPath:  testVideo,
			Translator: "",
		}
		if err := svc.prepareTaskInput(&in); err != nil {
			t.Fatalf("prepareTaskInput 失败: %v", err)
		}
		if in.Translator != "bing" {
			t.Errorf("期望兜底为 bing，实际为: %s", in.Translator)
		}
	})
}

func TestWailsServiceBindings(t *testing.T) {
	_ = events.Common.WindowFilesDropped
}
