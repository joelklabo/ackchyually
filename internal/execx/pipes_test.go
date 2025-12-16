package execx

import (
	"strings"
	"testing"
)

func TestRunPipes_Success(t *testing.T) {
	// Simple echo command
	res, err := runPipes("echo", []string{"hello"})
	if err != nil {
		t.Fatalf("runPipes failed: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", res.ExitCode)
	}
	if !strings.Contains(res.StdoutTail, "hello") {
		t.Errorf("expected output to contain 'hello', got %q", res.StdoutTail)
	}
	if res.Mode != "pipes" {
		t.Errorf("expected mode 'pipes', got %q", res.Mode)
	}
}

func TestRunPipes_Failure(t *testing.T) {
	// Command that exits with 1
	// We use 'sh -c exit 1' to ensure portability (mostly)
	res, err := runPipes("sh", []string{"-c", "exit 1"})
	if err == nil {
		// exec.Command.Run() returns an error if the command exits non-zero
		// but our wrapper returns it alongside the result.
		// wait, runPipes returns the error from cmd.Run() directly.
		t.Error("expected error from non-zero exit, got nil")
	}
	if res.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", res.ExitCode)
	}
}
