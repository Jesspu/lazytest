package engine

import "github.com/jesspatton/lazytest/filesystem"

// Messages

// WatcherMsg indicates a file system event occurred.
type WatcherMsg string

// TreeLoadedMsg carries the new file tree after a refresh.
type TreeLoadedMsg *filesystem.Node

// WatcherReadyMsg carries the initialized watcher.
type WatcherReadyMsg struct {
	watcher *filesystem.Watcher
}

// WatcherErrorMsg indicates an error from the filesystem watcher.
type WatcherErrorMsg struct {
	Err error
}

// NotificationMsg represents a temporary toast notification in the UI.
type NotificationMsg struct {
	Message string
	IsError bool
}

// ClearNotificationMsg signals that a notification should be dismissed.
type ClearNotificationMsg struct {
	ID int
}

