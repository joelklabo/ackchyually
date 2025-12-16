package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShimEnable_AppendsToExisting(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte("existing content"), 0o600); err != nil {
		t.Fatalf("write rc: %v", err)
	}

	code, _, _ := captureStdoutStderr(t, func() int {
		return shimEnable(nil)
	})
	if code != 0 {
		t.Fatalf("shimEnable returned %d want 0", code)
	}

	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "existing content") {
		t.Errorf("existing content lost")
	}
	if !strings.Contains(s, "# ackchyually shims") {
		t.Errorf("shims not appended")
	}
	// Check newline handling (it adds a blank line separator)
	if !strings.Contains(s, "existing content\n\n# ackchyually shims") {
		t.Errorf("missing newline before append, got:\n%q", s)
	}
}

func TestShimEnable_Flags(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)
	t.Setenv("HOME", home)

	rcPath := filepath.Join(home, "custom.rc")

	code, out, _ := captureStdoutStderr(t, func() int {
		return shimEnable([]string{"--shell", "bash", "--file", rcPath})
	})
	if code != 0 {
		t.Fatalf("shimEnable returned %d want 0", code)
	}
	if !strings.Contains(out, "enabled shims in: "+rcPath) {
		t.Errorf("expected enabled message, got:\n%s", out)
	}

	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	if !strings.Contains(string(content), "export PATH=") {
		t.Errorf("rc file missing export PATH")
	}
}

func TestShimEnable_Bash(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")

	code, out, _ := captureStdoutStderr(t, func() int {
		return shimEnable(nil)
	})
	if code != 0 {
		t.Fatalf("shimEnable returned %d want 0", code)
	}
	if !strings.Contains(out, ".bashrc") {
		t.Errorf("expected .bashrc, got:\n%s", out)
	}
}

func TestShimEnable_Fish(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/fish")

	code, out, _ := captureStdoutStderr(t, func() int {
		return shimEnable(nil)
	})
	if code != 0 {
		t.Fatalf("shimEnable returned %d want 0", code)
	}
	if !strings.Contains(out, "config.fish") {
		t.Errorf("expected config.fish, got:\n%s", out)
	}
}

func TestShimEnable_UnknownShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/unknown")
	code, _, _ := captureStdoutStderr(t, func() int {
		return shimEnable(nil)
	})
	if code != 2 {
		t.Fatalf("shimEnable returned %d want 2", code)
	}
}

func TestShimEnable_UnreadableRC(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	mkdirAll(t, home)
	t.Setenv("HOME", home)

	rcPath := filepath.Join(home, "custom.rc")
	if err := os.WriteFile(rcPath, []byte(""), 0o600); err != nil {
		t.Fatalf("write rc: %v", err)
	}
	if err := os.Chmod(rcPath, 0o000); err != nil {
		t.Fatalf("chmod rc: %v", err)
	}

	code, _, _ := captureStdoutStderr(t, func() int {
		return shimEnable([]string{"--shell", "bash", "--file", rcPath})
	})
	if code != 1 {
		t.Fatalf("shimEnable returned %d want 1", code)
	}
}
