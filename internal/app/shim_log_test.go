package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joelklabo/ackchyually/internal/store"
)

func TestRunShim_Exit0UsageishLoggedAsNonSuccess(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	bin := filepath.Join(tmp, "bin")

	mkdirAll(t, home)
	mkdirAll(t, bin)

	tool := "exit0usage"
	toolPath := filepath.Join(bin, tool)
	writeFile(t, toolPath, "#!/bin/sh\necho 'usage: tool [flags]' 1>&2\nexit 0\n", 0o755)

	t.Setenv("HOME", home)
	t.Setenv("PATH", bin)

	code := RunShim(tool, []string{"not-help"})
	if code != 0 {
		t.Fatalf("shim returned %d want 0", code)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	var exitCode int
	if err := db.QueryRow(`SELECT exit_code FROM invocations WHERE tool=? ORDER BY created_at DESC LIMIT 1`, tool).Scan(&exitCode); err != nil {
		t.Fatalf("query exit_code: %v", err)
	}
	if exitCode != 64 {
		t.Fatalf("logged exit_code=%d want 64", exitCode)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, contents string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
