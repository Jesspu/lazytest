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

	rootNode, err := Walk(tmpDir, nil)
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

func TestWalk_Excludes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-walker-excludes")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	files := []string{
		"src/component.test.tsx",
		"src/ignored/bad.test.ts",
		"e2e/login.spec.ts",
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

	// Exclude "src/ignored" and "e2e"
	excludes := []string{"src/ignored", "e2e"}

	rootNode, err := Walk(tmpDir, excludes)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	foundFiles := make(map[string]bool)
	var collectFiles func(*Node)
	collectFiles = func(n *Node) {
		if !n.IsDir {
			foundFiles[n.Path] = true
		}
		for _, child := range n.Children {
			collectFiles(child)
		}
	}
	collectFiles(rootNode)

	// Check that we only found src/component.test.tsx
	expected := filepath.Join(tmpDir, "src/component.test.tsx")
	if !foundFiles[expected] {
		t.Errorf("Expected to find %s", expected)
	}

	// Check excluded
	ignored := filepath.Join(tmpDir, "src/ignored/bad.test.ts")
	if foundFiles[ignored] {
		t.Errorf("Should have excluded %s", ignored)
	}

	e2e := filepath.Join(tmpDir, "e2e/login.spec.ts")
	if foundFiles[e2e] {
		t.Errorf("Should have excluded %s", e2e)
	}
}
