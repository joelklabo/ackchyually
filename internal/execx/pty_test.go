package execx

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/creack/pty"
)

func TestRun_UsesPTYWhenTTY(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no windows PTY support")
	}

	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open: %v", err)
	}

	oldIn := os.Stdin
	oldOut := os.Stdout
	oldErr := os.Stderr
	os.Stdin = tty
	os.Stdout = tty
	os.Stderr = tty

	t.Cleanup(func() {
		os.Stdin = oldIn
		os.Stdout = oldOut
		os.Stderr = oldErr
		_ = tty.Close()
		_ = ptmx.Close()
	})

	res, err := Run("sh", []string{"-c", "echo hi"})
	if err != nil {
		t.Fatalf("Run: %v (res=%#v)", err, res)
	}
	if res.Mode != "pty" {
		t.Fatalf("expected res.Mode=pty, got %q", res.Mode)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected ExitCode=0, got %d", res.ExitCode)
	}
	if !strings.Contains(res.CombinedTail, "hi") {
		t.Fatalf("expected CombinedTail to contain output, got %q", res.CombinedTail)
	}
}

func TestRunPTY_StartError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no windows PTY support")
	}
	// Missing executable should cause pty.Start to fail
	res, err := runPTY("missingtool_xyz", []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if res.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", res.ExitCode)
	}
}

func TestRun_NoTTY(t *testing.T) {
	// Ensure standard fds are NOT terminals (default in go test usually, but let's be safe)
	// We can't easily force them to be non-terminal if they are, but usually they are pipes.
	// We'll just call Run and expect it to work (via pipes).
	res, err := Run("echo", []string{"hi"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Mode == "pty" {
		// If it runs as PTY, then we are in a TTY environment.
		// That's fine, but we wanted to test the non-PTY path of Run.
		// To force non-PTY, we can pipe stdin?
		t.Skip("Running in TTY environment, skipping non-TTY test")
	}
	if res.Mode != "pipes" {
		t.Errorf("expected Mode=pipes, got %q", res.Mode)
	}
	if !strings.Contains(res.StdoutTail, "hi") {
		t.Errorf("output missing hi: %q", res.StdoutTail)
	}
}
