package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// RunnerInfo holds the name and command template for a detected test runner.
type RunnerInfo struct {
	Name    string
	Command string
}

// knownRunners defines the priority-ordered list of supported test runners.
var knownRunners = []RunnerInfo{
	{"vitest", "npx vitest run <path>"},
	{"jest", "npx jest <path> --colors"},
	{"mocha", "npx mocha <path>"},
	{"@playwright/test", "npx playwright test <path>"},
}

// DetectRunner reads package.json at root and returns the first recognized test
// runner found in devDependencies then dependencies. Falls back to Node's
// built-in test runner when nothing is matched.
func DetectRunner(root string) RunnerInfo {
	pkgPath := filepath.Join(root, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return RunnerInfo{"node", "node --test <path>"}
	}

	var pkg struct {
		DevDependencies map[string]string `json:"devDependencies"`
		Dependencies    map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return RunnerInfo{"node", "node --test <path>"}
	}

	for _, runner := range knownRunners {
		if _, ok := pkg.DevDependencies[runner.Name]; ok {
			return runner
		}
		if _, ok := pkg.Dependencies[runner.Name]; ok {
			return runner
		}
	}

	return RunnerInfo{"node", "node --test <path>"}
}

// Config holds the configuration for the test runner.
type Config struct {
	Command            string     `json:"command"`
	MaxConcurrentTests int        `json:"max_concurrent_tests,omitempty"`
	Overrides          []Override `json:"overrides,omitempty"`
	Excludes           []string   `json:"excludes,omitempty"`
	DetectedRunner     string     `json:"-"` // Not serialized; set at load time
}

// Override defines a custom command for a specific file pattern.
type Override struct {
	Pattern string `json:"pattern"`
	Command string `json:"command"`
}

// GetExecutionRoot finds the nearest package.json starting from the test file path and walking up.
func GetExecutionRoot(testFilePath string) (string, error) {
	dir := filepath.Dir(testFilePath)
	for {
		if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// LoadConfig looks for .lazytest.json starting from root and walking up.
// If not found, auto-detects the runner from package.json via DetectRunner.
// An explicit .lazytest.json always takes precedence over auto-detection.
func LoadConfig(root string) Config {
	defaultConcurrency := runtime.NumCPU() / 2
	if defaultConcurrency < 1 {
		defaultConcurrency = 1
	}

	dir := root
	for {
		configFile := filepath.Join(dir, ".lazytest.json")
		if _, err := os.Stat(configFile); err == nil {
			// Found explicit config — it always wins.
			data, err := os.ReadFile(configFile)
			if err != nil {
				break
			}

			var config Config
			if err := json.Unmarshal(data, &config); err != nil {
				break
			}

			detected := DetectRunner(root)
			if config.Command == "" {
				config.Command = detected.Command
			}
			if config.MaxConcurrentTests <= 0 {
				config.MaxConcurrentTests = defaultConcurrency
			}
			config.DetectedRunner = detected.Name
			return config
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached system root, not found
			break
		}
		dir = parent
	}

	// No .lazytest.json found — auto-detect from package.json.
	detected := DetectRunner(root)
	return Config{
		Command:            detected.Command,
		DetectedRunner:     detected.Name,
		MaxConcurrentTests: defaultConcurrency,
	}
}

// LoadConfigForPath resolves the config for a specific test file path.
// When the file falls inside a recognized workspace package, that package's
// config is returned. Otherwise it falls back to root-level LoadConfig.
func LoadConfigForPath(testPath string, workspaces []Workspace) Config {
	for _, ws := range workspaces {
		if strings.HasPrefix(testPath, ws.Root+string(filepath.Separator)) ||
			testPath == ws.Root {
			return ws.Config
		}
	}
	// Fallback: walk up from the test file's directory.
	return LoadConfig(filepath.Dir(testPath))
}

// BuildCommandString constructs the final command string to execute.
func BuildCommandString(template string, testPath string) (string, []string) {
	// Simple replacement for MVP
	// In a real app, we might use a template engine
	cmdStr := template
	if strings.Contains(template, "<path>") {
		cmdStr = strings.ReplaceAll(template, "<path>", testPath)
	} else {
		// If <path> is not specified, append it to the end
		cmdStr = fmt.Sprintf("%s %s", template, testPath)
	}

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}
