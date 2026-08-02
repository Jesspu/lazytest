package engine

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jesspatton/lazytest/filesystem"
	"github.com/jesspatton/lazytest/runner"
)

// Actions

func (e *Engine) TriggerTest(node *filesystem.Node) tea.Cmd {
	e.State.RunningNodes[node.Path] = node
	e.State.LastRunNode = node
	
	output := fmt.Sprintf("Running %s...\n", node.Name)
	e.State.TestOutputs[node.Path] = []string{output}
	e.State.NodeStatus[node.Path] = StatusRunning
	// Track in affected suite regardless of mode
	e.State.Affected[node.Path] = struct{}{}

	job, err := runner.PrepareJob(node.Path, e.Workspaces)
	if err != nil {
		e.State.TestOutputs[node.Path] = append(e.State.TestOutputs[node.Path], "Error: Could not find package.json\n")
		e.State.NodeStatus[node.Path] = StatusFail
		delete(e.State.RunningNodes, node.Path)
		return nil
	}

	return func() tea.Msg {
		e.runner.Run(job.Command, job.Args, job.Root, node.Path)
		return nil
	}
}

func (e *Engine) ReRunLast() tea.Cmd {
	if e.State.LastRunNode != nil {
		return e.TriggerTest(e.State.LastRunNode)
	}
	return nil
}

// ProcessQueue dequeues tests up to the MaxConcurrentTests limit and triggers them.
func (e *Engine) ProcessQueue() tea.Cmd {
	var cmds []tea.Cmd
	for len(e.State.RunningNodes) < e.ProjectConfig.MaxConcurrentTests && len(e.State.Queue) > 0 {
		nextPath := e.State.Queue[0]
		e.State.Queue = e.State.Queue[1:]
		cmds = append(cmds, e.TriggerTest(filesystem.NodeFromPath(nextPath)))
	}
	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

func (e *Engine) ToggleWatch(path string) {
	// Check if already watched
	if _, exists := e.State.Watched[path]; exists {
		// Remove
		delete(e.State.Watched, path)
	} else {
		// Add
		e.State.Watched[path] = struct{}{}
	}
}

func (e *Engine) ClearWatched() {
	e.State.Watched = make(map[string]struct{})
}

// ToggleSmartMode switches between Smart Mode and Manual Watch Mode.
func (e *Engine) ToggleSmartMode() {
	e.State.SmartMode = !e.State.SmartMode
}

func (e *Engine) FindRelatedTests(path string) []string {
	var tests []string
	seen := make(map[string]bool)

	// 1. Direct inclusion: if the changed path is itself a test file, include it first.
	if filesystem.IsTestFileByPath(path) {
		tests = append(tests, path)
		seen[path] = true
	}

	// 2. Query transitive dependents with mock-aware BFS.
	// GetAffectedDependents already prunes branches where the dependent mocks
	// the intermediate module, so no additional depType check is needed here.
	dependents := e.Graph.GetAffectedDependents(path)
	for _, dep := range dependents {
		if !seen[dep] && filesystem.IsTestFileByPath(dep) {
			tests = append(tests, dep)
			seen[dep] = true
		}
	}

	return tests
}
