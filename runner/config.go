package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
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

// LoadConfig looks for .lazytest.json in the project root.
// If not found, returns default config.
func LoadConfig(root string) Config {
	defaultConfig := Config{
		Command: "npx jest <path> --colors",
	}

	configFile := filepath.Join(root, ".lazytest.json")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return defaultConfig
	}

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

// BuildCommandString constructs the final command string to execute.
func BuildCommandString(template string, testPath string) (string, []string) {
	// Simple replacement for MVP
	// In a real app, we might use a template engine
	cmdStr := strings.ReplaceAll(template, "<path>", testPath)
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}
