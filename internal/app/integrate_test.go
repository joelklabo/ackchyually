package app

import (
	"os"
	"strings"
	"testing"

	"github.com/joelklabo/ackchyually/internal/integrations/codex"
)

func TestIntegrate_Codex_DryRunDoesNotWriteConfig(t *testing.T) {
	setTempHomeAndCWD(t)
	t.Setenv("PATH", "/usr/bin")

	cfg, err := codex.DefaultConfigPath()
	if err != nil {
		t.Fatalf("codex.DefaultConfigPath: %v", err)
	}
	if _, err := os.Stat(cfg); err == nil {
		t.Fatalf("expected %s to not exist before integration", cfg)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", cfg, err)
	}

	code, out, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "codex", "--dry-run"})
	})
	if code != 0 {
		t.Fatalf("integrate codex --dry-run returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	if !strings.Contains(out, "codex: would update") {
		t.Fatalf("expected dry-run output, got:\n%s", out)
	}

	if _, err := os.Stat(cfg); err == nil {
		t.Fatalf("expected %s to not exist after dry-run", cfg)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", cfg, err)
	}
}

func TestIntegrate_Codex_StatusApplyUndo(t *testing.T) {
	setTempHomeAndCWD(t)
	t.Setenv("PATH", "/usr/bin")

	code, out, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "status"})
	})
	if code != 0 {
		t.Fatalf("integrate status returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	if !strings.Contains(out, "codex:") {
		t.Fatalf("expected codex status line, got:\n%s", out)
	}
	if !strings.Contains(out, "integrated=no") {
		t.Fatalf("expected integrated=no before integration, got:\n%s", out)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "codex"})
	})
	if code != 0 {
		t.Fatalf("integrate codex returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "status"})
	})
	if code != 0 {
		t.Fatalf("integrate status returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	if !strings.Contains(out, "integrated=yes") {
		t.Fatalf("expected integrated=yes after integration, got:\n%s", out)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "codex", "--undo"})
	})
	if code != 0 {
		t.Fatalf("integrate codex --undo returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "status"})
	})
	if code != 0 {
		t.Fatalf("integrate status returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	if !strings.Contains(out, "integrated=no") {
		t.Fatalf("expected integrated=no after undo, got:\n%s", out)
	}
}
