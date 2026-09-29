package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ixugo/vdub/internal/conf"
)

const (
	parentExitTimeout = 120 * time.Second
	restartReadyDelay = 3 * time.Second
	maxPlanBytes      = 64 << 10
)

type Plan struct {
	Target             string   `json:"target"`
	Staged             string   `json:"staged"`
	Backup             string   `json:"backup"`
	ExecutableRelative string   `json:"executableRelative"`
	ParentPID          int      `json:"parentPID"`
	ReleaseVersion     string   `json:"releaseVersion"`
	BinaryVersion      string   `json:"binaryVersion"`
	OriginalSHA256     string   `json:"originalSHA256"`
	StagedSHA256       string   `json:"stagedSHA256"`
	StatePath          string   `json:"statePath"`
	RestartArgs        []string `json:"restartArgs"`
	WorkingDir         string   `json:"workingDir"`
	PlanPath           string   `json:"-"`
	started            bool
}

func planRoot() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "lark-studio", "updates"), nil
}

func helperName() string {
	if runtime.GOOS == "windows" {
		return "update-helper.exe"
	}
	return "update-helper"
}

func (p *Plan) persist(executable string) (err error) {
	root, err := planRoot()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return err
	}
	dir, err := os.MkdirTemp(root, "run-")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, os.RemoveAll(dir))
		}
	}()
	p.PlanPath = filepath.Join(dir, "plan.json")
	p.OriginalSHA256, err = fileSHA256(executable)
	if err != nil {
		return err
	}
	p.StagedSHA256, err = fileSHA256(filepath.Join(p.Staged, p.ExecutableRelative))
	if err != nil {
		return err
	}
	if err = copyFile(executable, filepath.Join(dir, helperName())); err != nil {
		return err
	}
	return p.write()
}

func (p *Plan) write() error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if len(data) > maxPlanBytes {
		return errors.New("更新计划超过大小限制")
	}
	temp, err := os.CreateTemp(filepath.Dir(p.PlanPath), ".plan-")
	if err != nil {
		return err
	}
	name := temp.Name()
	if _, err := temp.Write(data); err != nil {
		return errors.Join(err, temp.Close(), os.Remove(name))
	}
	if err := errors.Join(temp.Sync(), temp.Close()); err != nil {
		return errors.Join(err, os.Remove(name))
	}
	if err := os.Rename(name, p.PlanPath); err != nil {
		return errors.Join(err, os.Remove(name))
	}
	return nil
}

// Start 启动独立辅助进程。调用方在成功返回后退出当前应用。
func (p *Plan) Start() error {
	if err := p.validate(false); err != nil {
		return err
	}
	if err := p.write(); err != nil {
		return err
	}
	dir := filepath.Dir(p.PlanPath)
	log, err := os.OpenFile(filepath.Join(dir, "helper.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	command := exec.Command(filepath.Join(dir, helperName()), "--apply-update", p.PlanPath)
	command.Dir = dir
	command.Stdout = log
	command.Stderr = log
	detach(command)
	if err := command.Start(); err != nil {
		return errors.Join(err, log.Close())
	}
	p.started = true
	return errors.Join(command.Process.Release(), log.Close())
}

// Discard 只清理尚未交给辅助进程的本次暂存文件，不触碰安装目录。
func (p *Plan) Discard() error {
	if p.started {
		return errors.New("更新辅助进程已启动，不能删除其暂存文件")
	}
	root := filepath.Dir(p.Staged)
	if !filepath.IsAbs(p.Target) || filepath.Clean(p.Target) != p.Target || !filepath.IsAbs(root) || filepath.Clean(root) != root || root == p.Target || filepath.Dir(root) != filepath.Dir(p.Target) || !strings.HasPrefix(filepath.Base(root), ".lark-studio-update-") || p.Backup != filepath.Join(root, "previous") {
		return errors.New("不能清理不属于本次更新的暂存目录")
	}
	if _, err := os.Lstat(p.Backup); !errors.Is(err, os.ErrNotExist) {
		return errors.New("旧应用已备份，不能清理恢复目录")
	}
	privateRoot, err := planRoot()
	if err != nil {
		return err
	}
	if !filepath.IsAbs(p.PlanPath) || filepath.Clean(p.PlanPath) != p.PlanPath || filepath.Base(p.PlanPath) != "plan.json" || filepath.Dir(filepath.Dir(p.PlanPath)) != privateRoot || !strings.HasPrefix(filepath.Base(filepath.Dir(p.PlanPath)), "run-") {
		return errors.New("不能清理不属于本次更新的辅助目录")
	}
	if rel, err := filepath.Rel(filepath.Dir(p.PlanPath), p.Target); err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return errors.New("不能清理包含安装目录的辅助目录")
	}
	return errors.Join(os.RemoveAll(root), os.RemoveAll(filepath.Dir(p.PlanPath)))
}

// RunHelper 仅允许由本次复制到私有目录的辅助程序执行合法暂存计划。
func RunHelper(path string) (err error) {
	p, err := readPlan(path)
	if err != nil {
		return err
	}
	defer func() {
		message := "更新成功；旧应用保留于 " + p.Backup
		if err != nil {
			message = "更新失败：" + err.Error() + "；恢复目录：" + p.Backup
		}
		err = errors.Join(err, os.WriteFile(filepath.Join(filepath.Dir(path), "result.txt"), []byte(message+"\n"), 0600))
	}()
	if err = p.validate(true); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), parentExitTimeout)
	defer cancel()
	if err = waitParent(ctx, p.ParentPID); err != nil {
		return fmt.Errorf("等待原应用退出: %w", err)
	}
	if err = p.validate(true); err != nil {
		return err
	}
	return applyPlan(p, func(executable string, args []string) error {
		return restartApplication(executable, args, p.WorkingDir)
	})
}

func readPlan(path string) (*Plan, error) {
	if err := validatePlanPath(path); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxPlanBytes || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
		return nil, errors.New("更新计划文件不安全")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Plan
	if err := json.Unmarshal(data, &p, json.RejectUnknownMembers(true)); err != nil {
		return nil, err
	}
	p.PlanPath = path
	return &p, nil
}

func validatePlanPath(path string) error {
	root, err := planRoot()
	if err != nil {
		return err
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.Base(path) != "plan.json" || filepath.Dir(filepath.Dir(path)) != root || !strings.HasPrefix(filepath.Base(filepath.Dir(path)), "run-") {
		return errors.New("更新计划位置不合法")
	}
	for _, dir := range []string{root, filepath.Dir(path)} {
		info, err := os.Lstat(dir)
		if err != nil {
			return err
		}
		if !info.IsDir() || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
			return errors.New("更新计划目录权限不安全")
		}
	}
	return nil
}

func (p *Plan) validate(helper bool) error {
	if err := validatePlanPath(p.PlanPath); err != nil {
		return err
	}
	if p.ParentPID <= 0 || (helper && p.ParentPID == os.Getpid()) {
		return errors.New("更新计划进程无效")
	}
	if _, err := versionParts(p.ReleaseVersion); err != nil {
		return err
	}
	if _, err := versionParts(p.BinaryVersion); err != nil {
		return err
	}
	if err := p.validatePaths(); err != nil {
		return err
	}
	if err := validateRestartArgs(p.RestartArgs); err != nil {
		return err
	}
	if helper {
		self, err := os.Executable()
		if err != nil {
			return err
		}
		if self != filepath.Join(filepath.Dir(p.PlanPath), helperName()) {
			return errors.New("更新计划必须由专用辅助进程执行")
		}
	}
	for path, want := range map[string]string{filepath.Join(filepath.Dir(p.PlanPath), helperName()): p.OriginalSHA256, filepath.Join(p.Target, p.ExecutableRelative): p.OriginalSHA256, filepath.Join(p.Staged, p.ExecutableRelative): p.StagedSHA256} {
		got, err := fileSHA256(path)
		if err != nil {
			return err
		}
		if len(want) != sha256.Size*2 || got != want {
			return errors.New("更新计划程序校验值不符")
		}
	}
	if runtime.GOOS == "darwin" {
		return validateMac(context.Background(), p.Staged)
	}
	return nil
}

func (p *Plan) validatePaths() error {
	if !filepath.IsAbs(p.WorkingDir) || filepath.Clean(p.WorkingDir) != p.WorkingDir {
		return errors.New("更新计划工作目录无效")
	}
	workingInfo, err := os.Stat(p.WorkingDir)
	if err != nil {
		return err
	}
	if !workingInfo.IsDir() {
		return errors.New("更新计划工作目录不存在")
	}
	for _, path := range []string{p.Target, p.Staged, p.Backup} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return errors.New("更新计划路径无效")
		}
	}
	target, relative, err := installationTarget(filepath.Join(p.Target, p.ExecutableRelative), runtime.GOOS)
	if err != nil || target != p.Target || relative != p.ExecutableRelative {
		return errors.New("更新计划安装目录无效")
	}
	root := filepath.Dir(p.Staged)
	if filepath.Dir(root) != filepath.Dir(p.Target) || !strings.HasPrefix(filepath.Base(root), ".lark-studio-update-") || p.Backup != filepath.Join(root, "previous") {
		return errors.New("更新计划暂存目录无效")
	}
	for _, path := range []string{p.Target, root, p.Staged} {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return errors.New("更新计划目录不能是链接")
		}
	}
	if _, err := os.Lstat(p.Backup); !errors.Is(err, os.ErrNotExist) {
		return errors.New("更新恢复目录已存在")
	}
	if p.StatePath != "" {
		if !filepath.IsAbs(p.StatePath) || filepath.Clean(p.StatePath) != p.StatePath || filepath.Base(p.StatePath) != "update.toml" {
			return errors.New("更新状态文件路径无效")
		}
		if err := validateStateLocation(p.Target, p.StatePath); err != nil {
			return err
		}
		info, err := os.Lstat(p.StatePath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil && !info.Mode().IsRegular() {
			return errors.New("更新状态文件不能是链接")
		}
	}
	return nil
}

func validateStateLocation(target, statePath string) error {
	parent := filepath.Dir(statePath)
	resolved, err := filepath.EvalSymlinks(parent)
	if err == nil {
		parent = resolved
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	rel, err := filepath.Rel(target, parent)
	if err != nil {
		return err
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return errors.New("配置目录位于安装目录内，请先将配置移到应用目录外再更新")
	}
	return nil
}

func validateRestartArgs(args []string) error {
	if len(args) > 64 {
		return errors.New("重启参数过多")
	}
	for _, arg := range args {
		if len(arg) > 4096 || strings.ContainsRune(arg, 0) || strings.HasPrefix(arg, "--apply-update") || strings.HasPrefix(arg, "-apply-update") || strings.HasPrefix(arg, "--version") || strings.HasPrefix(arg, "-version") {
			return errors.New("更新计划包含不允许的重启参数")
		}
	}
	return nil
}

func applyPlan(p *Plan, restart func(string, []string) error) error {
	if err := os.Rename(p.Target, p.Backup); err != nil {
		return errors.Join(fmt.Errorf("保留旧应用: %w", err), restart(filepath.Join(p.Target, p.ExecutableRelative), p.RestartArgs))
	}
	if err := os.Rename(p.Staged, p.Target); err != nil {
		rollbackErr := os.Rename(p.Backup, p.Target)
		if rollbackErr == nil {
			rollbackErr = restart(filepath.Join(p.Target, p.ExecutableRelative), p.RestartArgs)
		}
		return errors.Join(fmt.Errorf("替换应用: %w", err), rollbackErr)
	}
	restoreState, err := p.writeReceipt()
	if err == nil {
		err = restart(filepath.Join(p.Target, p.ExecutableRelative), p.RestartArgs)
	}
	if err == nil {
		return nil
	}
	rollbackErr := errors.Join(os.Rename(p.Target, p.Staged), os.Rename(p.Backup, p.Target))
	if restoreState != nil {
		rollbackErr = errors.Join(rollbackErr, restoreState())
	}
	if rollbackErr == nil {
		rollbackErr = restart(filepath.Join(p.Target, p.ExecutableRelative), p.RestartArgs)
	}
	return errors.Join(fmt.Errorf("更新启动失败，已尝试恢复旧应用: %w", err), rollbackErr)
}

func (p *Plan) writeReceipt() (func() error, error) {
	if p.StatePath == "" {
		return nil, nil
	}
	previous, err := os.ReadFile(p.StatePath)
	existed := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	state, err := conf.ReadUpdateState(p.StatePath)
	if err != nil {
		return nil, err
	}
	state.InstalledReleaseVersion = p.ReleaseVersion
	state.InstalledBinarySHA256, err = fileSHA256(filepath.Join(p.Target, p.ExecutableRelative))
	if err != nil {
		return nil, err
	}
	restore := func() error {
		if existed {
			return os.WriteFile(p.StatePath, previous, 0600)
		}
		return os.Remove(p.StatePath)
	}
	if err := conf.WriteConfig(state, p.StatePath); err != nil {
		return nil, err
	}
	return restore, nil
}

func fileSHA256(path string) (string, error) {
	if err := regularExecutable(path); err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func restartApplication(executable string, args []string, workingDir string) error {
	command := exec.Command(executable, args...)
	command.Dir = workingDir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return err
	}
	completed := make(chan error, 1)
	go func() { completed <- command.Wait() }()
	timer := time.NewTimer(restartReadyDelay)
	defer timer.Stop()
	select {
	case err := <-completed:
		if err == nil {
			return errors.New("新应用启动后立即退出")
		}
		return fmt.Errorf("新应用启动后立即退出: %w", err)
	case <-timer.C:
		return nil
	}
}
