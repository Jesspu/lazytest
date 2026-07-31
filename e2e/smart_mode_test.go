package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSmartMode(t *testing.T) {
	tm, teardown := SetupTestEnv(t, "single_repo_jest")
	defer teardown()

	// Wait for TUI to load tree
	WaitForTexts(t, tm.Output(), []string{"Auto-detected: jest", "src"}, 5*time.Second)

	// Send 's' key to toggle Smart Mode ON
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'s'},
	})

	// Verify SMART MODE indicator appears in footer/badge
	WaitForText(t, tm.Output(), "SMART MODE", 3*time.Second)

	// Locate src/math.js in single_repo_jest fixture
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("TestSmartMode: unable to resolve test path")
	}
	repoRoot := filepath.Dir(filepath.Dir(filename))
	mathSrcPath := filepath.Join(repoRoot, "test_projects", "single_repo_jest", "src", "math.js")

	originalContent, err := os.ReadFile(mathSrcPath)
	if err != nil {
		t.Fatalf("TestSmartMode: failed to read %s: %v", mathSrcPath, err)
	}
	defer func() {
		_ = os.WriteFile(mathSrcPath, originalContent, 0644)
	}()

	// Allow fsnotify watcher initialization to complete
	time.Sleep(500 * time.Millisecond)

	// Simulate external file modification on math.js
	modifiedContent := append(originalContent, []byte("\n// trigger smart mode change\n")...)
	if err := os.WriteFile(mathSrcPath, modifiedContent, 0644); err != nil {
		t.Fatalf("TestSmartMode: failed to write %s: %v", mathSrcPath, err)
	}

	// Verify that graph analysis detects math.js modification and automatically triggers math.test.js
	WaitForText(t, tm.Output(), "math.test.js", 8*time.Second)
}
