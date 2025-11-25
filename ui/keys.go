package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings for the application.
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Tab       key.Binding
	ReRunLast key.Binding
	Refresh   key.Binding
	Help      key.Binding
	Quit      key.Binding

	// Search Keys
	Search     key.Binding
	NextMatch  key.Binding
	PrevMatch  key.Binding
	ExitSearch key.Binding

	// Tab Keys
	NextTab     key.Binding
	PrevTab     key.Binding
	ToggleWatch key.Binding
}

// NewKeyMap returns a set of default keybindings.
func NewKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "move down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "run test"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
		),
		ReRunLast: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "re-run last"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		NextMatch: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next match"),
		),
		PrevMatch: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "prev match"),
		),
		ExitSearch: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit search"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev tab"),
		),
		ToggleWatch: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "watch/unwatch"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini-help view. It's part of the help.KeyMap interface.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Search, k.NextTab, k.PrevTab, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the help.KeyMap interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Tab},
		{k.PrevTab, k.NextTab, k.ToggleWatch},
		{k.ReRunLast, k.Refresh, k.Help, k.Quit},
	}
}
