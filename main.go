package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jesspatton/lazytest/engine"
	"github.com/jesspatton/lazytest/ui"
)

// main is the entry point of the application.
func main() {
	targetDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	var initialNotify string
	var positionalArgs []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--notify" && i+1 < len(os.Args) {
			initialNotify = os.Args[i+1]
			i++
		} else if !strings.HasPrefix(arg, "-") {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	if len(positionalArgs) > 0 {
		argPath := positionalArgs[0]
		absPath, err := filepath.Abs(argPath)
		if err != nil {
			fmt.Printf("Invalid directory path: %v\n", err)
			os.Exit(1)
		}
		targetDir = absPath
	}

	eng := engine.New(targetDir)
	if initialNotify != "" {
		eng.InitialNotification = initialNotify
	}
	p := tea.NewProgram(ui.NewModel(eng), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
