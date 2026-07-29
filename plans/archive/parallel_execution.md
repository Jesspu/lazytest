# Parallel Test Execution Plan

This document outlines the plan to implement parallel test execution in LazyTest, allowing multiple watched files to run tests concurrently with a configurable limit.

## Goals
- Allow multiple tests to run in parallel.
- Limit the number of concurrent tests to prevent resource exhaustion.
- Make the concurrency limit configurable via `.lazytest.json`.
- Maintain UI responsiveness and correctly route output to the corresponding file view.

## Proposed Changes

### 1. Configuration (`runner/config.go`)

Add a `MaxConcurrentTests` field to the `Config` struct.

```go
type Config struct {
    Command            string `json:"command"`
    MaxConcurrentTests int    `json:"max_concurrent_tests"`
}
```

- Default `MaxConcurrentTests` to `1` (or a sensible default like `runtime.NumCPU() / 2`) if not specified.

### 2. Runner Architecture (`runner/runner.go`)

Refactor `Runner` to handle multiple concurrent processes.

#### Struct Changes
```go
type Runner struct {
    mu          sync.Mutex
    runningCmds map[string]context.CancelFunc // Map file path to cancel func
    sem         chan struct{}                 // Semaphore for concurrency limit
    Output      chan OutputMsg                // Channel to stream output
    Status      chan StatusMsg                // Channel to report completion
}

type OutputMsg struct {
    FilePath string
    Content  string
}

type StatusMsg struct {
    FilePath string
    Err      error
}
```

#### Logic Changes
- **`NewRunner(maxConcurrent int)`**: Initialize the semaphore channel with buffer size `maxConcurrent`.
- **`Run(command string, args []string, cwd string, filePath string)`**:
    - Acquire semaphore: `r.sem <- struct{}{}`.
    - Start goroutine.
    - Inside goroutine:
        - Defer releasing semaphore: `<-r.sem`.
        - Store cancel func in `runningCmds`.
        - Execute command.
        - Stream output wrapping it in `OutputMsg`.
        - Send final status in `StatusMsg`.
        - Remove from `runningCmds` on completion.

### 3. UI Integration (`ui/model.go`)

Update the UI to handle multiplexed output and status messages.

- **`Update`**:
    - Handle `OutputMsg`: Append content to `m.testOutputs[msg.FilePath]`. If `msg.FilePath` is the currently selected file in the "Watched" tab (or the active file in Explorer), update the viewport.
    - Handle `StatusMsg`: Update `m.nodeStatus[msg.FilePath]`. If it's the current file, update the status icon/text.

### 4. User Interface

- The "Watched" tab will need to show the status of each file (Running, Pass, Fail) dynamically.
- Ideally, add a visual indicator for "Queued" if the semaphore is full.

## Implementation Steps

1.  **Update Config**: Add `MaxConcurrentTests` to `runner/config.go`.
2.  **Refactor Runner**:
    - Change `Output` and `Status` channels to carry `FilePath`.
    - Implement semaphore logic.
    - Update `Run` method signature.
3.  **Update UI**:
    - Adapt to new `Runner` API.
    - Handle multiplexed messages.
    - Ensure viewport updates only when relevant data arrives.
4.  **Verify**:
    - Run multiple watched files.
    - Verify they run in parallel (up to the limit).
    - Verify output is correctly routed.
