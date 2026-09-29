package wails

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/update"
)

const (
	// 启动检查不长时间占用请求，安装允许下载较大的发布包。
	updateCheckTimeout   = 20 * time.Second
	updateInstallTimeout = 10 * time.Minute
)

// UpdateInfo 中的 available 已按自动或手动检查的忽略规则计算。
type UpdateInfo struct {
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	Available bool   `json:"available"`
	Supported bool   `json:"supported"`
	Reason    string `json:"reason"`
}

type UpdateStatus struct {
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

type updateSession struct {
	mu     sync.Mutex
	client *update.Client
	latest *update.Release
	status UpdateStatus
}

func (s *AppService) updateStatePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dir := s.bc.Runtime.ConfigDir
	if dir == "" {
		dir = filepath.Dir(s.bc.Runtime.ConfigPath)
	}
	return filepath.Join(dir, "update.toml")
}

func (s *AppService) updateClient() *update.Client {
	s.updates.mu.Lock()
	defer s.updates.mu.Unlock()
	if s.updates.client == nil {
		s.updates.client = update.NewClient()
	}
	return s.updates.client
}

func (s *AppService) CheckForUpdates(manual bool) (*UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()
	latest, err := s.updateClient().Fetch(ctx, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, fmt.Errorf("检查更新失败: %w", err)
	}
	state, err := conf.ReadUpdateState(s.updateStatePath())
	if err != nil {
		return nil, fmt.Errorf("读取更新偏好失败: %w", err)
	}
	s.mu.RLock()
	current := s.bc.Runtime.BuildVersion
	s.mu.RUnlock()
	current = installedReleaseVersion(current, state)
	available, err := update.ShouldPrompt(current, latest.Version, state.IgnoredVersion, manual)
	if err != nil {
		return nil, err
	}
	s.updates.mu.Lock()
	s.updates.latest = latest
	s.updates.mu.Unlock()
	return &UpdateInfo{Version: latest.Version, Notes: latest.Notes, Available: available, Supported: latest.Supported(), Reason: latest.Reason()}, nil
}

// 只有程序内容仍与已安装发布吻合时，才使用发布标签作为版本比较依据。
func installedReleaseVersion(current string, state conf.UpdateState) string {
	if state.InstalledReleaseVersion == "" || state.InstalledBinarySHA256 == "" {
		return current
	}
	exe, err := os.Executable()
	if err != nil {
		slog.Warn("校验已安装发布失败", "err", err)
		return current
	}
	file, err := os.Open(exe)
	if err != nil {
		slog.Warn("读取当前程序失败", "err", err)
		return current
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		slog.Warn("计算程序校验值失败", "err", err)
		return current
	}
	if hex.EncodeToString(digest.Sum(nil)) != state.InstalledBinarySHA256 {
		return current
	}
	if compared, err := update.CompareVersions(state.InstalledReleaseVersion, current); err == nil && compared > 0 {
		return state.InstalledReleaseVersion
	}
	return current
}

func (s *AppService) IgnoreUpdate(version string) error {
	if _, err := update.CompareVersions(version, version); err != nil {
		return err
	}
	s.updates.mu.Lock()
	defer s.updates.mu.Unlock()
	if s.updates.latest == nil || s.updates.latest.Version != version {
		return fmt.Errorf("请先检查更新，再忽略已显示的版本")
	}
	path := s.updateStatePath()
	state, err := conf.ReadUpdateState(path)
	if err != nil {
		return err
	}
	if state.IgnoredVersion != "" {
		compared, err := update.CompareVersions(version, state.IgnoredVersion)
		if err != nil {
			return err
		}
		if compared <= 0 {
			return nil
		}
	}
	state.IgnoredVersion = version
	return conf.WriteConfig(state, path)
}

func (s *AppService) GetUpdateStatus() UpdateStatus {
	s.updates.mu.Lock()
	defer s.updates.mu.Unlock()
	if s.updates.status.Phase == "" {
		return UpdateStatus{Phase: "idle"}
	}
	return s.updates.status
}

func (s *AppService) setUpdateStatus(phase, message string) {
	s.updates.mu.Lock()
	defer s.updates.mu.Unlock()
	s.updates.status = UpdateStatus{Phase: phase, Message: message}
}

func (s *AppService) InstallUpdate(version string) error {
	latest, err := s.beginUpdate(version)
	if err != nil {
		return err
	}
	err = s.prepareAndRestart(latest)
	if err != nil {
		s.setUpdateStatus("error", err.Error())
	}
	return err
}

func (s *AppService) beginUpdate(version string) (*update.Release, error) {
	s.updates.mu.Lock()
	defer s.updates.mu.Unlock()
	if s.updates.latest == nil || s.updates.latest.Version != version {
		return nil, fmt.Errorf("请先检查更新，再安装已显示的版本")
	}
	state, err := conf.ReadUpdateState(s.updateStatePath())
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	current := s.bc.Runtime.BuildVersion
	s.mu.RUnlock()
	current = installedReleaseVersion(current, state)
	newer, err := update.ShouldPrompt(current, version, "", true)
	if err != nil {
		return nil, err
	}
	if !newer {
		return nil, fmt.Errorf("当前已是最新版本，不能安装更早或相同版本")
	}
	if !s.updates.latest.Supported() {
		return nil, fmt.Errorf("%s", s.updates.latest.Reason())
	}
	if s.app == nil {
		return nil, fmt.Errorf("更新需要在桌面应用中运行")
	}
	switch s.updates.status.Phase {
	case "downloading", "preparing", "restarting":
		return nil, fmt.Errorf("更新正在进行，请稍候")
	}
	s.updates.status = UpdateStatus{Phase: "downloading", Message: "正在下载更新，请保持网络连接…"}
	return s.updates.latest, nil
}

func (s *AppService) prepareAndRestart(latest *update.Release) (err error) {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	client := update.NewClient()
	client.Progress = s.setUpdateStatus
	ctx, cancel := context.WithTimeout(context.Background(), updateInstallTimeout)
	defer cancel()
	plan, err := client.Prepare(ctx, latest, exe)
	if err != nil {
		return fmt.Errorf("准备更新失败: %w", err)
	}
	handedOff := false
	defer func() {
		if !handedOff {
			err = errors.Join(err, plan.Discard())
		}
	}()
	if err := s.validateUpdateDataPaths(plan.Target); err != nil {
		return err
	}
	plan.StatePath = s.updateStatePath()
	plan.RestartArgs = s.updateRestartArgs()
	if err := plan.Start(); err != nil {
		return fmt.Errorf("启动更新程序失败: %w", err)
	}
	handedOff = true
	s.setUpdateStatus("restarting", "更新已就绪，正在重启应用…")
	s.app.Quit()
	return nil
}

// 重启时使用已解析的配置目录，避免工作目录变化让相对 -conf 指向其他配置。
func (s *AppService) updateRestartArgs() []string {
	args := make([]string, 0, len(os.Args)+2)
	skipNext := false
	for _, arg := range os.Args[1:] {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "-conf" || arg == "--conf" {
			skipNext = true
			continue
		}
		if strings.HasPrefix(arg, "-conf=") || strings.HasPrefix(arg, "--conf=") {
			continue
		}
		args = append(args, arg)
	}
	s.mu.RLock()
	dir := s.bc.Runtime.ConfigDir
	s.mu.RUnlock()
	return append(args, "-conf", dir)
}

// 整包替换会移动安装目录，用户数据必须保持在包外的原位置。
func (s *AppService) validateUpdateDataPaths(target string) error {
	s.mu.RLock()
	paths := []string{conf.ResolveDSN(s.bc.Data.Database.Dsn), conf.TaskOutputDir(s.bc.Pipeline.DefaultOutputDir), s.bc.Pipeline.WhisperModel}
	s.mu.RUnlock()
	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := validateExternalDataPath(target, path); err != nil {
			return err
		}
	}
	return nil
}

func validateExternalDataPath(target, path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	candidates := []string{absolute}
	if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
		candidates = append(candidates, resolved)
	}
	roots := []string{target}
	if resolved, err := filepath.EvalSymlinks(target); err == nil {
		roots = append(roots, resolved)
	}
	for _, candidate := range candidates {
		for _, root := range roots {
			if !strings.EqualFold(filepath.VolumeName(root), filepath.VolumeName(candidate)) {
				continue
			}
			if runtime.GOOS == "windows" {
				candidate, root = strings.ToLower(candidate), strings.ToLower(root)
			}
			relative, err := filepath.Rel(root, candidate)
			if err != nil {
				return err
			}
			if relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return fmt.Errorf("用户数据位于应用安装目录内，请先将数据库、输出目录或模型移到应用目录外再更新: %s", path)
			}
		}
	}
	return nil
}
