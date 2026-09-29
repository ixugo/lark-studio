package update

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
)

func detach(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008 | 0x00000200}
}

func waitParent(ctx context.Context, pid int) (err error) {
	const synchronize = 0x00100000
	handle, err := syscall.OpenProcess(synchronize, false, uint32(pid))
	if errors.Is(err, syscall.Errno(87)) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, syscall.CloseHandle(handle)) }()
	for {
		result, err := syscall.WaitForSingleObject(handle, 200)
		if err != nil {
			return err
		}
		if result == syscall.WAIT_OBJECT_0 {
			return nil
		}
		if result != syscall.WAIT_TIMEOUT {
			return errors.New("等待旧程序退出失败")
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
}
