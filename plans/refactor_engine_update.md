# Epic: Engine Event Router Refactoring

## Objective
The current `engine.Update` method in `engine/engine.go` acts as a massive event router, handling significant inline business logic within a large `switch` statement (specifically for `WatcherMsg` which deals with config changes, graph rebuilding, and test re-queuing). This violates clean architecture principles by mixing routing concerns with domain logic.

The goal of this epic is to apply the **Extract Method** pattern to break down the `engine.Update` function into smaller, focused handler methods on the `Engine` struct.

## Tasks

### 1. Extract File Watcher Event Handlers
- **Task:** Create a new method `func (e *Engine) handleWatcherMsg(path string) tea.Cmd`.
- **Task:** Move the logic for identifying if a file is a config file vs. a source file into this new method.
- **Task:** Create sub-handlers for the two branches:
  - `func (e *Engine) handleConfigChange(path string) tea.Cmd`: Handle config reloading, graph rebuilding, and test re-queuing.
  - `func (e *Engine) handleSourceChange(path string) tea.Cmd`: Handle graph updating and related test discovery for Smart/Manual modes.

### 2. Extract Runner Output & Status Handlers
- **Task:** Create a method `func (e *Engine) handleOutputUpdate(msg runner.OutputUpdate) tea.Cmd` to manage appending output to the `TestOutputs` map.
- **Task:** Create a method `func (e *Engine) handleStatusUpdate(msg runner.StatusUpdate) tea.Cmd` to manage test completion, updating statuses, removing from the `RunningNodes` map, and triggering the next queue process.

### 3. Extract Tree/Initialization Handlers
- **Task:** Extract smaller handlers like `func (e *Engine) handleTreeLoaded(msg TreeLoadedMsg) tea.Cmd` and `func (e *Engine) handleWatcherReady(msg WatcherReadyMsg) tea.Cmd`.

### 4. Refactor `engine.Update`
- **Task:** Replace all inline logic in the `Update` switch statement with calls to the newly created handler methods. The switch statement should act purely as a dispatcher.

## Acceptance Criteria
- `engine/engine.go` is significantly smaller and easier to read.
- `engine.Update` contains no inline business logic, only a switch statement calling domain handlers.
- All tests continue to pass (`go test ./...`), verifying no behavioral changes were introduced.
