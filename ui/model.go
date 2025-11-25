package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jesspatton/lazytest/filesystem"
	"github.com/jesspatton/lazytest/runner"
)

// Pane represents a distinct section of the UI.
type Pane int

const (
	// PaneExplorer is the file explorer pane.
	PaneExplorer Pane = iota
	// PaneOutput is the test output pane.
	PaneOutput
)

// LeftTab represents the active tab in the left pane.
type LeftTab int

const (
	// TabExplorer is the file explorer tab.
	TabExplorer LeftTab = iota
	// TabWatched is the watched files tab.
	TabWatched
)

// TestStatus represents the current state of a test file.
type TestStatus int

const (
	// StatusIdle indicates the test is not running.
	StatusIdle TestStatus = iota
	// StatusRunning indicates the test is currently executing.
	StatusRunning
	// StatusPass indicates the last run passed.
	StatusPass
	// StatusFail indicates the last run failed.
	StatusFail
)

// Model represents the application state for the Bubbletea program.
type Model struct {
	// UI State
	activePane Pane
	width      int
	height     int
	ready      bool
	showHelp   bool
	cursor     int
	viewport   viewport.Model

	// Tab State
	activeTab     LeftTab
	watchedFiles  []string
	watchedCursor int

	// Search State
	searchMode        bool
	searchFocus       bool
	searchInput       textinput.Model
	searchMatches     []int
	currentMatchIndex int

	// Components
	keys KeyMap
	help help.Model

	// Data / Dependencies
	rootPath   string
	fileTree   *filesystem.Node
	flatNodes  []*filesystem.Node
	watcher    *filesystem.Watcher
	testRunner *runner.Runner

	// Application State
	output          string
	runningNodePath string
	lastRunNode     *filesystem.Node
	nodeStatus      map[string]TestStatus
}

// Messages

// WatcherMsg indicates a file system event occurred.
type WatcherMsg struct{}

// OutputMsg carries a line of output from the test runner.
type OutputMsg string

// TestResultMsg carries the final result (error or nil) of a test run.
type TestResultMsg struct{ Err error }

// TreeLoadedMsg carries the new file tree after a refresh.
type TreeLoadedMsg *filesystem.Node

// NewModel creates and initializes a new Model.
func NewModel() Model {
	cwd, _ := os.Getwd()
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#909090", Dark: "#A0A0A0"})
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#B0B0B0", Dark: "#808080"})
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#D0D0D0", Dark: "#606060"})
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#909090", Dark: "#A0A0A0"})
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#B0B0B0", Dark: "#808080"})
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#D0D0D0", Dark: "#606060"})
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Prompt = "/"
	ti.CharLimit = 156
	ti.Width = 20

	return Model{
		activePane:  PaneExplorer,
		rootPath:    cwd,
		testRunner:  runner.NewRunner(),
		nodeStatus:  make(map[string]TestStatus),
		keys:        NewKeyMap(),
		help:        h,
		searchInput: ti,
	}
}

// Init initializes the Bubbletea program.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshTree,
		m.startWatcher,
		m.waitForOutput,
		m.waitForTestResult,
	)
}

// Update handles incoming messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle global keys (except when in search mode, some keys might be overridden)
		if !m.searchMode {
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
			case key.Matches(msg, m.keys.ReRunLast):
				if m.lastRunNode != nil {
					return m, m.triggerTest(m.lastRunNode)
				}
			case key.Matches(msg, m.keys.NextTab):
				if m.activePane == PaneExplorer {
					if m.activeTab == TabExplorer {
						m.activeTab = TabWatched
					} else {
						m.activeTab = TabExplorer
					}
				}
			case key.Matches(msg, m.keys.PrevTab):
				if m.activePane == PaneExplorer {
					if m.activeTab == TabExplorer {
						m.activeTab = TabWatched
					} else {
						m.activeTab = TabExplorer
					}
				}
			}
		}

		// Handle pane-specific keys
		if m.activePane == PaneExplorer {
			if m.activeTab == TabWatched {
				switch {
				case key.Matches(msg, m.keys.Up):
					if m.watchedCursor > 0 {
						m.watchedCursor--
					}
				case key.Matches(msg, m.keys.Down):
					if m.watchedCursor < len(m.watchedFiles)-1 {
						m.watchedCursor++
					}
				case key.Matches(msg, m.keys.Enter):
					if m.watchedCursor < len(m.watchedFiles) {
						path := m.watchedFiles[m.watchedCursor]
						// Create a dummy node for triggering the test
						node := &filesystem.Node{
							Path: path,
							Name: path[strings.LastIndex(path, string(os.PathSeparator))+1:],
						}
						return m, m.triggerTest(node)
					}
				case key.Matches(msg, m.keys.ToggleWatch):
					if m.watchedCursor < len(m.watchedFiles) {
						// Remove from watched
						m.watchedFiles = append(m.watchedFiles[:m.watchedCursor], m.watchedFiles[m.watchedCursor+1:]...)
						if m.watchedCursor >= len(m.watchedFiles) && m.watchedCursor > 0 {
							m.watchedCursor--
						}
					}
				}
				return m, nil
			}

			if m.searchMode {
				if m.searchFocus {
					// Typing Mode
					switch {
					case key.Matches(msg, m.keys.ExitSearch):
						m.searchMode = false
						m.searchFocus = false
						m.searchInput.Blur()
						m.searchInput.Reset()
						m.searchMatches = nil
						return m, nil
					case key.Matches(msg, m.keys.Enter):
						// Switch to Navigation Mode
						m.searchFocus = false
						m.searchInput.Blur()
						// Jump to first match if exists
						if len(m.searchMatches) > 0 {
							m.currentMatchIndex = 0
							m.cursor = m.searchMatches[0]
						}
						return m, nil
					default:
						// Forward to text input
						var cmd tea.Cmd
						m.searchInput, cmd = m.searchInput.Update(msg)

						// Update matches
						m.searchMatches = []int{}
						if m.searchInput.Value() != "" {
							for i, node := range m.flatNodes {
								if strings.Contains(strings.ToLower(node.Name), strings.ToLower(m.searchInput.Value())) {
									m.searchMatches = append(m.searchMatches, i)
								}
							}
						}
						return m, cmd
					}
				} else {
					// Navigation Mode
					switch {
					case key.Matches(msg, m.keys.ExitSearch):
						m.searchMode = false
						m.searchInput.Reset()
						m.searchMatches = nil
						return m, nil
					case key.Matches(msg, m.keys.Search):
						// Re-enter typing mode?
						m.searchFocus = true
						m.searchInput.Focus()
						return m, textinput.Blink
					case key.Matches(msg, m.keys.NextMatch):
						if len(m.searchMatches) > 0 {
							m.currentMatchIndex = (m.currentMatchIndex + 1) % len(m.searchMatches)
							m.cursor = m.searchMatches[m.currentMatchIndex]
						}
					case key.Matches(msg, m.keys.PrevMatch):
						if len(m.searchMatches) > 0 {
							m.currentMatchIndex = (m.currentMatchIndex - 1 + len(m.searchMatches)) % len(m.searchMatches)
							m.cursor = m.searchMatches[m.currentMatchIndex]
						}
					case key.Matches(msg, m.keys.Enter):
						// Select/Run the file
						m.searchMode = false
						m.searchInput.Reset()
						m.searchMatches = nil
						if m.cursor < len(m.flatNodes) {
							node := m.flatNodes[m.cursor]
							if !node.IsDir {
								return m, m.triggerTest(node)
							}
						}
					}
				}
			} else {
				switch {
				case key.Matches(msg, m.keys.Search):
					m.searchMode = true
					m.searchFocus = true
					m.searchInput.Focus()
					return m, textinput.Blink
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
				case key.Matches(msg, m.keys.ToggleWatch):
					if m.cursor < len(m.flatNodes) {
						node := m.flatNodes[m.cursor]
						if !node.IsDir {
							// Toggle watch status
							found := false
							for i, path := range m.watchedFiles {
								if path == node.Path {
									// Remove
									m.watchedFiles = append(m.watchedFiles[:i], m.watchedFiles[i+1:]...)
									found = true
									break
								}
							}
							if !found {
								// Add
								m.watchedFiles = append(m.watchedFiles, node.Path)
							}
						}
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
			m.viewport.SetContent(m.wrapOutput(paneWidth, m.output))
			m.ready = true
		} else {
			m.viewport.Width = paneWidth
			m.viewport.Height = viewportHeight
			m.viewport.SetContent(m.wrapOutput(paneWidth, m.output))
		}

	case WatcherMsg:
		return m, m.refreshTree

	case TreeLoadedMsg:
		m.fileTree = msg
		m.flatNodes = flattenNodes(m.fileTree)
		return m, nil

	case OutputMsg:
		m.output += string(msg) + "\n"
		m.viewport.SetContent(m.wrapOutput(m.viewport.Width, m.output))
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
			m.viewport.SetContent(m.wrapOutput(m.viewport.Width, m.output))
			m.viewport.GotoBottom()
			m.runningNodePath = ""
		}
		return m, m.waitForTestResult
	}

	return m, tea.Batch(cmds...)
}

func (m Model) wrapOutput(width int, content string) string {
	if width <= 0 {
		return content
	}
	return lipgloss.NewStyle().Width(width).Render(content)
}

// View renders the UI based on the current state.
func (m Model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	if m.width == 0 {
		return "Loading..."
	}

	paneWidth := (m.width / 2) - 2
	paneHeight := m.height - 4

	// Explorer View
	explorerRender := m.renderExplorer(paneWidth, paneHeight)

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
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, panes, footer)
}

// Commands

func (m *Model) refreshTree() tea.Msg {
	tree, err := filesystem.Walk(m.rootPath)
	if err != nil {
		return nil
	}
	return TreeLoadedMsg(tree)
}

func (m *Model) startWatcher() tea.Msg {
	w, err := filesystem.NewWatcher(m.rootPath)
	if err != nil {
		return nil
	}
	m.watcher = w

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
