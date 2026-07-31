# Phase 1: Test Projects Infrastructure Implementation Plan

## Goal
Implement the foundational test fixtures required for E2E testing of `lazytest`. This involves creating a robust set of test projects representing the various structures and test runners that `lazytest` supports.

## 1. Directory Structure Setup
Create a top-level directory `test_projects/` in the repository root. This folder will be tracked by git, allowing CI and other developers to run the E2E tests immediately without complex setup steps.

```bash
mkdir -p test_projects/{single_repo_jest,single_repo_vitest,single_repo_mocha,monorepo_pnpm}
```

## 2. Project Implementations

### A. `single_repo_jest`
A standard JavaScript project using Jest.

**Structure:**
- `package.json`: Contains `"test": "jest"` and `"jest": "^29.0.0"` in `devDependencies`.
- `src/math.js`: Simple math functions (e.g., `add`, `subtract`).
- `src/math.test.js`: Jest tests importing from `math.js`.
- `src/utils.js`: Helper file without tests (useful for validating test vs. source file detection).

### B. `single_repo_vitest`
A modern TypeScript project using Vitest.

**Structure:**
- `package.json`: Contains `"test": "vitest"` and `"vitest": "^1.0.0"` in `devDependencies`.
- `vitest.config.ts`: Basic vitest configuration.
- `src/stringUtils.ts`: String manipulation functions.
- `src/stringUtils.test.ts`: Vitest assertions.

### C. `single_repo_mocha`
A legacy or standard JavaScript project using Mocha (and optionally Chai).

**Structure:**
- `package.json`: Contains `"test": "mocha"` and `"mocha": "^10.0.0"` in `devDependencies`.
- `src/api.js`: Dummy API helper.
- `test/api.test.js`: Mocha tests using `describe` and `it`.

### D. `monorepo_pnpm`
A complex monorepo using pnpm workspaces to test `lazytest`'s ability to handle multiple packages and mixed environments.

**Structure:**
- `pnpm-workspace.yaml`: Defining `packages/*`.
- `package.json`: Root package.
- `packages/pkg-a`: 
  - Uses Vitest.
  - `src/index.ts` and `src/index.test.ts`.
- `packages/pkg-b`:
  - Uses Jest.
  - `src/index.js` and `src/index.test.js`.

## 3. Dependency Management
Since we don't want developers to manually run `npm install` in 5 different directories to run E2E tests, we will provide a utility script to bootstrap the test environments.

**Tasks:**
- Create `scripts/setup_test_projects.sh` which iterates through the directories and runs `npm install` (or `pnpm install` where appropriate).
- *Consideration:* To keep the main repo clean and small, we might want to `git ignore` the `node_modules` inside `test_projects/` but keep the `package.json` and lockfiles tracked.

**`.gitignore` updates:**
```gitignore
test_projects/**/node_modules/
test_projects/**/dist/
```

## 4. Verification
Before proceeding to Phase 2 (teatest integration), we must verify the fixtures:
1. Run `setup_test_projects.sh` locally.
2. Manually launch `lazytest` inside each of the test project directories (e.g., `cd test_projects/single_repo_jest && ../../lazytest`).
3. Verify that `lazytest` detects the runner, parses the tests, and executes them successfully without errors.
