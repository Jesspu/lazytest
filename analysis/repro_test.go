package analysis

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestGraph_MockedDependency(t *testing.T) {
	// Setup temporary test directory
	tmpDir, err := os.MkdirTemp("", "lazytest_repro_mocked")
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

	// Build Graph
	g := NewGraph()
	if err := g.Build(tmpDir); err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Test GetDependents for utils.ts
	utilsPath := filepath.Join(tmpDir, "utils.ts")
	dependents := g.GetDependents(utilsPath)

	expected := []string{
		filepath.Join(tmpDir, "real.test.ts"),
		filepath.Join(tmpDir, "mocked.test.ts"),
		filepath.Join(tmpDir, "domock.test.ts"),
		filepath.Join(tmpDir, "setmock.test.ts"),
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

	// Check dependency types
	realTestPath := filepath.Join(tmpDir, "real.test.ts")
	mockedTestPath := filepath.Join(tmpDir, "mocked.test.ts")
	doMockTestPath := filepath.Join(tmpDir, "domock.test.ts")
	setMockTestPath := filepath.Join(tmpDir, "setmock.test.ts")

	if g.GetDependencyType(realTestPath, utilsPath) != DepRegular {
		t.Errorf("Expected real.test.ts to have regular dependency on utils.ts")
	}

	if g.GetDependencyType(mockedTestPath, utilsPath) != DepMocked {
		t.Errorf("Expected mocked.test.ts to have mocked dependency on utils.ts")
	}

	if g.GetDependencyType(doMockTestPath, utilsPath) != DepMocked {
		t.Errorf("Expected domock.test.ts to have mocked dependency on utils.ts")
	}

	if g.GetDependencyType(setMockTestPath, utilsPath) != DepMocked {
		t.Errorf("Expected setmock.test.ts to have mocked dependency on utils.ts")
	}
}
