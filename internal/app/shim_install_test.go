package app

import (
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
	writeFile(t, rcPath, enableSnippet("zsh", dir), 0o600)

	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin")

	code, out, _ := captureStdoutStderr(t, func() int {
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

	code, out, _ := captureStdoutStderr(t, func() int {
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

func TestShimInstall_NoArgs_ReturnsError(t *testing.T) {
	code, _, out := captureStdoutStderr(t, func() int {
		return shimInstall([]string{})
	})
	if code != 2 {
		t.Fatalf("shimInstall returned %d want 2", code)
	}
	if !strings.Contains(out, "shim install: specify tools") {
		t.Fatalf("expected specify tools error, got:\n%s", out)
	}
}

func TestShimUninstall_RemovesShim(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	shimPath := filepath.Join(dir, "git")
	writeFile(t, shimPath, "fake shim", 0o755)

	code, _, _ := captureStdoutStderr(t, func() int {
		return shimUninstall([]string{"git"})
	})
	if code != 0 {
		t.Fatalf("shimUninstall returned %d want 0", code)
	}

	if _, err := os.Stat(shimPath); !os.IsNotExist(err) {
		t.Fatalf("shim file still exists after uninstall")
	}
}

func TestShimUninstall_NoArgs_ReturnsError(t *testing.T) {
	code, _, out := captureStdoutStderr(t, func() int {
		return shimUninstall([]string{})
	})
	if code != 2 {
		t.Fatalf("shimUninstall returned %d want 2", code)
	}
	if !strings.Contains(out, "shim uninstall: specify tools") {
		t.Fatalf("expected specify tools error, got:\n%s", out)
	}
}

func TestShimDoctor_NoShims(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	code, out, _ := captureStdoutStderr(t, shimDoctor)
	if code != 0 {
		t.Fatalf("shimDoctor returned %d want 0", code)
	}
	if !strings.Contains(out, "installed shims: (none)") {
		t.Fatalf("expected no shims message, got:\n%s", out)
	}
}

func TestShimDoctor_Healthy(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	bin := filepath.Join(tmp, "bin")
	mkdirAll(t, home)
	mkdirAll(t, bin)

	t.Setenv("HOME", home)

	// Create a "real" tool
	realGit := filepath.Join(bin, "git")
	writeFile(t, realGit, "#!/bin/sh\necho git", 0o755)

	// Install shim
	// We need to mock os.Executable to point to something valid for symlinking
	// But shimInstall uses os.Executable(). We can't easily mock that without internal changes.
	// However, we can manually create the shim environment that doctor expects.

	dir := shimDir()
	mkdirAll(t, dir)

	// Create shim symlink pointing to our "ackchyually" (which we'll fake as the test binary)
	fakeAck := filepath.Join(bin, "ackchyually")
	writeFile(t, fakeAck, "fake ackchyually", 0o755)

	shimPath := filepath.Join(dir, "git")
	if err := os.Symlink(fakeAck, shimPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// Set PATH so shim is first, then bin (where real tool is)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+bin)

	code, out, _ := captureStdoutStderr(t, shimDoctor)
	if code != 0 {
		t.Fatalf("shimDoctor returned %d want 0, output:\n%s", code, out)
	}
	if !strings.Contains(out, "status:   ok (shims are active)") {
		t.Fatalf("expected ok status, got:\n%s", out)
	}
}

func TestShimDoctor_Inactive(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	bin := filepath.Join(tmp, "bin")
	mkdirAll(t, home)
	mkdirAll(t, bin)

	t.Setenv("HOME", home)

	// Create a "real" tool
	realGit := filepath.Join(bin, "git")
	writeFile(t, realGit, "#!/bin/sh\necho git", 0o755)

	dir := shimDir()
	mkdirAll(t, dir)

	fakeAck := filepath.Join(bin, "ackchyually")
	writeFile(t, fakeAck, "fake ackchyually", 0o755)

	shimPath := filepath.Join(dir, "git")
	if err := os.Symlink(fakeAck, shimPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// Set PATH so bin is first (shadowing shim)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+dir)

	code, out, _ := captureStdoutStderr(t, shimDoctor)
	if code != 1 {
		t.Fatalf("shimDoctor returned %d want 1", code)
	}
	if !strings.Contains(out, "inactive: git") {
		t.Fatalf("expected inactive git, got:\n%s", out)
	}
}

func TestShimDoctor_BrokenShim(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	// Create a file instead of symlink
	shimPath := filepath.Join(dir, "git")
	writeFile(t, shimPath, "not a symlink", 0o755)

	code, out, _ := captureStdoutStderr(t, shimDoctor)
	if code != 1 {
		t.Fatalf("shimDoctor returned %d want 1", code)
	}
	if !strings.Contains(out, "broken:   git") {
		t.Fatalf("expected broken git, got:\n%s", out)
	}
}

func TestShimDoctor_MissingRealTool(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	bin := filepath.Join(tmp, "bin")
	mkdirAll(t, home)
	mkdirAll(t, bin)

	t.Setenv("HOME", home)

	dir := shimDir()
	mkdirAll(t, dir)

	fakeAck := filepath.Join(bin, "ackchyually")
	writeFile(t, fakeAck, "fake ackchyually", 0o755)

	shimPath := filepath.Join(dir, "git")
	if err := os.Symlink(fakeAck, shimPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// Set PATH so shim is there, but real tool is nowhere
	t.Setenv("PATH", dir)

	code, out, _ := captureStdoutStderr(t, shimDoctor)
	if code != 1 {
		t.Fatalf("shimDoctor returned %d want 1", code)
	}
	if !strings.Contains(out, "missing:  git") {
		t.Fatalf("expected missing git, got:\n%s", out)
	}
}

func TestShimDoctor_DBPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	code, out, _ := captureStdoutStderr(t, shimDoctor)
	if code != 0 {
		t.Fatalf("shimDoctor returned %d want 0", code)
	}
	if !strings.Contains(out, "db:       ") {
		t.Errorf("output missing db path, got:\n%s", out)
	}
}

func TestShimDoctor_AckchyuallyShim(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	// Create ackchyually shim
	writeFile(t, filepath.Join(dir, "ackchyually"), "fake", 0o755)

	// Need another shim to trigger the note (if only ackchyually, it says (none))
	writeFile(t, filepath.Join(dir, "git"), "fake", 0o755)

	code, out, _ := captureStdoutStderr(t, shimDoctor)
	if code != 1 {
		t.Fatalf("shimDoctor returned %d want 1", code)
	}
	if !strings.Contains(out, "shim dir contains 'ackchyually'") {
		t.Errorf("output missing ackchyually shim note, got:\n%s", out)
	}
}

func TestShimDoctor_BrokenTarget(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	shimPath := filepath.Join(dir, "git")
	// Symlink to non-existent file
	if err := os.Symlink("/non/existent/file", shimPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	code, out, _ := captureStdoutStderr(t, shimDoctor)
	if code != 1 {
		t.Fatalf("shimDoctor returned %d want 1", code)
	}
	if !strings.Contains(out, "broken shim target") {
		t.Errorf("expected broken shim target message, got:\n%s", out)
	}
}

func TestShimInstall_SkipAckchyually(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()

	code, out, _ := captureStdoutStderr(t, func() int {
		return shimInstall([]string{"ackchyually", "git"})
	})
	if code != 0 {
		t.Fatalf("shimInstall returned %d want 0", code)
	}
	if !strings.Contains(out, "Installed shims in:") {
		t.Errorf("expected success message, got:\n%s", out)
	}

	if _, err := os.Lstat(filepath.Join(dir, "ackchyually")); err == nil {
		t.Errorf("ackchyually shim should not be created")
	}
	if _, err := os.Lstat(filepath.Join(dir, "git")); err != nil {
		t.Errorf("git shim should be created")
	}
}

func TestShimDir_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	dir := shimDir()
	if !strings.HasSuffix(dir, ".local/share/ackchyually/shims") {
		t.Errorf("shimDir() = %q, expected suffix .local/share/ackchyually/shims", dir)
	}
}

func TestShimInstall_MkdirError(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)
	t.Setenv("HOME", home)

	// Create a file where the shim dir parent should be
	// shimDir is .../ackchyually/shims
	// We create .../ackchyually as a file
	parent := filepath.Join(home, ".local", "share", "ackchyually")
	if err := os.MkdirAll(filepath.Dir(parent), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(parent, []byte("file"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	code, _, errOut := captureStdoutStderr(t, func() int {
		return shimInstall([]string{"git"})
	})
	if code != 1 {
		t.Fatalf("shimInstall returned %d want 1", code)
	}
	if !strings.Contains(errOut, "ackchyually:") {
		t.Fatalf("expected error message, got:\n%s", errOut)
	}
}

func TestShimDoctor_PermissionErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}

	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)
	t.Setenv("HOME", home)

	dir := shimDir()
	// Create parent dirs
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	// Create shim dir as a file
	if err := os.WriteFile(dir, []byte("file"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	// Ensure we can clean it up
	defer func() { _ = os.Remove(dir) }()

	code, _, errOut := captureStdoutStderr(t, shimDoctor)
	if code != 1 {
		t.Fatalf("shimDoctor returned %d want 1. Stderr: %s", code, errOut)
	}
	// Expect error message (not permission denied, but some error)
	if !strings.Contains(errOut, "ackchyually:") {
		t.Fatalf("expected error message in stderr, got:\n%s", errOut)
	}
}

func TestShimInstall_Success(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()

	code, out, _ := captureStdoutStderr(t, func() int {
		return shimInstall([]string{"git"})
	})
	if code != 0 {
		t.Fatalf("shimInstall returned %d want 0", code)
	}
	if !strings.Contains(out, "Installed shims in:") {
		t.Errorf("expected success message, got:\n%s", out)
	}

	shimPath := filepath.Join(dir, "git")
	if _, err := os.Lstat(shimPath); err != nil {
		t.Errorf("shim not created: %v", err)
	}
}

func TestShimUninstall_NonExistent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	code, _, _ := captureStdoutStderr(t, func() int {
		return shimUninstall([]string{"git"})
	})
	if code != 0 {
		t.Fatalf("shimUninstall returned %d want 0", code)
	}
}

func TestShimDoctor_TargetPermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	target := filepath.Join(t.TempDir(), "target")
	writeFile(t, target, "target", 0o000) // No permissions

	shimPath := filepath.Join(dir, "git")
	if err := os.Symlink(target, shimPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	code, out, _ := captureStdoutStderr(t, shimDoctor)
	if code != 1 {
		t.Fatalf("shimDoctor returned %d want 1", code)
	}
	if !strings.Contains(out, "broken shim target") {
		t.Errorf("expected broken shim target message, got:\n%s", out)
	}
}

func TestShimDoctor_TargetIsDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	target := filepath.Join(t.TempDir(), "targetDir")
	mkdirAll(t, target)

	shimPath := filepath.Join(dir, "git")
	if err := os.Symlink(target, shimPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	code, out, _ := captureStdoutStderr(t, shimDoctor)
	if code != 1 {
		t.Fatalf("shimDoctor returned %d want 1", code)
	}
	if !strings.Contains(out, "broken shim target") {
		t.Errorf("expected broken shim target message, got:\n%s", out)
	}
}

func TestShimDoctor_TargetNotExecutable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	target := filepath.Join(t.TempDir(), "targetFile")
	writeFile(t, target, "content", 0o600) // Not executable

	shimPath := filepath.Join(dir, "git")
	if err := os.Symlink(target, shimPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	code, out, _ := captureStdoutStderr(t, shimDoctor)
	if code != 1 {
		t.Fatalf("shimDoctor returned %d want 1", code)
	}
	if !strings.Contains(out, "broken shim target") {
		t.Errorf("expected broken shim target message, got:\n%s", out)
	}
}

func TestShimDoctor_ReadDirError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	// Create parent
	mkdirAll(t, filepath.Dir(dir))
	// Create dir with no permissions
	if err := os.Mkdir(dir, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.Chmod(dir, 0o755) //nolint:errcheck

	code, _, errOut := captureStdoutStderr(t, shimDoctor)
	if code != 1 {
		t.Fatalf("shimDoctor returned %d want 1", code)
	}
	if !strings.Contains(errOut, "ackchyually:") {
		t.Errorf("expected error message, got:\n%s", errOut)
	}
}

func TestShimDoctor_LstatError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	t.Setenv("HOME", t.TempDir())
	dir := shimDir()
	mkdirAll(t, dir)

	writeFile(t, filepath.Join(dir, "git"), "fake", 0o755)

	// Remove execute permission from dir
	if err := os.Chmod(dir, 0o600); err != nil { // Read/Write, no Execute
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(dir, 0o755) //nolint:errcheck

	code, out, errOut := captureStdoutStderr(t, shimDoctor)

	// If ReadDir fails, it prints to stderr.
	if strings.Contains(errOut, "ackchyually:") {
		t.Skip("ReadDir failed before Lstat, skipping")
	}

	if code != 1 {
		t.Fatalf("shimDoctor returned %d want 1", code)
	}
	if !strings.Contains(out, "stat failed") {
		t.Errorf("expected stat failed message, got:\n%s", out)
	}
}
