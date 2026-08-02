package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jesspatton/lazytest/engine"
	"github.com/jesspatton/lazytest/filesystem"
	"github.com/jesspatton/lazytest/runner"
)

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
	case tea.MouseMsg:
		return m.handleMouse(msg)
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
					return m, func() tea.Msg {
						return engine.NotificationMsg{
							Message: fmt.Sprintf("Failed to get changed files: %v", err),
							IsError: true,
						}
					}
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
				handled := false
				m, cmd, handled = m.handleSearchKey(msg)
				if handled {
					return m, cmd
				}
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			} else {
				switch {
				case key.Matches(msg, m.keys.Search):
					m.searchMode = true
					m.searchFocus = true
					m.searchInput.Focus()
					return m, tea.Batch(append(cmds, textinput.Blink)...)
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
			m.ready = true
		} else {
			m.viewport.Width = paneWidth
			m.viewport.Height = viewportHeight
		}
		m.syncViewportOutput()

	case engine.TreeLoadedMsg:
		m.flatNodes = flattenNodes(m.engine.GetTree())
		m.syncViewportOutput()
		return m, nil

	case engine.WatcherMsg:
		m.syncViewportOutput()
		return m, tea.Batch(cmds...)

	case runner.OutputUpdate:
		shouldShow := true
		switch m.activeTab {
		case TabWatched:
			tabList, _ := m.getTabList()
			if m.watchedCursor < len(tabList) && tabList[m.watchedCursor] != msg.FilePath {
				shouldShow = false
			}
		case TabExplorer:
			if m.cursor < len(m.flatNodes) && m.flatNodes[m.cursor].Path != msg.FilePath {
				shouldShow = false
			}
		}

		if shouldShow {
			out, _ := m.engine.GetTestOutput(msg.FilePath)
			m.viewport.SetContent(m.wrapOutput(m.viewport.Width, out))
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

	case engine.NotificationMsg:
		m.notificationID++
		m.activeNotification = msg.Message
		m.isNotificationError = msg.IsError
		id := m.notificationID
		cmds = append(cmds, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return engine.ClearNotificationMsg{ID: id}
		}))
		return m, tea.Batch(cmds...)

	case engine.ClearNotificationMsg:
		if msg.ID == m.notificationID {
			m.activeNotification = ""
			m.isNotificationError = false
		}
		return m, tea.Batch(cmds...)
	}

	return m, tea.Batch(cmds...)
}
