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
	Tree     *filesystem.Node
	Watched  map[string]struct{}
	Affected map[string]struct{}

	// Test Execution State
	Queue       []string
	NodeStatus  map[string]TestStatus
	TestOutputs map[string]string

	// Live State
	RunningNodes map[string]*filesystem.Node
	LastRunNode  *filesystem.Node
	RootPath     string

	// Mode
	SmartMode bool // If true, automatically queue all affected test files on file change
}

// NewState creates a new State instance.
func NewState(rootPath string) State {
	return State{
		RootPath:     rootPath,
		NodeStatus:   make(map[string]TestStatus),
		TestOutputs:  make(map[string]string),
		Watched:      make(map[string]struct{}),
		Affected:     make(map[string]struct{}),
		Queue:        make([]string, 0),
		RunningNodes: make(map[string]*filesystem.Node),
	}
}
