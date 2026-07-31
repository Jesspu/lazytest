# Phase 3: Core E2E Test Suite Implementation Plan

## Goal
Implement a comprehensive suite of End-to-End tests using the infrastructure and helpers established in Phases 1 and 2. This suite will simulate user interactions against real project fixtures to ensure `lazytest` behaves correctly across all its major workflows.

## 1. Test Suite Structure
We will organize the tests by functional area within the `e2e/` package to ensure maintainability.

**Proposed Files:**
- `e2e/runner_detection_test.go`: Tests verifying automatic detection of Jest, Vitest, Mocha, etc.
- `e2e/navigation_execution_test.go`: Tests for file tree traversal, selection, and manual test triggering.
- `e2e/search_mode_test.go`: Tests validating the search input state, filtering, and escaping.
- `e2e/smart_mode_test.go`: Tests validating the graph analysis and auto-queuing.
- `e2e/watch_mode_test.go`: Tests validating the file watcher integration and targeted re-runs.

## 2. Detailed Test Cases

### A. Runner Detection & Initial Load (`runner_detection_test.go`)
- **Test:** Boot TUI against `single_repo_jest`.
- **Assertions:**
  - Verify the UI outputs the correct configuration status (e.g., verifying a badge or output string that indicates "Jest").
  - Verify the initial file tree accurately lists `math.test.js`.
- **Test:** Repeat identical checks for `single_repo_vitest` and `single_repo_mocha`.

### B. Navigation & Test Execution (`navigation_execution_test.go`)
- **Test:** Run a passing test.
  - Setup: `single_repo_vitest` fixture.
  - Action: Send `j` or `k` keystrokes to select `stringUtils.test.ts`, then send `enter`.
  - Assertions: Wait for the TUI output to indicate "Running...", followed by "Passed". The right viewport should display the stdout test output from Vitest.
- **Test:** View test output history.
  - Action: After a test completes, use `l` or `tab` to switch focus to the output pane, ensure scrolling works.

### C. Search Mode (`search_mode_test.go`)
- **Test:** Filter file tree.
  - Setup: `single_repo_jest` fixture.
  - Action: Send `/` to enter search mode. Send keystrokes `m` `a` `t` `h`.
  - Assertions: Assert the footer or search bar shows the active query `math`. Assert the file tree is filtered to show only files matching `math`.
  - Action: Send `esc`.
  - Assertions: Assert search mode closes and the full file tree is restored to its original state.

### D. Smart Mode & Dependency Graph (`smart_mode_test.go`)
- **Test:** Auto-queue transitively affected tests.
  - Setup: `single_repo_jest` fixture.
  - Action: Send `s` to toggle Smart Mode ON.
  - Action: Use Go's `os.WriteFile` or `os.Chtimes` to simulate an external file system write to `src/math.js` (the source file, not the test file) within the fixture directory.
  - Assertions: 
    - `lazytest` should receive the `fsnotify` event.
    - The graph analyzer should trace the change from `math.js` to `math.test.js`.
    - Wait for the TUI to automatically transition `math.test.js` to "Running..." and subsequently display the results.

### E. Watch Mode (`watch_mode_test.go`)
- **Test:** Re-run specific file on change.
  - Setup: `single_repo_vitest` fixture.
  - Action: Navigate to `stringUtils.test.ts` and press `w` to toggle watch mode.
  - Action: Simulate a write to `stringUtils.test.ts`.
  - Assertions: Verify `lazytest` detects the change to the watched file and automatically re-runs it.

## 3. Dealing with Async Flakiness
E2E testing a TUI coupled with real background processes (node subprocesses, file watchers) can inherently introduce flakiness.

**Mitigation Strategy:**
- **Avoid Sleep:** Rely heavily on the `WaitForText` helper (from Phase 2) rather than fixed `time.Sleep` delays.
- **Timeouts:** Enforce strict but reasonable timeouts on assertions (e.g., max 5-10 seconds for a test run to complete).
- **Cleanup:** Ensure the `context.CancelFunc` from `SetupTestEnv` is *always* deferred at the very beginning of each test block to prevent zombie Node processes and lingering file watchers.
