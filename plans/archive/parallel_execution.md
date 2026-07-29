# Parallel Test Execution Plan

This document outlines the plan to implement parallel test execution in LazyTest, allowing multiple watched files to run tests concurrently with a configurable limit.

## Goals
- Allow multiple tests to run in parallel.
- Limit the number of concurrent tests to prevent resource exhaustion.
- Make the concurrency limit configurable via `.lazytest.json`.
- Maintain UI responsiveness and correctly route output to the corresponding test file.

## Proposed Changes

### 1. Configuration (`runner/config.go`)

Add a `MaxConcurrentTests` field to the `Config` struct.

```go
type Config struct {
    Command            string     `json:"command"`
    MaxConcurrentTests int        `json:"max_concurrent_tests"`
    // ... existing fields
}
```

- Default `MaxConcurrentTests` to `runtime.NumCPU() / 2` (min 1) if not specified or `0`.

### 2. Runner Architecture (`runner/runner.go`)

Refactor `Runner` to support executing multiple concurrent processes and identify output by file path.

#### Message Types
Update the message types to include the `FilePath` so the engine knows which test generated the update.

```go
type OutputUpdate struct {
    FilePath string
    Content  string
}

type StatusUpdate struct {
    FilePath string
    Err      error
}
```

#### Struct Changes
```go
type Runner struct {
    mu          sync.Mutex
    runningCmds map[string]context.CancelFunc // Map file path to cancel func
    Updates     chan Update                   // Single channel for ordered updates
}
```

#### Logic Changes
- **`NewRunner()`**: Initialize `runningCmds`.
- **`Run(command string, args []string, cwd string, filePath string)`**:
    - Remove the code that cancels the previous command.
    - Store the context cancel func in `runningCmds[filePath]`.
    - Pass `filePath` to `streamReader` so it can wrap output in `OutputUpdate{FilePath, ...}`.
    - Send `StatusUpdate{FilePath, ...}` upon completion.
    - Remove from `runningCmds` on completion.
- **`Kill(filePath string)`**: Add method to cancel a specific running test, replacing the existing `Kill()` which kills the current command.

### 3. Engine Architecture (`engine/state.go` & `engine/engine.go`)

Update the Engine to handle concurrency limits and state mapping for multiple running tests.

#### State Changes (`engine/state.go`)
- Remove `RunningNode *filesystem.Node` and `CurrentOutput string`.
- Add `RunningNodes map[string]*filesystem.Node` to track currently executing tests.
- `TestOutputs` map already exists and will be updated directly per file path.

#### Logic Changes (`engine/engine.go` & `engine/actions.go`)
- **Queue Processing**: In `Update()` when processing `runner.StatusUpdate` or queuing new tests, pop from `e.State.Queue` up to `e.ProjectConfig.MaxConcurrentTests` and trigger them.
- **Output Handling**: In `Update()` for `runner.OutputUpdate`, append to `e.State.TestOutputs[msg.FilePath]`.
- **Status Handling**: In `Update()` for `runner.StatusUpdate`, update `e.State.NodeStatus[msg.FilePath]`, remove from `RunningNodes`, and process the queue.
- **TriggerTest (`actions.go`)**: Update to add the node to `RunningNodes`, clear previous output in `TestOutputs`, and call `e.runner.Run(..., node.Path)`.

### 4. UI Integration

- Ensure the UI (e.g. `ui/explorer.go`, `ui/sync.go`) correctly queries `engine.State.NodeStatus` and `engine.State.TestOutputs` to display the state and output for the selected file, as there is no longer a single global `CurrentOutput`.

## Implementation Steps

1.  **Update Config**: Add `MaxConcurrentTests` to `runner/config.go`.
2.  **Refactor Runner**:
    - Change `OutputUpdate` and `StatusUpdate` to a struct carrying `FilePath`.
    - Update `Runner` to track multiple commands using a map.
    - Update `Run` method signature and logic.
3.  **Refactor Engine State**:
    - Replace `RunningNode` and `CurrentOutput` with `RunningNodes` map.
    - Update Engine's `Update` loop to handle the new `FilePath` mapping in updates.
    - Implement concurrency limiting in Engine's queue processing logic.
4.  **Refactor Engine Actions & UI**:
    - Fix all compile errors in `actions.go` and `ui/` packages related to `RunningNode` and `CurrentOutput` removal.
5.  **Verify**:
    - Run multiple watched files.
    - Verify they run in parallel (up to the limit).
    - Verify output is correctly routed.
