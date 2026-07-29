package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jesspatton/lazytest/filesystem"
)

// handleMouse processes mouse events based on the mouse support plan.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	splitPoint := m.width / 2

	// --- Output Pane ---
	if msg.X >= splitPoint {
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			m.activePane = PaneOutput
		}
		return m, nil
	}

	// --- Explorer Pane ---
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		m.activePane = PaneExplorer
	}

	// Handle Explorer Pane Scrolling
	if msg.Button == tea.MouseButtonWheelUp {
		if m.activeTab == TabExplorer {
			if m.cursor > 0 {
				m.cursor--
			}
		} else {
			if m.watchedCursor > 0 {
				m.watchedCursor--
			}
		}
		m.syncViewportOutput()
		return m, nil
	}

	if msg.Button == tea.MouseButtonWheelDown {
		if m.activeTab == TabExplorer {
			if m.cursor < len(m.flatNodes)-1 {
				m.cursor++
			}
		} else {
			tabList, _ := m.getTabList()
			if m.watchedCursor < len(tabList)-1 {
				m.watchedCursor++
			}
		}
		m.syncViewportOutput()
		return m, nil
	}

	// Handle Explorer Pane Clicks
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		// Border offsets from paneStyle
		contentX := msg.X - 2 // 1 border + 1 padding
		contentY := msg.Y - 1 // 1 top border

		if contentX < 0 || contentY < 0 {
			return m, nil
		}

		// Recreate tab header logic to find exact height
		activeTabStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)
		
		explorerTab := activeTabStyle.Render("Explorer")
		watchedTabLabel := "Watched"
		if m.engine.IsSmartMode() {
			watchedTabLabel = "Affected Suite"
		}
		watchedTab := activeTabStyle.Render(watchedTabLabel)
		tabs := lipgloss.JoinHorizontal(lipgloss.Bottom, explorerTab, watchedTab)
		
		headerOffset := lipgloss.Height(tabs) + 1
		headerPhysicalHeight := lipgloss.Height(tabs + "\n\n")
		tabAreaHeight := lipgloss.Height(tabs)
		explorerTabWidth := lipgloss.Width(explorerTab)

		// 1. Clicked Tabs Area
		if contentY < tabAreaHeight {
			if contentX < explorerTabWidth {
				m.activeTab = TabExplorer
			} else {
				m.activeTab = TabWatched
			}
			m.syncViewportOutput()
			return m, nil
		}

		// Calculate available heights for the tree
		paneHeight := m.height - 5
		treeHeight := paneHeight - headerPhysicalHeight

		searchHeight := 0
		if m.searchMode && m.activeTab == TabExplorer {
			searchHeight = 3
			treeHeight -= searchHeight
			
			// 2. Clicked Search Bar Area
			if contentY >= paneHeight-searchHeight {
				m.searchFocus = true
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(tea.MouseMsg{}) // just in case
				m.searchInput.Focus()
				return m, tea.Batch(cmd, textinput.Blink)
			} else {
				m.searchFocus = false
				m.searchInput.Blur()
			}
		}

		if treeHeight < 0 {
			treeHeight = 0
		}

		// 3. Clicked List Area
		if contentY >= headerOffset && contentY < headerOffset+treeHeight {
			visualIndex := contentY - headerOffset

			isDoubleClick := false
			if time.Since(m.lastClickTime) < 500*time.Millisecond &&
				msg.X == m.lastClickX && msg.Y == m.lastClickY {
				isDoubleClick = true
			}

			m.lastClickTime = time.Now()
			m.lastClickX = msg.X
			m.lastClickY = msg.Y

			if m.activeTab == TabExplorer {
				start, _ := m.calculateVisibleRange(treeHeight)
				index := start + visualIndex

				if index >= 0 && index < len(m.flatNodes) {
					m.cursor = index
					m.syncViewportOutput()

					if isDoubleClick {
						node := m.flatNodes[m.cursor]
						if !node.IsDir {
							return m, m.engine.TriggerTest(node.Node)
						}
					}
				}
			} else {
				tabList, _ := m.getTabList()
				// calculate visible range logic from explorer.go for watched tab
				start := 0
				if len(tabList) > treeHeight {
					if m.watchedCursor < treeHeight/2 {
						start = 0
					} else if m.watchedCursor > len(tabList)-treeHeight/2 {
						start = len(tabList) - treeHeight
					} else {
						start = m.watchedCursor - treeHeight/2
					}
				}

				index := start + visualIndex

				if index >= 0 && index < len(tabList) {
					m.watchedCursor = index
					m.syncViewportOutput()

					if isDoubleClick {
						path := tabList[m.watchedCursor]
						return m, m.engine.TriggerTest(filesystem.NodeFromPath(path))
					}
				}
			}
		}
	}

	return m, nil
}
