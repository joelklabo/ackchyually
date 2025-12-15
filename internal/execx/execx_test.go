package execx

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: "''"},
		{in: "abcXYZ012/._-=:,%+@", want: "abcXYZ012/._-=:,%+@"},
		{in: "hello world", want: "'hello world'"},
		{in: "a'b", want: `'a'"'"'b'`},
	}

	for _, tt := range tests {
		if got := shellQuote(tt.in); got != tt.want {
			t.Fatalf("shellQuote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestShellJoin(t *testing.T) {
	if got, want := ShellJoin([]string{"git", "status"}), "git status"; got != want {
		t.Fatalf("ShellJoin = %q, want %q", got, want)
	}
	if got, want := ShellJoin([]string{"echo", "hi there"}), "echo 'hi there'"; got != want {
		t.Fatalf("ShellJoin = %q, want %q", got, want)
	}
	if got, want := ShellJoin([]string{"echo", "a'b"}), `echo 'a'"'"'b'`; got != want {
		t.Fatalf("ShellJoin = %q, want %q", got, want)
	}
}

func TestTail_TrimsToCapacity(t *testing.T) {
	tail := NewTail(5)
	if _, err := tail.Write([]byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got, want := tail.String(), "hello"; got != want {
		t.Fatalf("tail=%q, want %q", got, want)
	}

	if _, err := tail.Write([]byte("world")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got, want := tail.String(), "world"; got != want {
		t.Fatalf("tail=%q, want %q", got, want)
	}
}

func TestWhichSkippingShims_SkipsShimDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	shimDir := ShimDir()
	realDir := filepath.Join(t.TempDir(), "real")
	mustMkdir(t, shimDir)
	mustMkdir(t, realDir)

	tool := "tool"
	writeExec(t, filepath.Join(shimDir, tool))
	realPath := filepath.Join(realDir, tool)
	writeExec(t, realPath)

	t.Setenv("PATH", stringsJoinPath(shimDir, realDir))

	got, err := WhichSkippingShims(tool)
	if err != nil {
		t.Fatalf("WhichSkippingShims: %v", err)
	}
	if got != realPath {
		t.Fatalf("WhichSkippingShims=%q, want %q", got, realPath)
	}
}

func TestWhichSkippingShims_NotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	shimDir := ShimDir()
	mustMkdir(t, shimDir)

	t.Setenv("PATH", shimDir)

	_, err := WhichSkippingShims("missing")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestIsExecutableFile(t *testing.T) {
	dir := t.TempDir()

	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if isExecutableFile(f) {
		t.Fatalf("expected non-executable file to be false")
	}
	if err := os.Chmod(f, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if !isExecutableFile(f) {
		t.Fatalf("expected executable file to be true")
	}
	if isExecutableFile(dir) {
		t.Fatalf("expected dir to be false")
	}
}

func TestExitCode(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Fatalf("exitCode(nil)=%d, want 0", got)
	}
	if got := exitCode(errors.New("boom")); got != 1 {
		t.Fatalf("exitCode(non-exit-error)=%d, want 1", got)
	}

	if runtime.GOOS == "windows" {
		t.Skip("unix-only exit status semantics")
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
	cmd := exec.Command("sh", "-c", "exit "+itoa(code))
	return cmd.Run()
}

func runShellKilledBySignal(t *testing.T, sig syscall.Signal) error {
	t.Helper()

	cmd := exec.Command("sh", "-c", "sleep 10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := cmd.Process.Signal(sig); err != nil {
		t.Fatalf("signal: %v", err)
	}
	return cmd.Wait()
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", path, err)
	}
}

func writeExec(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o755); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func stringsJoinPath(dirs ...string) string {
	sep := string(os.PathListSeparator)
	out := ""
	for i, d := range dirs {
		if i > 0 {
			out += sep
		}
		out += d
	}
	return out
}

// Small helpers to avoid pulling in extra deps in tests.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
