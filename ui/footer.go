package ui

func (m Model) renderFooter() string {
	m.help.ShowAll = false
	return statusStyle.Render(m.help.View(m.keys))
}
