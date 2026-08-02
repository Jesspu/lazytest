# Epic: UI Error Notification System

## Objective
Currently, the UI handles some subsystem interactions silently. For example, in `ui/update.go`, if `filesystem.GetChangedFiles` fails when handling the `AddRelated` keybinding, the error is swallowed and the user is given no indication that an action failed. 

The goal of this epic is to introduce a non-blocking toast/notification system within the Bubbletea UI that allows the `engine` or `ui` layer to surface temporary errors or warnings to the user without interrupting their workflow.

## Tasks

### 1. Define Notification Message Types
- **Task:** Define a new message type `type NotificationMsg struct { Message string; IsError bool }` (likely in `engine/messages.go` or a new `ui/messages.go` if strictly UI-bound).
- **Task:** Define a `type ClearNotificationMsg struct{}` to handle dismissing the notification after a delay.

### 2. Update the UI Model
- **Task:** Add fields to `ui.Model` to track the active notification: `activeNotification string`, `isNotificationError bool`, and potentially a timer reference.

### 3. Implement Notification Rendering
- **Task:** Update `ui.Model.View` to render the notification. This could be implemented as a toast overlay using `lipgloss.Place` over the main layout, or by taking over the footer space temporarily.
- **Task:** Define lipgloss styles for standard notifications (e.g., info blue) and error notifications (e.g., danger red).

### 4. Handle Notification Lifecycle in `Update`
- **Task:** Update `ui/update.go` to handle `NotificationMsg`. When received, update the model state and return a command using `tea.Tick` that fires a `ClearNotificationMsg` after a set duration (e.g., 3 seconds).
- **Task:** Handle `ClearNotificationMsg` to reset the notification state fields.

### 5. Surface Existing Silent Errors
- **Task:** Refactor `ui/update.go` around line 85 (`AddRelated` keybinding). Currently, `filesystem.GetChangedFiles` failure is explicitly swallowed (`// Silent error, no global output available`). This should return a `tea.Cmd` that emits a `NotificationMsg` (e.g., "Not a git repository" or similar git error).
- **Task:** Refactor `filesystem/watcher.go` around line 139. Currently, `w.fsWatcher.Errors` is channeled to `log.Println`, which can corrupt the TUI and doesn't get bubbled up. Update the watcher to emit an `engine.WatcherErrorMsg` (or similar) which the engine can catch and convert into a `NotificationMsg`.
- **Task:** Refactor these blocks to return or dispatch the newly created notification messages so the user is informed of background failures.

## Acceptance Criteria
- A temporary, non-blocking notification appears in the UI when a background error occurs.
- The notification auto-dismisses after a short timeout.
- The `AddRelated` functionality (and any other previously silent failures) accurately reports issues like missing git repositories or unreadable file paths to the user.
