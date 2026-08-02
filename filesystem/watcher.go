package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors the file system for changes.
type Watcher struct {
	fsWatcher    *fsnotify.Watcher
	Events       chan string // Signal to refresh the tree, carries the changed file path
	Errors       chan error  // Channel for filesystem watcher errors
	done         chan struct{}
	root         string
	mu           sync.Mutex
	pendingPaths map[string]struct{}
}

// NewWatcher creates a new Watcher for the given root directory.
func NewWatcher(root string) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher:    fsWatcher,
		Events:       make(chan string, 100), // Increased buffer to handle batch events
		Errors:       make(chan error, 10),
		done:         make(chan struct{}),
		root:         root,
		pendingPaths: make(map[string]struct{}),
	}

	// Use gocodewalker to find all relevant directories to watch
	fileListQueue := StreamFiles(root)

	// Always watch root
	_ = w.fsWatcher.Add(root)

	// Track added directories to avoid duplicates
	addedDirs := make(map[string]bool)
	addedDirs[root] = true

	for f := range fileListQueue {
		dir := filepath.Dir(f.Location)
		// Add this directory and all its parents up to root
		for dir != root && dir != "." && dir != "/" {
			if addedDirs[dir] {
				break
			}
			// We need to verify it is inside root, which it should be
			if strings.HasPrefix(dir, root) {
				_ = w.fsWatcher.Add(dir)
				addedDirs[dir] = true
			}
			dir = filepath.Dir(dir)
		}
	}

	go w.startLoop()

	return w, nil
}

// Close stops the watcher and releases resources.
func (w *Watcher) Close() {
	close(w.done)
	w.fsWatcher.Close()
}

func (w *Watcher) startLoop() {
	var timer *time.Timer
	debounceDuration := 100 * time.Millisecond

	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.fsWatcher.Events:

			if !ok {
				return
			}

			// Ignore CHMOD events which can be noisy
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}

			// If it's a directory creation, we need to add it to the watcher
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					w.fsWatcher.Add(event.Name)
					continue
				}
			}

			// Allowlist: Only process events for source files, test files, and config files
			if !IsSourceFile(event.Name) && !IsConfigFile(event.Name) {
				continue
			}

			// Accumulate the path into the pending set before resetting the timer.
			// This ensures no path is dropped when multiple files change within the
			// debounce window (e.g. git checkout, IDE bulk-save).
			w.mu.Lock()
			w.pendingPaths[event.Name] = struct{}{}
			w.mu.Unlock()

			// Reset the debounce timer. When it fires, all accumulated paths are
			// flushed to w.Events in one pass.
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounceDuration, func() {
				w.mu.Lock()
				pathsToEmit := make([]string, 0, len(w.pendingPaths))
				for path := range w.pendingPaths {
					pathsToEmit = append(pathsToEmit, path)
				}
				w.pendingPaths = make(map[string]struct{}) // Reset for next batch
				w.mu.Unlock()

				for _, path := range pathsToEmit {
					w.Events <- path
				}
			})

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			select {
			case w.Errors <- err:
			default:
			}
		}
	}
}
