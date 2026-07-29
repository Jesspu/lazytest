package engine

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jesspatton/lazytest/filesystem"
	"github.com/jesspatton/lazytest/runner"
)

// Actions

func (e *Engine) TriggerTest(node *filesystem.Node) tea.Cmd {
	e.State.RunningNode = node
	e.State.LastRunNode = node
	e.State.CurrentOutput = fmt.Sprintf("Running %s...\n", node.Name)
	e.State.TestOutputs[node.Path] = e.State.CurrentOutput
	e.State.NodeStatus[node.Path] = StatusRunning
	// Track in affected suite regardless of mode
	e.State.Affected[node.Path] = struct{}{}

	job, err := runner.PrepareJob(node.Path)
	if err != nil {
		e.State.CurrentOutput += "Error: Could not find package.json\n"
		e.State.NodeStatus[node.Path] = StatusFail
		return nil
	}

	e.State.TestOutputs[node.Path] = e.State.CurrentOutput

	return func() tea.Msg {
		e.runner.Run(job.Command, job.Args, job.Root)
		return nil
	}
}

func (e *Engine) ReRunLast() tea.Cmd {
	if e.State.LastRunNode != nil {
		return e.TriggerTest(e.State.LastRunNode)
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
	if filesystem.IsTestFile(path) {
		tests = append(tests, path)
		seen[path] = true
	}

	// 2. Query transitive dependents with mock-aware BFS.
	// GetAffectedDependents already prunes branches where the dependent mocks
	// the intermediate module, so no additional depType check is needed here.
	dependents := e.Graph.GetAffectedDependents(path)
	for _, dep := range dependents {
		if !seen[dep] && filesystem.IsTestFile(dep) {
			tests = append(tests, dep)
			seen[dep] = true
		}
	}

	return tests
}
