package integration

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func TestPTY_ShimRunsToolInPTY_AndPropagatesResize(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no windows support")
	}

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

	shimTool := filepath.Join(shimDir, "promptly")
	if err := os.Symlink(ack, shimTool); err != nil {
		t.Fatalf("symlink shim: %v", err)
	}

	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open: %v", err)
	}
	defer ptmx.Close()
	defer tty.Close()

	must(t, pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80}))

	cmd := exec.CommandContext(context.Background(), shimTool)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"PATH="+strings.Join([]string{shimDir, realDir}, string(os.PathListSeparator)),
	)

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

	// Resize the controlling terminal for ackchyually (the slave side `tty`) so
	// term.GetSize inside the shim reliably observes the updated size on all platforms.
	must(t, pty.Setsize(ptmx, &pty.Winsize{Rows: 40, Cols: 100}))
	must(t, pty.Setsize(tty, &pty.Winsize{Rows: 40, Cols: 100}))
	must(t, cmd.Process.Signal(syscall.SIGWINCH))
	waitForSize(t, tty, 40, 100)
	time.Sleep(200 * time.Millisecond)

	if _, err := ptmx.Write([]byte("y\n")); err != nil {
		t.Fatalf("ptmx.Write: %v", err)
	}

	waitContains(t, &buf, "SIZE_AFTER rows=40 cols=100")
	waitContains(t, &buf, "PROMPTLY_END")

	if err := waitCmd(t, cmd, 10*time.Second); err != nil {
		t.Fatalf("cmd failed: %v\nOUTPUT:\n%s", err, buf.String())
	}

	_ = ptmx.Close()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
	}
}

func waitForSize(t *testing.T, f *os.File, rows, cols uint16) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		gotCols, gotRows, err := term.GetSize(int(f.Fd()))
		if err == nil && gotRows == int(rows) && gotCols == int(cols) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	gotCols, gotRows, err := term.GetSize(int(f.Fd()))
	t.Fatalf("timeout waiting for size rows=%d cols=%d (got rows=%d cols=%d, err=%v)", rows, cols, gotRows, gotCols, err)
}

func TestPipes_NonInteractiveUsesPipes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no windows support")
	}

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

	shimTool := filepath.Join(shimDir, "promptly")
	must(t, os.Symlink(ack, shimTool))

	cmd := exec.CommandContext(context.Background(), shimTool)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"PATH="+strings.Join([]string{shimDir, realDir}, string(os.PathListSeparator)),
	)
	cmd.Stdin = strings.NewReader("y\n")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cmd failed: %v\nOUTPUT:\n%s", err, string(out))
	}
	s := string(out)
	if !strings.Contains(s, "TTY stdin=false") {
		t.Fatalf("expected non-tty tool under pipes, got:\n%s", s)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if exists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (go.mod)")
		}
		dir = parent
	}
}

func build(t *testing.T, dir, pkg, out string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", out, pkg)
	cmd.Dir = dir
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build %s failed: %v\n%s", pkg, err, string(b))
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	must(t, os.MkdirAll(path, 0o755))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func waitContains(t *testing.T, buf *safeBuffer, needle string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), needle) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %q\nOUTPUT:\n%s", needle, buf.String())
}

type safeBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.b.Write(p)
}

func (s *safeBuffer) String() string {
	s.m.Lock()
	defer s.m.Unlock()
	return s.b.String()
}

func waitCmd(t *testing.T, cmd *exec.Cmd, timeout time.Duration) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		if err := cmd.Process.Kill(); err != nil {
			_ = err // best-effort
		}
		return syscall.ETIMEDOUT
	}
}