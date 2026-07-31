# Phase 4: CI/CD Integration Implementation Plan

## Goal
Automate the execution of the E2E test suite within the Continuous Integration (CI) pipeline (e.g., GitHub Actions). This ensures that every pull request and push to the main branch is validated against the `test_projects/` fixtures, preventing regressions before they merge.

## 1. CI Environment Requirements
Because the `e2e` suite runs actual Node.js test runners (Jest, Vitest, etc.) in the background, the CI environment requires a hybrid setup containing both Go and Node.js ecosystems.

**Tasks:**
- Update the existing test workflow (e.g., `.github/workflows/test.yml`) or create a dedicated `.github/workflows/e2e.yml`.
- Add steps to configure the environment:
  - `actions/setup-go`: To run `go test`.
  - `actions/setup-node`: To provide `node` and `npm`.
  - `pnpm/action-setup`: To provide `pnpm` specifically for the `monorepo_pnpm` fixture.

## 2. Dependency Setup & Caching
To avoid downloading massive `node_modules` on every single CI run, caching is critical.

**Tasks:**
- Implement caching for `npm` and `pnpm` store directories in the workflow.
- Add a step to execute `scripts/setup_test_projects.sh` (created in Phase 1). This script will traverse the `test_projects/` directory and install all required JS dependencies before the Go tests begin.

## 3. E2E Test Execution
Execute the specific `e2e` package tests separately from standard unit tests if needed.

**Tasks:**
- Add the execution step: `go test -v ./e2e/...`
- **Parallelism Constraints:** Since tests like `smart_mode_test.go` and `watch_mode_test.go` actually mutate the file system (simulating file saves) inside the `test_projects/` directory, running these specific tests in parallel (`t.Parallel()`) could cause race conditions if they operate on the same fixture simultaneously. We must either:
  1. Run the E2E suite sequentially.
  2. Implement a fixture-cloning strategy in `SetupTestEnv` where each test gets a unique copy of the `test_projects` directory in a temporary folder. (Recommended if the suite grows large).

## 4. Mitigating CI Resource Constraints
CI runners are generally much slower than local development machines. A 2-second UI timeout that never fails locally might flake constantly in CI.

**Tasks:**
- Introduce an environment variable (e.g., `LAZYTEST_E2E_TIMEOUT=15s`) that overrides the default timeout values used in the `WaitForText` helpers.
- Set this variable to a generously high value in the GitHub Actions workflow to prevent flaky test failures due to slow CPU or I/O on the CI runner.
