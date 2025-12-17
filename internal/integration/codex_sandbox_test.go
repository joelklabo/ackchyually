//go:build !windows

package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIntegrateVerifyCodex_ConfigSanity(t *testing.T) {
	root := repoRoot(t)

	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	binDir := filepath.Join(tmp, "bin")

	mkdirAll(t, home)
	mkdirAll(t, binDir)

	ack := filepath.Join(binDir, "ackchyually")
	build(t, root, "./cmd/ackchyually", ack)

	// Fake codex so verify treats it as installed.
	fakeCodex := filepath.Join(binDir, "codex")
	must(t, os.WriteFile(fakeCodex, []byte("#!/bin/sh\necho codex 0.0.0\n"), 0o755)) //nolint:gosec

	env := append(os.Environ(),
		"HOME="+home,
		"PATH="+strings.Join([]string{binDir}, string(os.PathListSeparator)),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	{
		cmd := exec.CommandContext(ctx, ack, "shim", "install", "git")
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("shim install git failed: %v\nOUTPUT:\n%s", err, string(out))
		}
	}
	{
		cmd := exec.CommandContext(ctx, ack, "integrate", "codex")
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("integrate codex failed: %v\nOUTPUT:\n%s", err, string(out))
		}
	}
	{
		cmd := exec.CommandContext(ctx, ack, "integrate", "verify", "codex")
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("integrate verify codex failed: %v\nOUTPUT:\n%s", err, string(out))
		}
		if !strings.Contains(string(out), "codex: ok") {
			t.Fatalf("expected verify output to include ok, got:\n%s", string(out))
		}
	}
}
