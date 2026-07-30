# First-Run Welcome Banner Plan

This document outlines the plan to display a welcome banner when LazyTest starts, showing the auto-detected configuration so users have immediate confidence in the tool.

## Goals
- Show a brief, non-intrusive banner in the output pane on first launch.
- Display the detected test runner and command.
- Indicate whether a `.lazytest.json` was found or if auto-detection was used.
- Provide a hint for help and configuration.

## Current Behavior

When LazyTest starts, the output pane is empty until the user runs a test. There is no indication of which test command will be used, which runner was detected, or how to configure it. Users discover the wrong command only after their first test fails.

## Proposed Changes

### 1. Generate Welcome Message (`engine/engine.go`)

After initialization, populate a welcome message based on the loaded config.

```go
func (e *Engine) generateWelcome() string {
    var sb strings.Builder
    sb.WriteString("  LazyTest v0.0.6\n\n")

    if e.ProjectConfig.DetectedRunner != "" {
        sb.WriteString(fmt.Sprintf("  ⚡ Auto-detected: %s\n", e.ProjectConfig.DetectedRunner))
    } else {
        sb.WriteString("  📄 Config: .lazytest.json\n")
    }

    sb.WriteString(fmt.Sprintf("  ⌘ Command: %s\n", e.ProjectConfig.Command))
    sb.WriteString(fmt.Sprintf("  ⚙ Concurrency: %d\n", e.ProjectConfig.MaxConcurrentTests))

    if len(e.Workspaces) > 1 {
        sb.WriteString(fmt.Sprintf("  📦 Workspaces: %d packages detected\n", len(e.Workspaces)))
    }

    sb.WriteString("\n  Press ? for help • Press Enter to run a test\n")
    return sb.String()
}
```

### 2. Display in Output Pane (`engine/state.go`)

Add a `WelcomeMessage string` field to `State`. Populate it in `Engine.New()` after config is loaded.

The UI's `syncViewportOutput` in `ui/sync.go` should check: if no test is selected and the viewport would be empty, render `e.State.WelcomeMessage` instead.

```go
func (m *Model) syncViewportOutput() {
    // ... existing selected-path logic ...

    if output == "" && m.engine.State.WelcomeMessage != "" {
        output = m.engine.State.WelcomeMessage
    }

    m.viewport.SetContent(m.wrapOutput(m.viewport.Width, output))
}
```

### 3. Styling (`ui/styles.go`)

Use a subtle, dimmed style for the welcome text so it doesn't compete with test output. Use the existing `lipgloss` palette.

```go
var welcomeStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("241")).
    Padding(1, 2)
```

### 4. Auto-Dismiss

The welcome message should disappear as soon as any test is triggered. This happens naturally since `syncViewportOutput` will display the test's output once available.

## Implementation Steps

1. **Add `DetectedRunner`** field to `Config` (done as part of auto-detect runner plan).
2. **Add `WelcomeMessage`** field to `engine/state.go`.
3. **Implement `generateWelcome`** in `engine/engine.go`, called during `New()`.
4. **Update `syncViewportOutput`** to show welcome when viewport would be empty.
5. **Add welcome styling** to `ui/styles.go`.
6. **Verify** the banner appears on launch and disappears on first test run.
