package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jesspatton/lazytest/filesystem"
)

func (m Model) renderExplorer(paneWidth, paneHeight int) string {
	var explorerView strings.Builder
	explorerView.WriteString(titleStyle.Render("TEST EXPLORER") + "\n\n")

	// Calculate available height for the tree
	treeHeight := paneHeight
	searchHeight := 0
	if m.searchMode {
		searchHeight = 3 // 1 line text + 2 lines border
		treeHeight -= searchHeight
	}

	if m.fileTree == nil {
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

	// Fill remaining space to push search bar to bottom
	currentView := explorerView.String()
	currentHeight := lipgloss.Height(currentView)
	if currentHeight < treeHeight {
		currentView += strings.Repeat("\n", treeHeight-currentHeight)
	}

	if m.searchMode {
		searchStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			Width(paneWidth - 4) // Account for border width
		currentView += searchStyle.Render(m.searchInput.View())
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

func (m Model) renderNode(b *strings.Builder, node *filesystem.Node, index int) {
	cursor := " "
	if m.cursor == index {
		cursor = ">"
	}

	depth := strings.Count(node.Path, string(os.PathSeparator)) - strings.Count(m.rootPath, string(os.PathSeparator))
	indent := strings.Repeat("  ", depth)

	icon := m.getNodeIcon(node)

	name := node.Name
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

	line := fmt.Sprintf("%s %s%s %s", cursor, indent, icon, name)

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

	status, ok := m.nodeStatus[node.Path]
	if !ok {
		return "📄"
	}

	switch status {
	case StatusRunning:
		return "⏳"
	case StatusPass:
		return "✅"
	case StatusFail:
		return "❌"
	default:
		return "📄"
	}
}
