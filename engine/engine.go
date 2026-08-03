package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jesspatton/lazytest/analysis"
	"github.com/jesspatton/lazytest/filesystem"
	"github.com/jesspatton/lazytest/runner"
)

// Engine manages the application logic and side effects.
type Engine struct {
	State               State
	runner              *runner.Runner
	watcher             *filesystem.Watcher
	Graph               *analysis.Graph
	ProjectConfig       runner.Config
	Workspaces          []runner.Workspace // Nil for single-package repos
	InitialNotification string
}

// New creates a new Engine instance.
func New(rootPath string) *Engine {
	e := &Engine{
		State:         NewState(rootPath),
		runner:        runner.NewRunner(),
		Graph:         analysis.NewGraphWithRoot(rootPath),
		ProjectConfig: runner.LoadConfig(rootPath),
		Workspaces:    runner.DiscoverWorkspaces(rootPath),
	}
	e.State.WelcomeMessage = e.generateWelcome()
	return e
}

// generateWelcome builds the startup banner shown in the output pane.
func (e *Engine) generateWelcome() string {
	var sb strings.Builder
	sb.WriteString("  LazyTest v0.0.6\n\n")

	if e.ProjectConfig.DetectedRunner != "" {
		sb.WriteString(fmt.Sprintf("  ⚡ Auto-detected: %s\n", e.ProjectConfig.DetectedRunner))
	} else {
		sb.WriteString("  📄 Config: .lazytest.json\n")
	}

	sb.WriteString(fmt.Sprintf("  ⌘  Command: %s\n", e.ProjectConfig.Command))
	sb.WriteString(fmt.Sprintf("  ⚙  Concurrency: %d\n", e.ProjectConfig.MaxConcurrentTests))

	if len(e.Workspaces) > 1 {
		sb.WriteString(fmt.Sprintf("  📦 Workspaces: %d packages detected\n", len(e.Workspaces)))
	}

	sb.WriteString("\n  Press ? for help • Press Enter to run a test\n")
	return sb.String()
}

// Init initializes the engine's side effects.
func (e *Engine) Init() tea.Cmd {
	e.State.IsBuildingGraph = true
	cmds := []tea.Cmd{
		e.RefreshTree,
		e.startWatcher,
		e.buildGraph,
		e.waitForUpdates,
	}

	if e.InitialNotification != "" {
		msg := e.InitialNotification
		cmds = append(cmds, func() tea.Msg {
			return NotificationMsg{
				Message: msg,
				IsError: true,
			}
		})
	}

	return tea.Batch(cmds...)
}

// Update handles incoming messages and updates the engine state.
func (e *Engine) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case WatcherReadyMsg:
		return e.handleWatcherReady(msg)

	case WatcherMsg:
		return e.handleWatcherMsg(string(msg))

	case WatcherErrorMsg:
		return e.handleWatcherError(msg)

	case TreeLoadedMsg:
		return e.handleTreeLoaded(msg)

	case runner.OutputUpdate:
		return e.handleOutputUpdate(msg)

	case runner.StatusUpdate:
		return e.handleStatusUpdate(msg)

	case GraphBuildCompleteMsg:
		return e.handleGraphBuildComplete(msg)

	case GraphUpdateCompleteMsg:
		return e.handleGraphUpdateComplete(msg)
	}

	return nil
}

func (e *Engine) handleWatcherMsg(path string) tea.Cmd {
	if filesystem.IsConfigFile(path) {
		return e.handleConfigChange(path)
	}
	return e.handleSourceChange(path)
}

func (e *Engine) handleWatcherError(msg WatcherErrorMsg) tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return NotificationMsg{
				Message: fmt.Sprintf("Watcher error: %v", msg.Err),
				IsError: true,
			}
		},
		e.waitForWatcherEvents,
	)
}

func (e *Engine) handleConfigChange(path string) tea.Cmd {
	// 1. Reload runner configuration and workspace list
	e.ProjectConfig = runner.LoadConfig(e.State.RootPath)
	e.Workspaces = runner.DiscoverWorkspaces(e.State.RootPath)

	// 2. Rebuild graph asynchronously
	e.Graph = analysis.NewGraphWithRoot(e.State.RootPath)
	e.State.IsBuildingGraph = true

	return tea.Batch(
		e.RefreshTree,
		e.waitForWatcherEvents,
		func() tea.Msg {
			e.Graph.Build(e.State.RootPath)
			return GraphBuildCompleteMsg{ConfigPath: path}
		},
	)
}

func (e *Engine) handleSourceChange(path string) tea.Cmd {
	e.State.IsBuildingGraph = true

	if e.State.Tree != nil {
		if _, err := os.Stat(path); err == nil {
			if filesystem.IsTestFileByPath(path) {
				e.State.Tree.AddNode(path)
			}
		} else if os.IsNotExist(err) {
			if filesystem.IsTestFileByPath(path) {
				e.State.Tree.RemoveNode(path)
			}
		}
	}

	return tea.Batch(
		e.waitForWatcherEvents,
		func() tea.Msg {
			// Update dependency graph
			e.Graph.Update(path)
			return GraphUpdateCompleteMsg{SourcePath: path}
		},
	)
}

func (e *Engine) handleGraphBuildComplete(msg GraphBuildCompleteMsg) tea.Cmd {
	e.State.IsBuildingGraph = false
	if msg.ConfigPath == "" {
		return nil // Just initialization
	}

	// 3. Queue all watched tests (or all tests in Smart Mode) for re-execution
	var testsToQueue []string
	if e.State.SmartMode {
		for p := range e.Graph.Forward {
			if filesystem.IsTestFileByPath(p) {
				testsToQueue = append(testsToQueue, p)
			}
		}
	} else {
		for watchedPath := range e.State.Watched {
			testsToQueue = append(testsToQueue, watchedPath)
		}
	}
	sort.Strings(testsToQueue)

	msgStr := fmt.Sprintf("\nConfig change detected (%s). Reloaded settings and re-queued tests.\n", filepath.Base(msg.ConfigPath))
	for nodePath := range e.State.RunningNodes {
		e.State.TestOutputs[nodePath] = append(e.State.TestOutputs[nodePath], msgStr)
	}

	var nodes []*filesystem.Node
	for _, testPath := range testsToQueue {
		// Always record in Affected even if already queued
		e.State.Affected[testPath] = struct{}{}
		nodes = append(nodes, filesystem.NodeFromPath(testPath))
	}
	e.UpdateSortedAffected()

	return e.enqueueNodes(nodes)
}

func (e *Engine) handleGraphUpdateComplete(msg GraphUpdateCompleteMsg) tea.Cmd {
	e.State.IsBuildingGraph = false
	path := msg.SourcePath

	var testsToQueue []string
	if e.State.SmartMode {
		// Smart Mode: automatically queue every test transitively affected by this path
		testsToQueue = e.FindRelatedTests(path)
	} else {
		// Manual Mode: only queue watched tests that are in the affected set
		dependents := e.Graph.GetAffectedDependents(path)
		for watchedPath := range e.State.Watched {
			affected := watchedPath == path
			if !affected {
				_, affected = dependents[watchedPath]
			}
			if affected {
				testsToQueue = append(testsToQueue, watchedPath)
			}
		}
	}

	var nodes []*filesystem.Node
	for _, testPath := range testsToQueue {
		// Always record in Affected even if already queued
		e.State.Affected[testPath] = struct{}{}
		nodes = append(nodes, filesystem.NodeFromPath(testPath))
	}
	e.UpdateSortedAffected()

	return e.enqueueNodes(nodes)
}

func (e *Engine) handleOutputUpdate(msg runner.OutputUpdate) tea.Cmd {
	e.State.TestOutputs[msg.FilePath] = append(e.State.TestOutputs[msg.FilePath], msg.Content+"\n")
	return e.waitForUpdates
}

func (e *Engine) handleStatusUpdate(msg runner.StatusUpdate) tea.Cmd {
	if _, exists := e.State.RunningNodes[msg.FilePath]; exists {
		if msg.Err == nil {
			e.State.NodeStatus[msg.FilePath] = StatusPass
			e.State.TestOutputs[msg.FilePath] = append(e.State.TestOutputs[msg.FilePath], "\nPASS\n")
		} else {
			e.State.NodeStatus[msg.FilePath] = StatusFail
			e.State.TestOutputs[msg.FilePath] = append(e.State.TestOutputs[msg.FilePath], fmt.Sprintf("\nFAIL: %v\n", msg.Err))
		}
		delete(e.State.RunningNodes, msg.FilePath)
	}

	e.UpdateSortedAffected()

	// Process queue
	cmd := e.ProcessQueue()
	if cmd != nil {
		return tea.Batch(e.waitForUpdates, cmd)
	}
	return e.waitForUpdates
}

func (e *Engine) handleWatcherReady(msg WatcherReadyMsg) tea.Cmd {
	e.watcher = msg.watcher
	return e.waitForWatcherEvents
}

func (e *Engine) handleTreeLoaded(msg TreeLoadedMsg) tea.Cmd {
	e.State.Tree = msg
	return nil
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
		return NotificationMsg{
			Message: fmt.Sprintf("Failed to start file watcher: %v", err),
			IsError: true,
		}
	}
	return WatcherReadyMsg{watcher: w}
}

func (e *Engine) waitForWatcherEvents() tea.Msg {
	if e.watcher == nil {
		return nil
	}
	select {
	case eventPath, ok := <-e.watcher.Events:
		if !ok {
			return nil
		}
		return WatcherMsg(eventPath)
	case err, ok := <-e.watcher.Errors:
		if !ok {
			return nil
		}
		return WatcherErrorMsg{Err: err}
	}
}

func (e *Engine) waitForUpdates() tea.Msg {
	update, ok := <-e.runner.Updates
	if !ok {
		return nil
	}
	return update
}

func (e *Engine) buildGraph() tea.Msg {
	e.Graph.Build(e.State.RootPath)
	return GraphBuildCompleteMsg{ConfigPath: ""}
}

// Close stops background routines like the file watcher and running processes.
func (e *Engine) Close() {
	if e.watcher != nil {
		e.watcher.Close()
	}
	if e.runner != nil {
		e.runner.KillAll()
	}
}

