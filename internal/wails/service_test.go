package wails

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"github.com/wailsapp/wails/v3/pkg/events"
)

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
