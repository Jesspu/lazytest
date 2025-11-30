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

// IsConfigFile checks if a file is a configuration file that might affect tests.
func IsConfigFile(name string) bool {
	base := filepath.Base(name)
	return base == "package.json" ||
		base == "tsconfig.json" ||
		strings.HasPrefix(base, "vite.config.") ||
		strings.HasPrefix(base, "jest.config.") ||
		strings.HasPrefix(base, "babel.config.") ||
		strings.HasPrefix(base, "webpack.config.")
}
