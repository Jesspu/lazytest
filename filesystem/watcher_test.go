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
	testFile := filepath.Join(tmpDir, "test.txt")
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

func TestShouldIgnore(t *testing.T) {
	w := &Watcher{}

	tests := []struct {
		path string
		want bool
	}{
		{"/path/to/.git", true},
		{"/path/to/node_modules", true},
		{"/path/to/normal.go", false},
		{"/path/to/app.log", true},
		{"/path/to/dist", true},
	}

	for _, tt := range tests {
		if got := w.shouldIgnore(tt.path); got != tt.want {
			t.Errorf("shouldIgnore(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
