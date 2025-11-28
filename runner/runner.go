package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// Runner manages the execution of test commands.
type Runner struct {
	mu      sync.Mutex
	currCmd *exec.Cmd
	cancel  context.CancelFunc
	Updates chan Update // Single channel for ordered updates
}

// Update is a marker interface for runner updates.
type Update interface{}

// OutputUpdate carries a line of output.
type OutputUpdate string

// StatusUpdate carries the final result.
type StatusUpdate struct {
	Err error
}

// NewRunner creates a new Runner instance.
func NewRunner() *Runner {
	return &Runner{
		Updates: make(chan Update, 1024), // Buffered to prevent blocking
	}
}

// Run executes the test command. It kills any running command first.
func (r *Runner) Run(command string, args []string, cwd string) {
	r.mu.Lock()
	// Kill previous process if it exists
	if r.cancel != nil {
		r.cancel()
	}

	// Create new context
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = cwd
	// Set process group to ensure we can kill children if needed (though Context handles the main one)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Ensure we kill the whole process group when the context is cancelled
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	// Force color output
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "FORCE_COLOR=1", "CLICOLOR_FORCE=1")

	r.currCmd = cmd
	r.mu.Unlock()

	// Setup pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.Updates <- OutputUpdate(fmt.Sprintf("Error creating stdout pipe: %v", err))
		r.Updates <- StatusUpdate{Err: err}
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		r.Updates <- OutputUpdate(fmt.Sprintf("Error creating stderr pipe: %v", err))
		r.Updates <- StatusUpdate{Err: err}
		return
	}

	// Start command
	if err := cmd.Start(); err != nil {
		r.Updates <- OutputUpdate(fmt.Sprintf("Error starting command: %v", err))
		r.Updates <- StatusUpdate{Err: err}
		return
	}

	// Stream output in goroutines
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		streamReader(stdout, r.Updates)
	}()
	go func() {
		defer wg.Done()
		streamReader(stderr, r.Updates)
	}()

	// Wait for command to finish
	go func() {
		// Wait for process to exit first. This ensures pipes are closed.
		err := cmd.Wait()
		// Then wait for output streaming to finish
		wg.Wait()

		r.mu.Lock()
		// Only report status if this is still the current command
		shouldReport := false
		if r.currCmd == cmd {
			r.currCmd = nil
			r.cancel = nil
			shouldReport = true
		}
		r.mu.Unlock()

		if shouldReport {
			r.Updates <- StatusUpdate{Err: err}
		}
	}()
}

func streamReader(r io.Reader, out chan<- Update) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		out <- OutputUpdate(scanner.Text())
	}
}

// Kill explicitly stops the current command
func (r *Runner) Kill() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
	}
}
