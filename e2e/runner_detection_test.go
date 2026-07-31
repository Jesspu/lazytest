package e2e

import (
	"testing"
	"time"
)

func TestRunnerDetection_Jest(t *testing.T) {
	tm, teardown := SetupTestEnv(t, "single_repo_jest")
	defer teardown()

	// Verify welcome message auto-detection string and file tree src node for Jest
	WaitForTexts(t, tm.Output(), []string{"Auto-detected: jest", "src"}, 5*time.Second)
}

func TestRunnerDetection_Vitest(t *testing.T) {
	tm, teardown := SetupTestEnv(t, "single_repo_vitest")
	defer teardown()

	// Verify welcome message auto-detection string and file tree src node for Vitest
	WaitForTexts(t, tm.Output(), []string{"Auto-detected: vitest", "src"}, 5*time.Second)
}

func TestRunnerDetection_Mocha(t *testing.T) {
	tm, teardown := SetupTestEnv(t, "single_repo_mocha")
	defer teardown()

	// Verify welcome message auto-detection string and file tree test node for Mocha
	WaitForTexts(t, tm.Output(), []string{"Auto-detected: mocha", "test"}, 5*time.Second)
}
