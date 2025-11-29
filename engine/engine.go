package engine

import (
	"fmt"
	"os"
	"strings"

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
	State   State
	runner  *runner.Runner
	watcher *filesystem.Watcher
	Graph   *analysis.Graph
}

// New creates a new Engine instance.
func New(rootPath string) *Engine {
	return &Engine{
		State:  NewState(rootPath),
		runner: runner.NewRunner(),
		Graph:  analysis.NewGraph(),
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

		// Update dependency graph
		e.Graph.Update(path)

		// Queue watched files
		for _, watchedPath := range e.State.Watched {
			// If the changed file is a watched file, or if it's a dependency of a watched file?
			// The current logic just re-queues ALL watched files on ANY change.
			// This is inefficient but safe.
			// Ideally we only queue if 'path' affects 'watchedPath'.
			// But for now let's keep the existing logic of queuing all watched files,
			// but we MUST ensure the graph is updated first (done above).

			// Check if already in queue
			found := false
			for _, q := range e.State.Queue {
				if q == watchedPath {
					found = true
					break
				}
			}
			if !found {
				e.State.Queue = append(e.State.Queue, watchedPath)
			}
		}

		var cmd tea.Cmd
		// Trigger if idle
		if e.State.RunningNode == nil && len(e.State.Queue) > 0 {
			nextPath := e.State.Queue[0]
			e.State.Queue = e.State.Queue[1:]
			node := &filesystem.Node{
				Path: nextPath,
				Name: nextPath[strings.LastIndex(nextPath, string(os.PathSeparator))+1:],
			}
			cmd = e.TriggerTest(node)
		}

		return tea.Batch(e.RefreshTree, cmd, e.waitForWatcherEvents)

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
			node := &filesystem.Node{
				Path: nextPath,
				Name: nextPath[strings.LastIndex(nextPath, string(os.PathSeparator))+1:],
			}
			return tea.Batch(e.waitForUpdates, e.TriggerTest(node))
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
	for i, p := range e.State.Watched {
		if p == path {
			// Remove
			e.State.Watched = append(e.State.Watched[:i], e.State.Watched[i+1:]...)
			return
		}
	}
	// Add
	e.State.Watched = append(e.State.Watched, path)
}

func (e *Engine) ClearWatched() {
	e.State.Watched = []string{}
}

// Internal Commands

func (e *Engine) RefreshTree() tea.Msg {
	tree, err := filesystem.Walk(e.State.RootPath)
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
	return e.State.Watched
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
	for _, p := range e.State.Watched {
		if p == path {
			return true
		}
	}
	return false
}

func (e *Engine) FindRelatedTests(path string) []string {
	dependents := e.Graph.GetDependents(path)
	var tests []string
	for _, dep := range dependents {
		if strings.HasSuffix(dep, ".test.ts") || strings.HasSuffix(dep, ".test.js") ||
			strings.HasSuffix(dep, ".spec.ts") || strings.HasSuffix(dep, ".spec.js") ||
			strings.HasSuffix(dep, ".test.tsx") || strings.HasSuffix(dep, ".test.jsx") ||
			strings.HasSuffix(dep, ".spec.tsx") || strings.HasSuffix(dep, ".spec.jsx") {
			tests = append(tests, dep)
		}
	}
	return tests
}

func (e *Engine) buildGraph() tea.Msg {
	e.Graph.Build(e.State.RootPath)
	return nil
}
