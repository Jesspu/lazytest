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

func TestGraph_Update(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest_analysis_update")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initial state:
	// a.ts
	// b.ts -> imports a.ts
	files := map[string]string{
		"a.ts": "export const a = 1;",
		"b.ts": "import { a } from './a';",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	g := NewGraph()
	if err := g.Build(tmpDir); err != nil {
		t.Fatal(err)
	}

	aPath := filepath.Join(tmpDir, "a.ts")
	bPath := filepath.Join(tmpDir, "b.ts")

	// Verify initial dependency
	deps := g.GetDependents(aPath)
	if len(deps) != 1 || deps[0] != bPath {
		t.Errorf("Initial: Expected b.ts to depend on a.ts, got %v", deps)
	}

	// 1. Modify b.ts to REMOVE import of a.ts
	// b.ts -> (no imports)
	if err := os.WriteFile(bPath, []byte("export const b = 2;"), 0644); err != nil {
		t.Fatal(err)
	}
	g.Update(bPath)

	deps = g.GetDependents(aPath)
	if len(deps) != 0 {
		t.Errorf("After removal: Expected no dependents for a.ts, got %v", deps)
	}

	// 2. Modify b.ts to ADD import of a.ts back
	if err := os.WriteFile(bPath, []byte("import { a } from './a';"), 0644); err != nil {
		t.Fatal(err)
	}
	g.Update(bPath)

	deps = g.GetDependents(aPath)
	if len(deps) != 1 || deps[0] != bPath {
		t.Errorf("After re-add: Expected b.ts to depend on a.ts, got %v", deps)
	}

	// 3. Add pending import
	// c.ts -> imports d.ts (which doesn't exist yet)
	cPath := filepath.Join(tmpDir, "c.ts")
	if err := os.WriteFile(cPath, []byte("import { d } from './d';"), 0644); err != nil {
		t.Fatal(err)
	}
	g.Update(cPath) // Process new file

	// Check pending imports
	// d.ts (abs path) should be in pending
	dPathAbs := filepath.Join(tmpDir, "d") // Key is without extension
	if _, ok := g.PendingImports[dPathAbs]; !ok {
		// It might be stored with extension if the import had one, but here it's './d'
		// Let's check if it's there.
		t.Logf("Pending imports: %v", g.PendingImports)
		// Note: The key in PendingImports is the absolute path from resolvePaths.
		// resolvePaths uses filepath.Join(dir, imp).
	}

	// 4. Create d.ts
	dPath := filepath.Join(tmpDir, "d.ts")
	if err := os.WriteFile(dPath, []byte("export const d = 3;"), 0644); err != nil {
		t.Fatal(err)
	}
	g.Update(dPath) // This should trigger resolution of pending import from c.ts

	// Verify c.ts depends on d.ts
	dDeps := g.GetDependents(dPath)
	if len(dDeps) != 1 || dDeps[0] != cPath {
		t.Errorf("After creating d.ts: Expected c.ts to depend on d.ts, got %v", dDeps)
	}
}

func TestParser_Formats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest_analysis_parser")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `
	import { a } from './a';
	import b from "./b";
	import './c';
	const d = require('./d');
	`
	filePath := filepath.Join(tmpDir, "test.ts")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create the targets so they resolve
	for _, name := range []string{"a.ts", "b.ts", "c.ts", "d.js"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	p := NewParser()
	result, err := p.ParseImports(filePath)
	if err != nil {
		t.Fatalf("ParseImports failed: %v", err)
	}

	expected := map[string]bool{
		filepath.Join(tmpDir, "a.ts"): false,
		filepath.Join(tmpDir, "b.ts"): false,
		filepath.Join(tmpDir, "c.ts"): false,
		filepath.Join(tmpDir, "d.js"): false,
	}

	for _, res := range result.Resolved {
		if _, ok := expected[res.Path]; ok {
			expected[res.Path] = true
		} else {
			t.Errorf("Unexpected import found: %s", res.Path)
		}
	}

	for path, found := range expected {
		if !found {
			t.Errorf("Expected import not found: %s", path)
		}
	}
}

func TestParser_MultiLineImport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest_repro_multiline")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `
import {
  foo
} from './utils';
`
	path := filepath.Join(tmpDir, "test.ts")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create utils so it can be resolved
	if err := os.WriteFile(filepath.Join(tmpDir, "utils.ts"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	result, err := parser.ParseImports(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Resolved) == 0 {
		t.Error("Failed to parse multi-line import")
	}
}
