package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalk(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-walker-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files
	files := []string{
		"src/component.test.tsx",
		"src/utils/helper.spec.ts",
		"readme.md", // Should be ignored by isTestFile
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	rootNode, err := Walk(tmpDir)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if rootNode == nil {
		t.Fatal("rootNode is nil")
	}

	if rootNode.Name != filepath.Base(tmpDir) {
		t.Errorf("expected root name %s, got %s", filepath.Base(tmpDir), rootNode.Name)
	}

	// Helper to count nodes in tree
	var countTests func(*Node) int
	countTests = func(n *Node) int {
		count := 0
		if !n.IsDir && IsTestFile(n.Name) {
			count++
		}
		for _, child := range n.Children {
			count += countTests(child)
		}
		return count
	}

	testCount := countTests(rootNode)
	if testCount != 2 {
		t.Errorf("expected 2 test files in tree, got %d", testCount)
	}
}
