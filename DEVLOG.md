# LazyTest Dev Log

This log tracks the implementation of major features and structural changes based on the project plans.

### 2026-07-30
- **E2E teatest Integration (Phase 2)**: Integrated Charmbracelet's `teatest` framework, added `Engine.Close()` and `Runner.KillAll()` for clean resource teardown, created `e2e/helper.go`, and verified setup with a sanity test.
- **E2E Test Projects Infrastructure (Phase 1)**: Created test project fixtures (`single_repo_jest`, `single_repo_vitest`, `single_repo_mocha`, `monorepo_pnpm`) and `scripts/setup_test_projects.sh` bootstrap script for E2E testing.


### 2026-07-29
- **Monorepo Workspace Awareness**: Implemented workspace discovery (`package.json`/`pnpm-workspace.yaml`), adding support for per-package configurations and test routing.
- **Welcome Banner**: Added a stylized welcome message displaying the detected runner and configuration settings before the first test run.
- **Test File Detection**: Enhanced test discovery with directory-based heuristics, new file extensions (`.mjs`, `.cjs`), and helper file exclusion.
- **Auto-Detect Test Runner**: Automated test runner detection by inspecting `package.json` dependencies, prioritizing Vitest, Jest, Mocha, and Playwright.
- **Config File Awareness**: Expanded watcher support to dynamically reload settings upon changes to various testing and tool configuration files.

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
