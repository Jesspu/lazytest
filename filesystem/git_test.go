package filesystem

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGetChangedFiles(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "lazytest-git-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Create a file
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Get changed files
	files, err := GetChangedFiles(tmpDir)
	if err != nil {
		t.Fatalf("GetChangedFiles failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 changed file, got %d", len(files))
	}

	if files[0] != filePath {
		t.Errorf("expected file path %s, got %s", filePath, files[0])
	}
}
