# LazyTest Repository Analysis for Agents

## Project Overview
LazyTest is a Terminal User Interface (TUI) for running TypeScript and JavaScript tests (such as Jest and Vitest). It is written in Go and leverages the Charm ecosystem (`bubbletea`, `lipgloss`, `bubbles`).

## Architecture
The project follows a modular architecture separating presentation, engine coordination, test running, file watching, and code analysis:

- **`main.go`**: Application entry point. Initializes the engine (`engine.New`) and launches the Bubbletea program with `ui.Model`.
- **`engine/`**: Core coordinator layer separating business logic from UI rendering.
  - Manages application state (`state.go`), coordinates file watching events, tree updates, dependency graph generation, and test execution requests.
- **`ui/`**: Presentation layer implementing The Elm Architecture (Model-View-Update) via `bubbletea`.
  - Wraps the `Engine` and handles user interactions, keybindings, ANSI rendering, and view components (File Explorer, Output View, Help, Footer).
- **`analysis/`**: Static analysis engine for JavaScript and TypeScript projects.
  - `graph.go`: Builds forward and reverse dependency graphs with concurrent worker pools. Distinguishes regular vs. mocked imports.
  - `parser.go`: Parses import/require statements across JS, JSX, TS, and TSX files.
  - `config_resolver.go`: Resolves `tsconfig.json` path aliases (`compilerOptions.paths`).
- **`runner/`**: Test execution subsystem.
  - `runner.go`: Manages subprocess execution (`exec.Cmd`), output streaming via channels, context cancellation, and OS-specific process attributes.
  - `job.go`: Prepares test jobs (`TestJob`), matching path overrides and command templates.
  - `config.go`: Resolves workspace roots and project config (`.lazytest.json`).
- **`filesystem/`**: File system operations and events.
  - `walker.go`: Scans directory trees into `Node` hierarchies.
  - `watcher.go`: Monitors workspace file changes with debouncing using `fsnotify`.
  - `git.go`: Git repository detection and `.gitignore` aware status checks.
  - `stream.go`: Channel-based concurrent file walking.
  - `predicates.go`: Test file pattern detection.

## Key Components & Files

| Package | Key Files | Description |
| :--- | :--- | :--- |
| `main` | `main.go` | App startup, engine initialization, Bubbletea loop. |
| `engine` | `engine.go`, `state.go` | Central state management, side-effect commands, event orchestration. |
| `ui` | `model.go`, `explorer.go`, `keys.go`, `styles.go` | UI view rendering, keybindings, tree layout, Lipgloss styles. |
| `analysis` | `graph.go`, `parser.go`, `config_resolver.go` | TS/JS import parsing, tsconfig alias resolution, dependency graph. |
| `runner` | `runner.go`, `job.go`, `config.go`, `command_*.go` | Subprocess control, config & job resolution, platform command helpers. |
| `filesystem` | `walker.go`, `watcher.go`, `git.go`, `stream.go` | File tree walking, debounced fsnotify watching, Git ignore filtering. |

## Development & Testing Guidelines

### Build & Run
- **Build**: `go build -o lazytest .`
- **Run**: `./lazytest`
- **Test All Packages**: `go test ./...`

### Dependencies
- **Go**: 1.21+
- **TUI Libraries**: `github.com/charmbracelet/bubbletea`, `lipgloss`, `bubbles`
- **File Watching**: `github.com/fsnotify/fsnotify`
- **Runtime Environment**: `npx`, `jest` / `vitest` (or configured test runner in `.lazytest.json`)

## Conventions & Design Patterns
- **Decoupled Engine & UI**: UI state and view rendering are in `ui/`, but core state mutations and side effects originate in `engine/`.
- **Concurrency**: Background jobs (file analysis, process output streaming, fsnotify watching) produce messages sent into Bubbletea `Cmd` loops or engine update channels.
- **Cross-Platform Execution**: Platform-specific process attributes (e.g. process groups / cancellation) are abstracted in `command_unix.go` and `command_windows.go`.
- **Styling**: All UI color schemes and layout formatting are defined in `ui/styles.go` using Lipgloss.

