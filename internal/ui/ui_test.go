package ui

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/creack/pty"
)

func TestShouldStyle_GatesOnTerminalAndEnv(t *testing.T) {
	if shouldStyle(nil) {
		t.Fatalf("expected shouldStyle(nil)=false")
	}

	f, err := os.CreateTemp(t.TempDir(), "out-*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer f.Close()

	if shouldStyle(f) {
		t.Fatalf("expected shouldStyle(non-tty)=false")
	}

	if runtime.GOOS == "windows" {
		t.Skip("no windows PTY support")
	}

	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open: %v", err)
	}
	defer ptmx.Close()
	defer tty.Close()

	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	if !shouldStyle(tty) {
		t.Fatalf("expected shouldStyle(tty)=true")
	}

	t.Setenv("NO_COLOR", "1")
	if shouldStyle(tty) {
		t.Fatalf("expected shouldStyle(tty)=false when NO_COLOR is set")
	}

	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if shouldStyle(tty) {
		t.Fatalf("expected shouldStyle(tty)=false when TERM=dumb")
	}
}

func TestUI_RenderMethodsContainInput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no windows PTY support")
	}

	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open: %v", err)
	}
	defer ptmx.Close()
	defer tty.Close()

	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	u := New(tty)
	if !u.enabled {
		t.Fatalf("expected UI to be enabled when output is a tty")
	}

	if out := u.Warn("warn"); !strings.Contains(out, "warn") {
		t.Fatalf("Warn output did not contain input: %q", out)
	}
	if out := u.Error("err"); !strings.Contains(out, "err") {
		t.Fatalf("Error output did not contain input: %q", out)
	}
	if out := u.Label("label"); !strings.Contains(out, "label") {
		t.Fatalf("Label output did not contain input: %q", out)
	}
}
