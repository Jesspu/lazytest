package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the configuration for the test runner.
type Config struct {
	Command   string     `json:"command"`
	Overrides []Override `json:"overrides,omitempty"`
	Excludes  []string   `json:"excludes,omitempty"`
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
// If not found, returns default config.
func LoadConfig(root string) Config {
	defaultConfig := Config{
		Command: "npx jest <path> --colors",
	}

	dir := root
	for {
		configFile := filepath.Join(dir, ".lazytest.json")
		if _, err := os.Stat(configFile); err == nil {
			// Found it
			data, err := os.ReadFile(configFile)
			if err != nil {
				return defaultConfig
			}

			var config Config
			if err := json.Unmarshal(data, &config); err != nil {
				return defaultConfig
			}

			if config.Command == "" {
				config.Command = defaultConfig.Command
			}
			return config
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached system root, not found
			break
		}
		dir = parent
	}

	return defaultConfig
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
