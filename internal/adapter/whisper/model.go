package whisper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ixugo/vdub/internal/conf"
)

// maxModelPathLength 限制配置输入长度，避免异常路径进入文件查找。
const maxModelPathLength = 4096

// ResolveModel 校验识别模型并将旧模型简称转换为实际文件路径。
func ResolveModel(value string) (string, error) {
	model := strings.TrimSpace(value)
	if model == "" {
		return "", fmt.Errorf("请选择已下载的 Whisper 识别模型，或配置自定义模型路径")
	}
	if len(model) > maxModelPathLength {
		return "", fmt.Errorf("Whisper 模型路径超过长度限制")
	}
	name := strings.TrimPrefix(strings.ToLower(filepath.Base(model)), "ggml-")
	if strings.HasPrefix(name, "silero") {
		return "", fmt.Errorf("Silero 仅用于人声检测，请选择 Whisper 识别模型")
	}
	if !strings.ContainsAny(model, `/\\`) {
		name := strings.TrimSuffix(strings.TrimPrefix(model, "ggml-"), ".bin")
		model = filepath.Join(conf.StudioDir(), "models", "ggml-"+name+".bin")
	}
	path, err := filepath.Abs(model)
	if err != nil {
		return "", fmt.Errorf("解析 Whisper 模型路径失败: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("Whisper 识别模型不可用 %q，请检查自定义路径或先下载模型: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return "", fmt.Errorf("Whisper 识别模型必须是非空文件: %s", path)
	}
	return path, nil
}
