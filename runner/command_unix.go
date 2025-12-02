//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

func prepareCommand(cmd *exec.Cmd) {
	// Set process group to ensure we can kill children if needed
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Ensure we kill the whole process group when the context is cancelled
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
