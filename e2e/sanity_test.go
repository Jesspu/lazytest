package e2e

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestSanityBootAndQuit(t *testing.T) {
	tm, teardown := SetupTestEnv(t, "single_repo_jest")
	defer teardown()

	// Wait for welcome banner or title in TUI output stream
	WaitForText(t, tm.Output(), "LazyTest", 3*time.Second)

	// Send 'q' key message to quit
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'q'},
	})

	// Wait for application process to finish
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
