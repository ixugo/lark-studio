package whisper

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	_ "embed"
)

const (
	// 固定摘要保证内嵌的人声检测模型与验证过的 ONNX 文件一致。
	speechModelName     = "ggml-silero-v6.2.0.bin"
	speechModelSHA256   = "2aa269b785eeb53a82983a20501ddf7c1d9c48e33ab63a41391ac6c9f7fb6987"
	speechModelMaxBytes = 2 << 20
	// 给语音边缘保留辅音余量，避免检测窗口切掉句首。
	speechPaddingMillis = 150
)

//go:embed assets/ggml-silero-v6.2.0.bin
var embeddedSpeechModel []byte

// EnsureSpeechModel 将已校验的内嵌模型释放到统一资源目录，保证离线任务可用。
func EnsureSpeechModel(dir string) (string, error) {
	path := filepath.Join(dir, speechModelName)
	data, err := os.ReadFile(path)
	if err == nil {
		if err := validateSpeechModel(data); err == nil {
			return path, nil
		}
		if err := validateSpeechModel(embeddedSpeechModel); err != nil {
			return "", err
		}
		if err := saveSpeechModel(dir, embeddedSpeechModel); err != nil {
			return "", err
		}
		return path, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	if err := validateSpeechModel(embeddedSpeechModel); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := saveSpeechModel(dir, embeddedSpeechModel); err != nil {
		return "", err
	}
	return path, nil
}

// validateSpeechModel 只接受已确认版本，保证检测器不会加载损坏模型。
func validateSpeechModel(data []byte) error {
	if len(data) == 0 || len(data) > speechModelMaxBytes || fmt.Sprintf("%x", sha256.Sum256(data)) != speechModelSHA256 {
		return fmt.Errorf("Silero 模型校验失败: %s", speechModelName)
	}
	return nil
}

// saveSpeechModel 原子发布校验后的模型，避免并发任务读到半个文件。
func saveSpeechModel(dir string, data []byte) error {
	file, err := os.CreateTemp(dir, ".silero-*.bin")
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		closeErr := file.Close()
		return fmt.Errorf("写入人声模型: %w，关闭文件: %v", err, closeErr)
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), filepath.Join(dir, speechModelName))
}
