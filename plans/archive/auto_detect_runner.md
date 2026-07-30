# Auto-Detect Test Runner Plan

This document outlines the plan to automatically detect the project's test runner from `package.json`, eliminating the need for a `.lazytest.json` in the most common cases.

## Goals
- Detect Jest, Vitest, Mocha, Playwright, and Node's built-in test runner automatically.
- Remove the hard-coded `npx jest` default.
- Only fall back to a generic default when detection fails.
- Preserve `.lazytest.json` as an explicit override that always wins.

## Current Behavior

In `runner/config.go`, `LoadConfig` returns a hard-coded default of `npx jest <path> --colors` when no `.lazytest.json` is found. This means any non-Jest project requires manual configuration.

## Proposed Changes

### 1. Runner Detection Logic (`runner/config.go`)

Add a new function `DetectRunner(root string) string` that reads `package.json` and inspects `devDependencies` (then `dependencies`) for known test runner packages.

#### Detection Priority Order
1. `vitest` → `npx vitest run <path>`
2. `jest` → `npx jest <path> --colors`
3. `mocha` → `npx mocha <path>`
4. `@playwright/test` → `npx playwright test <path>`
5. None matched → `node --test <path>` (Node built-in runner)

#### Implementation

```go
type RunnerInfo struct {
    Name    string
    Command string
}

var knownRunners = []RunnerInfo{
    {"vitest", "npx vitest run <path>"},
    {"jest", "npx jest <path> --colors"},
    {"mocha", "npx mocha <path>"},
    {"@playwright/test", "npx playwright test <path>"},
}

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
```

### 2. Integrate Detection into `LoadConfig` (`runner/config.go`)

Update `LoadConfig` to call `DetectRunner` when no `.lazytest.json` is found, instead of using the hard-coded Jest default.

```go
func LoadConfig(root string) Config {
    // ... existing .lazytest.json search ...

    // No config file found — auto-detect
    detected := DetectRunner(root)
    return Config{
        Command:            detected.Command,
        DetectedRunner:     detected.Name,
        MaxConcurrentTests: defaultConcurrency,
    }
}
```

Add `DetectedRunner string` to the `Config` struct (non-serialized) so the UI can report what was detected.

### 3. Expose Detected Runner Name (`runner/config.go`)

```go
type Config struct {
    Command            string     `json:"command"`
    MaxConcurrentTests int        `json:"max_concurrent_tests,omitempty"`
    Overrides          []Override `json:"overrides,omitempty"`
    Excludes           []string   `json:"excludes,omitempty"`
    DetectedRunner     string     `json:"-"` // Not serialized
}
```

## Implementation Steps

1. **Add `DetectRunner`** function to `runner/config.go`.
2. **Update `LoadConfig`** to call `DetectRunner` when no `.lazytest.json` is found.
3. **Add `DetectedRunner` field** to `Config` struct for UI reporting.
4. **Update tests** in `runner/` to cover detection logic for each runner.
5. **Verify** that existing `.lazytest.json` overrides still take precedence.
