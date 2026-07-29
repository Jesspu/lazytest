package engine

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jesspatton/lazytest/filesystem"
)

// GetAffectedSuite returns all test paths that have been queued or executed
// during the session, sorted by status priority:
//
//  1. StatusFail
//  2. StatusRunning
//  3. StatusPass
//  4. StatusIdle / not yet run
//
// Within each group paths are sorted alphabetically.
func (e *Engine) GetAffectedSuite() []string {
	result := make([]string, 0, len(e.State.Affected))
	for path := range e.State.Affected {
		result = append(result, path)
	}

	// Priority: Fail=0, Running=1, Pass=2, Idle=3
	priority := func(path string) int {
		switch e.State.NodeStatus[path] {
		case StatusFail:
			return 0
		case StatusRunning:
			return 1
		case StatusPass:
			return 2
		default: // StatusIdle or not set
			return 3
		}
	}

	sort.Slice(result, func(i, j int) bool {
		pi, pj := priority(result[i]), priority(result[j])
		if pi != pj {
			return pi < pj
		}
		return result[i] < result[j]
	})

	return result
}

// GetSuiteStats returns the count of passed, failed, and running tests
// across all paths currently in the Affected suite.
func (e *Engine) GetSuiteStats() (passed, failed, running int) {
	for path := range e.State.Affected {
		switch e.State.NodeStatus[path] {
		case StatusPass:
			passed++
		case StatusFail:
			failed++
		case StatusRunning:
			running++
		}
	}
	return
}

// ClearAffectedSuite removes all passing (StatusPass) and idle/unrun tests
// from State.Affected, keeping only failing and currently running entries.
func (e *Engine) ClearAffectedSuite() {
	for path := range e.State.Affected {
		switch e.State.NodeStatus[path] {
		case StatusFail, StatusRunning:
			// keep
		default:
			delete(e.State.Affected, path)
		}
	}
}

// RunSuiteFailures queues all tests in the Affected suite that are currently
// failing (StatusFail) for re-execution.
func (e *Engine) RunSuiteFailures() tea.Cmd {
	var nodes []*filesystem.Node
	for path := range e.State.Affected {
		if e.State.NodeStatus[path] == StatusFail {
			nodes = append(nodes, filesystem.NodeFromPath(path))
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Path < nodes[j].Path })
	return e.enqueueNodes(nodes)
}

// RunAffectedSuite queues every test currently in the Affected suite for
// re-execution.
func (e *Engine) RunAffectedSuite() tea.Cmd {
	var nodes []*filesystem.Node
	for path := range e.State.Affected {
		nodes = append(nodes, filesystem.NodeFromPath(path))
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Path < nodes[j].Path })
	return e.enqueueNodes(nodes)
}

// enqueueNodes appends nodes to the queue (deduplicating) and triggers the
// first one if the runner is currently idle.
func (e *Engine) enqueueNodes(nodes []*filesystem.Node) tea.Cmd {
	queuedSet := make(map[string]struct{})
	for _, q := range e.State.Queue {
		queuedSet[q] = struct{}{}
	}
	for runningPath := range e.State.RunningNodes {
		queuedSet[runningPath] = struct{}{}
	}

	for _, node := range nodes {
		if _, exists := queuedSet[node.Path]; !exists {
			e.State.Queue = append(e.State.Queue, node.Path)
			queuedSet[node.Path] = struct{}{}
		}
	}

	return e.ProcessQueue()
}
