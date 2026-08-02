# Plan: Fix Synchronous Graph Building

## Issue Description
When a configuration file changes (e.g. `tsconfig.json`, `package.json`), `engine/engine.go` (`handleConfigChange`) invokes `e.Graph.Build(e.State.RootPath)` directly inside the Bubbletea `Update()` event loop. This blocks via `wg.Wait()` until the entire repository is parsed. Similarly, `handleSourceChange` calls `e.Graph.Update` synchronously, hitting the disk. This blocks the main UI thread, causing the TUI to freeze and drop keystrokes for several seconds in large monorepos.

## Proposed Fix
1. **Asynchronous Graph Building:**
   - Move the `Graph.Build` execution into a Goroutine.
   - Have it return a `tea.Cmd` that yields a custom Bubbletea message (e.g., `GraphBuildCompleteMsg`) when the wait group finishes.
   - During the build, the UI can display a loading spinner or progress indicator.

2. **Asynchronous Graph Updating:**
   - Similarly, `Graph.Update` should not block the main event loop. If a file is saved, the graph update should be queued and processed in the background.
   - Return a `GraphUpdateCompleteMsg` to notify the engine that the dependents list has been updated and the affected suite can be recalculated.

## Execution Steps
1. Create new message types in `engine/messages.go`: `GraphBuildCompleteMsg` and `GraphUpdateCompleteMsg`.
2. Refactor `handleConfigChange` in `engine/engine.go` to spawn a goroutine for `Graph.Build` and return a `tea.Cmd`.
3. Refactor `handleSourceChange` to do the same for `Graph.Update`.
4. Update the UI layer to handle a "building graph" state, displaying a loading indicator to the user.
5. Verify that changing `.lazytest.json` or saving a file no longer freezes the TUI.
