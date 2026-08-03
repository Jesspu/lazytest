# Plan: Fix Nested Loops and Redundant Sorting

## Issue Description
1. **Nested Loops in Watch Mode:** In `engine/engine.go` (`handleSourceChange`), resolving affected tests uses an $O(W \times D)$ nested loop, comparing every watched path against every affected dependent. In a monorepo where a core file has thousands of dependents, this nested loop runs on the main thread and can execute millions of iterations per save.
2. **Redundant Sorting on Render:** The UI calls `engine/suite.go` (`GetAffectedSuite`) on every view render (often 60fps). This function allocates a new slice from the `Affected` map and sorts it using a custom closure with multiple map lookups, causing constant CPU churn and GC pressure.

## Proposed Fix
1. **Optimize Dependent Lookups:**
   - The `Graph.GetDependents` method should return a `map[string]struct{}` instead of a `[]string`, providing $O(1)$ lookups.
   - Update `handleSourceChange` to check if a watched path exists in the dependents map directly, eliminating the nested loop.

2. **Cache Sorted Suite:**
   - Introduce a cached sorted slice in `engine.State` (e.g., `SortedAffectedSuite []string`).
   - Only recalculate and sort this slice when the underlying `Affected` map actually changes (e.g., when a test fails, passes, or is added/removed).
   - Update `GetAffectedSuite()` to simply return the cached slice, reducing its complexity to $O(1)$ and removing allocations during the UI render loop.

## Execution Steps
1. Refactor `analysis/graph.go` to return `map[string]struct{}` for dependents.
2. Update `engine/engine.go` to use $O(1)$ map lookups instead of the nested loop.
3. Add `SortedAffected []string` to `engine.State`.
4. Modify `engine/state.go` or `engine/suite.go` to manage the lifecycle of this cache, ensuring it updates only on state mutations.
5. Update `GetAffectedSuite()` to return the cached slice.
6. Profile the UI view loop to ensure `GetAffectedSuite` takes negligible time.
