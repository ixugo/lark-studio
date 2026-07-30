package api

import (
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
