# Spec Plan: Test File Self-Inclusion in Related Test Determination

## Problem Statement

Currently, `Engine.FindRelatedTests(path)` in `engine/engine.go` queries the dependency graph for reverse dependents:

```go
func (e *Engine) FindRelatedTests(path string) []string {
    dependents := e.Graph.GetDependents(path)
    var tests []string
    for _, dep := range dependents {
        if filesystem.IsTestFile(dep) {
            depType := e.Graph.GetDependencyType(dep, path)
            if depType != analysis.DepMocked {
                tests = append(tests, dep)
            }
        }
    }
    return tests
}
```

Because `GetDependents(path)` returns files that *import* `path`, if `path` is itself a test file (e.g., `app.test.ts`), `GetDependents("app.test.ts")` will return empty (unless another test imports `app.test.ts`). As a result:
- The `a` (`AddRelated`) keybinding skips direct test files when changed files include tests.
- Queries for test file relations fail to return the test file itself.

## Goals

1. Ensure `FindRelatedTests(path)` returns `path` if `path` is already a valid test file.
2. Maintain existing transitive dependency lookup for non-test source files.
3. Eliminate false empty returns when adding related tests for modified test files.

## Proposed Changes

### 1. Update `FindRelatedTests` (`engine/engine.go`)

Modify `FindRelatedTests` to inspect `path` first before querying transitive dependents:

```go
func (e *Engine) FindRelatedTests(path string) []string {
    var tests []string
    seen := make(map[string]bool)

    // 1. Direct inclusion if path is a test file
    if filesystem.IsTestFile(path) {
        tests = append(tests, path)
        seen[path] = true
    }

    // 2. Query transitive dependents
    dependents := e.Graph.GetDependents(path)
    for _, dep := range dependents {
        if !seen[dep] && filesystem.IsTestFile(dep) {
            depType := e.Graph.GetDependencyType(dep, path)
            if depType != analysis.DepMocked {
                tests = append(tests, dep)
                seen[dep] = true
            }
        }
    }

    return tests
}
```

### 2. Update Graph Query Utilities

Ensure duplicate entries are deduplicated using a map (`seen`).

## Verification Plan

1. **Unit Tests (`engine/engine_test.go`)**:
   - Test `FindRelatedTests("foo.test.ts")` -> should return `["/path/to/foo.test.ts"]`.
   - Test `FindRelatedTests("foo.ts")` where `foo.test.ts` imports `foo.ts` -> should return `["/path/to/foo.test.ts"]`.
   - Test `FindRelatedTests("foo.test.ts")` where `bar.test.ts` also imports `foo.test.ts` -> should return both test files without duplicates.
2. **UI Verification**:
   - Press `a` keybinding with git-modified `.test.ts` files and verify they are automatically added to watched list.
