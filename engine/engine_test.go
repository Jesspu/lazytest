package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jesspatton/lazytest/filesystem"
	"github.com/jesspatton/lazytest/runner"
)

func TestNewEngine(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-engine-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	e := New(tmpDir)

	if e.State.RootPath != tmpDir {
		t.Errorf("Expected RootPath %s, got %s", tmpDir, e.State.RootPath)
	}
	if e.runner == nil {
		t.Error("Expected runner to be initialized")
	}
	if e.Graph == nil {
		t.Error("Expected Graph to be initialized")
	}
}

func TestToggleWatch(t *testing.T) {
	e := New("/tmp")

	path := "/tmp/foo.test.js"
	e.ToggleWatch(path)

	if !e.IsWatched(path) {
		t.Error("Expected file to be watched")
	}

	e.ToggleWatch(path)

	if e.IsWatched(path) {
		t.Error("Expected file to be unwatched")
	}
}

func TestTriggerTest(t *testing.T) {
	// Setup temp dir with package.json and test file
	tmpDir, err := os.MkdirTemp("", "lazytest-trigger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(tmpDir, "foo.test.js")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create .lazytest.json to use echo
	config := `{"command": "echo test run"}`
	err = os.WriteFile(filepath.Join(tmpDir, ".lazytest.json"), []byte(config), 0644)
	if err != nil {
		t.Fatal(err)
	}

	e := New(tmpDir)
	node := &filesystem.Node{
		Path: testFile,
		Name: "foo.test.js",
	}

	// Trigger test
	cmd := e.TriggerTest(node)
	if cmd == nil {
		t.Fatal("Expected TriggerTest to return a command")
	}

	// Verify initial state
	if e.State.RunningNode != node {
		t.Error("Expected RunningNode to be set")
	}
	if status, _ := e.GetNodeStatus(testFile); status != StatusRunning {
		t.Errorf("Expected status Running, got %v", status)
	}

	// Execute the command (this runs runner.Run in a goroutine usually, but here we just call the function returned by tea.Cmd)
	// The tea.Cmd returned by TriggerTest is: func() tea.Msg { e.runner.Run(...); return nil }
	// So calling it will start the runner.
	go cmd()

	// Wait for updates from runner
	timeout := time.After(2 * time.Second)

	// We need to simulate the event loop processing updates
	done := make(chan bool)
	go func() {
		for {
			select {
			case update := <-e.runner.Updates:
				// Feed update back to engine
				switch u := update.(type) {
				case runner.OutputUpdate:
					e.Update(u)
				case runner.StatusUpdate:
					e.Update(u)
					done <- true
					return
				}
			case <-timeout:
				return
			}
		}
	}()

	select {
	case <-done:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("Timed out waiting for test to complete")
	}

	// Verify final state
	if status, _ := e.GetNodeStatus(testFile); status != StatusPass {
		t.Errorf("Expected status Pass, got %v", status)
	}

	output, _ := e.GetTestOutput(testFile)
	if output == "" {
		t.Error("Expected test output")
	}
}

func TestUpdateLoop(t *testing.T) {
	e := New("/tmp")
	node := &filesystem.Node{Path: "/tmp/foo.test.js", Name: "foo.test.js"}
	e.State.RunningNode = node
	e.State.TestOutputs[node.Path] = ""

	// Simulate OutputUpdate
	msg := runner.OutputUpdate("hello")
	e.Update(msg)

	if e.State.CurrentOutput != "hello\n" {
		t.Errorf("Expected output 'hello\\n', got '%s'", e.State.CurrentOutput)
	}

	// Simulate StatusUpdate (Pass)
	statusMsg := runner.StatusUpdate{Err: nil}
	e.Update(statusMsg)

	if status, _ := e.GetNodeStatus(node.Path); status != StatusPass {
		t.Errorf("Expected status Pass, got %v", status)
	}
	if e.State.RunningNode != nil {
		t.Error("Expected RunningNode to be nil after finish")
	}
}

func TestSmartQueueing(t *testing.T) {
	e := New("/tmp")

	// Watch three test files
	test1 := "/tmp/app.test.js"
	test2 := "/tmp/utils.test.js"
	test3 := "/tmp/unrelated.test.js"

	e.ToggleWatch(test1)
	e.ToggleWatch(test2)
	e.ToggleWatch(test3)

	// Verify all are watched
	if !e.IsWatched(test1) || !e.IsWatched(test2) || !e.IsWatched(test3) {
		t.Fatal("Expected all files to be watched")
	}

	// Set a running node so the queue won't be consumed immediately
	e.State.RunningNode = &filesystem.Node{Path: "/tmp/dummy.test.js"}

	// Simulate a change to test1 (which should only affect test1 itself)
	// Since we don't have a real graph with dependencies, this will queue only test1
	msg := WatcherMsg(test1)
	_ = e.Update(msg) // Call Update, which returns a tea.Cmd

	// Verify only test1 is queued (not test2 or test3)
	if len(e.State.Queue) != 1 {
		t.Errorf("Expected queue length 1, got %d. Queue: %v", len(e.State.Queue), e.State.Queue)
	}
	if len(e.State.Queue) > 0 && e.State.Queue[0] != test1 {
		t.Errorf("Expected queue to contain %s, got %s", test1, e.State.Queue[0])
	}

	// Verify that test1 is NOT queued again if we send the same message
	msg = WatcherMsg(test1)
	_ = e.Update(msg)
	if len(e.State.Queue) != 1 {
		t.Errorf("Expected queue to remain length 1 (deduplication), got %d", len(e.State.Queue))
	}

	// Clear the queue
	e.State.Queue = []string{}

	// Now simulate a change to a file that isn't watched
	// This should queue nothing (since no watched files depend on it in our empty graph)
	msg = WatcherMsg("/tmp/some-source.js")
	_ = e.Update(msg)

	if len(e.State.Queue) != 0 {
		t.Errorf("Expected queue to be empty for unrelated file change, got %d items: %v", len(e.State.Queue), e.State.Queue)
	}
}
