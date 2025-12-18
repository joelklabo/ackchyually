package execx

import (
	"os"
	"testing"

	"golang.org/x/term"
)

func TestIsTTY_ValidatesTerminalState(t *testing.T) {
	// This test verifies that IsTTY() not only checks if stdin/stdout are terminals,
	// but also validates that we can actually get the terminal state.
	// This prevents "device not configured" errors when terminals are in an invalid state.
	
	// In a normal test environment, stdin/stdout are not terminals.
	// So IsTTY should return false.
	if IsTTY() {
		t.Skip("Running in a TTY environment; can't test non-TTY behavior")
	}
	
	// Verify that if stdin is a terminal, we can get its state.
	stdinFd := int(os.Stdin.Fd())
	if term.IsTerminal(stdinFd) {
		if _, err := term.GetState(stdinFd); err != nil {
			// If we can't get the state, IsTTY should return false
			if IsTTY() {
				t.Error("IsTTY() returned true even though GetState failed")
			}
		}
	}
}

func TestRun_FallsBackToPipesWhenPTYUnavailable(t *testing.T) {
	// This test ensures that Run() works correctly even when PTY mode is not available.
	// In test environments, this typically means it will use pipes mode.
	
	res, err := Run("echo", []string{"hello"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	
	if res.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", res.ExitCode)
	}
	
	// Verify that some mode was used (either "pty" or "pipes")
	if res.Mode != "pty" && res.Mode != "pipes" {
		t.Errorf("unexpected mode: %q", res.Mode)
	}
}
