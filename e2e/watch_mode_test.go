package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWatchMode(t *testing.T) {
	tm, teardown := SetupTestEnv(t, "single_repo_vitest")
	defer teardown()

	// Wait for TUI to load tree
	WaitForTexts(t, tm.Output(), []string{"Auto-detected: vitest", "src"}, 5*time.Second)

	// Move cursor to stringUtils.test.ts ('j' from 'src' directory node)
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'j'},
	})

	// Send 'w' key to toggle watch mode on stringUtils.test.ts
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'w'},
	})

	// Verify watch eye icon indicator '👁' appears in output stream
	WaitForText(t, tm.Output(), "👁", 3*time.Second)

	// Locate stringUtils.test.ts in single_repo_vitest fixture
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("TestWatchMode: unable to resolve test path")
	}
	repoRoot := filepath.Dir(filepath.Dir(filename))
	testFilePath := filepath.Join(repoRoot, "test_projects", "single_repo_vitest", "src", "stringUtils.test.ts")

	originalContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("TestWatchMode: failed to read %s: %v", testFilePath, err)
	}
	defer func() {
		_ = os.WriteFile(testFilePath, originalContent, 0644)
	}()

	// Allow fsnotify watcher initialization to complete
	time.Sleep(500 * time.Millisecond)

	// Simulate external modification on watched file stringUtils.test.ts
	modifiedContent := append(originalContent, []byte("\n// trigger watch mode file change\n")...)
	if err := os.WriteFile(testFilePath, modifiedContent, 0644); err != nil {
		t.Fatalf("TestWatchMode: failed to write %s: %v", testFilePath, err)
	}

	// Verify lazytest detects watched file update and re-runs test
	WaitForText(t, tm.Output(), "stringUtils.test.ts", 8*time.Second)
}
