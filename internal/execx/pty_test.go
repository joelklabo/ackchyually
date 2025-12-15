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
