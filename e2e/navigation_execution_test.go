package e2e

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNavigationAndExecution(t *testing.T) {
	tm, teardown := SetupTestEnv(t, "single_repo_vitest")
	defer teardown()

	// Wait for TUI to finish scanning and display initial UI tree
	WaitForTexts(t, tm.Output(), []string{"Auto-detected: vitest", "src"}, 5*time.Second)

	// Send 'j' key to move cursor down from directory 'src' to test file 'stringUtils.test.ts'
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'j'},
	})

	// Send 'enter' key to trigger execution of selected test file
	tm.Send(tea.KeyMsg{
		Type: tea.KeyEnter,
	})

	// Assert that execution completes and output stream displays test results or status
	WaitForText(t, tm.Output(), "stringUtils.test.ts", 8*time.Second)

	// Switch tabs / panes using ']' or 'tab'
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{']'},
	})

	// Verify tab switch to Watched tab
	WaitForText(t, tm.Output(), "Watched", 3*time.Second)
}
