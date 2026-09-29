package conf

import (
	"errors"
	"os"
)

// UpdateState 单独保存更新偏好，避免配置页面保存或热重载覆盖忽略记录。
type UpdateState struct {
	InstalledReleaseVersion string `comment:"已安装的 GitHub 发布标签"`
	InstalledBinarySHA256   string `comment:"绑定发布标签的程序校验值，手动替换程序后自动失效"`
	IgnoredVersion          string `comment:"自动检查不再提示此版本及更早版本；手动检查不受影响"`
}

func ReadUpdateState(path string) (UpdateState, error) {
	var state UpdateState
	err := SetupConfig(&state, path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	return state, err
}
