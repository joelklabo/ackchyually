package app

import (
	"os"
	"strings"
	"testing"

	"github.com/joelklabo/ackchyually/internal/integrations/claude"
	"github.com/joelklabo/ackchyually/internal/integrations/codex"
)

func lineWithPrefix(out, prefix string) (string, bool) {
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line, true
		}
	}
	return "", false
}

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
	codexLine, ok := lineWithPrefix(out, "codex:")
	if !ok {
		t.Fatalf("expected codex status line, got:\n%s", out)
	}
	if !strings.Contains(codexLine, "integrated=no") {
		t.Fatalf("expected integrated=no before integration, got:\n%s", codexLine)
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
	codexLine, ok = lineWithPrefix(out, "codex:")
	if !ok {
		t.Fatalf("expected codex status line, got:\n%s", out)
	}
	if !strings.Contains(codexLine, "integrated=yes") {
		t.Fatalf("expected integrated=yes after integration, got:\n%s", codexLine)
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
	codexLine, ok = lineWithPrefix(out, "codex:")
	if !ok {
		t.Fatalf("expected codex status line, got:\n%s", out)
	}
	if !strings.Contains(codexLine, "integrated=no") {
		t.Fatalf("expected integrated=no after undo, got:\n%s", codexLine)
	}
}

func TestIntegrate_Claude_DryRunDoesNotWriteSettings(t *testing.T) {
	setTempHomeAndCWD(t)
	t.Setenv("PATH", "/usr/bin")

	settings, err := claude.DefaultSettingsPath()
	if err != nil {
		t.Fatalf("claude.DefaultSettingsPath: %v", err)
	}
	if _, err := os.Stat(settings); err == nil {
		t.Fatalf("expected %s to not exist before integration", settings)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", settings, err)
	}

	code, out, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "claude", "--dry-run"})
	})
	if code != 0 {
		t.Fatalf("integrate claude --dry-run returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	if !strings.Contains(out, "claude: would update") {
		t.Fatalf("expected dry-run output, got:\n%s", out)
	}

	if _, err := os.Stat(settings); err == nil {
		t.Fatalf("expected %s to not exist after dry-run", settings)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", settings, err)
	}
}

func TestIntegrate_Claude_StatusApplyUndo(t *testing.T) {
	setTempHomeAndCWD(t)
	t.Setenv("PATH", "/usr/bin")

	code, out, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "status"})
	})
	if code != 0 {
		t.Fatalf("integrate status returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	claudeLine, ok := lineWithPrefix(out, "claude:")
	if !ok {
		t.Fatalf("expected claude status line, got:\n%s", out)
	}
	if !strings.Contains(claudeLine, "integrated=no") {
		t.Fatalf("expected integrated=no before integration, got:\n%s", claudeLine)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "claude"})
	})
	if code != 0 {
		t.Fatalf("integrate claude returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "status"})
	})
	if code != 0 {
		t.Fatalf("integrate status returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	claudeLine, ok = lineWithPrefix(out, "claude:")
	if !ok {
		t.Fatalf("expected claude status line, got:\n%s", out)
	}
	if !strings.Contains(claudeLine, "integrated=yes") {
		t.Fatalf("expected integrated=yes after integration, got:\n%s", claudeLine)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "claude", "--undo"})
	})
	if code != 0 {
		t.Fatalf("integrate claude --undo returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "status"})
	})
	if code != 0 {
		t.Fatalf("integrate status returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	claudeLine, ok = lineWithPrefix(out, "claude:")
	if !ok {
		t.Fatalf("expected claude status line, got:\n%s", out)
	}
	if !strings.Contains(claudeLine, "integrated=no") {
		t.Fatalf("expected integrated=no after undo, got:\n%s", claudeLine)
	}
}
