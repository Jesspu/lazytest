package runner

import (
	"strings"
	"testing"
	"time"
)

func TestRunner(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r := NewRunner()
		r.Run("echo", []string{"hello"}, ".")

		var output []string
		var status *StatusUpdate

		timeout := time.After(2 * time.Second)
		done := false

		for !done {
			select {
			case update := <-r.Updates:
				switch u := update.(type) {
				case OutputUpdate:
					output = append(output, string(u))
				case StatusUpdate:
					status = &u
					done = true
				}
			case <-timeout:
				t.Fatal("Timeout waiting for command completion")
			}
		}

		if status.Err != nil {
			t.Errorf("Expected nil error, got %v", status.Err)
		}

		if len(output) == 0 {
			t.Error("Expected output, got none")
		} else {
			got := strings.Join(output, "")
			if !strings.Contains(got, "hello") {
				t.Errorf("Expected output to contain 'hello', got %q", got)
			}
		}
	})

	t.Run("Failure", func(t *testing.T) {
		r := NewRunner()
		// Run a command that fails (exit 1)
		r.Run("sh", []string{"-c", "exit 1"}, ".")

		var status *StatusUpdate
		timeout := time.After(2 * time.Second)
		done := false

		for !done {
			select {
			case update := <-r.Updates:
				if s, ok := update.(StatusUpdate); ok {
					status = &s
					done = true
				}
			case <-timeout:
				t.Fatal("Timeout waiting for command completion")
			}
		}

		if status.Err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("Kill", func(t *testing.T) {
		r := NewRunner()
		// Run a long running command
		r.Run("sleep", []string{"2"}, ".")

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		r.Kill()

		var status *StatusUpdate
		timeout := time.After(2 * time.Second)
		done := false

		for !done {
			select {
			case update := <-r.Updates:
				if s, ok := update.(StatusUpdate); ok {
					status = &s
					done = true
				}
			case <-timeout:
				t.Fatal("Timeout waiting for command completion")
			}
		}

		if status.Err == nil {
			t.Error("Expected error from killed process, got nil")
		}
	})

	t.Run("Concurrent Run", func(t *testing.T) {
		r := NewRunner()
		// Start first command
		r.Run("sleep", []string{"2"}, ".")

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		// Start second command immediately
		r.Run("echo", []string{"second"}, ".")

		// We expect the first command to be cancelled (killed) and the second to finish successfully
		// However, since they share the Updates channel, we might see updates from both.
		// The key behavior we want to verify is that the second command runs.

		foundSecond := false
		timeout := time.After(3 * time.Second)

		// Read updates until we see "second" or timeout
		for {
			select {
			case update := <-r.Updates:
				if out, ok := update.(OutputUpdate); ok {
					if strings.Contains(string(out), "second") {
						foundSecond = true
						// We can stop once we verify the second command ran
						return
					}
				}
			case <-timeout:
				if !foundSecond {
					t.Fatal("Timeout waiting for second command output")
				}
				return
			}
		}
	})
}
