package e2e

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSearchMode(t *testing.T) {
	tm, teardown := SetupTestEnv(t, "single_repo_jest")
	defer teardown()

	// Wait for TUI to load
	WaitForText(t, tm.Output(), "Auto-detected: jest", 5*time.Second)

	// Send '/' key to enter Search Mode
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'/'},
	})

	// Send keystrokes 'm', 'a', 't', 'h'
	for _, ch := range []rune{'m', 'a', 't', 'h'} {
		tm.Send(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{ch},
		})
	}

	// Assert that search bar query update is visible in output stream
	WaitForText(t, tm.Output(), "math", 3*time.Second)

	// Send 'Escape' key to exit search mode
	tm.Send(tea.KeyMsg{
		Type: tea.KeyEsc,
	})
}
