# Spec Plan: Environment & Config File Change Handling

## Problem Statement

The filesystem watcher allowlist (`filesystem/predicates.go:IsConfigFile`) monitors project configuration files such as:
- `package.json`
- `tsconfig.json`
- `jest.config.js` / `jest.config.ts`
- `vite.config.js`
- `babel.config.js`
- `webpack.config.js`

However, when a `WatcherMsg` arrives for one of these config files, `engine/engine.go` passes the path to `e.Graph.Update(path)`. In `analysis/graph.go`, `Update` immediately exits because `filesystem.IsSourceFile(path)` returns `false` for config files:

```go
func (g *Graph) Update(path string) {
    if !filesystem.IsSourceFile(filepath.Base(path)) {
        return
    }
    // ...
}
```

As a result, modifying a Jest configuration file or updating environment settings triggers a file event, but **no tests are re-run and the test environment is not refreshed**.

## Goals

1. Detect project configuration file modifications in `engine/engine.go`.
2. Automatically invalidate runner configurations and rebuild the dependency graph when config files change.
3. Queue all active watched (or smart mode affected) tests for re-execution when testing configuration changes.

## Proposed Changes

### 1. Update `Engine.Update(msg WatcherMsg)` (`engine/engine.go`)

Check if the modified file is a config file using `filesystem.IsConfigFile(path)`:

```go
case WatcherMsg:
    path := string(msg)

    if filesystem.IsConfigFile(path) {
        // 1. Reload runner configuration
        e.ProjectConfig = runner.LoadConfig(e.State.RootPath)

        // 2. Rebuild graph as paths/aliases might have changed (e.g. tsconfig.json)
        e.Graph = analysis.NewGraph()
        _ = e.Graph.Build(e.State.RootPath)

        // 3. Queue all watched tests for re-execution
        for watchedPath := range e.State.Watched {
            if !contains(e.State.Queue, watchedPath) {
                e.State.Queue = append(e.State.Queue, watchedPath)
            }
        }
        
        m.State.CurrentOutput += fmt.Sprintf("\nConfig change detected (%s). Reloaded settings and re-queued tests.\n", filepath.Base(path))
    } else {
        // Normal source/test file handling...
        e.Graph.Update(path)
        // ...
    }
```

### 2. Live Configuration Hot-Reload (`runner/config.go`)

Ensure `runner.LoadConfig` clears cached config instances if any, allowing updated `.lazytest.json` or Jest settings to take immediate effect for subsequent test job runs.

## Verification Plan

1. **Unit Tests (`engine/engine_test.go`)**:
   - Watch two test files.
   - Modify `package.json` or `.lazytest.json`.
   - Send `WatcherMsg("package.json")`.
   - Verify both test files are enqueued for re-run.
   - Verify `ProjectConfig` is updated with new configuration values.
2. **Manual UX Verification**:
   - Run `lazytest`.
   - Edit `.lazytest.json` or `jest.config.js`.
   - Confirm status output indicates config reload and test suite re-execution.
