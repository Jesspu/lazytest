# Phase 2: Integrate `teatest` Framework Implementation Plan

## Goal
Establish the core testing utilities and integrate the `teatest` package to programmatically test the `lazytest` TUI. This phase bridges the gap between the `test_projects/` fixtures created in Phase 1 and the actual E2E assertions to be written in Phase 3.

## 1. Dependency Management
We need to add the `teatest` package from Charmbracelet's experimental repository.

**Tasks:**
- Run `go get github.com/charmbracelet/x/exp/teatest` to update `go.mod` and `go.sum`.

## 2. E2E Package Structure
To keep E2E tests separate from unit tests and avoid slowing down the standard `go test ./...` run (if desired, though `go test` caches), we'll create a dedicated directory.

**Structure:**
- `e2e/`
  - `helper.go`: Core utilities for bootstrapping `teatest` with `lazytest`.
  - `sanity_test.go`: A basic sanity test to verify the setup works before diving into complex assertions.

## 3. Implementation of E2E Helpers
We need a robust way to spin up the `lazytest` application pointing at a specific fixture directory from Phase 1. `teatest` operates on a Bubbletea `tea.Model`.

**`e2e/helper.go` Specification:**

- **`SetupTestEnv(t *testing.T, fixtureName string) (*teatest.Program, context.CancelFunc)`**:
  1. Construct the absolute path to `test_projects/<fixtureName>`.
  2. Initialize the `runner.Config` pointing to this directory as the workspace root.
  3. Create the `engine.New(config)` instance.
  4. Create the `ui.NewModel(engine)` instance.
  5. Wrap the model in `teatest`: `tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(100, 30))`.
  6. Return the test model and a `context.CancelFunc` to cleanly shut down the engine's background workers (like file watchers).

- **Handling Background Processes (Crucial):**
  `lazytest`'s engine uses goroutines and channels to watch files and stream test output. It is critical that the teardown phase (usually deferred in tests) cleanly terminates these goroutines to prevent flaky tests or resource exhaustion.

- **Custom Assertions (Optional but recommended):**
  - `WaitForText(t *testing.T, out io.Reader, expected string, timeout time.Duration)`: Since `lazytest` updates asynchronously (e.g., waiting for test results to parse), we need a helper to block until the TUI renders specific text (like "✓ passing") or fails via timeout.

## 4. Verification
To ensure Phase 2 is successful:
1. Write a basic test in `e2e/sanity_test.go`.
2. The test should use `SetupTestEnv(t, "single_repo_jest")`.
3. Simulate pressing 'q' to quit: `tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})`.
4. Wait for the program to exit and assert no errors occurred.
5. Run `go test ./e2e/...` to verify the environment handles initialization and teardown properly.
