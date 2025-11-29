package filesystem

import "github.com/boyter/gocodewalker"

// StreamFiles starts a file walker and returns a channel of files.
// It abstracts the boilerplate of creating the channel and starting the goroutine.
func StreamFiles(root string) <-chan *gocodewalker.File {
	fileListQueue := make(chan *gocodewalker.File, 100)
	fileWalker := gocodewalker.NewFileWalker(root, fileListQueue)

	go func() {
		_ = fileWalker.Start()
	}()

	return fileListQueue
}
