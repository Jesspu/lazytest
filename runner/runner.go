package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
)

type Runner struct {
	mu      sync.Mutex
	currCmd *exec.Cmd
	cancel  context.CancelFunc
	Output  chan string // Channel to stream output lines
	Status  chan error  // Channel to report completion/error
}

func NewRunner() *Runner {
	return &Runner{
		Output: make(chan string, 100), // Buffered to prevent blocking
		Status: make(chan error, 1),
	}
}

// Run executes the test command. It kills any running command first.
func (r *Runner) Run(command string, args []string, cwd string) {
	r.mu.Lock()
	// Kill previous process if it exists
	if r.cancel != nil {
		r.cancel()
		// We could wait for it to exit, but for now let's rely on the context cancellation
		// and the fact that we are starting a new one.
		// Ideally we should wait for the previous Wait() to return to ensure cleanup.
	}
	
	// Create new context
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = cwd
	// Set process group to ensure we can kill children if needed (though Context handles the main one)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	
	r.currCmd = cmd
	r.mu.Unlock()

	// Setup pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.Output <- fmt.Sprintf("Error creating stdout pipe: %v", err)
		r.Status <- err
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		r.Output <- fmt.Sprintf("Error creating stderr pipe: %v", err)
		r.Status <- err
		return
	}

	// Start command
	if err := cmd.Start(); err != nil {
		r.Output <- fmt.Sprintf("Error starting command: %v", err)
		r.Status <- err
		return
	}

	// Stream output in goroutines
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		streamReader(stdout, r.Output)
	}()
	go func() {
		defer wg.Done()
		streamReader(stderr, r.Output)
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
			r.Status <- err
		}
	}()
}

func streamReader(r io.Reader, out chan<- string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		out <- scanner.Text()
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
