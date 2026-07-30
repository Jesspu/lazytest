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
		strings.HasSuffix(name, ".spec.jsx") ||
		strings.HasSuffix(name, ".test.mjs") ||
		strings.HasSuffix(name, ".spec.mjs") ||
		strings.HasSuffix(name, ".test.cjs") ||
		strings.HasSuffix(name, ".spec.cjs")
}

// IsSourceFile checks if a file is a compilable source file.
func IsSourceFile(name string) bool {
	exts := []string{".ts", ".js", ".tsx", ".jsx", ".mjs", ".cjs"}
	for _, ext := range exts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// testDirs lists directory names that conventionally contain test files.
var testDirs = []string{"test", "tests", "__tests__"}

// testHelperPatterns lists base filenames (without extension) that are
// conventionally helpers, fixtures, or mocks rather than runnable test cases.
var testHelperPatterns = []string{
	"setup", "helper", "helpers", "fixture", "fixtures",
	"mock", "mocks", "utils",
}

// isTestHelper returns true for files that live inside a test directory but
// are not themselves test cases (e.g. setup.js, _helpers.ts, fixtures/data.js).
func isTestHelper(name string) bool {
	base := filepath.Base(name)
	// Files starting with "_" or "." are helpers/hidden files.
	if strings.HasPrefix(base, "_") || strings.HasPrefix(base, ".") {
		return true
	}
	// Strip extension and check against known helper names.
	// Use substring matching so compound names like "setupTest" or "testHelper"
	// are also excluded, not just exact matches like "setup" or "helper".
	ext := filepath.Ext(base)
	stem := strings.ToLower(strings.TrimSuffix(base, ext))
	for _, pattern := range testHelperPatterns {
		if strings.Contains(stem, pattern) {
			return true
		}
	}
	// Any file inside a "fixtures" or "mocks" sub-directory.
	parts := strings.Split(filepath.ToSlash(name), "/")
	for _, part := range parts[:max(0, len(parts)-1)] {
		if part == "fixtures" || part == "mocks" || part == "mock" || part == "__mocks__" {
			return true
		}
	}
	return false
}

// IsTestFileByPath checks whether a file should be treated as a test based on
// its suffix (existing logic) OR its location inside a known test directory.
// Files that look like helpers/fixtures are excluded even when inside test dirs.
func IsTestFileByPath(absPath string) bool {
	// 1. Suffix-based — always wins.
	if IsTestFile(absPath) {
		return true
	}

	// 2. Directory-based: any path segment matching a test directory name.
	parts := strings.Split(filepath.ToSlash(absPath), "/")
	for _, part := range parts {
		for _, dir := range testDirs {
			if part == dir {
				// Must be a runnable source file (not json, yaml, md, …)
				if !IsSourceFile(absPath) {
					return false
				}
				// Exclude helpers and fixtures.
				if isTestHelper(absPath) {
					return false
				}
				return true
			}
		}
	}

	return false
}

// max is a small helper for slice bounds (Go 1.21+ has a built-in, but kept
// here for compatibility with older toolchains in this repo).
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
