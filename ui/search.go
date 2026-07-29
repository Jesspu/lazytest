package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// handleSearchKey processes a key message when the explorer is in search mode.
// It returns a boolean indicating whether the message was fully handled.
func (m Model) handleSearchKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.searchFocus {
		// Typing Mode
		switch {
		case key.Matches(msg, m.keys.ExitSearch):
			m.searchMode = false
			m.searchFocus = false
			m.searchInput.Blur()
			m.searchInput.Reset()
			m.searchMatches = nil
			return m, nil, true
		case key.Matches(msg, m.keys.Enter):
			// Switch to Navigation Mode
			m.searchFocus = false
			m.searchInput.Blur()
			// Jump to first match if exists
			if len(m.searchMatches) > 0 {
				m.currentMatchIndex = 0
				m.cursor = m.searchMatches[0]
			}
			return m, nil, true
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
			return m, cmd, true
		}
	} else {
		// Navigation Mode
		switch {
		case key.Matches(msg, m.keys.ExitSearch):
			m.searchMode = false
			m.searchInput.Reset()
			m.searchMatches = nil
			return m, nil, true
		case key.Matches(msg, m.keys.Search):
			// Re-enter typing mode?
			m.searchFocus = true
			m.searchInput.Focus()
			return m, textinput.Blink, true
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
					return m, m.engine.TriggerTest(node.Node), true
				}
			}
		}
		// Return handled=false so the rest of the flow can process commands
		return m, nil, false
	}
}
