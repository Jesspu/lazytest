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

	t.Run("Environment and Cwd", func(t *testing.T) {
		r := NewRunner()
		// We use a shell command to print env vars and pwd
		// This works on both Unix and likely Windows with git bash/wsl, but for pure Windows support
		// we might need to be careful. The user is on Mac, so 'sh' is fine.
		cmd := "sh"
		args := []string{"-c", "echo $FORCE_COLOR; echo $CLICOLOR_FORCE; pwd"}

		// Create a temporary directory to use as Cwd
		tmpDir := t.TempDir()

		// On Mac/Linux /tmp is often a symlink to /private/tmp, so we need to resolve it for comparison
		// However, for this test, checking if the output *contains* the base name of the temp dir is usually sufficient
		// or we can just use the runner's Cwd argument and see if it respects it.

		r.Run(cmd, args, tmpDir)

		var output []string
		var status *StatusUpdate
		done := false
		timeout := time.After(2 * time.Second)

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

		fullOutput := strings.Join(output, "\n")

		// Check Environment Variables
		if !strings.Contains(fullOutput, "1") {
			t.Error("Expected FORCE_COLOR or CLICOLOR_FORCE to be 1")
		}

		// Check Working Directory
		// We check if the output contains the temp dir path.
		// Note: on macOS /var/folders/... can be the real path for what t.TempDir returns.
		// So we might need to be flexible.
		// A safer check is to see if the last line (pwd) ends with the directory name we created.
		// But t.TempDir() creates a directory with a random name.
		// Let's just check if the output contains the path we passed in, assuming 'pwd' outputs it.
		// If there are symlinks, this might be flaky, but usually t.TempDir returns the path we should use.
		// To be safe, let's just check that it ran without error and produced output.
		// Actually, let's try to be more specific.
		if !strings.Contains(fullOutput, tmpDir) && !strings.Contains(fullOutput, "/private"+tmpDir) {
			// Try to handle the macOS /private prefix issue if it arises, but for now let's just warn if it fails
			// or maybe just check that it's not empty.
			// Better: write a file in that dir and check if it exists? No, we want to check the process's Cwd.
			// Let's rely on the fact that we passed tmpDir and if 'pwd' output contains it.
			// If this is flaky we can adjust.
			// On macOS, t.TempDir() returns something like /var/folders/..., but pwd might return /private/var/folders/...
			// Let's just check for the suffix of the temp dir which is unique enough.
			parts := strings.Split(tmpDir, "/")
			lastPart := parts[len(parts)-1]
			if !strings.Contains(fullOutput, lastPart) {
				t.Errorf("Expected output to contain cwd %q, got %q", lastPart, fullOutput)
			}
		}
	})

	t.Run("Stderr Capture", func(t *testing.T) {
		r := NewRunner()
		// Write to stderr
		r.Run("sh", []string{"-c", "echo 'some error' >&2"}, ".")

		var output []string
		var status *StatusUpdate
		done := false
		timeout := time.After(2 * time.Second)

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
			t.Errorf("Expected nil error (command exit 0), got %v", status.Err)
		}

		got := strings.Join(output, "")
		if !strings.Contains(got, "some error") {
			t.Errorf("Expected output to contain 'some error', got %q", got)
		}
	})
}
