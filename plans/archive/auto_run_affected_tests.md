# Spec Plan: Auto-Run / Smart Mode for Affected Tests

## Problem Statement

Currently, when a file changes on disk, `engine/engine.go` only queues tests that are **explicitly manually added** to `e.State.Watched`:

```go
for watchedPath := range e.State.Watched {
    // Check if this watched file is affected
    ...
}
```

If a developer modifies `src/utils.ts` and `src/utils.test.ts` exists in the repository but has not been manually watched (`'w'`), no tests are queued or executed. The user must manually discover and watch tests before receiving feedback, which differs from standard watch runners (e.g. `jest --watch`, Wallaby.js).

## Goals

1. Introduce an **Auto-Run (Smart Mode)** feature that automatically executes all tests affected by changed files without requiring manual watching.
2. Provide a UI toggle (e.g., keybinding `S` or mode switch) to toggle between:
   - **Manual Watch Mode** (current behavior): Only run manually watched tests (`e.State.Watched`).
   - **Smart Mode** (auto-run affected): Automatically run all test files transitively affected by file modifications.
3. Persist mode selection in state or `.lazytest.json`.

## Proposed Changes

### 1. State Expansion (`engine/state.go` & `engine/engine.go`)

Add `SmartMode` boolean flag to `engine.State`:

```go
type State struct {
    // ...
    SmartMode bool // If true, automatically queue all affected test files on file change
}
```

### 2. Update Engine Watcher Handling (`engine/engine.go`)

Modify `Engine.Update(msg WatcherMsg)` to evaluate test queuing based on `SmartMode`:

```go
case WatcherMsg:
    path := string(msg)
    e.Graph.Update(path)

    var testsToQueue []string

    if e.State.SmartMode {
        // Smart Mode: Find ALL tests transitively affected by this path
        testsToQueue = e.FindRelatedTests(path)
    } else {
        // Manual Mode: Only queue watched tests that are affected
        dependents := e.Graph.GetDependents(path)
        for watchedPath := range e.State.Watched {
            if watchedPath == path || contains(dependents, watchedPath) {
                testsToQueue = append(testsToQueue, watchedPath)
            }
        }
    }

    // Queue resolved tests (deduplicated)
    for _, testPath := range testsToQueue {
        if !isQueued(e.State.Queue, testPath) {
            e.State.Queue = append(e.State.Queue, testPath)
        }
    }
```

### 3. UI Toggle & Indicator (`ui/model.go`, `ui/keys.go`, `ui/footer.go`)

- Add keybinding `ToggleSmartMode` (e.g., `s` or `Ctrl+S`).
- Display active mode badge in footer: `[MODE: SMART]` vs `[MODE: MANUAL]`.
- Provide visual distinction in explorer for auto-triggered vs. manually watched tests.

## Verification Plan

1. **Unit Tests (`engine/engine_test.go`)**:
   - Set `e.State.SmartMode = true`.
   - Send `WatcherMsg("src/utils.ts")`.
   - Verify `src/utils.test.ts` is automatically enqueued without calling `ToggleWatch("src/utils.test.ts")`.
2. **Manual UX Testing**:
   - Toggle Smart Mode on/off in the TUI.
   - Edit a source file and verify affected tests execute automatically in Smart Mode.
