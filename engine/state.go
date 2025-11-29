package engine

import (
	"github.com/jesspatton/lazytest/filesystem"
)

// TestStatus represents the current state of a test file.
type TestStatus int

const (
	// StatusIdle indicates the test is not running.
	StatusIdle TestStatus = iota
	// StatusRunning indicates the test is currently executing.
	StatusRunning
	// StatusPass indicates the last run passed.
	StatusPass
	// StatusFail indicates the last run failed.
	StatusFail
)

// State represents the core business state of the application.
type State struct {
	// Data
	Tree    *filesystem.Node
	Watched []string

	// Test Execution State
	Queue       []string
	NodeStatus  map[string]TestStatus
	TestOutputs map[string]string

	// Live State
	RunningNode   *filesystem.Node
	LastRunNode   *filesystem.Node
	CurrentOutput string
	RootPath      string
}

// NewState creates a new State instance.
func NewState(rootPath string) State {
	return State{
		RootPath:    rootPath,
		NodeStatus:  make(map[string]TestStatus),
		TestOutputs: make(map[string]string),
		Watched:     make([]string, 0),
		Queue:       make([]string, 0),
	}
}
