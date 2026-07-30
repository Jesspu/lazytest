# Test File Detection Plan

This document outlines the plan to improve test file detection by adding directory-based heuristics alongside the existing suffix-based checks, covering Mocha, Node's built-in test runner, and other conventions.

## Goals
- Detect test files in `test/`, `tests/`, and `__tests__/` directories even without `.test.` or `.spec.` suffixes.
- Avoid false positives (e.g., test helpers, fixtures, config files inside test directories).
- Make detection aware of the detected runner so it can apply runner-specific patterns.

## Current Behavior

`filesystem/predicates.go` `IsTestFile` only matches files ending in `.test.{ts,js,tsx,jsx}` or `.spec.{ts,js,tsx,jsx}`. This misses:
- `test/math.js` (Mocha convention)
- `test/add.mjs` (Node `--test` convention)
- Files in `__tests__/` without `.test.` in the name (less common but valid in Jest)

## Proposed Changes

### 1. Directory-Based Detection (`filesystem/predicates.go`)

Add a new function `IsTestFileByPath(absPath string) bool` that considers the full path, not just the filename.

```go
// testDirs lists directory names that conventionally contain test files.
var testDirs = []string{"test", "tests", "__tests__"}

// IsTestFileByPath checks if a file is a test based on its name OR
// its location inside a known test directory.
func IsTestFileByPath(absPath string) bool {
    // 1. Suffix-based (existing logic)
    if IsTestFile(absPath) {
        return true
    }

    // 2. Directory-based: check if any path segment is a test directory
    parts := strings.Split(filepath.ToSlash(absPath), "/")
    for _, part := range parts {
        for _, dir := range testDirs {
            if part == dir {
                // Must still be a runnable source file, not a fixture/json/etc.
                return IsSourceFile(absPath)
            }
        }
    }

    return false
}
```

### 2. Exclude Helpers and Fixtures

Files inside test directories that are clearly not test cases should be excluded:
- Files starting with `_` or `.` (e.g., `_helpers.js`, `.eslintrc.js`)
- Files matching common helper patterns: `setup.{js,ts}`, `helper.{js,ts}`, `fixtures/**`

```go
var testHelperPatterns = []string{"setup", "helper", "helpers", "fixture", "fixtures", "mock", "mocks", "utils"}

func isTestHelper(name string) bool {
    base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
    base = strings.ToLower(base)
    for _, pattern := range testHelperPatterns {
        if base == pattern {
            return true
        }
    }
    return strings.HasPrefix(filepath.Base(name), "_")
}
```

### 3. Update Call Sites

Replace `IsTestFile(path)` with `IsTestFileByPath(path)` in:
- `filesystem/walker.go` — tree construction
- `engine/actions.go` — `FindRelatedTests`
- `engine/engine.go` — Smart Mode test discovery
- `analysis/graph.go` — graph node classification

### 4. Add `.mjs` and `.cjs` Support

Node's built-in test runner supports `.mjs` and `.cjs` files. Update `IsSourceFile` to include these extensions.

```go
func IsSourceFile(name string) bool {
    exts := []string{".ts", ".js", ".tsx", ".jsx", ".mjs", ".cjs"}
    // ...
}
```

## Implementation Steps

1. **Add `IsTestFileByPath`** to `filesystem/predicates.go`.
2. **Add helper exclusion logic** (`isTestHelper`).
3. **Add `.mjs` / `.cjs`** to `IsSourceFile`.
4. **Update call sites** to use `IsTestFileByPath` where full paths are available.
5. **Add tests** covering directory-based detection, helper exclusion, and new extensions.
6. **Verify** the explorer correctly shows test files from `test/` directories.
