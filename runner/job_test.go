package runner

import (
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

		job, err := PrepareJob(testFile)
		if err != nil {
			t.Fatalf("PrepareJob failed: %v", err)
		}

		if job.Root != tmpDir {
			t.Errorf("Expected root %s, got %s", tmpDir, job.Root)
		}

		// Default command is "npx jest <path> --colors"
		// Relative path from root to test file is src/foo.test.js
		expectedCmd := "npx"
		expectedArgsLen := 3 // jest, src/foo.test.js, --colors

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

		job, err := PrepareJob(testFile)
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
		job1, err := PrepareJob(pkgTest)
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
		job2, err := PrepareJob(specialTest)
		if err != nil {
			t.Fatal(err)
		}
		if job2.Command != "special" {
			t.Errorf("Expected special command, got %s", job2.Command)
		}

		// Test 3: Default fallback
		normalTest := filepath.Join(tmpDir, "src", "normal.test.js")
		job3, err := PrepareJob(normalTest)
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

		_, err = PrepareJob(testFile)
		if err == nil {
			t.Error("Expected error when no package.json found, got nil")
		}
	})
}
