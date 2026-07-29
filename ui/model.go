package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jesspatton/lazytest/engine"
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

// DisplayNode represents a node in the explorer list, potentially compacted.
type DisplayNode struct {
	*filesystem.Node
	DisplayName string
	Depth       int
}

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
	engine    *engine.Engine
	flatNodes []DisplayNode
}

// NewModel creates and initializes a new Model.
func NewModel(eng *engine.Engine) Model {
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
		engine:      eng,
		keys:        NewKeyMap(),
		help:        h,
		searchInput: ti,
	}
}

// Init initializes the Bubbletea program.
func (m Model) Init() tea.Cmd {
	return m.engine.Init()
}

// Update handles incoming messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Let engine handle business logic
	cmd = m.engine.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle global keys (except when in search mode, some keys might be overridden)
		if !m.searchMode {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.Help):
				m.showHelp = !m.showHelp
				m.help.ShowAll = m.showHelp
				return m, nil
			case key.Matches(msg, m.keys.Tab):
				if m.activePane == PaneExplorer {
					m.activePane = PaneOutput
				} else {
					m.activePane = PaneExplorer
				}
			case key.Matches(msg, m.keys.Refresh):
				return m, m.engine.RefreshTree
			case key.Matches(msg, m.keys.ReRunLast):
				return m, m.engine.ReRunLast()
			case key.Matches(msg, m.keys.NextTab):
				if m.activePane == PaneExplorer {
					if m.activeTab == TabExplorer {
						m.activeTab = TabWatched
					} else {
						m.activeTab = TabExplorer
					}
					m.syncViewportOutput()
				}
			case key.Matches(msg, m.keys.PrevTab):
				if m.activePane == PaneExplorer {
					if m.activeTab == TabExplorer {
						m.activeTab = TabWatched
					} else {
						m.activeTab = TabExplorer
					}
					m.syncViewportOutput()
				}
			case key.Matches(msg, m.keys.ClearWatched):
				if m.engine.IsSmartMode() {
					m.engine.ClearAffectedSuite()
					m.watchedCursor = 0
					m.syncViewportOutput()
				} else {
					m.engine.ClearWatched()
					m.watchedCursor = 0
					if m.activeTab == TabWatched {
						m.viewport.SetContent(m.wrapOutput(m.viewport.Width, "No watched files.\nPress 'w' on a file to watch it."))
					}
				}
			case key.Matches(msg, m.keys.RunFailures):
				if m.engine.IsSmartMode() {
					return m, m.engine.RunSuiteFailures()
				}
			case key.Matches(msg, m.keys.AddRelated):
				if m.engine.IsSmartMode() {
					return m, m.engine.RunAffectedSuite()
				}
				changedFiles, err := filesystem.GetChangedFiles(m.engine.State.RootPath)
				if err != nil {
					m.engine.State.CurrentOutput += fmt.Sprintf("Error getting changed files: %v\n", err)
				} else {
					count := 0
					for _, src := range changedFiles {
						related := m.engine.FindRelatedTests(src)
						for _, test := range related {
							if !m.engine.IsWatched(test) {
								m.engine.ToggleWatch(test)
								count++
							}
						}
					}
				}
			case key.Matches(msg, m.keys.ToggleSmartMode):
				m.engine.ToggleSmartMode()
				m.applySmartModeBindings()
			}
		}

		// Handle pane-specific keys
		if m.activePane == PaneExplorer {
			if m.activeTab == TabWatched {
				// In Smart Mode the watched tab shows the Affected Suite list.
				tabList, _ := m.getTabList()
				switch {
				case key.Matches(msg, m.keys.Up):
					if m.watchedCursor > 0 {
						m.watchedCursor--
						m.syncViewportOutput()
					}
				case key.Matches(msg, m.keys.Down):
					if m.watchedCursor < len(tabList)-1 {
						m.watchedCursor++
						m.syncViewportOutput()
					}
				case key.Matches(msg, m.keys.Enter):
					if m.watchedCursor < len(tabList) {
						path := tabList[m.watchedCursor]
						return m, m.engine.TriggerTest(filesystem.NodeFromPath(path))
					}
				case key.Matches(msg, m.keys.ToggleWatch):
					if !m.engine.IsSmartMode() && m.watchedCursor < len(tabList) {
						path := tabList[m.watchedCursor]
						m.engine.ToggleWatch(path)
						if m.watchedCursor >= len(m.engine.GetWatchedFiles()) && m.watchedCursor > 0 {
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
								if strings.Contains(strings.ToLower(node.DisplayName), strings.ToLower(m.searchInput.Value())) {
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
							m.syncViewportOutput()
						}
					case key.Matches(msg, m.keys.PrevMatch):
						if len(m.searchMatches) > 0 {
							m.currentMatchIndex = (m.currentMatchIndex - 1 + len(m.searchMatches)) % len(m.searchMatches)
							m.cursor = m.searchMatches[m.currentMatchIndex]
							m.syncViewportOutput()
						}
					case key.Matches(msg, m.keys.Enter):
						// Select/Run the file
						m.searchMode = false
						m.searchInput.Reset()
						m.searchMatches = nil
						if m.cursor < len(m.flatNodes) {
							node := m.flatNodes[m.cursor]
							if !node.IsDir {
								return m, m.engine.TriggerTest(node.Node)
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
					// Smart Navigation Up
					newCursor := m.cursor - 1
					for newCursor >= 0 {
						if !m.flatNodes[newCursor].IsDir {
							m.cursor = newCursor
							break
						}
						newCursor--
					}
					m.syncViewportOutput()
				case key.Matches(msg, m.keys.Down):
					// Smart Navigation Down
					newCursor := m.cursor + 1
					for newCursor < len(m.flatNodes) {
						if !m.flatNodes[newCursor].IsDir {
							m.cursor = newCursor
							break
						}
						newCursor++
					}
					m.syncViewportOutput()
				case key.Matches(msg, m.keys.Enter):
					if m.cursor < len(m.flatNodes) {
						node := m.flatNodes[m.cursor]
						if !node.IsDir {
							return m, m.engine.TriggerTest(node.Node)
						}
					}
				case key.Matches(msg, m.keys.ToggleWatch):
					if m.cursor < len(m.flatNodes) {
						node := m.flatNodes[m.cursor]
						if !node.IsDir {
							m.engine.ToggleWatch(node.Path)
						}
					}
				default:
					// No matching key
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
			m.viewport.SetContent(m.wrapOutput(paneWidth, m.engine.State.CurrentOutput))
			m.ready = true
		} else {
			m.viewport.Width = paneWidth
			m.viewport.Height = viewportHeight
			m.viewport.SetContent(m.wrapOutput(paneWidth, m.engine.GetCurrentOutput()))
		}

	case engine.TreeLoadedMsg:
		m.flatNodes = flattenNodes(m.engine.GetTree())
		m.syncViewportOutput()
		return m, nil

	case engine.WatcherMsg:
		m.syncViewportOutput()
		return m, tea.Batch(cmds...)

	case runner.OutputUpdate:
		shouldShow := true
		runningNode := m.engine.GetRunningNode()
		if runningNode != nil {
			if m.activeTab == TabWatched {
				tabList, _ := m.getTabList()
				if m.watchedCursor < len(tabList) && tabList[m.watchedCursor] != runningNode.Path {
					shouldShow = false
				}
			} else if m.activeTab == TabExplorer {
				if m.cursor < len(m.flatNodes) && m.flatNodes[m.cursor].Path != runningNode.Path {
					shouldShow = false
				}
			}
		}

		if shouldShow {
			m.viewport.SetContent(m.wrapOutput(m.viewport.Width, m.engine.GetCurrentOutput()))
			m.viewport.GotoBottom()
		}
		return m, tea.Batch(cmds...)

	case runner.StatusUpdate:
		// Zero-Touch Failure Auto-Focus (Smart Mode only):
		// When a test fails in Smart Mode, automatically jump to it.
		if msg.Err != nil && m.engine.IsSmartMode() {
			suite := m.engine.GetAffectedSuite()
			// The failed test will be first after re-sort (StatusFail priority)
			if len(suite) > 0 {
				m.activeTab = TabWatched
				m.watchedCursor = 0
				path := suite[0]
				if out, ok := m.engine.GetTestOutput(path); ok && out != "" {
					m.viewport.SetContent(m.wrapOutput(m.viewport.Width, out))
					m.viewport.GotoBottom()
				}
				return m, tea.Batch(cmds...)
			}
		}
		// After a test finishes, sync to show the final stored output for the
		// currently selected file (respects cursor position).
		m.syncViewportOutput()
		return m, tea.Batch(cmds...)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) wrapOutput(width int, content string) string {
	if width <= 0 {
		return content
	}
	return lipgloss.NewStyle().Width(width).Render(content)
}

// syncViewportOutput resolves and renders the correct output for the currently
// selected file/tab into the viewport. It is the single source of truth for
// deciding what the right-hand pane should display.
func (m *Model) syncViewportOutput() {
	if !m.ready {
		return
	}

	var content string

	if m.activeTab == TabWatched {
		tabList, emptyMsg := m.getTabList()
		if m.watchedCursor < len(tabList) {
			path := tabList[m.watchedCursor]
			if out, ok := m.engine.GetTestOutput(path); ok && out != "" {
				content = out
			} else {
				content = "No output yet."
			}
		} else {
			content = emptyMsg
		}
	} else {
		// TabExplorer
		if m.cursor < len(m.flatNodes) {
			node := m.flatNodes[m.cursor]
			if !node.IsDir {
				if out, ok := m.engine.GetTestOutput(node.Path); ok && out != "" {
					content = out
				} else if filesystem.IsTestFile(node.Name) {
					content = "No output yet for this test file.\nPress <Enter> to run, 'w' to watch, or 's' for Smart Mode."
				} else {
					content = fmt.Sprintf("Source file: %s\nPress 'w' to watch or 's' for Smart Mode.", node.Name)
				}
			} else {
				content = fmt.Sprintf("Directory: %s", node.Name)
			}
		}
	}

	if content == "" {
		content = m.engine.GetCurrentOutput()
	}

	m.viewport.SetContent(m.wrapOutput(m.viewport.Width, content))
}

// getTabList returns the list of paths and an empty-state hint message for the
// currently active tab, accounting for Smart Mode vs. Manual Watch Mode.
func (m *Model) getTabList() ([]string, string) {
	if m.engine.IsSmartMode() {
		return m.engine.GetAffectedSuite(), "No tests affected yet.\nEdit a source file to trigger Smart Mode."
	}
	return m.engine.GetWatchedFiles(), "No watched files.\nPress 'w' on a file to watch it."
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
	if m.engine.IsSmartMode() {
		passed, failed, running := m.engine.GetSuiteStats()
		badge := m.renderSuiteBadge(passed, failed, running)
		outputView.WriteString(badge + "\n")
	} else {
		outputView.WriteString(titleStyle.Render("OUTPUT") + "\n\n")
	}

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

// applySmartModeBindings updates key enabled states and help labels to reflect
// the current smart mode state. Call this immediately after toggling smart mode.
func (m *Model) applySmartModeBindings() {
	smartMode := m.engine.IsSmartMode()

	// ToggleWatch is meaningless in Smart Mode
	m.keys.ToggleWatch.SetEnabled(!smartMode)

	// RunFailures is only available in Smart Mode
	m.keys.RunFailures.SetEnabled(smartMode)

	// Repurpose ClearWatched and AddRelated labels in Smart Mode
	if smartMode {
		m.keys.ClearWatched = key.NewBinding(
			key.WithKeys("W"),
			key.WithHelp("W", "clear suite"),
		)
		m.keys.AddRelated = key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "run suite"),
		)
	} else {
		m.keys.ClearWatched = key.NewBinding(
			key.WithKeys("W"),
			key.WithHelp("W", "clear watched"),
		)
		m.keys.AddRelated = key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add related"),
		)
	}
}

// renderSuiteBadge renders the live suite stats header shown in Smart Mode.
// Example:  ⚡ SMART MODE | 3 Passed • 1 Failed • 0 Running
func (m Model) renderSuiteBadge(passed, failed, running int) string {
	label := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}).
		Bold(true).
		Padding(0, 1).
		Render("⚡ SMART MODE")

	sep := lipgloss.NewStyle().
		Foreground(subtle).
		Render(" | ")

	passedStr := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}).
		Render(fmt.Sprintf("%d Passed", passed))

	failedStr := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}).
		Render(fmt.Sprintf("%d Failed", failed))

	runningStr := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FCD34D"}).
		Render(fmt.Sprintf("%d Running", running))

	dot := lipgloss.NewStyle().Foreground(subtle).Render(" • ")

	return label + sep + passedStr + dot + failedStr + dot + runningStr
}
