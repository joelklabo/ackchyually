package contextkey

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetect_UsesGitRootWhenPresent(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")
	sub := filepath.Join(root, "a", "b")
	mkdirAll(t, sub)
	mkdirAll(t, filepath.Join(root, ".git"))

	old, err := os.Getwd()
	if err != nil {
		old = ""
	}
	if err := os.Chdir(sub); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if old != "" {
			if err := os.Chdir(old); err != nil {
				t.Fatalf("restore cwd: %v", err)
			}
		}
	})

	got := Detect()
	prefix, gotPath, ok := strings.Cut(got, ":")
	if !ok || prefix != "git" {
		t.Fatalf("Detect()=%q, want prefix git:", got)
	}
	if cleanPath(gotPath) != cleanPath(root) {
		t.Fatalf("Detect()=%q, want git:%s", got, cleanPath(root))
	}
}

func TestDetect_FallsBackToCwdWhenNoGitRoot(t *testing.T) {
	cwd := t.TempDir()

	old, err := os.Getwd()
	if err != nil {
		old = ""
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if old != "" {
			if err := os.Chdir(old); err != nil {
				t.Fatalf("restore cwd: %v", err)
			}
		}
	})

	got := Detect()
	prefix, gotPath, ok := strings.Cut(got, ":")
	if !ok || prefix != "cwd" {
		t.Fatalf("Detect()=%q, want prefix cwd:", got)
	}
	if cleanPath(gotPath) != cleanPath(cwd) {
		t.Fatalf("Detect()=%q, want cwd:%s", got, cleanPath(cwd))
	}
}

func TestFindGitRoot_AcceptsGitFileMarker(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")
	sub := filepath.Join(root, "sub")
	mkdirAll(t, sub)

	// Some git setups (e.g., worktrees/submodules) can use a .git *file*.
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: /tmp/elsewhere\n"), 0o600); err != nil {
		t.Fatalf("write .git file: %v", err)
	}

	if got := findGitRoot(sub); cleanPath(got) != cleanPath(root) {
		t.Fatalf("findGitRoot=%q, want %q", got, cleanPath(root))
	}
}

func TestFindGitRoot_ReturnsEmptyWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "a", "b")
	mkdirAll(t, sub)

	if got := findGitRoot(sub); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", path, err)
	}
}

func cleanPath(path string) string {
	path = filepath.Clean(path)
	// macOS temp directories often include symlinked prefixes like /var -> /private/var.
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved)
	}
	return path
}
