# Spec Plan: Smart Mode UI/UX Enhancements

## Problem Statement

When LazyTest starts, it defaults to Explorer mode (`TabExplorer`), which is designed for manually discovering and running tests or adding them to a watch list (`'w'`). However, when Smart Mode is activated (`'s'`), this manual watch list paradigm becomes obsolete because tests are automatically queued and executed based on file dependency analysis. 

In Smart Mode, background file changes can trigger multiple test executions. Without a specialized UI, developers are forced to manually scroll through a massive file tree or switch between tabs to hunt down which tests just ran, which tests failed, and what error output was generated.

## Goals

1. **Mode-Aware Tab Structure**: Replace the obsolete "Watched" tab with an "Affected Suite" tab whenever Smart Mode is active.
2. **Status-Prioritized Sorting**: Order test files in the Affected Suite tab by status (`Failed` -> `Running` -> `Passed` -> `Idle`) so that broken tests bubble immediately to the top of the list.
3. **Zero-Touch Debugging (Auto-Focus)**: Automatically jump the sidebar selection and right-hand output viewport to the first failed test file whenever a background Smart Mode run completes with failures.
4. **Live Suite Dashboard Badge**: Render a dynamic summary badge in the top header bar or above the output viewport displaying real-time suite statistics (`Passed`, `Failed`, and `Running` counts).
5. **Dynamic Suite Population**: Populate the Affected Suite list with any test files that have already executed during the session, dynamically appending newly triggered tests as file modifications occur.
6. **Dynamic Keybinding Adaptation**: Disable `ToggleWatch` (`w`), dynamically swap labels and behavior for `ClearWatched` (`W` -> clear suite) and `AddRelated` (`a` -> run suite), and enable `RunFailures` (`f`) when in Smart Mode.
7. **Suite Management**: Provide quick shortcuts in Smart Mode to clear passing/idle tests from the Affected Suite (`W`), re-run all failing tests (`f`), and re-run all tests in the Affected Suite (`a`).

## Proposed Changes

### Phase 1: Core Engine & Suite Management (`engine/` package)

#### 1. Engine State for Affected Suite (`engine/state.go` & `engine/engine.go`)
- Track a set or slice of affected/executed test paths in `State` (e.g., `State.Affected map[string]struct{}`).
- Whenever a test is queued or executed in Smart Mode (or whenever any test finishes running), add its path to `State.Affected`.
- Provide an accessor method `Engine.GetAffectedSuite() []string` that returns all affected/executed test file paths, sorted by test status:
  1. `StatusFail`
  2. `StatusRunning`
  3. `StatusPass`
  4. Alphabetical tie-breaking within each group.
- Calculate and expose suite summary counts via `Engine.GetSuiteStats() (passed, failed, running int)`.

#### 2. Suite Management Operations (`engine/engine.go`)
- Implement suite management methods on `Engine`:
  - `Engine.ClearAffectedSuite()`: Removes all passing (`StatusPass`) and unrun/idle tests from `State.Affected`, keeping only failing (`StatusFail`) or running tests.
  - `Engine.RunSuiteFailures()`: Queues all tests in `State.Affected` that currently have `StatusFail`.
  - `Engine.RunAffectedSuite()`: Queues all tests currently in `State.Affected`.

---

### Phase 2: UI Transformation & Zero-Touch UX (`ui/` package)

#### 3. Dynamic Keybinding Adaptation (`ui/keys.go`, `ui/model.go`)
- Add a new keybinding `RunFailures` (`f`, help: `"run failures"`) to `KeyMap`.
- Whenever Smart Mode is toggled on (`s`) or updated in `Model`:
  - Disable obsolete manual watch binding: `m.keys.ToggleWatch.SetEnabled(!smartMode)`.
  - Enable suite failure binding: `m.keys.RunFailures.SetEnabled(smartMode)`.
  - Dynamically swap help labels for repurposed keys:
    - If `smartMode`: set `m.keys.ClearWatched` help to `("W", "clear suite")`, and `m.keys.AddRelated` help to `("a", "run suite")`.
    - Else: set `m.keys.ClearWatched` help to `("W", "clear watched")`, and `m.keys.AddRelated` help to `("a", "add related")`.
- In `Model.Update`, when in Smart Mode:
  - Pressing `W` calls `m.engine.ClearAffectedSuite()` and refreshes viewport/cursor.
  - Pressing `a` calls `m.engine.RunAffectedSuite()`.
  - Pressing `f` calls `m.engine.RunSuiteFailures()`.

#### 4. UI Tab & Sidebar Transformation (`ui/model.go` & `ui/explorer.go`)
- In `renderExplorer()`, when `m.engine.IsSmartMode()` is true:
  - Change the header tab title from `"Watched"` to `"Affected Suite"` (or `"Smart Suite"`).
  - When rendering the second tab (`m.activeTab == TabWatched`, which acts as the Affected Suite tab in Smart Mode), render the list returned by `m.engine.GetAffectedSuite()` instead of `m.engine.GetWatchedFiles()`.
- Update cursor navigation and view rendering so that moving `m.watchedCursor` across the Affected Suite list synchronizes the viewport with `GetTestOutput(path)`.

#### 5. Header Dashboard Badge (`ui/model.go` or `ui/header.go`)
- In the top header bar or above the output viewport, check if `m.engine.IsSmartMode()` is active.
- If active, render a badge displaying the stats from `GetSuiteStats()`:
  ```
  ⚡ SMART MODE | 3 Passed • 1 Failed • 0 Running
  ```
- Use distinct styling (e.g., green for passed, red for failed, yellow for running) via Lipgloss.

#### 6. Zero-Touch Failure Auto-Focus (`ui/model.go`)
- In `Model.Update(msg tea.Msg)`, under `case runner.StatusUpdate:`:
  - When `msg.Err != nil` (a test failed) and `m.engine.IsSmartMode()` is true:
    - Switch `m.activeTab` to `TabWatched` (the Affected Suite tab).
    - Find the index of the failed test in `m.engine.GetAffectedSuite()`.
    - Set `m.watchedCursor` to that index.
    - Synchronize the viewport content with the failed test's error output and scroll to bottom (`m.viewport.GotoBottom()`).

## Verification Plan

### Phase 1 Verification (Engine Layer)
- **Automated Unit Tests (`engine/engine_test.go`)**:
  - Test `GetAffectedSuite()` sorting: add 3 test files with status Pass, Fail, and Running. Verify the returned slice puts the Failed test first, Running second, and Pass third.
  - Test `GetSuiteStats()` returns exact counts of passed, failed, and running tests.
  - Test `ClearAffectedSuite()` removes passing tests while keeping failing tests.
  - Test `RunSuiteFailures()` and `RunAffectedSuite()` enqueue the expected test subsets.

### Phase 2 Verification (UI Layer)
- **Automated Unit Tests (`ui/model_test.go` or equivalent)**:
  - Verify that when `IsSmartMode()` is true, a failing `runner.StatusUpdate` automatically sets `m.activeTab` to the Affected Suite tab and positions `m.watchedCursor` on the failing test.
  - Verify that toggling Smart Mode dynamically updates keybinding enabled states and help label strings.
- **Manual Verification**:
  1. Run `lazytest` in a terminal.
  2. Toggle Smart Mode on (`'s'`). Verify the tab label changes from "Watched" to "Affected Suite" and the dashboard badge appears.
  3. Edit a source file that causes a test to fail.
  4. Confirm that as soon as the test fails, the UI automatically switches to the Affected Suite tab, highlights the failed test at the top of the list, and displays its error output in the viewport without touching the keyboard.
  5. Fix the code in the editor and save. Confirm the test re-runs, passes, moves down in the status-sorted list, and updates the stats badge.
  6. In Smart Mode, press `f` and verify only failing tests are re-run. Press `a` and verify all tests in the Affected Suite are re-run. Press `W` and verify passing tests are cleared while failing tests remain in the list.

