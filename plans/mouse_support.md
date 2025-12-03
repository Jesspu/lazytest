# Mouse Support Implementation Plan

This plan outlines the steps to add mouse support to the LazyTest UI. The goal is to enable users to interact with the application using the mouse for navigation, selection, and execution of tests.

## Goals
1.  **Pane Selection**: Single click to switch focus between the Explorer (left) and Output (right) panes.
2.  **Tab Selection**: Single click to switch between "Explorer" and "Watched" tabs within the Explorer pane.
3.  **File Selection**: Single click to select a file in the Explorer or Watched list.
4.  **Run Test**: Double click on a file to run the test (equivalent to pressing Enter).

## Proposed Changes

### 1. Enable Mouse Events
In `ui/model.go`, update the `Init` function to enable mouse events.

```go
func (m Model) Init() tea.Cmd {
    return tea.Batch(
        m.engine.Init(),
        tea.EnableMouseCellMotion, // Enable mouse click, release, and wheel events
    )
}
```

### 2. Update Model State
In `ui/model.go`, add fields to the `Model` struct to track click timing and position for double-click detection.

```go
type Model struct {
    // ... existing fields
    
    // Mouse State
    lastClickTime time.Time
    lastClickX    int
    lastClickY    int
    // ...
}
```

### 3. Handle Mouse Messages
In `ui/model.go`, update the `Update` function to handle `tea.MouseMsg`.

#### Logic Flow:
1.  **Capture Event**: Listen for `tea.MouseMsg`.
2.  **Determine Pane**:
    *   Calculate the split point (approx `m.width / 2`).
    *   If `msg.X < splitPoint`, user clicked in **Explorer Pane**.
    *   If `msg.X >= splitPoint`, user clicked in **Output Pane**.
    *   Update `m.activePane` accordingly.
3.  **Handle Explorer Pane Clicks**:
    *   **Header/Tabs Area** (Top ~3 lines):
        *   Check `msg.Y`. If it's within the tab header range (e.g., row 0-2):
            *   Check `msg.X` to determine if "Explorer" or "Watched" tab was clicked.
            *   Update `m.activeTab`.
    *   **List Area** (Below Header):
        *   Calculate the list index based on `msg.Y`.
        *   **Visual Index**: `visualIndex = msg.Y - headerHeight`.
        *   **Scroll Offset**: Re-calculate the current scroll offset (`start` index) using the same logic as `calculateVisibleRange`.
        *   **Actual Index**: `index = start + visualIndex`.
        *   **Bounds Check**: Ensure `index` is valid for the current list (`flatNodes` or `watchedFiles`).
        *   **Action**:
            *   **Single Click**: Update `m.cursor` (Explorer) or `m.watchedCursor` (Watched).
            *   **Double Click**:
                *   Check if `time.Since(m.lastClickTime) < 500ms`.
                *   Check if `msg.X` and `msg.Y` are close to `lastClickX/Y` (or identical).
                *   If match: Trigger test run (same as `Enter` key).
                *   If no match: Just select.
            *   Update `lastClickTime`, `lastClickX`, `lastClickY`.

### 4. Implementation Details

#### Constants
Define constants for layout dimensions if they aren't dynamic, or calculate them dynamically.
*   `HeaderHeight`: The tabs take up some vertical space. Based on `explorer.go`, it seems to be `lipgloss.Height(tabs + "\n\n")`. This is likely 3 lines (1 line text + 2 borders/padding/newlines).

#### Coordinate Mapping
*   **Explorer Tab**: `X` range `0` to `width_of_explorer_label`.
*   **Watched Tab**: `X` range `width_of_explorer_label + padding` to `end`.
*   **List Item**:
    *   `Y` coordinate corresponds to the row.
    *   Need to account for `m.height` and `paneHeight`.

#### Double Click Logic
```go
// Inside tea.MouseMsg handler
if msg.Type == tea.MouseLeft {
    isDoubleClick := false
    if time.Since(m.lastClickTime) < 500*time.Millisecond && 
       msg.X == m.lastClickX && msg.Y == m.lastClickY {
        isDoubleClick = true
    }
    
    m.lastClickTime = time.Now()
    m.lastClickX = msg.X
    m.lastClickY = msg.Y
    
    if isDoubleClick {
        // Trigger Run Action
    } else {
        // Trigger Select Action
    }
}
```

## Verification
1.  **Manual Test**:
    *   Run the app.
    *   Click on "Output" pane -> Focus should switch.
    *   Click on "Explorer" pane -> Focus should switch.
    *   Click on "Watched" tab -> Tab should switch.
    *   Click on a file in the list -> Cursor should move to that file.
    *   Double click on a file -> Test should run.
    *   Scroll (if wheel enabled) -> List should scroll (optional, but `EnableMouseCellMotion` handles wheel events as key presses usually, or separate events).

## Notes
*   Ensure `tea.EnableMouseCellMotion` is used.
*   Be careful with off-by-one errors in coordinate calculations.
*   The `calculateVisibleRange` logic needs to be accessible or replicated in `Update` to correctly map Y coordinates to list indices, especially when scrolled.
