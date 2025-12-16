package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShimEnable_Usage(t *testing.T) {
	code, _, errOut := captureStdoutStderr(t, func() int {
		return shimEnable([]string{"arg"})
	})
	if code != 2 {
		t.Errorf("shimEnable(args) = %d; want 2", code)
	}
	if !strings.Contains(errOut, "usage:") {
		t.Errorf("expected usage, got:\n%s", errOut)
	}
}

func TestShimEnable_UnknownShell_Arg(t *testing.T) {
	code, _, errOut := captureStdoutStderr(t, func() int {
		return shimEnable([]string{"--shell=ksh"})
	})
	if code != 2 {
		t.Errorf("shimEnable(ksh) = %d; want 2", code)
	}
	if !strings.Contains(errOut, "unsupported shell") {
		t.Errorf("expected unsupported shell msg, got:\n%s", errOut)
	}
}

func TestShimEnable_DetectShell(t *testing.T) {
	// Mock SHELL env
	t.Setenv("SHELL", "/bin/zsh")
	
	// Mock HOME to avoid editing real RC
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	code, out, _ := captureStdoutStderr(t, func() int {
		return shimEnable([]string{})
	})
	if code != 0 {
		t.Errorf("shimEnable() = %d; want 0", code)
	}
	if !strings.Contains(out, "enabled shims in") {
		t.Errorf("expected success msg, got:\n%s", out)
	}
	
	rcPath := filepath.Join(tmp, ".zshrc")
	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	if !strings.Contains(string(content), "ackchyually/shims") {
		t.Errorf("expected rc content to contain shims path, got:\n%s", string(content))
	}

	// Run again (idempotency)
	code, out, _ = captureStdoutStderr(t, func() int {
		return shimEnable([]string{})
	})
	if code != 0 {
		t.Errorf("shimEnable() = %d; want 0", code)
	}
	if !strings.Contains(out, "already enabled") {
		t.Errorf("expected already enabled msg, got:\n%s", out)
	}
}

func TestShimEnable_File(t *testing.T) {
	tmp := t.TempDir()
	rc := filepath.Join(tmp, "custom.rc")
	
	code, _, _ := captureStdoutStderr(t, func() int {
		return shimEnable([]string{"--shell=bash", "--file=" + rc})
	})
	if code != 0 {
		t.Errorf("shimEnable(file) = %d; want 0", code)
	}
	
	content, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	if !strings.Contains(string(content), "export PATH=") {
		t.Errorf("expected bash export, got:\n%s", string(content))
	}
}

func TestShimEnable_WriteError(t *testing.T) {
	tmp := t.TempDir()
	// Create directory where file should be to cause error?
	// No, writeFileAtomic will try to write to tmp file then rename.
	// Make directory read-only?
	
	rcDir := filepath.Join(tmp, "readonly")
	if err := os.Mkdir(rcDir, 0o500); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	rc := filepath.Join(rcDir, ".bashrc")
	
	code, _, errOut := captureStdoutStderr(t, func() int {
		return shimEnable([]string{"--shell=bash", "--file=" + rc})
	})
	if code != 1 {
		t.Errorf("shimEnable(readonly) = %d; want 1", code)
	}
	if !strings.Contains(errOut, "permission denied") && !strings.Contains(errOut, "access is denied") {
		// Error message varies by OS
		t.Logf("stderr: %s", errOut)
	}
}
