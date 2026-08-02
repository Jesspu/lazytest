package e2e

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/jesspatton/lazytest/engine"
	"github.com/jesspatton/lazytest/ui"
)

func TestErrorNotification_GitError(t *testing.T) {
	// Create an isolated temporary directory that is not inside any git repo
	nonGitDir := t.TempDir()

	eng := engine.New(nonGitDir)
	model := ui.NewModel(eng)
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(100, 30))
	defer func() {
		tm.Send(tea.QuitMsg{})
		tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
		eng.Close()
	}()

	// Wait for TUI initialization
	time.Sleep(300 * time.Millisecond)

	// Send 'a' key in manual mode (AddRelated) when directory is not a git repository
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'a'},
	})

	// Verify error notification appears in UI
	WaitForText(t, tm.Output(), "Failed to get changed files", 3*time.Second)
}

func TestNotification_DirectMessageRendering(t *testing.T) {
	tm, teardown := SetupTestEnv(t, "single_repo_jest")
	defer teardown()

	// Wait for initial UI render
	WaitForText(t, tm.Output(), "src", 5*time.Second)

	// Send a custom NotificationMsg into the Bubbletea event loop
	tm.Send(engine.NotificationMsg{
		Message: "Custom E2E Toast Notification",
		IsError: true,
	})

	// Verify notification appears in output
	WaitForText(t, tm.Output(), "Custom E2E Toast Notification", 3*time.Second)
}
