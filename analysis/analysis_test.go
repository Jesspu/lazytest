package analysis

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestGraph(t *testing.T) {
	// Setup temporary test directory
	tmpDir, err := os.MkdirTemp("", "lazytest_analysis_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files
	files := map[string]string{
		"utils.ts":          "export const foo = 'bar';",
		"component.ts":      "import { foo } from './utils';",
		"utils.test.ts":     "import { foo } from './utils';",
		"component.test.ts": "import { Component } from './component';",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Build Graph
	g := NewGraph()
	if err := g.Build(tmpDir); err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Test GetDependents for utils.ts
	utilsPath := filepath.Join(tmpDir, "utils.ts")
	dependents := g.GetDependents(utilsPath)

	expected := []string{
		filepath.Join(tmpDir, "component.ts"),
		filepath.Join(tmpDir, "utils.test.ts"),
		filepath.Join(tmpDir, "component.test.ts"), // Transitive dependency via component.ts
	}

	sort.Strings(dependents)
	sort.Strings(expected)

	if len(dependents) != len(expected) {
		t.Errorf("Expected %d dependents, got %d", len(expected), len(dependents))
	}

	for i := range expected {
		if dependents[i] != expected[i] {
			t.Errorf("Expected dependent %s, got %s", expected[i], dependents[i])
		}
	}
}

func TestGraph_RelativeImports(t *testing.T) {
	// Setup temporary test directory
	tmpDir, err := os.MkdirTemp("", "lazytest_analysis_relative")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure:
	// src/app.tsx
	// test/app.test.tsx (imports ../src/app)

	if err := os.MkdirAll(filepath.Join(tmpDir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"src/app.tsx":       "export const App = () => {};",
		"test/app.test.tsx": "import App from '../src/app';",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Build Graph
	g := NewGraph()
	if err := g.Build(tmpDir); err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Test GetDependents for src/app.tsx
	appPath := filepath.Join(tmpDir, "src/app.tsx")
	dependents := g.GetDependents(appPath)

	expected := []string{
		filepath.Join(tmpDir, "test/app.test.tsx"),
	}

	if len(dependents) != len(expected) {
		t.Errorf("Expected %d dependents, got %d", len(expected), len(dependents))
	}

	if len(dependents) > 0 && dependents[0] != expected[0] {
		t.Errorf("Expected dependent %s, got %s", expected[0], dependents[0])
	}
}

func TestGraph_CaseSensitivity(t *testing.T) {
	// Setup temporary test directory
	tmpDir, err := os.MkdirTemp("", "lazytest_analysis_case")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure:
	// src/App.tsx (TitleCase)
	// test/app.test.tsx (imports ../src/app - lowercase)

	if err := os.MkdirAll(filepath.Join(tmpDir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"src/App.tsx":       "export const App = () => {};",
		"test/app.test.tsx": "import App from '../src/app';",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Build Graph
	g := NewGraph()
	if err := g.Build(tmpDir); err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Test GetDependents for src/App.tsx
	// Note: We query with the actual file path (TitleCase) because that's what the UI/Walker would provide.
	appPath := filepath.Join(tmpDir, "src/App.tsx")
	dependents := g.GetDependents(appPath)

	expected := []string{
		filepath.Join(tmpDir, "test/app.test.tsx"),
	}

	if len(dependents) != len(expected) {
		t.Errorf("Expected %d dependents, got %d", len(expected), len(dependents))
	}
}
