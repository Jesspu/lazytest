package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHelp() string {
	title := titleStyle.Render("HELP")
	helpView := m.help.View(m.keys)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		paneStyle.Render(fmt.Sprintf("%s\n\n%s", title, helpView)),
	)
}
