# Plan: Fix Output Concatenation and UI Blocking

## Issue Description
Test runners emit output that the engine currently accumulates using standard Go string concatenation (`+=`) in `engine/engine.go` (`handleOutputUpdate`). Because Go strings are immutable, this results in $O(N^2)$ memory allocation and copying for every new line of output. Additionally, for each new line received, the UI re-renders and word-wraps the entire accumulated string using `lipgloss.Render` on the main UI thread in `ui/sync.go` (`wrapOutput`). In verbose test suites, this causes memory leaks, aggressive GC pauses, and completely freezes the TUI.

## Proposed Fix
1. **Engine Layer:**
   - Modify the `State.TestOutputs` map to store a more efficient data structure instead of a raw `string`. Options include a custom ring buffer (to cap maximum output size and prevent indefinite growth) or simply a slice of strings (`[]string`), where each element represents a line.
   - Update `handleOutputUpdate` to append to this new structure instead of using string concatenation.

2. **UI Layer:**
   - Update `ui/sync.go` to only process delta updates when possible, or at least avoid re-wrapping and rendering the entire history on every single character/line received.
   - Implement debouncing for the `OutputUpdate` Bubbletea messages so that rapid bursts of output don't flood the `Update` loop. Render at most every X milliseconds (e.g., 16ms for ~60fps) during heavy I/O.
   - If using a slice of lines, the UI viewport can render only the visible lines rather than the entire buffer.

## Execution Steps
1. Refactor `engine.State.TestOutputs` from `map[string]string` to `map[string][]string` or a custom `OutputBuffer` struct.
2. Update the `handleOutputUpdate` method to append new lines to the buffer.
3. Update `ui/sync.go` (`wrapOutput`) to operate on the new data structure and only render visible lines based on the viewport state.
4. Add a debouncer to the UI update loop for `runner.OutputUpdate` messages.
5. Run tests with highly verbose output to verify memory stability and UI responsiveness.
