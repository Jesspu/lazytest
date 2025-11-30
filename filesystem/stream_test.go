package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStreamFiles(t *testing.T) {
	// Create a temp directory structure
	tmpDir, err := os.MkdirTemp("", "lazytest-stream-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filesToCreate := []string{
		"file1.txt",
		"dir1/file2.txt",
		"dir1/dir2/file3.txt",
	}

	for _, f := range filesToCreate {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Stream files
	fileChan := StreamFiles(tmpDir)

	count := 0
	for range fileChan {
		count++
	}

	if count != 3 {
		t.Errorf("expected 3 files, got %d", count)
	}
}
