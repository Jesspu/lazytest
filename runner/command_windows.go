//go:build windows

package runner

import (
	"os/exec"
)

func prepareCommand(cmd *exec.Cmd) {
	// Windows doesn't support Setpgid or syscall.Kill for process groups in the same way.
	// The default behavior of exec.CommandContext will kill the process when the context is cancelled.
}
