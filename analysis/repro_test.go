package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestGraph_LateCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest_repro_late")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Create dependent file (test) FIRST
	// It imports './utils', which does not exist yet.
	testContent := "import { foo } from './utils';"
	testPath := filepath.Join(tmpDir, "utils.test.ts")
	if err := os.WriteFile(testPath, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Build Graph
	g := NewGraph()
	if err := g.Build(tmpDir); err != nil {
		t.Fatal(err)
	}

	// 3. Create dependency file (utils) LATER
	utilsPath := filepath.Join(tmpDir, "utils.ts")
	if err := os.WriteFile(utilsPath, []byte("export const foo = 'bar';"), 0644); err != nil {
		t.Fatal(err)
	}

	// 4. Update Graph for the new file
	g.Update(utilsPath)

	// 5. Check if dependency is linked
	dependents := g.GetDependents(utilsPath)
	if len(dependents) == 0 {
		t.Error("Failed to link dependency when file is created after dependent")
	}
}
