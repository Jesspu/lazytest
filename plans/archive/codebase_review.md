# Codebase Review & Refactoring Plan

**Date**: 2026-07-28  
**Scope**: Full sweep of all packages — duplication, over-complication, file structure, documentation accuracy.

---

## Summary of Metrics

| Package | Files | Lines (code) | Lines (test) | Largest File |
| :--- | ---: | ---: | ---: | :--- |
| `engine/` | 3 | 540 | 723 | `engine.go` (487) |
| `ui/` | 8 | 1,025 | 137 | `model.go` (627) |
| `analysis/` | 5 | 625 | 781 | `graph.go` (275) |
| `runner/` | 6 | 308 | 491 | `runner.go` (134) |
| `filesystem/` | 5 | 358 | 420 | `watcher.go` (143) |
| `main` | 1 | 39 | 0 | `main.go` |
| **Total** | **28** | **~2,895** | **~2,552** | |

---

## 1. Duplication

### 1.1 — `nodeFromPath()` inline construction (HIGH)

The pattern of constructing a `&filesystem.Node{Path: p, Name: basename(p)}` from a raw path string is copy-pasted **7 times** across `engine/engine.go` (lines 144, 181, 434, 449, 479) and `ui/model.go` (line 202). Each instance inlines the same `strings.LastIndex(path, string(os.PathSeparator))+1` expression for the basename.

**Fix**: Add a `filesystem.NodeFromPath(path string) *Node` constructor and replace all inline constructions.

---

### 1.2 — Tab list resolution (`smartMode ? GetAffectedSuite : GetWatchedFiles`) (HIGH)

This `if/else` branch to resolve the correct tab list is duplicated **5 times** across `model.go` (lines 183, 408, 475) and `explorer.go` (line 79). Each callsite builds the same `tabList` variable with the same `emptyMsg` logic.

**Fix**: Add a `(m *Model) getTabList() ([]string, string)` helper on Model that encapsulates the smart-mode vs. manual branching, returns `(list, emptyMessage)`.

---

### 1.3 — Status icon switch (`StatusRunning → ⏳, StatusPass → ✅, StatusFail → ❌`) (MEDIUM)

The status → emoji mapping appears **twice** in `explorer.go`: once in the watched/affected tab renderer (line 114–126) and again in `getNodeIcon()` (line 252–272). The same mapping also exists implicitly in the engine tests for verification.

**Fix**: Centralise in a single `StatusIcon(status TestStatus) string` function, either on the `engine` package (exported) or a shared `ui` helper.

---

### 1.4 — Queue dedup + trigger-if-idle (MEDIUM)

The queue deduplication + "pop-and-trigger if idle" pattern is written three separate ways:
1. In `WatcherMsg` handler (engine.go:123–148) — inline, builds `queuedSet`, pops, constructs node, calls `TriggerTest`.
2. In `StatusUpdate` handler (engine.go:177–186) — inline, pops, constructs node.
3. In `enqueueNodes()` (engine.go:460–486) — proper helper, but used only by `RunSuiteFailures` / `RunAffectedSuite`.

The `WatcherMsg` and `StatusUpdate` handlers should reuse `enqueueNodes()` (or a lighter variant) instead of duplicating the dedup and trigger logic.

---

### 1.5 — `NextTab` / `PrevTab` identical logic (LOW)

Both handlers in `model.go` (lines 138–155) have exactly the same body — they toggle between the two tabs. With only two tabs, "next" and "prev" are identical. This is fine semantically (they'll diverge if a third tab is added), but worth noting.

---

## 2. Over-complication

### 2.1 — `Graph.GetDependents()` is unused (MEDIUM)

`GetDependents()` (graph.go:158) performs a standard BFS **without** mock-pruning. It is not called anywhere in the codebase — only `GetAffectedDependents()` (the mock-aware version) is used. Dead code that could confuse contributors.

**Fix**: Remove `GetDependents()` or mark it explicitly as a public API if it's intended for future use.

---

### 2.2 — `GetDependencyType()` is unused (LOW)

`GetDependencyType()` (graph.go:239) is never called by any production or test code.

**Fix**: Remove, or add test coverage if it's intended for future use.

---

### 2.3 — `validate/` is an empty directory (LOW)

The `validate/` directory exists but contains no files. It may have been scaffolded for future work but contributes noise to the project structure.

**Fix**: Remove, or add a TODO/README if planned.

---

### 2.4 — Stale `coverage.out` and `debug.log` committed (LOW)

Both files are in `.gitignore` but `debug.log` may have been committed previously and `coverage.out` is tracked. These are build/runtime artifacts.

**Fix**: Ensure both are gitignored and remove any tracked versions with `git rm --cached`.

---

## 3. File Structure

### 3.1 — `engine/engine.go` has grown to 487 lines (HIGH)

This file now contains:
- Bubbletea message types (3 types)
- Engine struct + constructor
- `Init()` + `Update()` (the core event loop — 130 lines)
- Action methods (`TriggerTest`, `ReRunLast`, `ToggleWatch`, `ClearWatched`, `ToggleSmartMode`)
- Internal commands (watcher, tree refresh, graph build)
- Accessor methods (7 getters)
- Affected suite methods (`GetAffectedSuite`, `GetSuiteStats`, `ClearAffectedSuite`, `RunSuiteFailures`, `RunAffectedSuite`, `enqueueNodes` — 130 lines)
- `FindRelatedTests()`

These are ~5 distinct responsibility clusters in one file.

**Fix**: Split `engine/engine.go` into:
- `engine/engine.go` — Struct, constructor, `Init()`, `Update()` (core loop)
- `engine/actions.go` — `TriggerTest`, `ReRunLast`, `ToggleWatch`, `ClearWatched`, `ToggleSmartMode`, `FindRelatedTests`
- `engine/suite.go` — `GetAffectedSuite`, `GetSuiteStats`, `ClearAffectedSuite`, `RunSuiteFailures`, `RunAffectedSuite`, `enqueueNodes`
- `engine/accessors.go` — All getter methods (`GetWatchedFiles`, `GetTestOutput`, etc.)
- `engine/messages.go` — `WatcherMsg`, `TreeLoadedMsg`, `WatcherReadyMsg`

---

### 3.2 — `ui/model.go` has grown to 627 lines (HIGH)

The `Update()` method alone is ~340 lines of deeply nested `switch/case/if`. It interleaves:
- Global key handlers
- Per-pane key handlers (TabWatched, TabExplorer, search modes)
- Message handlers (WindowSize, TreeLoaded, Watcher, OutputUpdate, StatusUpdate)
- Helper methods (`wrapOutput`, `syncViewportOutput`, `applySmartModeBindings`, `renderSuiteBadge`)

**Fix**: Split `ui/model.go` into:
- `ui/model.go` — Struct, types, `NewModel()`, `Init()`, `View()`
- `ui/update.go` — `Update()` method + message handlers
- `ui/search.go` — All search-mode key handling (typing mode, navigation mode)
- `ui/badge.go` — `renderSuiteBadge`, `applySmartModeBindings`
- `ui/sync.go` — `syncViewportOutput`, `wrapOutput`

---

### 3.3 — `analysis/repro_test.go` purpose unclear (LOW)

This file appears to be a reproduction test (likely for a specific bug). If the bug is fixed, consider folding the test into `analysis_test.go` or adding a comment explaining what it reproduces.

---

## 4. Documentation Gaps

### 4.1 — `README.md` — Missing Smart Mode UX features (HIGH)

The README documents the basic Smart Mode toggle (`s`) and the `[SMART MODE]` footer badge, but is missing the entire Phase 1/Phase 2 Affected Suite feature set:

| Missing | Section |
| :--- | :--- |
| "Affected Suite" tab (replaces Watched tab in Smart Mode) | Features, Keybindings |
| Status-prioritized sorting (Fail → Running → Pass) | Features |
| Zero-Touch Auto-Focus on failure | Features |
| Suite Stats Badge (`⚡ SMART MODE \| N Passed • N Failed • N Running`) | Features |
| `f` — Run Failures | Keybindings |
| `W` — Clear Suite (in Smart Mode) | Keybindings |
| `a` — Run Suite (in Smart Mode) | Keybindings |
| Dynamic keybinding label swaps | Keybindings |

**Fix**: Update Features section and Keybindings table to document the new Smart Mode behaviour with notes that `W`, `a`, and `f` change meaning based on mode.

---

### 4.2 — `agents.md` — Stale architecture description (HIGH)

`agents.md` is missing:
- `engine/state.go` `Affected` map and suite management methods
- `engine/engine.go` suite accessors (`GetAffectedSuite`, `GetSuiteStats`, etc.)
- `ui/model.go` Smart Mode UX methods (`applySmartModeBindings`, `renderSuiteBadge`, `getTabList`)
- `ui/keys.go` `RunFailures` binding
- The `ui/footer.go` and `ui/help.go` files are not mentioned in the Key Components table
- The `plans/` directory is not mentioned
- `filesystem/predicates.go` mentions test file detection but not the `IsConfigFile()` predicate
- Git operations (`git.go` `GetChangedFiles()`) described as "Git repository detection and `.gitignore` aware status checks" — inaccurate; it runs `git status --porcelain` for diff-based test selection, not gitignore checks

**Fix**: Full update to `agents.md` reflecting current architecture, state shape, and package responsibilities.

---

### 4.3 — `README.md` — "Refresh" key description inaccurate (LOW)

The README says `R` will "Refresh file tree and clear test states", but `R` only calls `RefreshTree` — it does **not** clear test states (`NodeStatus`, `TestOutputs`).

---

## 5. Refactoring Plan (Priority Order)

### Phase A: Extract helpers to eliminate duplication (small, safe, testable)

| Task | Files | Impact |
| :--- | :--- | :--- |
| Add `filesystem.NodeFromPath(path)` constructor | `filesystem/walker.go`, `engine/engine.go`, `ui/model.go` | Eliminates 7 inline constructions |
| Add `(m *Model) getTabList() ([]string, string)` helper | `ui/model.go`, `ui/explorer.go` | Eliminates 5 duplicate branches |
| Centralise status-to-icon mapping | `ui/explorer.go` | Removes 1 duplicate switch |
| Refactor `WatcherMsg` / `StatusUpdate` to use `enqueueNodes()` | `engine/engine.go` | Unifies 3 dedup patterns |

### Phase B: File splits (medium, structural)

| Task | From → To |
| :--- | :--- |
| Split `engine/engine.go` (487 → ~5 files) | `engine.go`, `actions.go`, `suite.go`, `accessors.go`, `messages.go` |
| Split `ui/model.go` (627 → ~5 files) | `model.go`, `update.go`, `search.go`, `badge.go`, `sync.go` |

### Phase C: Dead code & cleanup

| Task |
| :--- |
| Remove `Graph.GetDependents()` (unused, untested) |
| Remove `Graph.GetDependencyType()` (unused) |
| Remove empty `validate/` directory |
| Remove `coverage.out` / `debug.log` from tracking if committed |
| Fold or annotate `analysis/repro_test.go` |

### Phase D: Documentation update

| Task |
| :--- |
| Update `README.md` keybindings table and Features section for Smart Mode UX |
| Full rewrite of `agents.md` to reflect current architecture |
| Fix `R`/Refresh description in README |
