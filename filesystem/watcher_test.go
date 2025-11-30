package filesystem

import (
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
