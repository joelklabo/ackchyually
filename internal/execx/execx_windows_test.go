//go:build windows

package execx

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestExitCode(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Fatalf("exitCode(nil)=%d, want 0", got)
	}
	if got := exitCode(errors.New("boom")); got != 1 {
		t.Fatalf("exitCode(non-exit-error)=%d, want 1", got)
	}

	// Test specific exit code (e.g. 7)
	// We can't rely on 'sh' being present on Windows.
	// We can try 'cmd /c exit 7'.
	err := runCmdExitStatus(t, 7)
	if got := exitCode(err); got != 7 {
		t.Fatalf("exitCode(exit 7)=%d, want 7 (err=%v)", got, err)
	}
}

func runCmdExitStatus(t *testing.T, code int) error {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "cmd", "/c", "exit", itoa(code))
	return cmd.Run()
}
