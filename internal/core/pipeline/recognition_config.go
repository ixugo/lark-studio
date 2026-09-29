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
