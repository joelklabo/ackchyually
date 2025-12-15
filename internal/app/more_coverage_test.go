package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joelklabo/ackchyually/internal/store"
)

func TestShimCmd_NoArgs_ShowsUsage(t *testing.T) {
	setTempHomeAndCWD(t)

	code, _, errOut := captureStdoutStderr(t, func() int {
		return shimCmd(nil)
	})
	if code != 2 {
		t.Fatalf("shimCmd returned %d want 2", code)
	}
	if !strings.Contains(errOut, "Commands:") {
		t.Fatalf("expected usage on stderr, got:\n%s", errOut)
	}
}

func TestTagCmd_NoArgs_ShowsUsage(t *testing.T) {
	setTempHomeAndCWD(t)

	code, _, errOut := captureStdoutStderr(t, func() int {
		return tagCmd(nil)
	})
	if code != 2 {
		t.Fatalf("tagCmd returned %d want 2", code)
	}
	if !strings.Contains(errOut, "Commands:") {
		t.Fatalf("expected usage on stderr, got:\n%s", errOut)
	}
}

func TestShimEnable_CreatesRCFileAndIsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)

	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	rcPath := filepath.Join(home, ".zshrc")

	code, out := captureStdout(t, func() int { return shimEnable(nil) })
	if code != 0 {
		t.Fatalf("shimEnable returned %d want 0, got:\n%s", code, out)
	}
	if !strings.Contains(out, "OK: enabled shims in: "+rcPath) {
		t.Fatalf("expected enable ok line, got:\n%s", out)
	}
	rc, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	if !strings.Contains(string(rc), "ackchyually shims") || !strings.Contains(string(rc), "export PATH=") {
		t.Fatalf("expected snippet in rc, got:\n%s", string(rc))
	}

	// Calling again should be a no-op.
	code, out = captureStdout(t, func() int { return shimEnable(nil) })
	if code != 0 {
		t.Fatalf("shimEnable (second) returned %d want 0, got:\n%s", code, out)
	}
	if !strings.Contains(out, "OK: already enabled in: "+rcPath) {
		t.Fatalf("expected already-enabled message, got:\n%s", out)
	}
}

func TestShimEnable_UnsupportedShell(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)

	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/nonesuch")

	code, _, errOut := captureStdoutStderr(t, func() int { return shimEnable(nil) })
	if code != 2 {
		t.Fatalf("shimEnable returned %d want 2", code)
	}
	if !strings.Contains(errOut, "unsupported shell") {
		t.Fatalf("expected unsupported shell error, got:\n%s", errOut)
	}
	if !strings.Contains(errOut, "export PATH=") {
		t.Fatalf("expected PATH export hint, got:\n%s", errOut)
	}
}

func TestShimUninstall_RemovesSymlink(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)
	t.Setenv("HOME", home)

	dir := shimDir()
	mkdirAll(t, dir)

	target := filepath.Join(tmp, "ack")
	writeFile(t, target, "#!/bin/sh\nexit 0\n", 0o755)

	shim := filepath.Join(dir, "git")
	if err := os.Symlink(target, shim); err != nil {
		t.Fatalf("symlink shim: %v", err)
	}

	if code := shimUninstall([]string{"git"}); code != 0 {
		t.Fatalf("shimUninstall returned %d want 0", code)
	}
	if _, err := os.Lstat(shim); !os.IsNotExist(err) {
		t.Fatalf("expected shim removed, got err=%v", err)
	}
}

func TestShimList_EmptyAndPopulated(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)
	t.Setenv("HOME", home)

	// No shim dir.
	code, out := captureStdout(t, func() int { return shimList(nil) })
	if code != 0 {
		t.Fatalf("shimList returned %d want 0", code)
	}
	if !strings.Contains(out, "(no shims installed)") {
		t.Fatalf("expected empty message, got:\n%s", out)
	}

	// Create shim dir with a couple shims and some ignored entries.
	dir := shimDir()
	mkdirAll(t, dir)
	writeFile(t, filepath.Join(dir, ".hidden"), "x", 0o644)
	writeFile(t, filepath.Join(dir, "ackchyually"), "x", 0o755)
	writeFile(t, filepath.Join(dir, "git"), "x", 0o755)
	writeFile(t, filepath.Join(dir, "gh"), "x", 0o755)

	code, out = captureStdout(t, func() int { return shimList(nil) })
	if code != 0 {
		t.Fatalf("shimList returned %d want 0, got:\n%s", code, out)
	}
	if !strings.Contains(out, "gh\n") || !strings.Contains(out, "git\n") {
		t.Fatalf("expected listed shims, got:\n%s", out)
	}
	if strings.Contains(out, "ackchyually") {
		t.Fatalf("did not expect ackchyually shim to be listed, got:\n%s", out)
	}
}

func TestShimDoctor_ActiveShimsReturnsOK(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)
	t.Setenv("HOME", home)

	dir := shimDir()
	realDir := filepath.Join(tmp, "real")
	mkdirAll(t, dir)
	mkdirAll(t, realDir)

	// Target for shims (doesn't need to be a real ackchyually binary for doctor).
	target := filepath.Join(tmp, "ack")
	writeFile(t, target, "#!/bin/sh\nexit 0\n", 0o755)

	// Real tool in PATH (excluding shims).
	realTool := filepath.Join(realDir, "git")
	writeFile(t, realTool, "#!/bin/sh\nexit 0\n", 0o755)

	shim := filepath.Join(dir, "git")
	if err := os.Symlink(target, shim); err != nil {
		t.Fatalf("symlink shim: %v", err)
	}

	t.Setenv("PATH", strings.Join([]string{dir, realDir}, string(os.PathListSeparator)))

	code, out := captureStdout(t, shimDoctor)
	if code != 0 {
		t.Fatalf("shimDoctor returned %d want 0, got:\n%s", code, out)
	}
	if !strings.Contains(out, "installed shims: git") {
		t.Fatalf("expected shim list, got:\n%s", out)
	}
	if !strings.Contains(out, "status:   ok") {
		t.Fatalf("expected ok status, got:\n%s", out)
	}
}

func TestAutoExecKnownSuccess_EarlyReturns(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)
	t.Setenv("HOME", home)

	ctxKey := "cwd:/tmp/repo"

	if code, ok := autoExecKnownSuccess("git", ctxKey, []string{"git", "status"}); ok || code != 0 {
		t.Fatalf("expected no auto-exec when empty, got code=%d ok=%v", code, ok)
	}

	// A single candidate equal to argvSafe should not auto-exec.
	if err := store.WithDB(func(db *store.DB) error {
		return db.InsertInvocation(store.Invocation{
			At:         time.Now(),
			DurationMS: 1,
			ContextKey: ctxKey,
			Tool:       "git",
			ExePath:    "/usr/bin/git",
			ArgvJSON:   store.MustJSON([]string{"git", "status"}),
			ExitCode:   0,
			Mode:       "pipes",
		})
	}); err != nil {
		t.Fatalf("seed invocation: %v", err)
	}
	if code, ok := autoExecKnownSuccess("git", ctxKey, []string{"git", "status"}); ok || code != 0 {
		t.Fatalf("expected no auto-exec when cmd==argvSafe, got code=%d ok=%v", code, ok)
	}

	// A redacted candidate should never auto-exec.
	if err := store.WithDB(func(db *store.DB) error {
		return db.InsertInvocation(store.Invocation{
			At:         time.Now(),
			DurationMS: 1,
			ContextKey: ctxKey,
			Tool:       "git",
			ExePath:    "/usr/bin/git",
			ArgvJSON:   store.MustJSON([]string{"git", "--token", "<redacted>"}),
			ExitCode:   0,
			Mode:       "pipes",
		})
	}); err != nil {
		t.Fatalf("seed invocation (redacted): %v", err)
	}
	if code, ok := autoExecKnownSuccess("git", ctxKey, []string{"git", "--token", "<redacted>"}); ok || code != 0 {
		t.Fatalf("expected no auto-exec when candidate contains redacted, got code=%d ok=%v", code, ok)
	}
}

func TestVersionHelpers(t *testing.T) {
	if got, want := shortSHA("0123456789abcdef"), "0123456789ab"; got != want {
		t.Fatalf("shortSHA=%q, want %q", got, want)
	}
	if got, want := shortSHA("short"), "short"; got != want {
		t.Fatalf("shortSHA=%q, want %q", got, want)
	}

	for _, tc := range []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "dev", want: "dev"},
		{in: "1.2.3", want: "v1.2.3"},
		{in: "v1.2.3", want: "v1.2.3"},
	} {
		if got := normalizeVersion(tc.in); got != tc.want {
			t.Fatalf("normalizeVersion(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}
