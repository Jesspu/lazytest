package runner

import (
	"path/filepath"
)

type TestJob struct {
	Command string
	Args    []string
	Root    string
}

// PrepareJob encapsulates the logic to prepare a test execution.
// It finds the execution root, loads the config, and builds the command.
func PrepareJob(nodePath string) (*TestJob, error) {
	execRoot, err := GetExecutionRoot(nodePath)
	if err != nil {
		return nil, err
	}

	config := LoadConfig(execRoot)
	relToRoot, _ := filepath.Rel(execRoot, nodePath)
	cmd, args := BuildCommandString(config.Command, relToRoot)

	return &TestJob{
		Command: cmd,
		Args:    args,
		Root:    execRoot,
	}, nil
}
