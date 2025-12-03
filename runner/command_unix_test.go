//go:build !windows

package runner

import (
	"os/exec"
	"testing"
)

func TestPrepareCommand_Unix(t *testing.T) {
	cmd := exec.Command("echo", "hello")
	prepareCommand(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr should not be nil")
	}

	if !cmd.SysProcAttr.Setpgid {
		t.Error("Setpgid should be true")
	}

	if cmd.Cancel == nil {
		t.Error("Cancel function should be set")
	}
}
