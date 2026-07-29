package engine

import (
	"sort"

	"github.com/jesspatton/lazytest/filesystem"
)

// Accessors

func (e *Engine) GetWatchedFiles() []string {
	// Convert map to slice and sort for consistent ordering
	// (maps have non-deterministic iteration order in Go)
	result := make([]string, 0, len(e.State.Watched))
	for path := range e.State.Watched {
		result = append(result, path)
	}
	// Sort alphabetically for stable UI rendering
	sort.Strings(result)
	return result
}

func (e *Engine) GetTestOutput(path string) (string, bool) {
	val, ok := e.State.TestOutputs[path]
	return val, ok
}

func (e *Engine) GetNodeStatus(path string) (TestStatus, bool) {
	val, ok := e.State.NodeStatus[path]
	return val, ok
}

func (e *Engine) GetTree() *filesystem.Node {
	return e.State.Tree
}

func (e *Engine) GetRunningNode() *filesystem.Node {
	return e.State.RunningNode
}

func (e *Engine) GetCurrentOutput() string {
	return e.State.CurrentOutput
}

func (e *Engine) IsWatched(path string) bool {
	_, exists := e.State.Watched[path]
	return exists
}

// IsSmartMode returns whether Smart Mode is currently active.
func (e *Engine) IsSmartMode() bool {
	return e.State.SmartMode
}
