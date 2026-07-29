package engine

import (
	"fmt"
	"path/filepath"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jesspatton/lazytest/analysis"
	"github.com/jesspatton/lazytest/filesystem"
	"github.com/jesspatton/lazytest/runner"
)

// Messages

// WatcherMsg indicates a file system event occurred.
type WatcherMsg string

// TreeLoadedMsg carries the new file tree after a refresh.
type TreeLoadedMsg *filesystem.Node

// WatcherReadyMsg carries the initialized watcher.
type WatcherReadyMsg struct {
	watcher *filesystem.Watcher
}

// Engine manages the application logic and side effects.
type Engine struct {
	State         State
	runner        *runner.Runner
	watcher       *filesystem.Watcher
	Graph         *analysis.Graph
	ProjectConfig runner.Config
}

// New creates a new Engine instance.
func New(rootPath string) *Engine {
	return &Engine{
		State:         NewState(rootPath),
		runner:        runner.NewRunner(),
		Graph:         analysis.NewGraphWithRoot(rootPath),
		ProjectConfig: runner.LoadConfig(rootPath),
	}
}

// Init initializes the engine's side effects.
func (e *Engine) Init() tea.Cmd {
	return tea.Batch(
		e.RefreshTree,
		e.startWatcher,
		e.buildGraph,
		e.waitForUpdates,
	)
}

// Update handles incoming messages and updates the engine state.
func (e *Engine) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case WatcherReadyMsg:
		e.watcher = msg.watcher
		return e.waitForWatcherEvents

	case WatcherMsg:
		path := string(msg)

		var testsToQueue []string

		if filesystem.IsConfigFile(path) {
			// 1. Reload runner configuration
			e.ProjectConfig = runner.LoadConfig(e.State.RootPath)

			// 2. Rebuild graph as paths/aliases might have changed (e.g. tsconfig.json)
			e.Graph = analysis.NewGraphWithRoot(e.State.RootPath)
			_ = e.Graph.Build(e.State.RootPath)

			// 3. Queue all watched tests (or all tests in Smart Mode) for re-execution
			if e.State.SmartMode {
				for p := range e.Graph.Forward {
					if filesystem.IsTestFile(p) {
						testsToQueue = append(testsToQueue, p)
					}
				}
			} else {
				for watchedPath := range e.State.Watched {
					testsToQueue = append(testsToQueue, watchedPath)
				}
			}
			sort.Strings(testsToQueue)

			e.State.CurrentOutput += fmt.Sprintf("\nConfig change detected (%s). Reloaded settings and re-queued tests.\n", filepath.Base(path))
			if e.State.RunningNode != nil {
				e.State.TestOutputs[e.State.RunningNode.Path] = e.State.CurrentOutput
			}
		} else {
			// Update dependency graph
			e.Graph.Update(path)

			if e.State.SmartMode {
				// Smart Mode: automatically queue every test transitively affected by this path
				testsToQueue = e.FindRelatedTests(path)
			} else {
				// Manual Mode: only queue watched tests that are in the affected set
				dependents := e.Graph.GetAffectedDependents(path)
				for watchedPath := range e.State.Watched {
					affected := watchedPath == path
					if !affected {
						for _, dep := range dependents {
							if dep == watchedPath {
								affected = true
								break
							}
						}
					}
					if affected {
						testsToQueue = append(testsToQueue, watchedPath)
					}
				}
			}
		}

		// Enqueue (deduplicated) and track in Affected suite
		var nodes []*filesystem.Node
		for _, testPath := range testsToQueue {
			// Always record in Affected even if already queued
			e.State.Affected[testPath] = struct{}{}
			nodes = append(nodes, filesystem.NodeFromPath(testPath))
		}

		return tea.Batch(e.RefreshTree, e.enqueueNodes(nodes), e.waitForWatcherEvents)

	case TreeLoadedMsg:
		e.State.Tree = msg
		return nil

	case runner.OutputUpdate:
		e.State.CurrentOutput += string(msg) + "\n"
		if e.State.RunningNode != nil {
			e.State.TestOutputs[e.State.RunningNode.Path] = e.State.CurrentOutput
		}
		return e.waitForUpdates

	case runner.StatusUpdate:
		if e.State.RunningNode != nil {
			if msg.Err == nil {
				e.State.NodeStatus[e.State.RunningNode.Path] = StatusPass
				e.State.CurrentOutput += "\nPASS\n"
			} else {
				e.State.NodeStatus[e.State.RunningNode.Path] = StatusFail
				e.State.CurrentOutput += fmt.Sprintf("\nFAIL: %v\n", msg.Err)
			}
			e.State.TestOutputs[e.State.RunningNode.Path] = e.State.CurrentOutput
			e.State.RunningNode = nil
		}

		// Process queue
		if len(e.State.Queue) > 0 {
			nextPath := e.State.Queue[0]
			e.State.Queue = e.State.Queue[1:]
			return tea.Batch(e.waitForUpdates, e.TriggerTest(filesystem.NodeFromPath(nextPath)))
		}

		return e.waitForUpdates
	}

	return nil
}

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

// IsSmartMode returns whether Smart Mode is currently active.
func (e *Engine) IsSmartMode() bool {
	return e.State.SmartMode
}

// Internal Commands

func (e *Engine) RefreshTree() tea.Msg {
	tree, err := filesystem.Walk(e.State.RootPath, e.ProjectConfig.Excludes)
	if err != nil {
		return nil
	}
	return TreeLoadedMsg(tree)
}

func (e *Engine) startWatcher() tea.Msg {
	w, err := filesystem.NewWatcher(e.State.RootPath)
	if err != nil {
		return nil
	}
	return WatcherReadyMsg{watcher: w}
}

func (e *Engine) waitForWatcherEvents() tea.Msg {
	if e.watcher == nil {
		return nil
	}
	eventPath, ok := <-e.watcher.Events
	if !ok {
		return nil
	}
	return WatcherMsg(eventPath)
}

func (e *Engine) waitForUpdates() tea.Msg {
	update, ok := <-e.runner.Updates
	if !ok {
		return nil
	}
	return update
}

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

func (e *Engine) buildGraph() tea.Msg {
	e.Graph.Build(e.State.RootPath)
	return nil
}

// GetAffectedSuite returns all test paths that have been queued or executed
// during the session, sorted by status priority:
//
//	1. StatusFail
//	2. StatusRunning
//	3. StatusPass
//	4. StatusIdle / not yet run
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
	if e.State.RunningNode != nil {
		queuedSet[e.State.RunningNode.Path] = struct{}{}
	}

	for _, node := range nodes {
		if _, exists := queuedSet[node.Path]; !exists {
			e.State.Queue = append(e.State.Queue, node.Path)
			queuedSet[node.Path] = struct{}{}
		}
	}

	if e.State.RunningNode == nil && len(e.State.Queue) > 0 {
		nextPath := e.State.Queue[0]
		e.State.Queue = e.State.Queue[1:]
		return e.TriggerTest(filesystem.NodeFromPath(nextPath))
	}
	return nil
}
