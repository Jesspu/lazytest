package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jesspatton/lazytest/filesystem"
	"github.com/jesspatton/lazytest/runner"
)

type Pane int

const (
	PaneExplorer Pane = iota
	PaneOutput
)

type TestStatus int

const (
	StatusIdle TestStatus = iota
	StatusRunning
	StatusPass
	StatusFail
)

type Model struct {
	activePane Pane
	width      int
	height     int
	ready      bool
	showHelp   bool
	
	keys       KeyMap
	help       help.Model
	
	// File System
	rootPath   string
	fileTree   *filesystem.Node
	flatNodes  []*filesystem.Node // Flattened list for navigation
	cursor     int
	watcher    *filesystem.Watcher

	// Runner
	testRunner      *runner.Runner
	output          string
	viewport        viewport.Model
	runningNodePath string
	lastRunNode     *filesystem.Node
	
	// State
	nodeStatus map[string]TestStatus
}

// Messages
type WatcherMsg struct{}
type OutputMsg string
type TestResultMsg struct{ Err error }
type TreeLoadedMsg *filesystem.Node

func NewModel() Model {
	cwd, _ := os.Getwd()
	h := help.New()
	h.ShowAll = true
	return Model{
		activePane: PaneExplorer,
		rootPath:   cwd,
		testRunner: runner.NewRunner(),
		nodeStatus: make(map[string]TestStatus),
		keys:       NewKeyMap(),
		help:       h,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshTree,
		m.startWatcher,
		m.waitForOutput,
		m.waitForTestResult,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.watcher != nil {
				m.watcher.Close()
			}
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, m.keys.Tab):
			if m.activePane == PaneExplorer {
				m.activePane = PaneOutput
			} else {
				m.activePane = PaneExplorer
			}
		case key.Matches(msg, m.keys.Refresh):
			return m, m.refreshTree
		case key.Matches(msg, m.keys.ReRunLast):
			if m.lastRunNode != nil {
				return m, m.triggerTest(m.lastRunNode)
			}
		}

		// Handle pane-specific keys
		if m.activePane == PaneExplorer {
			switch {
			case key.Matches(msg, m.keys.Up):
				if m.cursor > 0 {
					m.cursor--
				}
			case key.Matches(msg, m.keys.Down):
				if m.cursor < len(m.flatNodes)-1 {
					m.cursor++
				}
			case key.Matches(msg, m.keys.Enter):
				if m.cursor < len(m.flatNodes) {
					node := m.flatNodes[m.cursor]
					if !node.IsDir {
						return m, m.triggerTest(node)
					}
				}
			}
		} else {
			// Forward keys to viewport
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		
		// Calculate available space
		// Width: (Total / 2) - Border(2) - Padding(2) = Total/2 - 4
		paneWidth := (m.width / 2) - 4
		// Height: Total - Footer(1) - Border(2) - Padding(0) = Total - 3
		// Let's reserve 2 extra lines for safety/margins
		paneHeight := m.height - 5

		// Viewport Height: PaneHeight - Header("OUTPUT\n\n")
		// Header takes 2 lines (Title + Empty line)
		viewportHeight := paneHeight - 2

		if !m.ready {
			m.viewport = viewport.New(paneWidth, viewportHeight)
			m.viewport.SetContent(m.output)
			m.ready = true
		} else {
			m.viewport.Width = paneWidth
			m.viewport.Height = viewportHeight
		}

	case WatcherMsg:
		return m, m.refreshTree

	case TreeLoadedMsg:
		m.fileTree = msg
		m.flattenNodes()
		return m, nil

	case OutputMsg:
		m.output += string(msg) + "\n"
		m.viewport.SetContent(m.output)
		m.viewport.GotoBottom()
		return m, m.waitForOutput

	case TestResultMsg:
		if m.runningNodePath != "" {
			if msg.Err == nil {
				m.nodeStatus[m.runningNodePath] = StatusPass
				m.output += "\nPASS\n"
			} else {
				m.nodeStatus[m.runningNodePath] = StatusFail
				m.output += fmt.Sprintf("\nFAIL: %v\n", msg.Err)
			}
			m.viewport.SetContent(m.output)
			m.viewport.GotoBottom()
			m.runningNodePath = ""
		}
		return m, m.waitForTestResult
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.showHelp {
		return m.helpView()
	}

	if m.width == 0 {
		return "Loading..."
	}

	paneWidth := (m.width / 2) - 2
	paneHeight := m.height - 4

	// Explorer View
	var explorerView strings.Builder
	explorerView.WriteString(titleStyle.Render("TEST EXPLORER") + "\n\n")
	
	if m.fileTree == nil {
		explorerView.WriteString("Scanning...")
	} else {
		// Render flattened list
		// We need to limit the view to the height (scrolling).
		// For MVP, just slice around cursor or show all if fits.
		// Let's implement basic scrolling window.
		start := 0
		end := len(m.flatNodes)
		
		if len(m.flatNodes) > paneHeight {
			if m.cursor < paneHeight/2 {
				start = 0
				end = paneHeight
			} else if m.cursor > len(m.flatNodes)-paneHeight/2 {
				start = len(m.flatNodes) - paneHeight
				end = len(m.flatNodes)
			} else {
				start = m.cursor - paneHeight/2
				end = m.cursor + paneHeight/2
			}
		}

		for i := start; i < end; i++ {
			if i >= len(m.flatNodes) {
				break
			}
			node := m.flatNodes[i]
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			
			// Indentation
			depth := strings.Count(node.Path, string(os.PathSeparator)) - strings.Count(m.rootPath, string(os.PathSeparator))
			indent := strings.Repeat("  ", depth)
			
			icon := "•"
			if node.IsDir {
				icon = "📁"
			} else {
				status, ok := m.nodeStatus[node.Path]
				if ok {
					switch status {
					case StatusRunning:
						icon = "⏳"
					case StatusPass:
						icon = "✅"
					case StatusFail:
						icon = "❌"
					default:
						icon = "📄"
					}
				} else {
					icon = "📄"
				}
			}

			line := fmt.Sprintf("%s %s%s %s", cursor, indent, icon, node.Name)
			
			if m.cursor == i {
				explorerView.WriteString(lipgloss.NewStyle().Foreground(highlight).Render(line) + "\n")
			} else {
				explorerView.WriteString(line + "\n")
			}
		}
	}

	explorerStyle := paneStyle
	if m.activePane == PaneExplorer {
		explorerStyle = activePaneStyle
	}
	explorerRender := explorerStyle.
		Width(paneWidth).
		Height(paneHeight).
		Render(explorerView.String())

	// Output View
	var outputView strings.Builder
	outputView.WriteString(titleStyle.Render("OUTPUT") + "\n\n")
	
	if !m.ready {
		outputView.WriteString("Initializing...")
	} else {
		outputView.WriteString(m.viewport.View())
	}

	outputStyle := paneStyle
	if m.activePane == PaneOutput {
		outputStyle = activePaneStyle
	}
	outputRender := outputStyle.
		Width(paneWidth).
		Height(paneHeight).
		Render(outputView.String())

	panes := lipgloss.JoinHorizontal(lipgloss.Top, explorerRender, outputRender)
	footer := statusStyle.Render("? Help  q Quit  Tab Switch Pane  Enter Run  R Refresh")

	return lipgloss.JoinVertical(lipgloss.Left, panes, footer)
}

// Commands

func (m *Model) refreshTree() tea.Msg {
	tree, err := filesystem.Walk(m.rootPath)
	if err != nil {
		return nil // Handle error
	}
	return TreeLoadedMsg(tree)
}

func (m *Model) startWatcher() tea.Msg {
	w, err := filesystem.NewWatcher(m.rootPath)
	if err != nil {
		return nil
	}
	m.watcher = w
	
	// Listen for events
	go func() {
		for range w.Events {
			// We can't send directly to program update from here easily without the program reference
			// But we can't access program here.
			// Wait, the standard way is to have a Cmd that waits on the channel.
		}
	}()
	
	return m.waitForWatcherEvents()
}

func (m Model) waitForWatcherEvents() tea.Msg {
	if m.watcher == nil {
		return nil
	}
	_, ok := <-m.watcher.Events
	if !ok {
		return nil
	}
	return WatcherMsg{}
}

func (m Model) waitForOutput() tea.Msg {
	line, ok := <-m.testRunner.Output
	if !ok {
		return nil
	}
	return OutputMsg(line)
}

func (m Model) waitForTestResult() tea.Msg {
	err, ok := <-m.testRunner.Status
	if !ok {
		return nil
	}
	return TestResultMsg{Err: err}
}

func (m *Model) triggerTest(node *filesystem.Node) tea.Cmd {
	m.lastRunNode = node
	m.output = fmt.Sprintf("Running %s...\n", node.Name)
	m.viewport.SetContent(m.output)
	m.viewport.GotoBottom()
	
	m.runningNodePath = node.Path
	m.nodeStatus[node.Path] = StatusRunning

	// Prepare test job
	job, err := runner.PrepareJob(node.Path)
	if err != nil {
		m.output += "Error: Could not find package.json\n"
		m.viewport.SetContent(m.output)
		m.nodeStatus[node.Path] = StatusFail
		return nil
	}
	
	return func() tea.Msg {
		m.testRunner.Run(job.Command, job.Args, job.Root)
		return nil
	}
}

// Helpers

func (m *Model) flattenNodes() {
	m.flatNodes = []*filesystem.Node{}
	if m.fileTree == nil {
		return
	}
	// Depth-first traversal
	var traverse func(*filesystem.Node)
	traverse = func(n *filesystem.Node) {
		// Don't add root itself if it's just "."
		if n != m.fileTree {
			m.flatNodes = append(m.flatNodes, n)
		}
		for _, child := range n.Children {
			traverse(child)
		}
	}
	traverse(m.fileTree)
}

func (m Model) helpView() string {
	title := titleStyle.Render("HELP")
	helpView := m.help.View(m.keys)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		paneStyle.Render(fmt.Sprintf("%s\n\n%s", title, helpView)),
	)
}
