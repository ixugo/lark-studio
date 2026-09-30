package api

import (
	"fmt"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"os"
	"path/filepath"
	"testing"

	"github.com/ixugo/vdub/internal/conf"
)

// TestUpdateConfigMigratesFFmpegWhisperMode 验证旧客户端提交的 FFmpeg 选项会迁移至 whisper.cpp。
func TestUpdateConfigMigratesFFmpegWhisperMode(t *testing.T) {
	cfg := conf.DefaultConfig()
	cfg.Runtime.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	mode := "ffmpeg"
	uc := &Usecase{Conf: &cfg}

	if _, err := uc.updateConfig(nil, &updateConfigInput{
		Pipeline: &pipelineInput{WhisperMode: &mode},
	}); err != nil {
		t.Fatal(err)
	}
	if cfg.Pipeline.WhisperMode != "whisper-cpp" {
		t.Fatalf("听写引擎 = %q，期望 whisper-cpp", cfg.Pipeline.WhisperMode)
	}
}

func TestUpdateWorkersRejectsInvalidWithoutMutation(t *testing.T) {
	for _, n := range []int{-1, 0, 9} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			cfg := conf.DefaultConfig()
			cfg.Runtime.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
			uc := &Usecase{Conf: &cfg}
			if _, err := uc.updateConfig(nil, &updateConfigInput{Pipeline: &pipelineInput{Workers: new(n)}}); err == nil {
				t.Fatal("接受无效并发数")
			}
			if cfg.Pipeline.Workers != 2 {
				t.Fatal("无效输入改变配置")
			}
			if _, err := os.Stat(cfg.Runtime.ConfigPath); !os.IsNotExist(err) {
				t.Fatal("无效输入写入配置")
			}
		})
	}
}

func TestUpdateWorkersPersists(t *testing.T) {
	cfg := conf.DefaultConfig()
	cfg.Runtime.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	uc := &Usecase{Conf: &cfg, Scheduler: pipeline.NewScheduler(pipeline.NewCore(pipeline.Config{}, nil, nil, nil), nil)}
	t.Cleanup(uc.Scheduler.Stop)
	if _, err := uc.updateConfig(nil, &updateConfigInput{Pipeline: &pipelineInput{Workers: new(3)}}); err != nil {
		t.Fatal(err)
	}
	var loaded conf.Bootstrap
	if err := conf.SetupConfig(&loaded, cfg.Runtime.ConfigPath); err != nil {
		t.Fatal(err)
	}
	if loaded.Pipeline.Workers != 3 || cfg.Pipeline.Workers != 3 {
		t.Fatal("并发数未持久化")
	}
}
