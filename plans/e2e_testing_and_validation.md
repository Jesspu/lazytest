# Epic: E2E Testing and Validation Framework

## Overview
As `lazytest` continues to grow, it is critical to establish a robust validation framework to ensure stability, prevent regressions, and verify that new features work correctly. This epic focuses on hardening the application by introducing end-to-end (E2E) testing using `teatest` and creating a diverse set of real-world test projects for validation.

## Goals
1. **TUI Validation:** Use `teatest` (from `charmbracelet/x`) to programmatically test the Bubbletea UI, simulating user input and asserting view states.
2. **Real-world Scenarios:** Create a `test_projects/` directory containing various project structures and test runners to validate `lazytest` against realistic environments.
3. **Automated Verification:** Build a test suite that runs against these projects to ensure the engine, file watching, analysis, and UI all coordinate correctly.

## Phase 1: Establish Test Projects Infrastructure
[View Detailed Implementation Plan for Phase 1](file:///Users/jesspatton/Documents/Projects/lazytest/plans/e2e_phase1_test_projects.md)

We will create a `test_projects/` folder in the root repository. This folder will not be git-ignored and will serve as the fixture source for our E2E tests. 

Planned test project templates:
- `single_repo_jest`: Standard JavaScript/TypeScript project utilizing Jest.
- `single_repo_vitest`: Standard project utilizing Vitest.
- `single_repo_mocha`: Standard project utilizing Mocha.
- `monorepo_pnpm`: A monorepo structure (e.g., pnpm workspaces) demonstrating multiple packages and potentially mixed runners.
- *Future*: `playwright_project`, `mixed_configs`, etc.

**Tasks:**
- Create the `test_projects/` directory structure.
- Populate each template with a minimal `package.json`, source files, test files, and configuration files.
- Ensure all test projects can run their respective test suites independently.

## Phase 2: Integrate `teatest` Framework
[View Detailed Implementation Plan for Phase 2](file:///Users/jesspatton/Documents/Projects/lazytest/plans/e2e_phase2_teatest_integration.md)

`teatest` will allow us to simulate keypresses and capture the visual output of the Bubbletea application.

**Tasks:**
- Add `github.com/charmbracelet/x/exp/teatest` as a dependency.
- Create an `e2e/` package or a suite of `*_test.go` files dedicated to E2E validation.
- Implement a helper function to bootstrap the `lazytest` engine and UI model against a specific `test_projects/` directory.

## Phase 3: Core E2E Test Suite
[View Detailed Implementation Plan for Phase 3](file:///Users/jesspatton/Documents/Projects/lazytest/plans/e2e_phase3_core_test_suite.md)

With the framework and fixtures in place, we will implement the core tests that validate the primary workflows of `lazytest`.

**Test Scenarios to Implement:**
1. **Initial Load & Runner Detection:**
   - Boot the TUI against each test project.
   - Assert that the correct test runner is detected and the file tree is accurately populated.
2. **Navigation & Execution:**
   - Simulate keypresses (`j`, `k`, `enter`) to navigate the file tree and trigger a test.
   - Assert that the UI updates to show the test status (running, passed, failed) and the output pane contains the expected text.
3. **Search Mode:**
   - Simulate entering search mode (`/`), typing a query, and asserting that the file tree is filtered correctly.
4. **Smart Mode & Graph Analysis:**
   - Enable Smart Mode via keypress.
   - Simulate a file system change (e.g., modifying a source file in a test project).
   - Assert that `lazytest` correctly identifies the affected tests and queues them for execution.
5. **Watch Mode:**
   - Toggle watch mode and modify a test file, ensuring the specific test is re-run.

## Phase 4: CI/CD Integration
[View Detailed Implementation Plan for Phase 4](file:///Users/jesspatton/Documents/Projects/lazytest/plans/e2e_phase4_ci_cd_integration.md)

To prevent regressions, the E2E test suite must be integrated into the continuous integration pipeline.

**Tasks:**
- Update GitHub Actions (or the relevant CI configuration) to execute the E2E test suite.
- Ensure the CI environment has the necessary runtimes (Node.js, pnpm, etc.) to execute the test projects.

## Open Questions & Future Considerations
- **Test execution time:** E2E tests executing real Node.js test runners might be slow. We may need to configure the CI pipeline to run these tests in parallel or only on main/PRs.
- **Teatest capabilities:** We'll need to evaluate how `teatest` handles background processes and channel-based state updates in `lazytest` to ensure we don't encounter race conditions during assertions.
