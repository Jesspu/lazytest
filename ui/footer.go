package ui

import "github.com/charmbracelet/lipgloss"

func (m Model) renderFooter() string {
	m.help.ShowAll = false

	var leftComponent string
	if m.activeNotification != "" {
		if m.isNotificationError {
			leftComponent = notificationErrorStyle.Render("✖ " + m.activeNotification)
		} else {
			leftComponent = notificationInfoStyle.Render("ℹ " + m.activeNotification)
		}
	} else {
		leftComponent = statusStyle.Render(m.help.View(m.keys))
	}

	var modeLabel string
	if m.engine.IsSmartMode() {
		modeLabel = lipgloss.NewStyle().
			Foreground(special).
			Bold(true).
			Padding(0, 1).
			Render("[SMART MODE]")
	} else {
		modeLabel = lipgloss.NewStyle().
			Foreground(subtle).
			Padding(0, 1).
			Render("[MANUAL]")
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, leftComponent, modeLabel)
}
