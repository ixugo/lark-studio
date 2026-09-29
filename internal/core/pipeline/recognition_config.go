package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ixugo/vdub/internal/adapter/asr"
	"github.com/ixugo/vdub/internal/adapter/whisper"
)

// ValidateRecognitionConfig 在创建任务及暂存源文件前检查识别条件。
// 纯文本任务与已有字幕的合成任务不需要调用识别引擎。
func ValidateRecognitionConfig(inputPath, outputDir string, mode int, config asr.Config) error {
	if err := ValidateResourceMode(inputPath, mode); err != nil {
		return err
	}
	if mode == ModeDubOnly || mode == ModeTextTranslate {
		return nil
	}
	sourceSRT := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".srt"
	for _, path := range []string{filepath.Join(outputDir, "src.srt"), sourceSRT} {
		if path == "src.srt" {
			continue
		}
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
			return nil
		}
	}
	if config.Engine == "" {
		return fmt.Errorf("请选择语音识别引擎")
	}
	if err := asr.ValidateConfig(config); err != nil {
		return err
	}
	if config.Engine == "openai" {
		return nil
	}
	_, err := whisper.ResolveModel(config.WhisperModel)
	return err
}

// ValidateResourceMode 防止媒体资源误入跳过听写的纯文本配方。
func ValidateResourceMode(inputPath string, mode int) error {
	if mode != ModeDubOnly && mode != ModeTextTranslate {
		return nil
	}
	switch strings.ToLower(filepath.Ext(inputPath)) {
	case ".mp4", ".mkv", ".mov", ".avi", ".webm", ".flv", ".mp3", ".wav", ".m4a", ".aac", ".flac", ".ogg":
		return fmt.Errorf("视频和音频必须先经过听写转录，不能使用纯文本配方")
	}
	return nil
}
