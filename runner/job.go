package runner

import (
	"path/filepath"
	"strings"
)

// TestJob represents a test execution job.
type TestJob struct {
	Command string
	Args    []string
	Root    string
}

// PrepareJob encapsulates the logic to prepare a test execution.
// It finds the execution root, resolves the per-package config (using the
// workspace list for monorepo routing), and builds the command.
func PrepareJob(nodePath string, workspaces []Workspace) (*TestJob, error) {
	execRoot, err := GetExecutionRoot(nodePath)
	if err != nil {
		return nil, err
	}

	config := LoadConfigForPath(nodePath, workspaces)
	relToRoot, _ := filepath.Rel(execRoot, nodePath)

	// Normalize path separators for matching
	matchPath := filepath.ToSlash(relToRoot)

	commandTemplate := config.Command
	for _, override := range config.Overrides {
		if matchPattern(override.Pattern, matchPath) {
			commandTemplate = override.Command
			break
		}
	}

	cmd, args := BuildCommandString(commandTemplate, relToRoot)

	return &TestJob{
		Command: cmd,
		Args:    args,
		Root:    execRoot,
	}, nil
}

func matchPattern(pattern, path string) bool {
	// Simple support for recursive directory matching
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "**")
		return strings.HasPrefix(path, prefix)
	}

	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false
	}
	return matched
}
