# Spec Plan: Cursor Output Synchronization

## Problem Statement

In Explorer mode (`TabExplorer`), navigating through the file tree using the arrow keys (`Up`, `Down`, or smart navigation) only updates the tree selection index (`m.cursor`). The UI does not check whether the newly selected file has previously recorded test output in `m.engine.GetTestOutput(path)`. 

Furthermore, when switching tabs or when background test runners emit status updates, the viewport frequently defaults to rendering `m.engine.GetCurrentOutput()` (the global stream of whatever test executed last across the entire engine) rather than the specific output of the file currently highlighted by the user's cursor. 

As a result, users cannot inspect past results of specific test files simply by navigating to them in the file explorer. To see a test's output, they are forced to either re-run the test or switch to the Watched tab (if the file happens to be watched).

## Goals

1. **Dynamic Output Synchronization**: Ensure the right-hand output viewport dynamically synchronizes with the currently selected file in both Explorer mode (`TabExplorer`) and Watched mode (`TabWatched`).
2. **Unified Output Helper**: Create a clean, DRY helper method (`syncViewportOutput()`) on `Model` to centralize all viewport content resolution and scrolling logic.
3. **Contextual Placeholders**: Provide clear, actionable placeholder text when navigating to test files that have not yet executed in the current session (e.g., `"No output yet for this test file.\nPress <Enter> to run."`).
4. **Active Inspection Guard**: Prevent asynchronous background runner updates (`OutputUpdate`, `StatusUpdate`) from overwriting the viewport when the developer is actively inspecting a different test file than the one currently executing in the background.

## Proposed Changes

### 1. Centralize Output Resolution (`ui/model.go`)

Create a helper method `syncViewportOutput() Model` (or update `updateViewport`) that determines what content should be displayed in `m.viewport` based on `m.activeTab` and cursor position:

```go
func (m *Model) syncViewportOutput() {
    if !m.ready {
        return
    }

    var content string

    if m.activeTab == TabWatched {
        watchedFiles := m.engine.GetWatchedFiles()
        if m.watchedCursor < len(watchedFiles) {
            path := watchedFiles[m.watchedCursor]
            if out, ok := m.engine.GetTestOutput(path); ok && out != "" {
                content = out
            } else {
                content = "No output yet."
            }
        } else {
            content = "No watched files.\nPress 'w' on a file to watch it."
        }
    } else {
        // TabExplorer
        if m.cursor < len(m.flatNodes) {
            node := m.flatNodes[m.cursor]
            if !node.IsDir {
                if out, ok := m.engine.GetTestOutput(node.Path); ok && out != "" {
                    content = out
                } else if filesystem.IsTestFile(node.Name) {
                    content = "No output yet for this test file.\nPress <Enter> to run, 'w' to watch, or 's' for Smart Mode."
                } else {
                    content = fmt.Sprintf("Source file: %s\nPress 'w' to watch or 's' for Smart Mode.", node.Name)
                }
            } else {
                content = fmt.Sprintf("Directory: %s", node.Name)
            }
        }
    }

    if content == "" {
        content = m.engine.GetCurrentOutput()
    }

    m.viewport.SetContent(m.wrapOutput(m.viewport.Width, content))
}
```

### 2. Hook Up Navigation & Tab Switch Events (`ui/model.go`)

In `Model.Update(msg tea.Msg)`, invoke `m.syncViewportOutput()` after any cursor movement or view transition:
- **Explorer Navigation**: On `Up`, `Down`, search next/prev match, search enter.
- **Watched Navigation**: On `Up`, `Down` in `TabWatched`.
- **Tab Switching**: On `Tab`, `NextTab`, `PrevTab`.
- **File System Events**: On `engine.TreeLoadedMsg` and `engine.WatcherMsg`.

### 3. Guard Asynchronous Runner Updates (`ui/model.go`)

Update `case runner.OutputUpdate:` and `case runner.StatusUpdate:` in `Model.Update` so that live streaming output only refreshes the viewport if the user is currently viewing the active running test:

```go
case runner.OutputUpdate:
    shouldShow := true
    runningNode := m.engine.GetRunningNode()
    
    if runningNode != nil {
        if m.activeTab == TabWatched {
            watched := m.engine.GetWatchedFiles()
            if m.watchedCursor < len(watched) && watched[m.watchedCursor] != runningNode.Path {
                shouldShow = false
            }
        } else if m.activeTab == TabExplorer {
            if m.cursor < len(m.flatNodes) && m.flatNodes[m.cursor].Path != runningNode.Path {
                shouldShow = false
            }
        }
    }

    if shouldShow {
        m.viewport.SetContent(m.wrapOutput(m.viewport.Width, m.engine.GetCurrentOutput()))
        m.viewport.GotoBottom()
    }
    return m, tea.Batch(cmds...)
```

## Verification Plan

### Automated Tests
- **Unit Tests (`ui/model_test.go` or equivalent)**:
  - Initialize a `Model` with two test files (`testA.js`, `testB.js`).
  - Pre-populate `e.State.TestOutputs` with distinct output strings for `testA.js` and `testB.js`.
  - Simulate keypresses (`key.Up`, `key.Down`) changing `m.cursor` in Explorer mode.
  - Verify that `m.viewport.View()` reflects the exact stored test output corresponding to the currently selected file.
  - Simulate a `runner.OutputUpdate` for a background test while `m.cursor` points to a different file; verify the viewport is *not* overwritten.

### Manual Verification
1. Run `lazytest` in a terminal.
2. In Explorer mode, navigate to `app.test.js` and press `<Enter>` to execute it.
3. Navigate to `utils.test.js` and press `<Enter>` to execute it.
4. Move the cursor up and down between `app.test.js` and `utils.test.js`.
5. Confirm that the right-hand output pane immediately toggles between the recorded test results of `app.test.js` and `utils.test.js` without re-executing them.
6. Navigate to an un-run test file and verify the helpful placeholder text is rendered.
