package execx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShellJoin(t *testing.T) {
	if got, want := ShellJoin([]string{"git", "status"}), "git status"; got != want {
		t.Fatalf("ShellJoin = %q, want %q", got, want)
	}
	if got, want := ShellJoin([]string{"echo", "hi there"}), "echo 'hi there'"; got != want {
		t.Fatalf("ShellJoin = %q, want %q", got, want)
	}
	if got, want := ShellJoin([]string{"echo", "a'b"}), `echo a\'b`; got != want {
		t.Fatalf("ShellJoin = %q, want %q", got, want)
	}
}

func TestContainsFold(t *testing.T) {
	tests := []struct {
		haystack string
		needle   string
		want     bool
	}{
		{"Hello World", "world", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "Foo", false},
		{"", "foo", false},
		{"foo", "", true},
	}

	for _, tt := range tests {
		if got := ContainsFold(tt.haystack, tt.needle); got != tt.want {
			t.Errorf("ContainsFold(%q, %q) = %v; want %v", tt.haystack, tt.needle, got, tt.want)
		}
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

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", path, err)
	}
}

func writeExec(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o700); err != nil { //nolint:gosec
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
