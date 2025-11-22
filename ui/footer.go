package ui



func (m Model) renderFooter() string {
	return statusStyle.Render("? Help  q Quit  Tab Switch Pane  Enter Run  R Refresh")
}
