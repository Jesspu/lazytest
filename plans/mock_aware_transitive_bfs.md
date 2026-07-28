# Spec Plan: Mock-Aware Dependency Traversal in Graph BFS

## Problem Statement

The dependency graph (`analysis/graph.go`) tracks mock dependencies (`DepMocked`) per edge. However, the traversal method `GetDependents(path)` performs a standard Breadth-First Search (BFS) through `g.Reverse` **without examining `DependencyType` on intermediate edges**:

```go
func (g *Graph) GetDependents(path string) []string {
    // ...
    if deps, ok := g.Reverse[current]; ok {
        for dep := range deps {
            if !visited[dep] {
                visited[dep] = true
                dependents = append(dependents, dep)
                queue = append(queue, dep)
            }
        }
    }
    // ...
}
```

This creates two issues:
1. **Intermediate Mocks Are Ignored**: If `A.ts` -> `B.ts` -> `C.test.ts`, and `C.test.ts` mocks `B.ts` (`jest.mock('./B')`), editing `A.ts` will still trigger `C.test.ts` because `GetDependents("A.ts")` reaches `C.test.ts` via `B.ts`.
2. **False Positive Queuing**: `FindRelatedTests` only checks the direct dependency type between `dep` and `path`, failing to detect when an intermediate module in the chain was mocked.

## Goals

1. Update dependency graph traversal to respect mocked boundaries in dependency chains.
2. Stop BFS propagation when traversing through a `DepMocked` edge.
3. Eliminate false positive test runs caused by mocked intermediate modules.

## Proposed Changes

### 1. Implement `GetAffectedDependents` (`analysis/graph.go`)

Add a new method `GetAffectedDependents(path string)` that prunes BFS branches when `depType == DepMocked`:

```go
// GetAffectedDependents returns files transitively depending on path,
// stopping traversal along edges where the dependency is mocked.
func (g *Graph) GetAffectedDependents(path string) []string {
    g.mu.RLock()
    defer g.mu.RUnlock()

    visited := make(map[string]bool)
    var dependents []string

    queue := []string{path}
    visited[path] = true

    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]

        if deps, ok := g.Reverse[current]; ok {
            for dep, depType := range deps {
                if !visited[dep] {
                    visited[dep] = true
                    dependents = append(dependents, dep)
                    
                    // Only continue traversing upstream dependents if this edge is NOT mocked.
                    // If 'dep' mocks 'current', changes in 'current' (and its dependencies)
                    // do not affect 'dep' or anything above it.
                    if depType != DepMocked {
                        queue = append(queue, dep)
                    }
                }
            }
        }
    }

    return dependents
}
```

### 2. Update Engine Query (`engine/engine.go`)

Update `Engine.FindRelatedTests` and `Engine.Update(WatcherMsg)` to call `e.Graph.GetAffectedDependents(path)` instead of `GetDependents(path)`.

```go
func (e *Engine) FindRelatedTests(path string) []string {
    var tests []string
    seen := make(map[string]bool)

    if filesystem.IsTestFile(path) {
        tests = append(tests, path)
        seen[path] = true
    }

    dependents := e.Graph.GetAffectedDependents(path)
    for _, dep := range dependents {
        if !seen[dep] && filesystem.IsTestFile(dep) {
            tests = append(tests, dep)
            seen[dep] = true
        }
    }
    return tests
}
```

## Verification Plan

1. **Unit Tests (`analysis/analysis_test.go`)**:
   - Create test structure:
     - `leaf.ts`
     - `middle.ts` imports `leaf.ts`
     - `unmocked.test.ts` imports `middle.ts`
     - `mocked.test.ts` imports `middle.ts` and calls `jest.mock('./middle')`
   - Modify `leaf.ts`.
   - Call `g.GetAffectedDependents(leafPath)`.
   - Verify `unmocked.test.ts` IS included in returned dependents.
   - Verify `mocked.test.ts` IS NOT included in returned dependents.
2. **Regression Verification**:
   - Run existing `analysis_test.go` suite to ensure non-mocked reverse dependencies behave identically.
