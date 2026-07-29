# Mouse Support Implementation Plan

This plan outlines the steps to add mouse support to the LazyTest UI. The goal is to enable users to interact with the application using the mouse for navigation, selection, and execution of tests.

## Goals
1.  **Pane Selection**: Single click to switch focus between the Explorer (left) and Output (right) panes.
2.  **Tab Selection**: Single click to switch between "Explorer" and "Watched / Affected Suite" tabs within the Explorer pane.
3.  **File Selection**: Single click to select a file in the Explorer or Watched / Affected Suite list.
4.  **Run Test**: Double click on a file to run the test (equivalent to pressing Enter).
5.  **Scrolling**: Use the mouse wheel to scroll both the Explorer file list and the Output pane.

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
Create a new file `ui/mouse.go` and implement a `handleMouse` function. Then, in `ui/update.go`, update the `Update` function to forward `tea.MouseMsg` to this new handler.

#### Logic Flow:
1.  **Capture Event**: Listen for `tea.MouseMsg`.
2.  **Determine Pane**:
    *   Calculate the split point (`paneWidth` is approx `m.width / 2 - 4` due to borders).
    *   If `msg.X` falls in the left pane bounds, it's the **Explorer Pane**.
    *   If `msg.X` falls in the right pane bounds, it's the **Output Pane**.
    *   *Note on Scrolling Output*: If the event is in the Output Pane, and it's a `tea.MouseWheelUp` or `tea.MouseWheelDown`, pass it directly to `m.viewport.Update(msg)` so the output scrolls natively.
    *   Update `m.activePane` accordingly based on left clicks.
3.  **Handle Explorer Pane Events**:
    *   **Mouse Wheel Events**: Handle `tea.MouseWheelUp` and `tea.MouseWheelDown` by decrementing/incrementing `m.cursor` (or `m.watchedCursor`).
    *   **Border/Padding Offsets**: Subtract any `lipgloss` margins/borders from `msg.X` and `msg.Y` before calculating row/column indices.
    *   **Header/Tabs Area** (Top ~3 lines):
        *   Check `msg.Y`. If it's within the tab header range (e.g., row 0-2, after offsetting for the top border):
            *   Check `msg.X` to determine if "Explorer" or "Watched / Affected Suite" tab was clicked. (Note: the label changes based on `m.engine.IsSmartMode()`).
            *   Update `m.activeTab`.
    *   **Search Bar Area**:
        *   If `m.searchMode` is true, check if `msg.Y` is at the bottom bounds (`treeHeight` + `headerHeight`). If so, focus the search input.
    *   **List Area** (Below Header):
        *   Calculate the list index based on `msg.Y`.
        *   **Visual Index**: `visualIndex = msg.Y - headerHeight`.
        *   **Scroll Offset**: Re-calculate the current scroll offset (`start` index) using the same logic as `calculateVisibleRange`. Note that `treeHeight` depends on `paneHeight` (which is `m.height - 5`) and whether `m.searchMode` is active.
        *   **Actual Index**: `index = start + visualIndex`.
        *   **Bounds Check**: Ensure `index` is valid for the current list (`flatNodes` or `getTabList()`).
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
*   **Watched Tab**: `X` range `width_of_explorer_label + padding` to `end`. The label width is dynamic ("Watched" vs "Affected Suite").
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
    *   Use the mouse wheel over the Explorer list -> List selection should move up/down.
    *   Use the mouse wheel over the Output pane -> The output text should scroll.

## Notes
*   Ensure `tea.EnableMouseCellMotion` is used.
*   Be careful with off-by-one errors in coordinate calculations.
*   The `calculateVisibleRange` logic needs to be accessible or replicated in `Update` to correctly map Y coordinates to list indices, especially when scrolled.
*   Remember to import the `"time"` package in `ui/model.go` for the `time.Time` fields.
