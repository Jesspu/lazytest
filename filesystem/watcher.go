package filesystem

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors the file system for changes.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	Events    chan string // Signal to refresh the tree, carries the changed file path
	done      chan struct{}
	root      string
}

// NewWatcher creates a new Watcher for the given root directory.
func NewWatcher(root string) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher: fsWatcher,
		Events:    make(chan string, 10), // Buffered to prevent blocking
		done:      make(chan struct{}),
		root:      root,
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

			// Debounce logic
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounceDuration, func() {
				w.Events <- event.Name
			})

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Println("Watcher error:", err)
		}
	}
}
