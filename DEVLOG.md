# LazyTest Dev Log

This log tracks the implementation of major features and structural changes based on the project plans.

### 2026-07-28
- **Codebase Review & Refactoring**: Split `engine/engine.go` and `ui/model.go` into multiple focused files, eliminating duplication and extracting sync, search, and badge concerns.
- **Smart Mode UX / Phase 1 & 2**: Overhauled the Smart Mode UI to include the Affected Suite tab, live suite stats badge, zero-touch auto-focus on failures, and dynamic keybindings.
- **Cursor Output Sync**: Centralized output rendering logic, ensuring the viewport accurately reflects the currently selected file or active test.

### 2026-07-27
- **Auto-Run Affected Tests**: Introduced the persistent Smart Mode feature, auto-queueing transitively affected tests on any file system change.
- **Config Change Handling**: Added live-reload capabilities for `.lazytest.json`, dynamically reapplying configuration changes without restart.
- **Mock-Aware Transitive BFS**: Implemented forward and reverse dependency graph analysis, explicitly identifying mocked paths to ensure accurate affected-test calculation.
- **Parser Enhancements**: Upgraded the AST parser to support dynamic imports, re-exports, and TS path aliases via `tsconfig.json` resolution.
- **Test File Self-Inclusion**: Fixed an engine bug to correctly include modified test files in the related-test queue.
- **Watcher Debounce Batching**: Upgraded `fsnotify` processing to emit a batched slice of all files changed within the debounce window.

### 2025-11-29
- **Parallel Execution (Partial/Planned)**: Started work on optimizing watch operations and implementing a smart test execution queue.
