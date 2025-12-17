//go:build !windows

package copilot

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWrapper_InstallRunUndo_RegularFile(t *testing.T) {
	tmp := t.TempDir()
	wrapperPath := filepath.Join(tmp, "copilot")

	writeExec(t, wrapperPath, `#!/bin/sh
echo "OUT:ORIG"
echo "OUT:PATH=$PATH"
echo "ERR:ORIG" 1>&2
exit 42
`)

	plan, err := PlanInstall(wrapperPath, filepath.Join(tmp, "shims"))
	if err != nil {
		t.Fatalf("PlanInstall: %v", err)
	}
	if err := Apply(plan); err != nil {
		t.Fatalf("Apply(install): %v", err)
	}

	stdout, stderr, code := run(t, wrapperPath, map[string]string{"PATH": "AAA"})
	if code != 42 {
		t.Fatalf("exit=%d, want 42\nSTDOUT:\n%s\nSTDERR:\n%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "OUT:ORIG") {
		t.Fatalf("missing stdout marker\nSTDOUT:\n%s", stdout)
	}
	if !strings.Contains(stderr, "ERR:ORIG") {
		t.Fatalf("missing stderr marker\nSTDERR:\n%s", stderr)
	}
	if !strings.Contains(stdout, "OUT:PATH="+filepath.Join(tmp, "shims")+":AAA") {
		t.Fatalf("expected PATH to be shim-first\nSTDOUT:\n%s", stdout)
	}

	undo, err := PlanUndo(wrapperPath)
	if err != nil {
		t.Fatalf("PlanUndo: %v", err)
	}
	if err := Apply(undo); err != nil {
		t.Fatalf("Apply(undo): %v", err)
	}

	if _, err := os.Stat(wrapperPath); err != nil {
		t.Fatalf("expected original restored at %s: %v", wrapperPath, err)
	}
	if _, err := os.Stat(wrapperPath + backupSuffix); err == nil {
		t.Fatalf("expected backup removed")
	}
}

func TestWrapper_InstallUndo_Symlink(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "copilot-target")
	wrapperPath := filepath.Join(tmp, "copilot")

	writeExec(t, target, `#!/bin/sh
echo "OUT:TARGET"
exit 0
`)
	if err := os.Symlink(target, wrapperPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	plan, err := PlanInstall(wrapperPath, filepath.Join(tmp, "shims"))
	if err != nil {
		t.Fatalf("PlanInstall: %v", err)
	}
	if err := Apply(plan); err != nil {
		t.Fatalf("Apply(install): %v", err)
	}

	stdout, stderr, code := run(t, wrapperPath, nil)
	if code != 0 || stderr != "" || !strings.Contains(stdout, "OUT:TARGET") {
		t.Fatalf("unexpected run result: code=%d\nSTDOUT:\n%s\nSTDERR:\n%s", code, stdout, stderr)
	}

	undo, err := PlanUndo(wrapperPath)
	if err != nil {
		t.Fatalf("PlanUndo: %v", err)
	}
	if err := Apply(undo); err != nil {
		t.Fatalf("Apply(undo): %v", err)
	}

	if _, err := os.Lstat(wrapperPath); err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if _, err := os.Readlink(wrapperPath); err != nil {
		t.Fatalf("expected restored symlink: %v", err)
	}
}

func writeExec(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
}

func run(t *testing.T, path string, env map[string]string) (stdout string, stderr string, code int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err == nil {
		return outBuf.String(), errBuf.String(), 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return outBuf.String(), errBuf.String(), ee.ExitCode()
	}
	t.Fatalf("run failed: %v", err)
	return "", "", 0
}
