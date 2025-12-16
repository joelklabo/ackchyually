//go:build !windows

package execx

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"testing"
)

func TestExitCode(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Fatalf("exitCode(nil)=%d, want 0", got)
	}
	if got := exitCode(errors.New("boom")); got != 1 {
		t.Fatalf("exitCode(non-exit-error)=%d, want 1", got)
	}

	err := runShellExitStatus(t, 7)
	if got := exitCode(err); got != 7 {
		t.Fatalf("exitCode(exit 7)=%d, want 7 (err=%v)", got, err)
	}

	err = runShellKilledBySignal(t, syscall.SIGKILL)
	if got, want := exitCode(err), 128+int(syscall.SIGKILL); got != want {
		t.Fatalf("exitCode(SIGKILL)=%d, want %d (err=%v)", got, want, err)
	}
}

func runShellExitStatus(t *testing.T, code int) error {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "exit "+itoa(code)) //nolint:gosec
	return cmd.Run()
}

func runShellKilledBySignal(t *testing.T, sig syscall.Signal) error {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "sh", "-c", "sleep 10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := cmd.Process.Signal(sig); err != nil {
		t.Fatalf("signal: %v", err)
	}
	return cmd.Wait()
}
