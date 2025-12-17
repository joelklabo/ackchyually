//go:build !windows

package integration

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestAgentCLI_SpawnViaPATH_LogsSuccessForBest(t *testing.T) {
	root := repoRoot(t)

	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	shimDir := filepath.Join(home, ".local", "share", "ackchyually", "shims")
	realDir := filepath.Join(tmp, "real")
	binDir := filepath.Join(tmp, "bin")

	mkdirAll(t, shimDir)
	mkdirAll(t, realDir)
	mkdirAll(t, binDir)

	ack := filepath.Join(binDir, "ackchyually")
	build(t, root, "./cmd/ackchyually", ack)

	realTool := filepath.Join(realDir, "promptly")
	build(t, root, "./internal/testtools/promptly", realTool)

	// Simulate an agent CLI: it runs "tool" via PATH under a PTY. We want the shim
	// to be found, the real tool to be run, and a success to be recorded.
	must(t, os.Symlink(ack, filepath.Join(shimDir, "promptly")))

	origPath := os.Getenv("PATH")
	t.Setenv("HOME", home)
	t.Setenv("PATH", strings.Join([]string{shimDir, realDir, origPath}, string(os.PathListSeparator)))

	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open: %v", err)
	}
	defer ptmx.Close()
	defer tty.Close()

	must(t, pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80}))

	cmd := exec.CommandContext(context.Background(), "promptly")
	cmd.Dir = root
	cmd.Env = os.Environ()
	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty

	var buf safeBuffer
	done := make(chan struct{})
	go func() {
		if _, err := io.Copy(&buf, ptmx); err != nil {
			_ = err // best-effort
		}
		close(done)
	}()

	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	waitContains(t, &buf, "PROMPTLY_START")
	waitContains(t, &buf, "TTY stdin=true stdout=true")
	waitContains(t, &buf, "PROMPT enter y:")

	if _, err := ptmx.Write([]byte("y\n")); err != nil {
		t.Fatalf("ptmx.Write: %v", err)
	}

	waitContains(t, &buf, "PROMPTLY_END")
	if err := waitCmd(t, cmd, 10*time.Second); err != nil {
		t.Fatalf("cmd failed: %v\nOUTPUT:\n%s", err, buf.String())
	}

	best := exec.CommandContext(context.Background(), ack, "best", "--tool", "promptly")
	best.Dir = root
	best.Env = os.Environ()

	out, err := best.CombinedOutput()
	if err != nil {
		t.Fatalf("ackchyually best failed: %v\nOUTPUT:\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "promptly") {
		t.Fatalf("expected best output to include promptly, got:\n%s", string(out))
	}
}
