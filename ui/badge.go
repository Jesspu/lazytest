package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

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
