package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jesspatton/lazytest/engine"
	"github.com/jesspatton/lazytest/filesystem"
)

func (m Model) renderExplorer(paneWidth, paneHeight int) string {
	var explorerView strings.Builder

	// Render Tabs
	activeTabStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlight).
		Padding(0, 1).
		Foreground(highlight)

	inactiveTabStyle := lipgloss.NewStyle().
		Border(lipgloss.HiddenBorder()).
		BorderForeground(subtle).
		Padding(0, 1).
		Foreground(subtle)

	var explorerTab, watchedTab string
	if m.activeTab == TabExplorer {
		explorerTab = activeTabStyle.Render("Explorer")
		watchedTab = inactiveTabStyle.Render("Watched")
	} else {
		explorerTab = inactiveTabStyle.Render("Explorer")
		watchedTab = activeTabStyle.Render("Watched")
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Bottom, explorerTab, watchedTab)
	explorerView.WriteString(tabs + "\n\n")

	// Calculate available height for the tree
	treeHeight := paneHeight
	searchHeight := 0
	if m.searchMode && m.activeTab == TabExplorer {
		searchHeight = 3 // 1 line text + 2 lines border
		treeHeight -= searchHeight
	}

	if m.activeTab == TabExplorer {
		if m.engine.GetTree() == nil {
			explorerView.WriteString("Scanning...")
		} else {
			start, end := m.calculateVisibleRange(treeHeight)

			for i := start; i < end; i++ {
				if i >= len(m.flatNodes) {
					break
				}
				node := m.flatNodes[i]
				m.renderNode(&explorerView, node, i)
			}
		}
	} else {
		// Render Watched Files
		if len(m.engine.GetWatchedFiles()) == 0 {
			explorerView.WriteString("No watched files.\nPress 'w' on a file to watch it.")
		} else {
			start := 0
			end := len(m.engine.GetWatchedFiles())
			if len(m.engine.GetWatchedFiles()) > treeHeight {
				if m.watchedCursor < treeHeight/2 {
					start = 0
					end = treeHeight
				} else if m.watchedCursor > len(m.engine.GetWatchedFiles())-treeHeight/2 {
					start = len(m.engine.GetWatchedFiles()) - treeHeight
					end = len(m.engine.GetWatchedFiles())
				} else {
					start = m.watchedCursor - treeHeight/2
					end = m.watchedCursor + treeHeight/2
				}
			}

			for i := start; i < end; i++ {
				path := m.engine.GetWatchedFiles()[i]
				name := path[strings.LastIndex(path, string(os.PathSeparator))+1:]

				cursor := " "
				if m.watchedCursor == i {
					cursor = ">"
				}

				// Get status for this file
				status, ok := m.engine.GetNodeStatus(path)
				icon := "📄"
				if ok {
					switch status {
					case engine.StatusRunning:
						icon = "⏳"
					case engine.StatusPass:
						icon = "✅"
					case engine.StatusFail:
						icon = "❌"
					}
				}

				line := fmt.Sprintf("%s %s %s", cursor, icon, name)
				if m.watchedCursor == i {
					explorerView.WriteString(lipgloss.NewStyle().Foreground(highlight).Render(line) + "\n")
				} else {
					explorerView.WriteString(line + "\n")
				}
			}
		}
	}

	// Fill remaining space to push search bar to bottom
	currentView := explorerView.String()
	currentHeight := lipgloss.Height(currentView)
	if currentHeight < treeHeight {
		currentView += strings.Repeat("\n", treeHeight-currentHeight)
	}

	if m.searchMode && m.activeTab == TabExplorer {
		searchStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			Width(paneWidth - 4) // Account for border width

		searchContent := m.searchInput.View()
		if !m.searchFocus {
			hints := "n: next • N: prev • Esc: exit"
			hintsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

			// Calculate available space
			availableWidth := paneWidth - 6 // -4 for outer margin, -2 for border
			contentWidth := lipgloss.Width(searchContent)
			hintsWidth := lipgloss.Width(hints)

			if contentWidth+hintsWidth+1 < availableWidth {
				padding := strings.Repeat(" ", availableWidth-contentWidth-hintsWidth)
				searchContent += padding + hintsStyle.Render(hints)
			}
		}
		currentView += searchStyle.Render(searchContent)
	}

	explorerStyle := paneStyle
	if m.activePane == PaneExplorer {
		explorerStyle = activePaneStyle
	}

	return explorerStyle.
		Width(paneWidth).
		Height(paneHeight).
		Render(currentView)
}

func (m Model) calculateVisibleRange(paneHeight int) (int, int) {
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
	return start, end
}

func (m Model) renderNode(b *strings.Builder, node DisplayNode, index int) {
	cursor := " "
	if m.cursor == index {
		cursor = ">"
	}

	// Use the pre-calculated depth from DisplayNode
	indent := strings.Repeat("  ", node.Depth)

	icon := m.getNodeIcon(node.Node)

	// Check if watched
	watchIcon := "  "
	for _, watched := range m.engine.GetWatchedFiles() {
		if watched == node.Path {
			watchIcon = "👁 "
			break
		}
	}

	name := node.DisplayName
	// Highlight search matches
	if m.searchMode && m.searchInput.Value() != "" {
		lowerName := strings.ToLower(name)
		lowerQuery := strings.ToLower(m.searchInput.Value())
		if strings.Contains(lowerName, lowerQuery) {
			// Find all occurrences
			var sb strings.Builder
			lastIdx := 0
			for {
				idx := strings.Index(lowerName[lastIdx:], lowerQuery)
				if idx == -1 {
					sb.WriteString(name[lastIdx:])
					break
				}
				idx += lastIdx
				sb.WriteString(name[lastIdx:idx])
				sb.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("212")).Foreground(lipgloss.Color("0")).Render(name[idx : idx+len(lowerQuery)]))
				lastIdx = idx + len(lowerQuery)
			}
			name = sb.String()
		}
	}

	line := fmt.Sprintf("%s %s%s%s %s", cursor, indent, watchIcon, icon, name)

	if m.cursor == index {
		b.WriteString(lipgloss.NewStyle().Foreground(highlight).Render(line) + "\n")
	} else {
		b.WriteString(line + "\n")
	}
}

func (m Model) getNodeIcon(node *filesystem.Node) string {
	if node.IsDir {
		return "📁"
	}

	status, ok := m.engine.GetNodeStatus(node.Path)
	if !ok {
		return "📄"
	}

	switch status {
	case engine.StatusRunning:
		return "⏳"
	case engine.StatusPass:
		return "✅"
	case engine.StatusFail:
		return "❌"
	default:
		return "📄"
	}
}
