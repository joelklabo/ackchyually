package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShimList(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	writeFile(t, filepath.Join(dir, "git"), "fake", 0o755)
	writeFile(t, filepath.Join(dir, "npm"), "fake", 0o755)

	code, out, _ := captureStdoutStderr(t, func() int {
		return shimList(nil)
	})
	if code != 0 {
		t.Fatalf("shimList returned %d want 0", code)
	}
	if !strings.Contains(out, "git") {
		t.Errorf("output missing git")
	}
	if !strings.Contains(out, "npm") {
		t.Errorf("output missing npm")
	}
}

func TestShimList_Empty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	code, out, _ := captureStdoutStderr(t, func() int {
		return shimList(nil)
	})
	if code != 0 {
		t.Fatalf("shimList returned %d want 0", code)
	}
	if !strings.Contains(out, "(no shims installed)") {
		t.Errorf("expected (no shims installed), got:\n%s", out)
	}
}

func TestShimList_HiddenFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	writeFile(t, filepath.Join(dir, ".hidden"), "fake", 0o755)
	writeFile(t, filepath.Join(dir, "visible"), "fake", 0o755)

	code, out, _ := captureStdoutStderr(t, func() int {
		return shimList(nil)
	})
	if code != 0 {
		t.Fatalf("shimList returned %d want 0", code)
	}
	if strings.Contains(out, ".hidden") {
		t.Errorf("output should not contain hidden file")
	}
	if !strings.Contains(out, "visible") {
		t.Errorf("output missing visible file")
	}
}

func TestShimList_PermissionError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(dir, 0o755) }() //nolint:errcheck

	code, _, errOut := captureStdoutStderr(t, func() int {
		return shimList(nil)
	})
	if code != 1 {
		t.Fatalf("shimList returned %d want 1", code)
	}
	if !strings.Contains(errOut, "ackchyually:") {
		t.Errorf("expected error message, got:\n%s", errOut)
	}
}

func TestShimList_Ignored(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	// Directory should be ignored
	mkdirAll(t, filepath.Join(dir, "subdir"))

	// ackchyually binary should be ignored
	writeFile(t, filepath.Join(dir, "ackchyually"), "fake", 0o755)
	writeFile(t, filepath.Join(dir, "ackchyually.exe"), "fake", 0o755)

	// Visible file should be shown
	writeFile(t, filepath.Join(dir, "visible"), "fake", 0o755)

	code, out, _ := captureStdoutStderr(t, func() int {
		return shimList(nil)
	})
	if code != 0 {
		t.Fatalf("shimList returned %d want 0", code)
	}
	if strings.Contains(out, "subdir") {
		t.Errorf("output should not contain subdir")
	}
	if strings.Contains(out, "ackchyually") {
		t.Errorf("output should not contain ackchyually")
	}
	if !strings.Contains(out, "visible") {
		t.Errorf("output missing visible file")
	}
}

func TestShimList_Args(t *testing.T) {
	code, _, errOut := captureStdoutStderr(t, func() int {
		return shimList([]string{"arg"})
	})
	if code != 2 {
		t.Fatalf("shimList returned %d want 2", code)
	}
	if !strings.Contains(errOut, "usage:") {
		t.Errorf("expected usage, got:\n%s", errOut)
	}
}
