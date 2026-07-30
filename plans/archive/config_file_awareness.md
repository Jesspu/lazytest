# Config File Awareness Plan

This document outlines the plan to expand `IsConfigFile` to recognize configuration files for all supported test runners, ensuring live-reload and graph rebuilds trigger correctly.

## Goals
- Detect config files for Vitest, Mocha, Playwright, and pnpm in addition to existing Jest/Babel/Webpack support.
- Trigger config reload and test re-queue when any of these files change.
- Keep the predicate maintainable as new runners are added.

## Current Behavior

`filesystem/predicates.go` `IsConfigFile` recognizes:
- `package.json`
- `tsconfig.json`
- `.lazytest.json`
- `vite.config.*`
- `jest.config.*`
- `babel.config.*`
- `webpack.config.*`

Missing config files that affect test behavior:
- `vitest.config.*` (Vitest-specific config, separate from `vite.config`)
- `.mocharc.*` (`.mocharc.yml`, `.mocharc.json`, `.mocharc.js`, etc.)
- `playwright.config.*`
- `pnpm-workspace.yaml`
- `.babelrc` (Babel legacy config format)
- `vitest.workspace.*` (Vitest workspace config)

## Proposed Changes

### 1. Expand `IsConfigFile` (`filesystem/predicates.go`)

Refactor the function to use a data-driven approach for cleaner maintenance.

```go
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
```

### 2. Runner-Specific Reload Behavior (`engine/engine.go`)

Currently, any config file change triggers a full config reload and test re-queue. This is appropriate for most cases, but certain config files (like `tsconfig.json`) should additionally rebuild the dependency graph since they affect path resolution.

The current behavior already handles this correctly — `tsconfig.json` changes trigger a graph rebuild because it's caught by the generic `IsConfigFile` check in the `WatcherMsg` handler. No change needed here, but document the mapping:

| Config File | Reload Config | Rebuild Graph | Re-queue Tests |
|---|---|---|---|
| `.lazytest.json` | ✅ | ✅ | ✅ |
| `package.json` | ✅ | ✅ | ✅ |
| `tsconfig.json` | ✅ | ✅ | ✅ |
| `jest.config.*` | ✅ | ❌ | ✅ |
| `vitest.config.*` | ✅ | ❌ | ✅ |
| `.mocharc.*` | ✅ | ❌ | ✅ |
| `playwright.config.*` | ✅ | ❌ | ✅ |
| `pnpm-workspace.yaml` | ✅ | ✅ | ✅ |

### 3. Update Existing Tests

Update `engine/engine_test.go` config change tests to verify the new config file patterns are recognized.

## Implementation Steps

1. **Refactor `IsConfigFile`** to use `configBasenames` and `configPrefixes` slices.
2. **Add the missing entries**: `vitest.config.*`, `.mocharc.*`, `playwright.config.*`, `pnpm-workspace.yaml`, `.babelrc`, `vitest.workspace.*`.
3. **Add unit tests** in `filesystem/` for each new pattern.
4. **Verify** that modifying a `vitest.config.ts` or `.mocharc.yml` triggers a proper reload.
