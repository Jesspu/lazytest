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
