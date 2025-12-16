package helpcount

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunner_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	// We need the repo root.
	root, err := filepath.Abs("../../../")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}

	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer runner.Close()

	opts := Options{
		Mode:           ModeCompare,
		ScenarioFilter: "", // Run all scenarios
	}

	if len(BuiltinScenarios()) == 0 {
		t.Skip("no builtin scenarios found")
	}

	report, err := runner.Run(opts)
	if err != nil {
		t.Logf("Runner.Run failed: %v", err)
	}
	t.Logf("Ran %d scenarios", len(report.Results))

	if len(report.Results) == 0 && err == nil {
		t.Log("No scenarios matched filter 'fake_exit0_usage'")
	}
}
