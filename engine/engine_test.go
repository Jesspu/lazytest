package engine

import (
	"os"
	"path/filepath"
	"strings"
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
	e.ProjectConfig.MaxConcurrentTests = 0

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
	e.ProjectConfig.MaxConcurrentTests = 0

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
	e.ProjectConfig.MaxConcurrentTests = 0
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
	if len(e.State.RunningNodes) == 0 {
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
	e.ProjectConfig.MaxConcurrentTests = 0
	node := &filesystem.Node{Path: "/tmp/foo.test.js", Name: "foo.test.js"}
	e.State.RunningNodes[node.Path] = node
	e.State.TestOutputs[node.Path] = ""

	// Simulate OutputUpdate
	msg := runner.OutputUpdate{FilePath: node.Path, Content: "hello"}
	e.Update(msg)

	if e.State.TestOutputs[node.Path] != "hello\n" {
		t.Errorf("Expected output 'hello\\n', got '%s'", e.State.TestOutputs[node.Path])
	}

	// Simulate StatusUpdate (Pass)
	statusMsg := runner.StatusUpdate{FilePath: node.Path, Err: nil}
	e.Update(statusMsg)

	if status, _ := e.GetNodeStatus(node.Path); status != StatusPass {
		t.Errorf("Expected status Pass, got %v", status)
	}
	if len(e.State.RunningNodes) > 0 {
		t.Error("Expected RunningNode to be nil after finish")
	}
}

func TestSmartQueueing(t *testing.T) {
	e := New("/tmp")
	e.ProjectConfig.MaxConcurrentTests = 0

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
	e.State.RunningNodes["/tmp/dummy.test.js"] = &filesystem.Node{Path: "/tmp/dummy.test.js"}

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

// TestFindRelatedTests_SelfInclusion verifies that FindRelatedTests returns the
// path itself when it is already a test file.
func TestFindRelatedTests_SelfInclusion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-related-self")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "foo.test.ts")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	e := New(tmpDir)
	e.ProjectConfig.MaxConcurrentTests = 0
	e.Graph.Build(tmpDir)

	related := e.FindRelatedTests(testFile)

	if len(related) != 1 || related[0] != testFile {
		t.Errorf("FindRelatedTests(%q) = %v, want [%q]", testFile, related, testFile)
	}
}

// TestFindRelatedTests_SourceFile verifies that FindRelatedTests returns the
// dependent test file when the changed path is a source (non-test) file.
func TestFindRelatedTests_SourceFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-related-source")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sourceFile := filepath.Join(tmpDir, "foo.ts")
	testFile := filepath.Join(tmpDir, "foo.test.ts")

	if err := os.WriteFile(sourceFile, []byte("export const x = 1;"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile, []byte("import { x } from './foo';"), 0644); err != nil {
		t.Fatal(err)
	}

	e := New(tmpDir)
	e.ProjectConfig.MaxConcurrentTests = 0
	e.Graph.Build(tmpDir)

	related := e.FindRelatedTests(sourceFile)

	if len(related) != 1 || related[0] != testFile {
		t.Errorf("FindRelatedTests(%q) = %v, want [%q]", sourceFile, related, testFile)
	}
}

// TestFindRelatedTests_NoDuplicates verifies that when a test file is both the
// queried path and imported by another test file, both are returned without duplicates.
func TestFindRelatedTests_NoDuplicates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-related-dedup")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// foo.test.ts is the changed file (a test file itself).
	// bar.test.ts imports foo.test.ts.
	fooTest := filepath.Join(tmpDir, "foo.test.ts")
	barTest := filepath.Join(tmpDir, "bar.test.ts")

	if err := os.WriteFile(fooTest, []byte("export const helper = () => {};"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(barTest, []byte("import { helper } from './foo.test';"), 0644); err != nil {
		t.Fatal(err)
	}

	e := New(tmpDir)
	e.ProjectConfig.MaxConcurrentTests = 0
	e.Graph.Build(tmpDir)

	related := e.FindRelatedTests(fooTest)

	// Expect exactly 2 unique entries: fooTest (self) + barTest (transitive).
	seen := make(map[string]bool)
	for _, r := range related {
		if seen[r] {
			t.Errorf("FindRelatedTests returned duplicate entry: %s", r)
		}
		seen[r] = true
	}

	if !seen[fooTest] {
		t.Errorf("FindRelatedTests missing self-inclusion of %s; got %v", fooTest, related)
	}
	if !seen[barTest] {
		t.Errorf("FindRelatedTests missing transitive dependent %s; got %v", barTest, related)
	}
	if len(related) != 2 {
		t.Errorf("FindRelatedTests returned %d results, want 2: %v", len(related), related)
	}
}

// TestSmartMode verifies that when SmartMode is enabled, a WatcherMsg for a
// source file automatically enqueues its dependent test file without requiring
// it to be manually watched.
func TestSmartMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-smartmode")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source and test files with an import relationship
	sourceFile := filepath.Join(tmpDir, "utils.ts")
	testFile := filepath.Join(tmpDir, "utils.test.ts")

	if err := os.WriteFile(sourceFile, []byte("export const add = (a: number, b: number) => a + b;"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile, []byte("import { add } from './utils';"), 0644); err != nil {
		t.Fatal(err)
	}

	e := New(tmpDir)
	e.ProjectConfig.MaxConcurrentTests = 0
	// Build the dependency graph so the relationship is known
	e.Graph.Build(tmpDir)

	// Enable Smart Mode — testFile is NOT manually watched
	e.ToggleSmartMode()
	if !e.IsSmartMode() {
		t.Fatal("Expected SmartMode to be true")
	}

	// Pin a running node so the queue won't be consumed immediately
	e.State.RunningNodes["/tmp/dummy.test.ts"] = &filesystem.Node{Path: "/tmp/dummy.test.ts"}

	// Simulate a file-change event on the source file
	_ = e.Update(WatcherMsg(sourceFile))

	// The test file should have been auto-queued even though it was never watched
	if len(e.State.Queue) != 1 {
		t.Fatalf("Expected queue length 1 in Smart Mode, got %d: %v", len(e.State.Queue), e.State.Queue)
	}
	if e.State.Queue[0] != testFile {
		t.Errorf("Expected %s in queue, got %s", testFile, e.State.Queue[0])
	}
}

// TestSmartModeToggle verifies that ToggleSmartMode flips the flag and
// IsSmartMode reflects the current state correctly.
func TestSmartModeToggle(t *testing.T) {
	e := New("/tmp")
	e.ProjectConfig.MaxConcurrentTests = 0

	if e.IsSmartMode() {
		t.Error("Expected SmartMode to start as false")
	}

	e.ToggleSmartMode()
	if !e.IsSmartMode() {
		t.Error("Expected SmartMode to be true after first toggle")
	}

	e.ToggleSmartMode()
	if e.IsSmartMode() {
		t.Error("Expected SmartMode to be false after second toggle")
	}
}

func TestConfigChangeHandling(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-config-change")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, ".lazytest.json")
	if err := os.WriteFile(configFile, []byte(`{"command": "jest"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "lazytest"}`), 0644); err != nil {
		t.Fatal(err)
	}

	test1 := filepath.Join(tmpDir, "app.test.js")
	test2 := filepath.Join(tmpDir, "utils.test.js")
	if err := os.WriteFile(test1, []byte("test 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(test2, []byte("test 2"), 0644); err != nil {
		t.Fatal(err)
	}

	e := New(tmpDir)
	e.ProjectConfig.MaxConcurrentTests = 0
	if e.ProjectConfig.Command != "jest" {
		t.Fatalf("Expected initial command 'jest', got '%s'", e.ProjectConfig.Command)
	}

	e.ToggleWatch(test1)
	e.ToggleWatch(test2)

	// Pin a running node so the queue won't be consumed immediately
	e.State.RunningNodes["/tmp/dummy.test.js"] = &filesystem.Node{Path: "/tmp/dummy.test.js"}

	// Modify config on disk
	if err := os.WriteFile(configFile, []byte(`{"command": "vitest", "max_concurrent_tests": 0}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Trigger config change event
	_ = e.Update(WatcherMsg(configFile))

	// Verify config reload
	if e.ProjectConfig.Command != "vitest" {
		t.Errorf("Expected reloaded command 'vitest', got '%s'", e.ProjectConfig.Command)
	}

	// Verify both watched test files are enqueued for re-run
	// Depending on MaxConcurrentTests, the tests might be queued or running
	totalQueuedOrRunning := len(e.State.Queue) + len(e.State.RunningNodes)
	if totalQueuedOrRunning != 3 { // 2 watched tests + 1 dummy
		t.Fatalf("Expected 3 total queued/running tests, got %d", totalQueuedOrRunning)
	}


	// Verify output message on the dummy node which wasn't re-triggered
	expectedMsg := "Config change detected (.lazytest.json). Reloaded settings and re-queued tests."
	if !strings.Contains(e.State.TestOutputs["/tmp/dummy.test.js"], expectedMsg) {
		t.Errorf("Expected output to contain %q, got %q", expectedMsg, e.State.TestOutputs["/tmp/dummy.test.js"])
	}
}

func TestConfigChangeHandling_SmartMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-config-smart")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pkgFile := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(pkgFile, []byte(`{"name": "lazytest-test", "max_concurrent_tests": 0}`), 0644); err != nil {
		t.Fatal(err)
	}

	test1 := filepath.Join(tmpDir, "bar.test.ts")
	test2 := filepath.Join(tmpDir, "foo.test.ts")
	if err := os.WriteFile(test1, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(test2, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	e := New(tmpDir)
	e.ProjectConfig.MaxConcurrentTests = 0
	e.Graph.Build(tmpDir)

	e.ToggleSmartMode()
	if !e.IsSmartMode() {
		t.Fatal("Expected SmartMode to be enabled")
	}

	// No watched files in Smart Mode
	if len(e.State.Watched) != 0 {
		t.Fatalf("Expected 0 watched files, got %d", len(e.State.Watched))
	}

	// Pin a running node
	e.State.RunningNodes["/tmp/dummy.test.ts"] = &filesystem.Node{Path: "/tmp/dummy.test.ts"}

	_ = e.Update(WatcherMsg(pkgFile))

	// Verify both test files discovered in the graph are enqueued
	totalQueuedOrRunning := len(e.State.Queue) + len(e.State.RunningNodes)
	if totalQueuedOrRunning != 3 { // 2 smart tests + 1 dummy
		t.Fatalf("Expected 3 total queued/running tests, got %d", totalQueuedOrRunning)
	}


	expectedMsg := "Config change detected (package.json). Reloaded settings and re-queued tests."
	if !strings.Contains(e.State.TestOutputs["/tmp/dummy.test.ts"], expectedMsg) {
		t.Errorf("Expected output to contain %q, got %q", expectedMsg, e.State.TestOutputs["/tmp/dummy.test.ts"])
	}
}

// ---------------------------------------------------------------------------
// Phase 1: Affected Suite tests
// ---------------------------------------------------------------------------

// TestGetAffectedSuite_Sorting verifies that GetAffectedSuite returns paths
// ordered: Fail → Running → Pass → Idle, with alphabetical tie-breaking.
func TestGetAffectedSuite_Sorting(t *testing.T) {
	e := New("/tmp")
	e.ProjectConfig.MaxConcurrentTests = 0

	fail := "/tmp/fail.test.js"
	running := "/tmp/running.test.js"
	pass := "/tmp/pass.test.js"
	idle := "/tmp/idle.test.js"

	for _, p := range []string{pass, idle, running, fail} {
		e.State.Affected[p] = struct{}{}
	}
	e.State.NodeStatus[fail] = StatusFail
	e.State.NodeStatus[running] = StatusRunning
	e.State.NodeStatus[pass] = StatusPass
	// idle has no status set (StatusIdle / zero value)

	suite := e.GetAffectedSuite()

	if len(suite) != 4 {
		t.Fatalf("Expected 4 items, got %d: %v", len(suite), suite)
	}
	if suite[0] != fail {
		t.Errorf("Expected first item to be fail, got %s", suite[0])
	}
	if suite[1] != running {
		t.Errorf("Expected second item to be running, got %s", suite[1])
	}
	if suite[2] != pass {
		t.Errorf("Expected third item to be pass, got %s", suite[2])
	}
	if suite[3] != idle {
		t.Errorf("Expected fourth item to be idle, got %s", suite[3])
	}
}

// TestGetAffectedSuite_AlphabeticalTieBreak verifies that two tests in the
// same status bucket are sorted alphabetically.
func TestGetAffectedSuite_AlphabeticalTieBreak(t *testing.T) {
	e := New("/tmp")
	e.ProjectConfig.MaxConcurrentTests = 0

	a := "/tmp/alpha.test.js"
	b := "/tmp/beta.test.js"

	e.State.Affected[b] = struct{}{}
	e.State.Affected[a] = struct{}{}
	e.State.NodeStatus[a] = StatusFail
	e.State.NodeStatus[b] = StatusFail

	suite := e.GetAffectedSuite()
	if len(suite) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(suite))
	}
	if suite[0] != a || suite[1] != b {
		t.Errorf("Expected alphabetical order [%s, %s], got %v", a, b, suite)
	}
}

// TestGetSuiteStats verifies exact counts for passed, failed, and running.
func TestGetSuiteStats(t *testing.T) {
	e := New("/tmp")
	e.ProjectConfig.MaxConcurrentTests = 0

	paths := []struct {
		path   string
		status TestStatus
	}{
		{"/tmp/a.test.js", StatusPass},
		{"/tmp/b.test.js", StatusPass},
		{"/tmp/c.test.js", StatusFail},
		{"/tmp/d.test.js", StatusRunning},
		{"/tmp/e.test.js", StatusIdle},
	}

	for _, p := range paths {
		e.State.Affected[p.path] = struct{}{}
		e.State.NodeStatus[p.path] = p.status
	}

	passed, failed, running := e.GetSuiteStats()
	if passed != 2 {
		t.Errorf("Expected 2 passed, got %d", passed)
	}
	if failed != 1 {
		t.Errorf("Expected 1 failed, got %d", failed)
	}
	if running != 1 {
		t.Errorf("Expected 1 running, got %d", running)
	}
}

// TestClearAffectedSuite verifies that only failing/running tests are kept.
func TestClearAffectedSuite(t *testing.T) {
	e := New("/tmp")
	e.ProjectConfig.MaxConcurrentTests = 0

	fail := "/tmp/fail.test.js"
	running := "/tmp/running.test.js"
	pass := "/tmp/pass.test.js"
	idle := "/tmp/idle.test.js"

	for _, p := range []string{fail, running, pass, idle} {
		e.State.Affected[p] = struct{}{}
	}
	e.State.NodeStatus[fail] = StatusFail
	e.State.NodeStatus[running] = StatusRunning
	e.State.NodeStatus[pass] = StatusPass

	e.ClearAffectedSuite()

	if _, ok := e.State.Affected[fail]; !ok {
		t.Error("Expected failing test to remain after ClearAffectedSuite")
	}
	if _, ok := e.State.Affected[running]; !ok {
		t.Error("Expected running test to remain after ClearAffectedSuite")
	}
	if _, ok := e.State.Affected[pass]; ok {
		t.Error("Expected passing test to be removed by ClearAffectedSuite")
	}
	if _, ok := e.State.Affected[idle]; ok {
		t.Error("Expected idle test to be removed by ClearAffectedSuite")
	}
}

// TestRunSuiteFailures verifies that only failing tests are enqueued.
func TestRunSuiteFailures(t *testing.T) {
	e := New("/tmp")
	e.ProjectConfig.MaxConcurrentTests = 0

	fail1 := "/tmp/a_fail.test.js"
	fail2 := "/tmp/b_fail.test.js"
	pass := "/tmp/pass.test.js"
	idle := "/tmp/idle.test.js"

	for _, p := range []string{fail1, fail2, pass, idle} {
		e.State.Affected[p] = struct{}{}
	}
	e.State.NodeStatus[fail1] = StatusFail
	e.State.NodeStatus[fail2] = StatusFail
	e.State.NodeStatus[pass] = StatusPass

	// Pin a running node so nothing fires immediately
	e.State.RunningNodes["/tmp/dummy.test.js"] = &filesystem.Node{Path: "/tmp/dummy.test.js"}

	e.RunSuiteFailures()

	if len(e.State.Queue) != 2 {
		t.Fatalf("Expected 2 items in queue (only failures), got %d: %v", len(e.State.Queue), e.State.Queue)
	}
	if e.State.Queue[0] != fail1 || e.State.Queue[1] != fail2 {
		t.Errorf("Expected queue [%s, %s], got %v", fail1, fail2, e.State.Queue)
	}
}

// TestRunAffectedSuite verifies that every test in the Affected suite is enqueued.
func TestRunAffectedSuite(t *testing.T) {
	e := New("/tmp")
	e.ProjectConfig.MaxConcurrentTests = 0

	a := "/tmp/a.test.js"
	b := "/tmp/b.test.js"
	c := "/tmp/c.test.js"

	for _, p := range []string{a, b, c} {
		e.State.Affected[p] = struct{}{}
	}
	e.State.NodeStatus[a] = StatusPass
	e.State.NodeStatus[b] = StatusFail

	// Pin a running node so nothing fires immediately
	e.State.RunningNodes["/tmp/dummy.test.js"] = &filesystem.Node{Path: "/tmp/dummy.test.js"}

	e.RunAffectedSuite()

	if len(e.State.Queue) != 3 {
		t.Fatalf("Expected all 3 items queued, got %d: %v", len(e.State.Queue), e.State.Queue)
	}

	found := make(map[string]bool)
	for _, q := range e.State.Queue {
		found[q] = true
	}
	for _, p := range []string{a, b, c} {
		if !found[p] {
			t.Errorf("Expected %s in queue, not found: %v", p, e.State.Queue)
		}
	}
}

// TestAffectedPopulatedOnEnqueue verifies that tests added to the queue via a
// WatcherMsg are automatically recorded in State.Affected.
func TestAffectedPopulatedOnEnqueue(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-affected-enqueue")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sourceFile := filepath.Join(tmpDir, "utils.ts")
	testFile := filepath.Join(tmpDir, "utils.test.ts")

	if err := os.WriteFile(sourceFile, []byte("export const x = 1;"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile, []byte("import { x } from './utils';"), 0644); err != nil {
		t.Fatal(err)
	}

	e := New(tmpDir)
	e.ProjectConfig.MaxConcurrentTests = 0
	e.Graph.Build(tmpDir)
	e.ToggleSmartMode()

	// Pin a running node so the queue won't be consumed immediately
	e.State.RunningNodes["/tmp/dummy.test.ts"] = &filesystem.Node{Path: "/tmp/dummy.test.ts"}

	_ = e.Update(WatcherMsg(sourceFile))

	if _, ok := e.State.Affected[testFile]; !ok {
		t.Errorf("Expected %s to be in State.Affected after enqueue, got %v", testFile, e.State.Affected)
	}
}

func TestHandleTreeLoaded(t *testing.T) {
	e := New("/tmp")
	tree := &filesystem.Node{Name: "root", Path: "/tmp", IsDir: true}
	cmd := e.Update(TreeLoadedMsg(tree))
	if cmd != nil {
		t.Error("Expected handleTreeLoaded to return nil tea.Cmd")
	}
	if e.State.Tree != tree {
		t.Errorf("Expected e.State.Tree to be %v, got %v", tree, e.State.Tree)
	}
}

func TestHandleWatcherReady(t *testing.T) {
	e := New("/tmp")
	w, err := filesystem.NewWatcher("/tmp")
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Close()

	cmd := e.Update(WatcherReadyMsg{watcher: w})
	if cmd == nil {
		t.Error("Expected handleWatcherReady to return non-nil tea.Cmd")
	}
	if e.watcher != w {
		t.Errorf("Expected e.watcher to be %v, got %v", w, e.watcher)
	}
}

func TestUpdate_UnknownMsg(t *testing.T) {
	e := New("/tmp")
	type unknownMsg struct{}
	cmd := e.Update(unknownMsg{})
	if cmd != nil {
		t.Errorf("Expected nil tea.Cmd for unknown message type, got %v", cmd)
	}
}
