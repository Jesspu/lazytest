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
	
	if m.fileTree == nil {
		explorerView.WriteString("Scanning...")
	} else {
		start, end := m.calculateVisibleRange(paneHeight)
		
		for i := start; i < end; i++ {
			if i >= len(m.flatNodes) {
				break
			}
			node := m.flatNodes[i]
			m.renderNode(&explorerView, node, i)
		}
	}

	explorerStyle := paneStyle
	if m.activePane == PaneExplorer {
		explorerStyle = activePaneStyle
	}
	return explorerStyle.
		Width(paneWidth).
		Height(paneHeight).
		Render(explorerView.String())
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
	line := fmt.Sprintf("%s %s%s %s", cursor, indent, icon, node.Name)
	
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
