package app

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShimInstall_PathFirst_PersistedInRC_NoPersistTip(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)

	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	dir := shimDir()
	rcPath := filepath.Join(home, ".zshrc")
	writeFile(t, rcPath, enableSnippet("zsh", dir), 0o644)

	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin")

	code, out := captureStdout(t, func() int {
		return shimInstall([]string{"git"})
	})
	if code != 0 {
		t.Fatalf("shimInstall returned %d want 0, got:\n%s", code, out)
	}

	if !strings.Contains(out, "OK: shim dir is first in PATH\n") {
		t.Fatalf("expected shim PATH-first ok line, got:\n%s", out)
	}
	if !strings.Contains(out, "OK: shims are enabled in: "+rcPath+"\n") {
		t.Fatalf("expected persisted rc ok line, got:\n%s", out)
	}
	if strings.Contains(out, "Tip: persist this with: ackchyually shim enable\n") {
		t.Fatalf("did not expect persist tip, got:\n%s", out)
	}
}

func TestShimInstall_PathFirst_NotPersistedInRC_ShowsPersistTip(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)

	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	dir := shimDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin")

	code, out := captureStdout(t, func() int {
		return shimInstall([]string{"git"})
	})
	if code != 0 {
		t.Fatalf("shimInstall returned %d want 0, got:\n%s", code, out)
	}

	if !strings.Contains(out, "OK: shim dir is first in PATH\n") {
		t.Fatalf("expected shim PATH-first ok line, got:\n%s", out)
	}
	if !strings.Contains(out, "Tip: persist this with: ackchyually shim enable\n") {
		t.Fatalf("expected persist tip, got:\n%s", out)
	}
}

func captureStdout(t *testing.T, fn func() int) (int, string) {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	code := fn()
	os.Stdout = old

	_ = w.Close()
	b, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return code, string(b)
}
