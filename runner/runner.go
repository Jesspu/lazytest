package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// Runner manages the execution of test commands.
type Runner struct {
	mu          sync.Mutex
	runningCmds map[string]context.CancelFunc
	Updates     chan Update // Single channel for ordered updates
}

// Update is a marker interface for runner updates.
type Update interface{}

// OutputUpdate carries a line of output.
type OutputUpdate struct {
	FilePath string
	Content  string
}

// StatusUpdate carries the final result.
type StatusUpdate struct {
	FilePath string
	Err      error
}

// NewRunner creates a new Runner instance.
func NewRunner() *Runner {
	return &Runner{
		runningCmds: make(map[string]context.CancelFunc),
		Updates:     make(chan Update, 1024), // Buffered to prevent blocking
	}
}

func (r *Runner) Run(command string, args []string, cwd string, filePath string) {
	r.mu.Lock()
	// Kill existing process for this file if it's already running
	if cancel, exists := r.runningCmds[filePath]; exists {
		cancel()
	}
	
	// Create new context
	ctx, cancel := context.WithCancel(context.Background())
	r.runningCmds[filePath] = cancel

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = cwd

	prepareCommand(cmd)

	// Force color output
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "FORCE_COLOR=1", "CLICOLOR_FORCE=1")

	r.mu.Unlock()

	// Setup pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.Updates <- OutputUpdate{FilePath: filePath, Content: fmt.Sprintf("Error creating stdout pipe: %v", err)}
		r.Updates <- StatusUpdate{FilePath: filePath, Err: err}
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		r.Updates <- OutputUpdate{FilePath: filePath, Content: fmt.Sprintf("Error creating stderr pipe: %v", err)}
		r.Updates <- StatusUpdate{FilePath: filePath, Err: err}
		return
	}

	// Start command
	if err := cmd.Start(); err != nil {
		r.Updates <- OutputUpdate{FilePath: filePath, Content: fmt.Sprintf("Error starting command: %v", err)}
		r.Updates <- StatusUpdate{FilePath: filePath, Err: err}
		return
	}

	// Stream output in goroutines
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		streamReader(stdout, filePath, r.Updates)
	}()
	go func() {
		defer wg.Done()
		streamReader(stderr, filePath, r.Updates)
	}()

	// Wait for command to finish
	go func() {
		// Wait for output streaming to finish first
		wg.Wait()
		// Then wait for process to exit and close pipes
		err := cmd.Wait()

		r.mu.Lock()
		delete(r.runningCmds, filePath)
		r.mu.Unlock()

		r.Updates <- StatusUpdate{FilePath: filePath, Err: err}
	}()
}

func streamReader(r io.Reader, filePath string, out chan<- Update) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		out <- OutputUpdate{FilePath: filePath, Content: scanner.Text()}
	}
	if err := scanner.Err(); err != nil {
		out <- OutputUpdate{FilePath: filePath, Content: fmt.Sprintf("error reading output: %v", err)}
	}
}

// Kill explicitly stops a specific running command
func (r *Runner) Kill(filePath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if cancel, exists := r.runningCmds[filePath]; exists {
		cancel()
		delete(r.runningCmds, filePath)
	}
}

// KillAll explicitly stops all currently running commands.
func (r *Runner) KillAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for filePath, cancel := range r.runningCmds {
		cancel()
		delete(r.runningCmds, filePath)
	}
}

