# LazyTest Dev Log

This log tracks the implementation of major features and structural changes based on the project plans.

### 2026-07-29
- **Test File Detection**: Extended `IsTestFile` to cover `.test.mjs`, `.spec.mjs`, `.test.cjs`, and `.spec.cjs` suffixes. Extended `IsSourceFile` to treat `.mjs` and `.cjs` as source files. Added `IsTestFileByPath` which augments suffix-based detection with directory-based heuristics — files inside `test/`, `tests/`, and `__tests__` directories are now recognised as tests even without a `.test.` or `.spec.` suffix (Mocha/Node built-in conventions). A companion `isTestHelper` guard uses substring matching on the lowercased stem to exclude setup, helper, fixture, mock, and utility files (including compound names like `setupTest.ts` and `testHelper.tsx`). All call sites in `walker.go`, `engine/actions.go`, and `engine/engine.go` updated to use the new function.
- **Auto-Detect Test Runner**: Added `DetectRunner` to `runner/config.go`, which reads `package.json` and identifies the project's test runner by inspecting `devDependencies` then `dependencies` in priority order (vitest → jest → mocha → playwright → node built-in). `LoadConfig` now calls this instead of hard-coding `npx jest`, and exposes the detected runner name via a new `Config.DetectedRunner` field for UI reporting. Explicit `.lazytest.json` configs continue to take precedence.
- **Config File Awareness**: Refactored `IsConfigFile` in `filesystem/predicates.go` to a data-driven slice approach, extending recognition to `vitest.config.*`, `vitest.workspace.*`, `.mocharc.*`, `playwright.config.*`, `pnpm-workspace.yaml`, and `.babelrc`. These files now correctly trigger config reloads and test re-queues when changed.

### 2026-07-28
- **Parallel Execution**: Refactored the engine and runner to support executing multiple tests concurrently based on a configurable limit, replacing the global output state with per-path stream multiplexing.
- **Mouse Support**: Implemented comprehensive mouse support for pane selection, tab switching, double-click test execution, and native scrolling, utilizing proper layout margin coordinate translation.
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
