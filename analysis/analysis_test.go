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

	// Test GetAffectedDependents for utils.ts
	utilsPath := filepath.Join(tmpDir, "utils.ts")
	dependents := g.GetAffectedDependents(utilsPath)

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

	// Test GetAffectedDependents for src/app.tsx
	appPath := filepath.Join(tmpDir, "src/app.tsx")
	dependents := g.GetAffectedDependents(appPath)

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

	// Test GetAffectedDependents for src/App.tsx
	// Note: We query with the actual file path (TitleCase) because that's what the UI/Walker would provide.
	appPath := filepath.Join(tmpDir, "src/App.tsx")
	dependents := g.GetAffectedDependents(appPath)

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
	deps := g.GetAffectedDependents(aPath)
	if len(deps) != 1 || deps[0] != bPath {
		t.Errorf("Initial: Expected b.ts to depend on a.ts, got %v", deps)
	}

	// 1. Modify b.ts to REMOVE import of a.ts
	// b.ts -> (no imports)
	if err := os.WriteFile(bPath, []byte("export const b = 2;"), 0644); err != nil {
		t.Fatal(err)
	}
	g.Update(bPath)

	deps = g.GetAffectedDependents(aPath)
	if len(deps) != 0 {
		t.Errorf("After removal: Expected no dependents for a.ts, got %v", deps)
	}

	// 2. Modify b.ts to ADD import of a.ts back
	if err := os.WriteFile(bPath, []byte("import { a } from './a';"), 0644); err != nil {
		t.Fatal(err)
	}
	g.Update(bPath)

	deps = g.GetAffectedDependents(aPath)
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
	dDeps := g.GetAffectedDependents(dPath)
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

// TestParser_DynamicImport verifies that dynamic import() expressions are captured.
func TestParser_DynamicImport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest_dynamic_import")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create target file so the import resolves.
	if err := os.WriteFile(filepath.Join(tmpDir, "utils.ts"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	content := `
const mod = await import('./utils');
const lazy = import('./utils'); // without await
`
	filePath := filepath.Join(tmpDir, "test.ts")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewParser()
	result, err := p.ParseImports(filePath)
	if err != nil {
		t.Fatalf("ParseImports failed: %v", err)
	}

	utilsPath := filepath.Join(tmpDir, "utils.ts")
	found := false
	for _, r := range result.Resolved {
		if r.Path == utilsPath {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Dynamic import of utils.ts not resolved; resolved=%v unresolved=%v", result.Resolved, result.Unresolved)
	}
}

// TestParser_ReExports verifies that re-export statements are captured.
func TestParser_ReExports(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest_reexport")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create target files.
	for _, name := range []string{"helper.ts", "types.ts"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	content := `
export { foo } from './helper';
export * from './types';
`
	filePath := filepath.Join(tmpDir, "index.ts")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewParser()
	result, err := p.ParseImports(filePath)
	if err != nil {
		t.Fatalf("ParseImports failed: %v", err)
	}

	expected := map[string]bool{
		filepath.Join(tmpDir, "helper.ts"): false,
		filepath.Join(tmpDir, "types.ts"):  false,
	}

	for _, r := range result.Resolved {
		if _, ok := expected[r.Path]; ok {
			expected[r.Path] = true
		}
	}

	for path, found := range expected {
		if !found {
			t.Errorf("Re-export target not resolved: %s (resolved=%v)", path, result.Resolved)
		}
	}
}

// TestResolveAlias_Unit tests the ResolveAlias helper in isolation.
func TestResolveAlias_Unit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest_alias_unit")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write a tsconfig.json with @/* -> ./src/*
	tsconfig := `{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"],
      "~/*": ["./src/*"],
      "@utils": ["./src/utils"]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte(tsconfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Invalidate cache so we pick up the temp tsconfig.
	InvalidateTSConfigCache(tmpDir)

	tests := []struct {
		importPath string
		wantSuffix string // suffix of expected absolute path
		wantOK     bool
	}{
		{"@/services/api", "src/services/api", true},
		{"~/components/button", "src/components/button", true},
		{"@utils", "src/utils", true},
		{"react", "", false},
		{"./relative", "", false},
	}

	for _, tc := range tests {
		got, ok := ResolveAlias(tc.importPath, tmpDir)
		if ok != tc.wantOK {
			t.Errorf("ResolveAlias(%q) ok=%v, want %v", tc.importPath, ok, tc.wantOK)
			continue
		}
		if tc.wantOK && !filepath.IsAbs(got) {
			t.Errorf("ResolveAlias(%q) returned non-absolute path: %q", tc.importPath, got)
		}
		if tc.wantSuffix != "" {
			if !hasSuffixPath(got, tc.wantSuffix) {
				t.Errorf("ResolveAlias(%q) = %q, want suffix %q", tc.importPath, got, tc.wantSuffix)
			}
		}
	}
}

// hasSuffixPath checks that path ends with the given suffix (OS-agnostic).
func hasSuffixPath(path, suffix string) bool {
	// Normalise separators.
	path = filepath.ToSlash(path)
	suffix = filepath.ToSlash(suffix)
	return len(path) >= len(suffix) &&
		(path == suffix || path[len(path)-len(suffix)-1] == '/' && path[len(path)-len(suffix):] == suffix)
}

// TestGraph_TSAlias is an integration test: graph correctly links a test file
// importing via @/... alias to its target source file.
func TestGraph_TSAlias(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest_alias_graph")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure:
	// src/utils.ts
	// src/api.ts      -> imports @/utils
	// tests/api.test.ts -> imports @/api
	srcDir := filepath.Join(tmpDir, "src")
	testDir := filepath.Join(tmpDir, "tests")
	for _, d := range []string{srcDir, testDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		"src/utils.ts":      "export const util = 1;",
		"src/api.ts":        "import { util } from '@/utils';",
		"tests/api.test.ts": "import { api } from '@/api';",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// tsconfig.json at project root.
	tsconfig := `{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte(tsconfig), 0644); err != nil {
		t.Fatal(err)
	}

	InvalidateTSConfigCache(tmpDir)

	g := NewGraphWithRoot(tmpDir)
	if err := g.Build(tmpDir); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	utilsPath := filepath.Join(tmpDir, "src/utils.ts")
	dependents := g.GetAffectedDependents(utilsPath)

	// Expect src/api.ts and transitively tests/api.test.ts
	depSet := make(map[string]bool)
	for _, d := range dependents {
		depSet[d] = true
	}

	apiPath := filepath.Join(tmpDir, "src/api.ts")
	testPath := filepath.Join(tmpDir, "tests/api.test.ts")

	if !depSet[apiPath] {
		t.Errorf("Expected src/api.ts to depend on src/utils.ts; got dependents=%v", dependents)
	}
	if !depSet[testPath] {
		t.Errorf("Expected tests/api.test.ts to transitively depend on src/utils.ts; got dependents=%v", dependents)
	}
}

// TestGetAffectedDependents_MockBoundary is the primary verification test for
// mock-aware BFS traversal. It sets up:
//
//	leaf.ts → middle.ts → unmocked.test.ts  (regular import)
//	                    → mocked.test.ts    (regular import + jest.mock('./middle'))
//
// GetAffectedDependents("leaf.ts") should return middle.ts and unmocked.test.ts
// but NOT mocked.test.ts, because mocked.test.ts has mocked middle.ts away,
// insulating itself from changes in leaf.ts.
func TestGetAffectedDependents_MockBoundary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest_affected_deps")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	files := map[string]string{
		"leaf.ts":          "export const value = 1;",
		"middle.ts":        "import { value } from './leaf';",
		"unmocked.test.ts": "import { value } from './middle';",
		"mocked.test.ts": `
import { value } from './middle';
jest.mock('./middle');
`,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	g := NewGraph()
	if err := g.Build(tmpDir); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	leafPath := filepath.Join(tmpDir, "leaf.ts")
	middlePath := filepath.Join(tmpDir, "middle.ts")
	unmockedPath := filepath.Join(tmpDir, "unmocked.test.ts")
	mockedPath := filepath.Join(tmpDir, "mocked.test.ts")

	affected := g.GetAffectedDependents(leafPath)

	affectedSet := make(map[string]bool)
	for _, p := range affected {
		affectedSet[p] = true
	}

	// middle.ts directly imports leaf.ts and is not mocked — must be included.
	if !affectedSet[middlePath] {
		t.Errorf("Expected middle.ts to be in affected dependents of leaf.ts; got %v", affected)
	}

	// unmocked.test.ts imports middle.ts without mocking — must be included.
	if !affectedSet[unmockedPath] {
		t.Errorf("Expected unmocked.test.ts to be in affected dependents of leaf.ts; got %v", affected)
	}

	// mocked.test.ts mocks middle.ts, so changes in leaf.ts should NOT reach it.
	if affectedSet[mockedPath] {
		t.Errorf("Expected mocked.test.ts NOT to be in affected dependents of leaf.ts (mock boundary); got %v", affected)
	}
}

// TestGraph_MockedDependency verifies that jest.mock, jest.doMock, and jest.setMock
// mark dependencies as DepMocked, insulating tests from GetAffectedDependents.
func TestGraph_MockedDependency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest_mocked_dep")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	files := map[string]string{
		"utils.ts":     "export const foo = 'bar';",
		"real.test.ts": "import { foo } from './utils';",
		"mocked.test.ts": `
import { foo } from './utils';
jest.mock('./utils');
`,
		"domock.test.ts": `
import { foo } from './utils';
jest.doMock('./utils', () => {});
`,
		"setmock.test.ts": `
import { foo } from './utils';
jest.setMock('./utils', {});
`,
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	g := NewGraph()
	if err := g.Build(tmpDir); err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	utilsPath := filepath.Join(tmpDir, "utils.ts")
	affected := g.GetAffectedDependents(utilsPath)

	realTestPath := filepath.Join(tmpDir, "real.test.ts")
	mockedTestPath := filepath.Join(tmpDir, "mocked.test.ts")
	doMockTestPath := filepath.Join(tmpDir, "domock.test.ts")
	setMockTestPath := filepath.Join(tmpDir, "setmock.test.ts")

	// GetAffectedDependents should include real.test.ts but exclude mocked files
	if len(affected) != 1 || affected[0] != realTestPath {
		t.Errorf("Expected affected dependents [%s], got %v", realTestPath, affected)
	}

	// Verify Forward map has DepMocked for mocked tests
	if g.Forward[mockedTestPath][utilsPath] != DepMocked {
		t.Errorf("Expected mocked.test.ts to have DepMocked for utils.ts")
	}
	if g.Forward[doMockTestPath][utilsPath] != DepMocked {
		t.Errorf("Expected domock.test.ts to have DepMocked for utils.ts")
	}
	if g.Forward[setMockTestPath][utilsPath] != DepMocked {
		t.Errorf("Expected setmock.test.ts to have DepMocked for utils.ts")
	}
}
