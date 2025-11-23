# LazyTest Repository Analysis for Agents

## Project Overview
LazyTest is a Terminal User Interface (TUI) for running TypeScript and JavaScript tests (specifically Jest). It is written in Go and leverages the Charm ecosystem (Bubbletea, Lipgloss, Bubbles).

## Architecture
The project follows a modular architecture:

- **`main.go`**: Entry point. Initializes the Bubbletea program.
- **`ui/`**: Contains the presentation layer and UI logic.
    - Uses The Elm Architecture (Model-View-Update) via Bubbletea.
    - Key components: File Explorer, Output View, Help Menu, Footer.
- **`runner/`**: Handles the business logic of executing tests.
    - Manages `exec.Cmd` processes.
    - Handles configuration (`.lazytest.json`) and `package.json` discovery.
- **`filesystem/`**: Handles file system interactions.
    - Recursive directory walking.
    - File watching using `fsnotify`.

## Key Files & Components

### UI (`ui/`)
- `model.go`: Central application state (`Model` struct) and the main `Update` loop.
- `explorer.go`: Logic for the file tree view.
- `keys.go`: Keybinding definitions using `bubbles/key`.
- `styles.go`: UI styling using `lipgloss`.

### Runner (`runner/`)
- `runner.go`: Executes test commands and streams output.
- `config.go`: Resolves project root and custom configurations.

### Filesystem (`filesystem/`)
- `watcher.go`: Watches for file changes to trigger updates.
- `walker.go`: Scans directories to build the test file tree.

## Development Guidelines

### Build & Run
- **Build**: `go build -o lazytest .`
- **Run**: `./lazytest`

### Dependencies
- **Go**: 1.21+
- **TUI Libs**: `github.com/charmbracelet/bubbletea`, `lipgloss`, `bubbles`.
- **System**: Requires `npx` and `jest` (or configured test runner) in the environment for actual test execution.

## Conventions
- **State Management**: All UI state changes happen in `ui/model.go`'s `Update` function.
- **Styling**: Styles are defined in `ui/styles.go` to keep `View` methods clean.
- **Error Handling**: Errors are generally passed through the Bubbletea `Cmd` system or logged to `debug.log` if critical.
