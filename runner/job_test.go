package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareJob(t *testing.T) {
	t.Run("Default Config", func(t *testing.T) {
		// Create temp dir acting as root
		tmpDir, err := os.MkdirTemp("", "lazytest-job-default")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Create package.json to mark it as root
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a dummy test file
		testFile := filepath.Join(tmpDir, "src", "foo.test.js")
		if err := os.MkdirAll(filepath.Dir(testFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		job, err := PrepareJob(testFile, nil)
		if err != nil {
			t.Fatalf("PrepareJob failed: %v", err)
		}

		if job.Root != tmpDir {
			t.Errorf("Expected root %s, got %s", tmpDir, job.Root)
		}

		// Default when package.json has no recognized runner is node --test <path>.
		// Relative path from root to test file is src/foo.test.js
		expectedCmd := "node"
		expectedArgsLen := 2 // --test, src/foo.test.js

		if job.Command != expectedCmd {
			t.Errorf("Expected command %s, got %s", expectedCmd, job.Command)
		}

		if len(job.Args) != expectedArgsLen {
			t.Errorf("Expected %d args, got %d: %v", expectedArgsLen, len(job.Args), job.Args)
		}
	})

	t.Run("Custom Config", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "lazytest-job-custom")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Create package.json
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create .lazytest.json with custom command
		configContent := `{"command": "go test -v <path>"}`
		if err := os.WriteFile(filepath.Join(tmpDir, ".lazytest.json"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		testFile := filepath.Join(tmpDir, "pkg", "foo_test.go")
		if err := os.MkdirAll(filepath.Dir(testFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		job, err := PrepareJob(testFile, nil)
		if err != nil {
			t.Fatalf("PrepareJob failed: %v", err)
		}

		expectedCmd := "go"
		// args: test, -v, pkg/foo_test.go

		if job.Command != expectedCmd {
			t.Errorf("Expected command %s, got %s", expectedCmd, job.Command)
		}

		// Check if args contains the file path
		foundPath := false
		for _, arg := range job.Args {
			if arg == filepath.Join("pkg", "foo_test.go") {
				foundPath = true
				break
			}
		}
		if !foundPath {
			t.Errorf("Expected args to contain test file path, got %v", job.Args)
		}
	})

	t.Run("Overrides", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "lazytest-job-overrides")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		configContent := `{
			"command": "default <path>",
			"overrides": [
				{"pattern": "pkg/**", "command": "pkg-test <path>"},
				{"pattern": "src/special.test.js", "command": "special <path>"}
			]
		}`
		if err := os.WriteFile(filepath.Join(tmpDir, ".lazytest.json"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Test 1: Pattern match (directory)
		pkgTest := filepath.Join(tmpDir, "pkg", "sub", "foo_test.go")
		if err := os.MkdirAll(filepath.Dir(pkgTest), 0755); err != nil {
			t.Fatal(err)
		}
		job1, err := PrepareJob(pkgTest, nil)
		if err != nil {
			t.Fatal(err)
		}
		if job1.Command != "pkg-test" {
			t.Errorf("Expected pkg-test command, got %s", job1.Command)
		}

		// Test 2: Exact match
		specialTest := filepath.Join(tmpDir, "src", "special.test.js")
		if err := os.MkdirAll(filepath.Dir(specialTest), 0755); err != nil {
			t.Fatal(err)
		}
		job2, err := PrepareJob(specialTest, nil)
		if err != nil {
			t.Fatal(err)
		}
		if job2.Command != "special" {
			t.Errorf("Expected special command, got %s", job2.Command)
		}

		// Test 3: Default fallback
		normalTest := filepath.Join(tmpDir, "src", "normal.test.js")
		job3, err := PrepareJob(normalTest, nil)
		if err != nil {
			t.Fatal(err)
		}
		if job3.Command != "default" {
			t.Errorf("Expected default command, got %s", job3.Command)
		}
	})

	t.Run("No Root", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "lazytest-job-noroot")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// No package.json created

		testFile := filepath.Join(tmpDir, "foo.test.js")
		if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		_, err = PrepareJob(testFile, nil)
		if err == nil {
			t.Error("Expected error when no package.json found, got nil")
		}
	})
}

// TestWorkspaceRouting verifies that PrepareJob selects the correct
// per-package runner when workspaces are provided.
func TestWorkspaceRouting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-ws-routing")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Root package.json (no workspaces field needed for this test)
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Package A: uses vitest
	pkgA := filepath.Join(tmpDir, "packages", "app-a")
	if err := os.MkdirAll(pkgA, 0755); err != nil {
		t.Fatal(err)
	}
	pkgAJSON, _ := json.Marshal(map[string]interface{}{
		"name":           "@scope/app-a",
		"devDependencies": map[string]string{"vitest": "^1.0.0"},
	})
	if err := os.WriteFile(filepath.Join(pkgA, "package.json"), pkgAJSON, 0644); err != nil {
		t.Fatal(err)
	}

	// Package B: uses jest
	pkgB := filepath.Join(tmpDir, "packages", "app-b")
	if err := os.MkdirAll(pkgB, 0755); err != nil {
		t.Fatal(err)
	}
	pkgBJSON, _ := json.Marshal(map[string]interface{}{
		"name":           "@scope/app-b",
		"devDependencies": map[string]string{"jest": "^29.0.0"},
	})
	if err := os.WriteFile(filepath.Join(pkgB, "package.json"), pkgBJSON, 0644); err != nil {
		t.Fatal(err)
	}

	workspaces := []Workspace{
		{Name: "@scope/app-a", Root: pkgA, Config: LoadConfig(pkgA)},
		{Name: "@scope/app-b", Root: pkgB, Config: LoadConfig(pkgB)},
	}

	// Test file in app-a
	testA := filepath.Join(pkgA, "src", "foo.test.ts")
	if err := os.MkdirAll(filepath.Dir(testA), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testA, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	jobA, err := PrepareJob(testA, workspaces)
	if err != nil {
		t.Fatalf("PrepareJob(app-a): %v", err)
	}
	if jobA.Command != "npx" {
		t.Errorf("app-a: expected 'npx', got %q", jobA.Command)
	}
	if len(jobA.Args) == 0 || jobA.Args[0] != "vitest" {
		t.Errorf("app-a: expected first arg 'vitest', got %v", jobA.Args)
	}

	// Test file in app-b
	testB := filepath.Join(pkgB, "src", "bar.test.ts")
	if err := os.MkdirAll(filepath.Dir(testB), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testB, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	jobB, err := PrepareJob(testB, workspaces)
	if err != nil {
		t.Fatalf("PrepareJob(app-b): %v", err)
	}
	if jobB.Command != "npx" {
		t.Errorf("app-b: expected 'npx', got %q", jobB.Command)
	}
	if len(jobB.Args) == 0 || jobB.Args[0] != "jest" {
		t.Errorf("app-b: expected first arg 'jest', got %v", jobB.Args)
	}
}

// TestDiscoverWorkspaces verifies npm, yarn, and pnpm workspace parsing.
func TestDiscoverWorkspaces(t *testing.T) {
	t.Run("npm plain array", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "lazytest-ws-npm")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		pkgJSON := `{"workspaces": ["packages/*"]}`
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
			t.Fatal(err)
		}

		// Create two packages
		for _, name := range []string{"alpha", "beta"} {
			dir := filepath.Join(tmpDir, "packages", name)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			data, _ := json.Marshal(map[string]string{"name": name})
			if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0644); err != nil {
				t.Fatal(err)
			}
		}

		ws := DiscoverWorkspaces(tmpDir)
		if len(ws) != 2 {
			t.Fatalf("expected 2 workspaces, got %d", len(ws))
		}
	})

	t.Run("pnpm-workspace.yaml", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "lazytest-ws-pnpm")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// No npm workspaces in package.json
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		yaml := "packages:\n  - 'apps/*'\n  - 'libs/*'\n"
		if err := os.WriteFile(filepath.Join(tmpDir, "pnpm-workspace.yaml"), []byte(yaml), 0644); err != nil {
			t.Fatal(err)
		}

		// Create packages
		for _, path := range []string{"apps/web", "libs/utils"} {
			dir := filepath.Join(tmpDir, filepath.FromSlash(path))
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			data, _ := json.Marshal(map[string]string{"name": filepath.Base(dir)})
			if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0644); err != nil {
				t.Fatal(err)
			}
		}

		ws := DiscoverWorkspaces(tmpDir)
		if len(ws) != 2 {
			t.Fatalf("expected 2 workspaces, got %d", len(ws))
		}
	})

	t.Run("no workspaces", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "lazytest-ws-none")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		ws := DiscoverWorkspaces(tmpDir)
		if len(ws) != 0 {
			t.Fatalf("expected 0 workspaces, got %d", len(ws))
		}
	})
}
