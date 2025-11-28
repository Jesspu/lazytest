package ui

import (
	"os"
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
				// TODO: Implement ReRunLast in Engine
				if m.engine.State.RunningNode != nil {
					// This logic is slightly different now, we might need a LastRunNode in State
				}
			case key.Matches(msg, m.keys.NextTab):
				if m.activePane == PaneExplorer {
					if m.activeTab == TabExplorer {
						m.activeTab = TabWatched
						if m.watchedCursor < len(m.engine.State.Watched) {
							path := m.engine.State.Watched[m.watchedCursor]
							if out, ok := m.engine.State.TestOutputs[path]; ok {
								m.viewport.SetContent(m.wrapOutput(m.viewport.Width, out))
							} else {
								m.viewport.SetContent(m.wrapOutput(m.viewport.Width, "No output yet."))
							}
							m.viewport.GotoBottom()
						}
					} else {
						m.activeTab = TabExplorer
						m.viewport.SetContent(m.wrapOutput(m.viewport.Width, m.engine.State.CurrentOutput))
						m.viewport.GotoBottom()
					}
				}
			case key.Matches(msg, m.keys.PrevTab):
				if m.activePane == PaneExplorer {
					if m.activeTab == TabExplorer {
						m.activeTab = TabWatched
						if m.watchedCursor < len(m.engine.State.Watched) {
							path := m.engine.State.Watched[m.watchedCursor]
							if out, ok := m.engine.State.TestOutputs[path]; ok {
								m.viewport.SetContent(m.wrapOutput(m.viewport.Width, out))
							} else {
								m.viewport.SetContent(m.wrapOutput(m.viewport.Width, "No output yet."))
							}
							m.viewport.GotoBottom()
						}
					} else {
						m.activeTab = TabExplorer
						m.viewport.SetContent(m.wrapOutput(m.viewport.Width, m.engine.State.CurrentOutput))
						m.viewport.GotoBottom()
					}
				}
			case key.Matches(msg, m.keys.ClearWatched):
				m.engine.State.Watched = []string{}
				m.watchedCursor = 0
				if m.activeTab == TabWatched {
					m.viewport.SetContent(m.wrapOutput(m.viewport.Width, "No watched files.\nPress 'w' on a file to watch it."))
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
						path := m.engine.State.Watched[m.watchedCursor]
						if out, ok := m.engine.State.TestOutputs[path]; ok {
							m.viewport.SetContent(m.wrapOutput(m.viewport.Width, out))
						} else {
							m.viewport.SetContent(m.wrapOutput(m.viewport.Width, "No output yet."))
						}
						m.viewport.GotoBottom()
					}
				case key.Matches(msg, m.keys.Down):
					if m.watchedCursor < len(m.engine.State.Watched)-1 {
						m.watchedCursor++
						path := m.engine.State.Watched[m.watchedCursor]
						if out, ok := m.engine.State.TestOutputs[path]; ok {
							m.viewport.SetContent(m.wrapOutput(m.viewport.Width, out))
						} else {
							m.viewport.SetContent(m.wrapOutput(m.viewport.Width, "No output yet."))
						}
						m.viewport.GotoBottom()
					}
				case key.Matches(msg, m.keys.Enter):
					if m.watchedCursor < len(m.engine.State.Watched) {
						path := m.engine.State.Watched[m.watchedCursor]
						// Create a dummy node for triggering the test
						node := &filesystem.Node{
							Path: path,
							Name: path[strings.LastIndex(path, string(os.PathSeparator))+1:],
						}
						return m, m.engine.TriggerTest(node)
					}
				case key.Matches(msg, m.keys.ToggleWatch):
					if m.watchedCursor < len(m.engine.State.Watched) {
						path := m.engine.State.Watched[m.watchedCursor]
						m.engine.ToggleWatch(path)
						if m.watchedCursor >= len(m.engine.State.Watched) && m.watchedCursor > 0 {
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
			m.viewport.SetContent(m.wrapOutput(paneWidth, m.engine.State.CurrentOutput))
		}

	case engine.TreeLoadedMsg:
		m.flatNodes = flattenNodes(m.engine.State.Tree)
		return m, nil

	case runner.OutputUpdate:
		shouldShow := true
		if m.activeTab == TabWatched {
			if m.watchedCursor < len(m.engine.State.Watched) && m.engine.State.Watched[m.watchedCursor] != m.engine.State.RunningNode.Path {
				shouldShow = false
			}
		}

		if shouldShow {
			m.viewport.SetContent(m.wrapOutput(m.viewport.Width, m.engine.State.CurrentOutput))
			m.viewport.GotoBottom()
		}
		return m, tea.Batch(cmds...)

	case runner.StatusUpdate:
		m.viewport.SetContent(m.wrapOutput(m.viewport.Width, m.engine.State.CurrentOutput))
		m.viewport.GotoBottom()
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
