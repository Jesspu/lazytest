package e2e

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/jesspatton/lazytest/engine"
	"github.com/jesspatton/lazytest/ui"
)

// SetupTestEnv spins up an isolated lazytest application instance pointing at
// a fixture directory under test_projects/<fixtureName>. It returns the teatest.TestModel
// and a context.CancelFunc to cleanly tear down background processes (e.g. file watchers).
func SetupTestEnv(t *testing.T, fixtureName string) (*teatest.TestModel, context.CancelFunc) {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("SetupTestEnv: unable to determine caller directory")
	}

	repoRoot := filepath.Dir(filepath.Dir(filename))
	fixturePath := filepath.Join(repoRoot, "test_projects", fixtureName)

	eng := engine.New(fixturePath)
	model := ui.NewModel(eng)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(100, 30))

	teardown := func() {
		tm.Send(tea.QuitMsg{})
		tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
		eng.Close()
	}

	return tm, teardown
}

// WaitForText blocks until the teatest output stream contains expected text or times out.
func WaitForText(t *testing.T, out io.Reader, expected string, timeout time.Duration) {
	t.Helper()
	teatest.WaitFor(
		t,
		out,
		func(bts []byte) bool {
			return strings.Contains(string(bts), expected)
		},
		teatest.WithDuration(resolveTimeout(timeout)),
	)
}

// WaitForTexts blocks until the teatest output stream contains ALL expected strings or times out.
func WaitForTexts(t *testing.T, out io.Reader, expected []string, timeout time.Duration) {
	t.Helper()
	teatest.WaitFor(
		t,
		out,
		func(bts []byte) bool {
			str := string(bts)
			for _, exp := range expected {
				if !strings.Contains(str, exp) {
					return false
				}
			}
			return true
		},
		teatest.WithDuration(resolveTimeout(timeout)),
	)
}

func resolveTimeout(defaultTimeout time.Duration) time.Duration {
	if env := os.Getenv("LAZYTEST_E2E_TIMEOUT"); env != "" {
		if parsed, err := time.ParseDuration(env); err == nil {
			return parsed
		}
	}
	return defaultTimeout
}
