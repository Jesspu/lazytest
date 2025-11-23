package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory structure
	// /tmp/test-monorepo
	// ├── .lazytest.json
	// └── packages
	//     └── app (execRoot)

	tmpDir, err := os.MkdirTemp("", "lazytest-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .lazytest.json at root
	configContent := `{"command": "echo 'Monorepo Config' --"}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".lazytest.json"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create nested directory
	appDir := filepath.Join(tmpDir, "packages", "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test LoadConfig from appDir
	config := LoadConfig(appDir)

	expected := "echo 'Monorepo Config' --"
	if config.Command != expected {
		t.Errorf("Expected command %q, got %q", expected, config.Command)
	}
}

func TestLoadConfig_Default(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lazytest-config-default-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := LoadConfig(tmpDir)
	expected := "npx jest <path> --colors"
	if config.Command != expected {
		t.Errorf("Expected default command %q, got %q", expected, config.Command)
	}
}
