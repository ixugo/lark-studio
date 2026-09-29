//go:build !windows

package update

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"time"
)

func detach(command *exec.Cmd) { command.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }

func waitParent(ctx context.Context, pid int) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
