# Spec Plan: Batching Watcher Debounce Events

## Problem Statement

In `filesystem/watcher.go`, file modification events are debounced using `time.AfterFunc` with a single timer instance:

```go
if timer != nil {
    timer.Stop()
}
timer = time.AfterFunc(debounceDuration, func() {
    w.Events <- event.Name
})
```

When multiple files are modified in rapid succession within the 100ms debounce window (common during `git checkout`, rebase, or saving multiple files in an IDE), `timer.Stop()` cancels the timer scheduled for previous files. As a result, only the **last** modified file path is sent to `w.Events`, and all previous file change notifications are silently lost.

## Goals

1. Ensure all file modification events occurring within the debounce window are captured and processed.
2. Prevent event dropping during batch file changes (e.g. bulk saves, branch switching).
3. Maintain low-latency notification for single file edits without unnecessary delays.

## Proposed Changes

### 1. Update `Watcher` Struct (`filesystem/watcher.go`)

Add thread-safe tracking for pending paths during the debounce window.

```go
type Watcher struct {
    fsWatcher    *fsnotify.Watcher
    Events       chan string
    done         chan struct{}
    root         string
    mu           sync.Mutex
    pendingPaths map[string]struct{}
}
```

Initialize `pendingPaths` in `NewWatcher`:

```go
w := &Watcher{
    fsWatcher:    fsWatcher,
    Events:       make(chan string, 100), // Increase buffer capacity
    done:         make(chan struct{}),
    root:         root,
    pendingPaths: make(map[string]struct{}),
}
```

### 2. Update Debounce Logic (`filesystem/watcher.go`)

Collect all unique paths encountered during the debounce window and flush them when the timer expires.

```go
w.mu.Lock()
w.pendingPaths[event.Name] = struct{}{}
w.mu.Unlock()

if timer != nil {
    timer.Stop()
}

timer = time.AfterFunc(debounceDuration, func() {
    w.mu.Lock()
    pathsToEmit := make([]string, 0, len(w.pendingPaths))
    for path := range w.pendingPaths {
        pathsToEmit = append(pathsToEmit, path)
    }
    w.pendingPaths = make(map[string]struct{}) // Reset pending paths
    w.mu.Unlock()

    for _, path := range pathsToEmit {
        w.Events <- path
    }
})
```

### 3. (Optional) Batch Message Support in Engine (`engine/engine.go`)

Optionally update `WatcherMsg` in `engine/engine.go` or support receiving a slice of paths `BatchWatcherMsg([]string)` so the graph and queue updates can be processed in a single batch pass rather than loop iterations.

## Verification Plan

1. **Unit Tests (`filesystem/watcher_test.go`)**:
   - Write a unit test that writes to 5 different files within 10ms.
   - Verify that `w.Events` receives events for all 5 files after the 100ms debounce window.
2. **Integration Verification**:
   - Run `lazytest`, modify multiple files simultaneously via `touch src/file1.ts src/file2.ts`.
   - Verify all dependent test files are properly queued and executed.
