package runner

import (
	"os"
	"path/filepath"
	"testing"
)

// makePackageJSON writes a package.json with the given deps into dir.
func makePackageJSON(t *testing.T, dir string, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// --- DetectRunner ---

func TestDetectRunner_Vitest(t *testing.T) {
	tmp := t.TempDir()
	makePackageJSON(t, tmp, `{"devDependencies": {"vitest": "^1.0.0"}}`)
	r := DetectRunner(tmp)
	if r.Name != "vitest" {
		t.Errorf("expected vitest, got %q", r.Name)
	}
	if r.Command != "npx vitest run <path>" {
		t.Errorf("unexpected command %q", r.Command)
	}
}

func TestDetectRunner_Jest(t *testing.T) {
	tmp := t.TempDir()
	makePackageJSON(t, tmp, `{"devDependencies": {"jest": "^29.0.0"}}`)
	r := DetectRunner(tmp)
	if r.Name != "jest" {
		t.Errorf("expected jest, got %q", r.Name)
	}
	if r.Command != "npx jest <path> --colors" {
		t.Errorf("unexpected command %q", r.Command)
	}
}

func TestDetectRunner_Mocha(t *testing.T) {
	tmp := t.TempDir()
	makePackageJSON(t, tmp, `{"devDependencies": {"mocha": "^10.0.0"}}`)
	r := DetectRunner(tmp)
	if r.Name != "mocha" {
		t.Errorf("expected mocha, got %q", r.Name)
	}
}

func TestDetectRunner_Playwright(t *testing.T) {
	tmp := t.TempDir()
	makePackageJSON(t, tmp, `{"devDependencies": {"@playwright/test": "^1.0.0"}}`)
	r := DetectRunner(tmp)
	if r.Name != "@playwright/test" {
		t.Errorf("expected @playwright/test, got %q", r.Name)
	}
	if r.Command != "npx playwright test <path>" {
		t.Errorf("unexpected command %q", r.Command)
	}
}

func TestDetectRunner_DependenciesFallback(t *testing.T) {
	// Runner listed in `dependencies` (not devDependencies) should still be detected.
	tmp := t.TempDir()
	makePackageJSON(t, tmp, `{"dependencies": {"jest": "^29.0.0"}}`)
	r := DetectRunner(tmp)
	if r.Name != "jest" {
		t.Errorf("expected jest, got %q", r.Name)
	}
}

func TestDetectRunner_Priority_VitestBeforeJest(t *testing.T) {
	// When both vitest and jest are present, vitest must win.
	tmp := t.TempDir()
	makePackageJSON(t, tmp, `{"devDependencies": {"jest": "^29.0.0", "vitest": "^1.0.0"}}`)
	r := DetectRunner(tmp)
	if r.Name != "vitest" {
		t.Errorf("expected vitest to take priority, got %q", r.Name)
	}
}

func TestDetectRunner_NoPackageJSON(t *testing.T) {
	// No package.json → node built-in.
	tmp := t.TempDir()
	r := DetectRunner(tmp)
	if r.Name != "node" {
		t.Errorf("expected node fallback, got %q", r.Name)
	}
	if r.Command != "node --test <path>" {
		t.Errorf("unexpected command %q", r.Command)
	}
}

func TestDetectRunner_NoKnownRunner(t *testing.T) {
	// package.json exists but has no recognized runner → node built-in.
	tmp := t.TempDir()
	makePackageJSON(t, tmp, `{"devDependencies": {"typescript": "^5.0.0"}}`)
	r := DetectRunner(tmp)
	if r.Name != "node" {
		t.Errorf("expected node fallback, got %q", r.Name)
	}
}

// --- LoadConfig ---

func TestLoadConfig_ExplicitOverride(t *testing.T) {
	// .lazytest.json must take precedence over auto-detection.
	tmp := t.TempDir()
	configContent := `{"command": "echo 'Monorepo Config' --"}`
	if err := os.WriteFile(filepath.Join(tmp, ".lazytest.json"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Also place a vitest package.json — the explicit config should still win for Command.
	makePackageJSON(t, tmp, `{"devDependencies": {"vitest": "^1.0.0"}}`)

	config := LoadConfig(tmp)
	expected := "echo 'Monorepo Config' --"
	if config.Command != expected {
		t.Errorf("expected command %q, got %q", expected, config.Command)
	}
}

func TestLoadConfig_MonorepoWalkUp(t *testing.T) {
	// .lazytest.json at a parent should be found when searching from a nested dir.
	tmpDir, err := os.MkdirTemp("", "lazytest-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configContent := `{"command": "echo 'Monorepo Config' --"}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".lazytest.json"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	appDir := filepath.Join(tmpDir, "packages", "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	config := LoadConfig(appDir)
	expected := "echo 'Monorepo Config' --"
	if config.Command != expected {
		t.Errorf("expected command %q, got %q", expected, config.Command)
	}
}

func TestLoadConfig_Default_NodeFallback(t *testing.T) {
	// No .lazytest.json and no package.json → node built-in.
	tmp := t.TempDir()
	config := LoadConfig(tmp)
	expected := "node --test <path>"
	if config.Command != expected {
		t.Errorf("expected default command %q, got %q", expected, config.Command)
	}
	if config.DetectedRunner != "node" {
		t.Errorf("expected DetectedRunner %q, got %q", "node", config.DetectedRunner)
	}
}

func TestLoadConfig_Default_AutoDetectVitest(t *testing.T) {
	// No .lazytest.json but package.json with vitest → auto-detect.
	tmp := t.TempDir()
	makePackageJSON(t, tmp, `{"devDependencies": {"vitest": "^1.0.0"}}`)
	config := LoadConfig(tmp)
	if config.Command != "npx vitest run <path>" {
		t.Errorf("expected vitest command, got %q", config.Command)
	}
	if config.DetectedRunner != "vitest" {
		t.Errorf("expected DetectedRunner vitest, got %q", config.DetectedRunner)
	}
}
