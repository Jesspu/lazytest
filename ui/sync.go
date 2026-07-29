package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/jesspatton/lazytest/filesystem"
)

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
