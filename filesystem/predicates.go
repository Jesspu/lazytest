package filesystem

import (
	"path/filepath"
	"strings"
)

// IsTestFile checks if a file is a test file based on its extension.
func IsTestFile(name string) bool {
	return strings.HasSuffix(name, ".test.ts") ||
		strings.HasSuffix(name, ".test.js") ||
		strings.HasSuffix(name, ".spec.ts") ||
		strings.HasSuffix(name, ".spec.js") ||
		strings.HasSuffix(name, ".test.tsx") ||
		strings.HasSuffix(name, ".test.jsx") ||
		strings.HasSuffix(name, ".spec.tsx") ||
		strings.HasSuffix(name, ".spec.jsx")
}

// IsSourceFile checks if a file is a compilable source file.
func IsSourceFile(name string) bool {
	exts := []string{".ts", ".js", ".tsx", ".jsx"}
	for _, ext := range exts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// configBasenames are exact-match config file names.
var configBasenames = []string{
	"package.json",
	"tsconfig.json",
	".lazytest.json",
	"pnpm-workspace.yaml",
	".babelrc",
}

// configPrefixes are prefix-match patterns for config files
// that may have varying extensions (e.g., .js, .ts, .json, .yml).
var configPrefixes = []string{
	"vite.config.",
	"vitest.config.",
	"vitest.workspace.",
	"jest.config.",
	"babel.config.",
	"webpack.config.",
	"playwright.config.",
	".mocharc.",
}

// IsConfigFile checks if a file is a configuration file that might affect tests.
func IsConfigFile(name string) bool {
	base := filepath.Base(name)
	for _, exact := range configBasenames {
		if base == exact {
			return true
		}
	}
	for _, prefix := range configPrefixes {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}
	return false
}
