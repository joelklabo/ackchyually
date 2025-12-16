package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomic_Error(t *testing.T) {
	tmp := t.TempDir()
	// Create a file
	f := filepath.Join(tmp, "file")
	if err := os.WriteFile(f, []byte(""), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Try to write to a path where 'file' is a directory component
	// e.g. tmp/file/target
	target := filepath.Join(f, "target")

	err := writeFileAtomic(target, []byte("data"), 0o600)
	if err == nil {
		t.Error("expected error writing to path with file as parent dir")
	}
}

func TestShimInstall_SymlinkError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	// Create a non-empty directory where the shim should be
	shimPath := filepath.Join(dir, "git")
	mkdirAll(t, shimPath)
	if err := os.WriteFile(filepath.Join(shimPath, "file"), []byte(""), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	code, _, errOut := captureStdoutStderr(t, func() int {
		return shimInstall([]string{"git"})
	})
	if code != 1 {
		t.Fatalf("shimInstall returned %d want 1", code)
	}
	if !strings.Contains(errOut, "symlink failed") && !strings.Contains(errOut, "remove") {
		// It might fail at remove or symlink depending on OS behavior with non-empty dir
		t.Fatalf("expected failure (symlink or remove), got:\n%s", errOut)
	}
}

func TestWriteFileAtomic_RenameError(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err := writeFileAtomic(target, []byte("data"), 0o600)
	if err == nil {
		t.Error("expected error renaming over directory")
	}
}

func TestWriteFileAtomic_TempFileError(t *testing.T) {
	// To trigger temp file error, we can try to write to a non-existent directory
	// But writeFileAtomic does MkdirAll.
	// So we need a path where MkdirAll fails or CreateTemp fails.
	// If we make the parent directory a file, MkdirAll should fail.

	tmp := t.TempDir()
	parent := filepath.Join(tmp, "file")
	if err := os.WriteFile(parent, []byte(""), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	path := filepath.Join(parent, "target")

	err := writeFileAtomic(path, []byte("content"), 0o600)
	if err == nil {
		t.Fatal("expected error writing to path where parent is a file")
	}
}

func TestWriteFileAtomic_ReadOnlyDir(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "readonly")
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	target := filepath.Join(dir, "file")
	err := writeFileAtomic(target, []byte("data"), 0o600)
	if err == nil {
		t.Error("expected error writing to read-only directory")
	}
}
