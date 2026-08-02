# Plan: Fix Full Workspace Directory Walk

## Issue Description
Currently, every time a source file changes, `engine/engine.go` (`handleSourceChange`) triggers a `tea.Batch(e.RefreshTree, ...)` command. `RefreshTree` performs a complete disk crawl (`filesystem.Walk`) of the entire workspace root. Furthermore, the tree insertion algorithm in `filesystem/walker.go` (`addPathToTree`) linearly searches child arrays to find existing directories, creating an $O(N^2)$ bottleneck during tree construction. In a large monorepo (100k+ files), this exhausts disk I/O, creates immense GC pressure, and stalls the file watcher.

## Proposed Fix
1. **Filesystem Layer (Tree Insertion):**
   - Modify the `Node` struct in `filesystem/walker.go` to include a map of children for $O(1)$ lookups during tree construction: `ChildrenMap map[string]*Node`.
   - Update `addPathToTree` to use this map instead of iterating over the `Children` slice. The slice can be generated once at the end or maintained concurrently for ordered rendering.

2. **Engine Layer (Incremental Updates):**
   - Stop calling `RefreshTree` (which does a full `filesystem.Walk`) on every file modification.
   - Implement incremental tree updates. When `fsnotify` reports a file change (Create, Remove, Rename), the engine should mutate the specific branch of the in-memory `Node` tree.
   - Only perform a full `RefreshTree` on initialization or major workspace changes.

## Execution Steps
1. Refactor `filesystem/walker.go` to use `ChildrenMap` for $O(1)$ node insertion.
2. Create incremental tree update functions: `AddNode(path string)`, `RemoveNode(path string)`.
3. Update `engine/engine.go` to handle `fsnotify` events by dispatching incremental updates instead of a full `RefreshTree`.
4. Verify that formatting a file or saving a file no longer triggers a full disk crawl.
