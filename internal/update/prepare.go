package update

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Prepare 只下载并暂存应用，运行中的安装目录在辅助进程接管前保持原样。
func (c *Client) Prepare(ctx context.Context, release *Release, executable string) (plan *Plan, err error) {
	if !release.Supported() {
		return nil, errors.New(release.Reason())
	}
	if release.goos != runtime.GOOS || release.goarch != runtime.GOARCH {
		return nil, errors.New("安装包平台与当前系统不符")
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return nil, err
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return nil, err
	}
	target, relative, err := installationTarget(executable, runtime.GOOS)
	if err != nil {
		return nil, err
	}
	stageRoot, err := os.MkdirTemp(filepath.Dir(target), ".lark-studio-update-")
	if err != nil {
		return nil, fmt.Errorf("安装目录不可写: %w", err)
	}
	privateDir := ""
	defer func() {
		if err != nil {
			err = errors.Join(err, os.RemoveAll(stageRoot))
			if privateDir != "" {
				err = errors.Join(err, os.RemoveAll(privateDir))
			}
		}
	}()
	archive := filepath.Join(stageRoot, release.asset.Name)
	c.progress("downloading", "正在下载更新安装包", 0)
	if err = c.download(ctx, release.asset, archive); err != nil {
		return nil, err
	}
	c.progress("preparing", "正在校验并准备更新", 99)
	staged, err := stageApplication(ctx, archive, stageRoot, runtime.GOOS)
	if err != nil {
		return nil, err
	}
	binaryVersion, err := inspectVersion(ctx, filepath.Join(staged, relative))
	if err != nil {
		return nil, err
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	plan = &Plan{Target: target, Staged: staged, Backup: filepath.Join(stageRoot, "previous"), ExecutableRelative: relative, ParentPID: os.Getpid(), ReleaseVersion: release.Version, BinaryVersion: binaryVersion, RestartArgs: append([]string(nil), os.Args[1:]...), WorkingDir: workingDir}
	if err = plan.persist(executable); err != nil {
		return nil, err
	}
	privateDir = filepath.Dir(plan.PlanPath)
	if err = os.Remove(archive); err != nil {
		return nil, err
	}
	return plan, nil
}

func installationTarget(executable, goos string) (string, string, error) {
	switch goos {
	case "darwin":
		if filepath.Base(executable) != "lark-studio" || filepath.Base(filepath.Dir(executable)) != "MacOS" || filepath.Base(filepath.Dir(filepath.Dir(executable))) != "Contents" {
			return "", "", errors.New("自动更新仅支持安装后的 macOS 应用")
		}
		target := filepath.Dir(filepath.Dir(filepath.Dir(executable)))
		if !strings.HasSuffix(target, ".app") {
			return "", "", errors.New("程序不在 macOS 应用包内")
		}
		return target, filepath.Join("Contents", "MacOS", "lark-studio"), nil
	case "windows":
		if !strings.EqualFold(filepath.Base(executable), "lark-studio.exe") {
			return "", "", errors.New("自动更新仅支持已解压的 Windows 应用")
		}
		target := filepath.Dir(executable)
		if !strings.EqualFold(filepath.Base(target), "lark-studio") {
			return "", "", errors.New("请先将 Windows 安装包解压到独立的 lark-studio 文件夹，再执行自动更新")
		}
		if target == filepath.VolumeName(target)+string(filepath.Separator) {
			return "", "", errors.New("不能更新磁盘根目录中的程序")
		}
		return target, "lark-studio.exe", nil
	default:
		return "", "", errors.New("当前系统不支持自动安装")
	}
}

func stageApplication(ctx context.Context, archive, root, goos string) (string, error) {
	if goos == "windows" {
		if err := extractZip(ctx, archive, root); err != nil {
			return "", err
		}
		app := filepath.Join(root, "lark-studio")
		if err := regularExecutable(filepath.Join(app, "lark-studio.exe")); err != nil {
			return "", err
		}
		if err := regularExecutable(filepath.Join(app, "ffmpeg.exe")); err != nil {
			return "", err
		}
		return app, nil
	}
	return stageMac(ctx, archive, root)
}

func stageMac(ctx context.Context, archive, root string) (app string, err error) {
	mount := filepath.Join(root, "mount")
	if err = os.Mkdir(mount, 0700); err != nil {
		return "", err
	}
	if err = runCommand(ctx, "/usr/bin/hdiutil", "attach", "-readonly", "-nobrowse", "-noautoopen", "-mountpoint", mount, archive); err != nil {
		return "", err
	}
	defer func() {
		detachCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err = errors.Join(err, runCommand(detachCtx, "/usr/bin/hdiutil", "detach", mount))
	}()
	entries, err := os.ReadDir(mount)
	if err != nil {
		return "", err
	}
	source := ""
	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".app") {
			if source != "" {
				return "", errors.New("安装包包含多个应用")
			}
			source = filepath.Join(mount, entry.Name())
		}
	}
	if source == "" {
		return "", errors.New("安装包没有 macOS 应用")
	}
	if err = validateMac(ctx, source); err != nil {
		return "", err
	}
	app = filepath.Join(root, "application.app")
	if err = runCommand(ctx, "/usr/bin/ditto", source, app); err != nil {
		return "", err
	}
	return app, validateMac(ctx, app)
}

func validateMac(ctx context.Context, app string) error {
	if err := regularExecutable(filepath.Join(app, "Contents", "MacOS", "lark-studio")); err != nil {
		return err
	}
	if err := filepath.WalkDir(app, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(app, resolved)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return errors.New("应用包含指向包外的符号链接")
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return runCommand(ctx, "/usr/bin/codesign", "--verify", "--deep", "--strict", app)
}

func regularExecutable(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return errors.New("安装包缺少有效的可执行文件")
	}
	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		return errors.New("安装包程序没有执行权限")
	}
	return nil
}

type boundedOutput struct {
	bytes.Buffer
	limit int
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len() < b.limit {
		if _, err := b.Buffer.Write(p[:min(len(p), b.limit-b.Len())]); err != nil {
			return 0, err
		}
	}
	return n, nil
}

func runCommand(ctx context.Context, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	output := &boundedOutput{limit: 4096}
	command.Stdout = output
	command.Stderr = output
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s 失败: %w (%s)", filepath.Base(name), err, strings.TrimSpace(output.String()))
	}
	return nil
}

func inspectVersion(ctx context.Context, executable string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "--version")
	output := &boundedOutput{limit: 4096}
	command.Stdout = output
	command.Stderr = output
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("检查更新程序版本: %w (%s)", err, output.String())
	}
	return parseVersionOutput(output.String())
}

func parseVersionOutput(output string) (string, error) {
	fields := strings.Fields(output)
	if len(fields) < 2 || (fields[0] != "lark-studio" && fields[0] != "vdub") {
		return "", errors.New("更新程序未返回有效版本信息")
	}
	if _, err := versionParts(fields[1]); err != nil {
		return "", err
	}
	return fields[1], nil
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		return err
	}
	copyErr := error(nil)
	if _, err := io.Copy(output, input); err != nil {
		copyErr = err
	}
	return errors.Join(copyErr, output.Sync(), output.Close())
}
