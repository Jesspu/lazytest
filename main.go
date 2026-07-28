package main

import (
	"fmt"
	"os"
	"path/filepath"

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

	if len(os.Args) > 1 {
		argPath := os.Args[1]
		absPath, err := filepath.Abs(argPath)
		if err != nil {
			fmt.Printf("Invalid directory path: %v\n", err)
			os.Exit(1)
		}
		targetDir = absPath
	}

	eng := engine.New(targetDir)
	p := tea.NewProgram(ui.NewModel(eng), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

