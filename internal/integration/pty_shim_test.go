package integration

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
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

	cmd := exec.Command(shimTool)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"PATH="+strings.Join([]string{shimDir, realDir}, string(os.PathListSeparator)),
	)

	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty

	var buf bytes.Buffer
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
	_ = tty.Close()

	waitContains(t, &buf, "PROMPTLY_START")
	waitContains(t, &buf, "TTY stdin=true stdout=true")
	waitContains(t, &buf, "PROMPT enter y:")

	must(t, pty.Setsize(ptmx, &pty.Winsize{Rows: 40, Cols: 100}))
	must(t, cmd.Process.Signal(syscall.SIGWINCH))
	time.Sleep(50 * time.Millisecond)

	if _, err := ptmx.Write([]byte("y\n")); err != nil {
		t.Fatalf("ptmx.Write: %v", err)
	}

	waitContains(t, &buf, "SIZE_AFTER rows=40 cols=100")
	waitContains(t, &buf, "PROMPTLY_END")

	if err := waitCmd(t, cmd, 5*time.Second); err != nil {
		t.Fatalf("cmd failed: %v\nOUTPUT:\n%s", err, buf.String())
	}

	_ = ptmx.Close()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
	}
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

	cmd := exec.Command(shimTool)
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
	cmd := exec.Command("go", "build", "-o", out, pkg)
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

func waitContains(t *testing.T, buf *bytes.Buffer, needle string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), needle) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %q\nOUTPUT:\n%s", needle, buf.String())
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
