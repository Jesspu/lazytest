# LazyTest

LazyTest is a terminal user interface (TUI) for running tests in TypeScript and JavaScript projects, heavily inspired by the excellent [lazygit](https://github.com/jesseduffield/lazygit). It provides a fast, keyboard-centric workflow for navigating test files, executing them, and viewing results instantly.

![LazyTest Screenshot](media/example.png)

## Features

*   **Vim-style Navigation**: Navigate your file tree with `j`, `k`, `h`, `l`.
*   **Instant Feedback**: Real-time output streaming with ANSI color support.
*   **Smart Test Selection**: Automatically watch and run tests related to changed files using dependency graph analysis.
*   **File Watching**: Automatically detects new test files. Manually toggle watch mode for specific files.
*   **Context Awareness**: Automatically finds the nearest `package.json` to run tests in the correct context (great for monorepos).
*   **Status Indicators**: Visual feedback for running (⏳), passed (✅), and failed (❌) tests.
*   **Watched Files Tab**: View and manage your list of watched files in a dedicated tab.
*   **Search**: Quickly find files with `/` and navigate matches with `n`/`N`.
*   **.gitignore Support**: Automatically respects `.gitignore` patterns and common ignore patterns.
*   **Customizable**: Configure custom test commands via `.lazytest.json`.

## Quick Start

### Prerequisites

*   [Go](https://go.dev/dl/) (1.21 or later recommended)
*   A JavaScript/TypeScript project with Jest installed.

### Installation

Clone the repository and build the binary:

```bash
git clone https://github.com/jesspatton/lazytest.git
cd lazytest
go build -o lazytest .
```

### Usage

Navigate to your project directory and run the binary:

```bash
./lazytest
```

(Optional) Move the binary to your PATH to run it from anywhere:

```bash
mv lazytest /usr/local/bin/
```

### Keybindings

| Key | Action |
| :--- | :--- |
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `Enter` | Run the selected test file |
| `Tab` | Switch between File Explorer and Output panes |
| `a` | Auto-watch tests related to changed files |
| `r` | Re-run the last executed test |
| `R` | Refresh the file tree |
| `/` | Enter Search Mode |
| `n` | Next Search Match |
| `N` | Previous Search Match |
| `Esc` | Exit Search Mode |
| `]` | Next Tab |
| `[` | Previous Tab |
| `w` | Toggle Watch Mode for selected file |
| `W` | Clear all Watched Files |
| `?` | Toggle Help Menu |
| `q` / `Ctrl+C` | Quit |

## Configuration & Test Runner Support

By default, LazyTest attempts to run tests using:
```bash
npx jest <relative_path_to_file> --colors
```

It automatically detects the execution root by searching up the directory tree for a `package.json` file.

### Custom Configuration

You can customize the behavior by creating a `.lazytest.json` file in your project root (where `package.json` is located).

**Supported Fields:**
*   `command`: The default command to run for tests. `<path>` is replaced by the test file path.
*   `overrides`: specific commands for file patterns (supports glob patterns and `/**` suffix).
*   `excludes`: directories to completely hide from the explorer.

**Example `.lazytest.json`:**

```json
{
  "command": "npm test --",
  "excludes": [
    "e2e",
    "examples/ignored_folder"
  ],
  "overrides": [
    {
      "pattern": "packages/backend/**/*.go",
      "command": "npx jest"
    },
    {
      "pattern": "packages/ui/**",
      "command": "npm run test:unit --"
    }
  ]
}
```

## Tech Stack & Architecture

LazyTest is built with Go and uses the [Charm](https://charm.sh/) ecosystem for its TUI components.

*   **Language**: Go (Golang)
*   **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea) (The Elm Architecture for Go)
*   **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss) (CSS-like styling)
*   **Components**: [Bubbles](https://github.com/charmbracelet/bubbles) (Viewport, etc.)
*   **File Watching**: [fsnotify](https://github.com/fsnotify/fsnotify)
*   **File Walking**: [gocodewalker](https://github.com/boyter/gocodewalker) (Fast directory traversal with ignore support)

### Project Structure

*   `main.go`: Entry point. Initializes the Bubbletea program.
*   `ui/`: Contains the TUI logic.
    *   `model.go`: The core application state and update loop.
    *   `explorer.go`: File explorer view logic.
    *   `footer.go`: Status bar/footer view logic.
    *   `help.go`: Help menu view logic.
    *   `keys.go`: Keybinding definitions.
    *   `styles.go`: Lipgloss style definitions.
    *   `utils.go`: Helper functions.
*   `engine/`: Business logic and state management.
    *   `engine.go`: Core engine that coordinates test execution and file watching.
    *   `state.go`: Application state management.
*   `runner/`: Handles test execution.
    *   `runner.go`: Manages `exec.Cmd`, process cancellation, and output streaming.
    *   `config.go`: Handles `package.json` discovery and `.lazytest.json` parsing.
    *   `job.go`: Encapsulates test job preparation logic.
*   `analysis/`: Dependency graph analysis.
    *   `graph.go`: Builds and maintains the dependency graph for smart test selection.
    *   `parser.go`: Parses import statements from source files.
*   `filesystem/`: File system operations.
    *   `walker.go`: Directory walking using gocodewalker to build the test tree.
    *   `watcher.go`: `fsnotify` implementation for detecting file changes.
    *   `ignore.go`: Handles `.gitignore` parsing and ignore pattern matching.
