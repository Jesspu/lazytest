package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcher(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-watcher-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close()

	// Wait for watcher to start up
	time.Sleep(100 * time.Millisecond)

	// Create a file
	testFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for event
	select {
	case event := <-w.Events:
		if event != testFile {
			t.Errorf("expected event for %s, got %s", testFile, event)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for file creation event")
	}

	// Test ignore logic
	logFile := filepath.Join(tmpDir, "app.log")
	if err := os.WriteFile(logFile, []byte("log"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-w.Events:
		t.Errorf("unexpected event for ignored file: %s", event)
	case <-time.After(500 * time.Millisecond):
		// Success, no event received
	}
}

func TestWatcherAllowlist(t *testing.T) {
	tests := []struct {
		path      string
		wantEvent bool
		isConfig  bool
	}{
		{"/path/to/test.ts", true, false},
		{"/path/to/package.json", true, true},
		{"/path/to/vite.config.ts", true, true},
		{"/path/to/app.log", false, false},
		{"/path/to/README.md", false, false},
		{"/path/to/node_modules/pkg/index.js", true, false}, // Technically allowed by file extension, but usually ignored by walker
	}

	for _, tt := range tests {
		isSource := IsSourceFile(tt.path)
		isConfig := IsConfigFile(tt.path)

		gotEvent := isSource || isConfig

		if gotEvent != tt.wantEvent {
			t.Errorf("File %s: want event=%v, got source=%v, config=%v", tt.path, tt.wantEvent, isSource, isConfig)
		}
	}
}

// TestWatcher_BatchDebounce verifies that all file paths modified within a single
// debounce window are emitted — none are silently dropped.
func TestWatcher_BatchDebounce(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-watcher-batch")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close()

	// Give the watcher time to initialise its fsnotify watch.
	time.Sleep(100 * time.Millisecond)

	// Write 5 distinct source files in rapid succession (<10ms total).
	const numFiles = 5
	filePaths := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		p := filepath.Join(tmpDir, fmt.Sprintf("file%d.ts", i))
		filePaths[i] = p
		if err := os.WriteFile(p, []byte("export {}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Collect all events that arrive within a generous timeout window.
	// The debounce is 100ms, so all 5 events should arrive together shortly after.
	received := make(map[string]bool)
	deadline := time.After(2 * time.Second)

collect:
	for len(received) < numFiles {
		select {
		case path := <-w.Events:
			received[path] = true
		case <-deadline:
			break collect
		}
	}

	// Verify every file was received.
	for _, p := range filePaths {
		if !received[p] {
			t.Errorf("Event not received for %s (got %d/%d events: %v)", p, len(received), numFiles, received)
		}
	}
}
