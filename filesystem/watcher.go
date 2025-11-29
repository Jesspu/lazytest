package filesystem

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors the file system for changes.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	Events    chan string // Signal to refresh the tree, carries the changed file path
	done      chan struct{}
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
	}

	// Add root and all subdirectories to watcher
	// Note: fsnotify is not recursive by default on Linux, but we'll walk and add.
	// On Mac (kqueue) it might be different, but explicit add is safer.
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if shouldIgnore(info.Name()) {
				return filepath.SkipDir
			}
			return w.fsWatcher.Add(path)
		}
		return nil
	})
	if err != nil {
		fsWatcher.Close()
		return nil, err
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
			// Ignore ignored files
			if shouldIgnore(filepath.Base(event.Name)) {
				continue
			}

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
				}
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
