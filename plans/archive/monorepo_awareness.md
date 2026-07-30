# Monorepo Workspace Awareness Plan

This document outlines the plan to make LazyTest workspace-aware so it automatically uses the correct test runner per package in monorepo setups.

## Goals
- Parse `workspaces` from root `package.json` (npm/yarn) and `pnpm-workspace.yaml` (pnpm).
- Auto-detect the test runner independently for each workspace package.
- Route test execution to the correct package root without requiring per-package overrides in `.lazytest.json`.

## Current Behavior

`GetExecutionRoot` in `runner/job.go` walks up from the test file to find the nearest `package.json`. This correctly finds the package root in monorepos, but `LoadConfig` is only called once at startup with the top-level root. The result is that all tests run with the same command, even when packages use different runners (e.g., one package uses Jest, another uses Vitest).

## Proposed Changes

### 1. Workspace Discovery (`runner/workspace.go` — NEW)

Add a new file to detect and parse workspace definitions.

```go
type Workspace struct {
    Name    string
    Root    string // Absolute path to the package directory
    Config  Config // Per-package resolved config
}

// DiscoverWorkspaces scans the project root for workspace definitions
// and returns a list of Workspace entries.
func DiscoverWorkspaces(projectRoot string) []Workspace {
    // 1. Try npm/yarn: parse package.json "workspaces" field
    // 2. Try pnpm: parse pnpm-workspace.yaml "packages" field
    // 3. Expand glob patterns to actual directories
    // 4. For each directory, call DetectRunner + LoadConfig
}
```

#### npm/yarn `package.json` format
```json
{
    "workspaces": ["packages/*", "apps/*"]
}
```

#### pnpm `pnpm-workspace.yaml` format
```yaml
packages:
  - 'packages/*'
  - 'apps/*'
```

### 2. Per-Package Config Resolution (`runner/config.go`)

Update `LoadConfig` to accept an optional workspace context. When a test file is inside a recognized workspace package, use that package's detected runner instead of the root-level config.

```go
// LoadConfigForPath resolves the config for a specific test file path,
// checking workspace-level configs before falling back to root.
func LoadConfigForPath(testPath string, workspaces []Workspace) Config {
    for _, ws := range workspaces {
        if strings.HasPrefix(testPath, ws.Root) {
            return ws.Config
        }
    }
    // Fall back to root config
    return LoadConfig(filepath.Dir(testPath))
}
```

### 3. Update `PrepareJob` (`runner/job.go`)

Modify `PrepareJob` to accept the workspace list and resolve the config per test file.

```go
func PrepareJob(nodePath string, workspaces []Workspace) (*TestJob, error) {
    execRoot, err := GetExecutionRoot(nodePath)
    if err != nil {
        return nil, err
    }

    config := LoadConfigForPath(nodePath, workspaces)
    // ... rest unchanged
}
```

### 4. Engine Integration (`engine/engine.go`)

Store the discovered workspaces on the `Engine` struct. Rebuild on config change events.

```go
type Engine struct {
    // ...existing fields
    Workspaces []runner.Workspace
}

func New(rootPath string) *Engine {
    e := &Engine{
        // ...existing init
    }
    e.Workspaces = runner.DiscoverWorkspaces(rootPath)
    return e
}
```

Pass `e.Workspaces` through to `PrepareJob` in `TriggerTest`.

### 5. Config Reload (`engine/engine.go`)

When a `package.json` or `pnpm-workspace.yaml` change is detected, rebuild the workspace list:

```go
case WatcherMsg:
    if filesystem.IsConfigFile(path) {
        e.Workspaces = runner.DiscoverWorkspaces(e.State.RootPath)
        // ...existing reload logic
    }
```

Add `pnpm-workspace.yaml` to `IsConfigFile` in `filesystem/predicates.go`.

## Implementation Steps

1. **Create `runner/workspace.go`** with `DiscoverWorkspaces` and glob expansion logic.
2. **Add `LoadConfigForPath`** to `runner/config.go`.
3. **Update `PrepareJob`** to accept and use workspace context.
4. **Store workspaces on `Engine`** and pass through to `TriggerTest`.
5. **Add `pnpm-workspace.yaml`** to `IsConfigFile`.
6. **Add tests** for workspace discovery with npm, yarn, and pnpm formats.
7. **Verify** in a real monorepo that each package uses its own detected runner.
