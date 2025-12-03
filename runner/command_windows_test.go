//go:build windows

package runner

import (
	"os/exec"
	"testing"
)

func TestPrepareCommand_Windows(t *testing.T) {
	cmd := exec.Command("echo", "hello")
	// Should not panic or error
	prepareCommand(cmd)

	// On Windows, we expect no specific SysProcAttr changes for now
}
