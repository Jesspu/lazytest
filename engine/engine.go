package engine

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jesspatton/lazytest/analysis"
	"github.com/jesspatton/lazytest/filesystem"
	"github.com/jesspatton/lazytest/runner"
)

var testFileRegex = regexp.MustCompile(`\.(test|spec)\.[jt]sx?$`)

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

		// Smart queueing: Only queue watched tests that are affected by this change
		// Build a set of queued items for O(1) lookup
		queuedSet := make(map[string]struct{})
		for _, q := range e.State.Queue {
			queuedSet[q] = struct{}{}
		}

		// Find all files affected by this change (transitive dependents)
		dependents := e.Graph.GetDependents(path)

		// Queue watched tests that are in the affected set
		for watchedPath := range e.State.Watched {
			// Check if this watched file is affected
			affected := false
			if watchedPath == path {
				// The watched file itself was changed
				affected = true
			} else {
				// Check if it's in the dependents list
				for _, dep := range dependents {
					if dep == watchedPath {
						affected = true
						break
					}
				}
			}

			// Only queue if affected and not already queued
			if affected {
				if _, alreadyQueued := queuedSet[watchedPath]; !alreadyQueued {
					e.State.Queue = append(e.State.Queue, watchedPath)
					queuedSet[watchedPath] = struct{}{}
				}
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
	dependents := e.Graph.GetDependents(path)
	var tests []string
	for _, dep := range dependents {
		if testFileRegex.MatchString(dep) {
			tests = append(tests, dep)
		}
	}
	return tests
}

func (e *Engine) buildGraph() tea.Msg {
	e.Graph.Build(e.State.RootPath)
	return nil
}
