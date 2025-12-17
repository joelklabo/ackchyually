package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joelklabo/ackchyually/internal/execx"
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
	tmp := t.TempDir()
	writeExec(
		t,
		tmp,
		"codex",
		"#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"codex 0.0.0\"; exit 0; fi\necho ok\n",
		"@echo off\r\nif \"%1\"==\"--version\" (\r\necho codex 0.0.0\r\nexit /b 0\r\n)\r\necho ok\r\n",
	)
	t.Setenv("PATH", tmp)

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
	tmp := t.TempDir()
	writeExec(
		t,
		tmp,
		"codex",
		"#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"codex 0.0.0\"; exit 0; fi\necho ok\n",
		"@echo off\r\nif \"%1\"==\"--version\" (\r\necho codex 0.0.0\r\nexit /b 0\r\n)\r\necho ok\r\n",
	)
	t.Setenv("PATH", tmp)

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
	if !strings.Contains(out, "fix: ackchyually integrate codex") {
		t.Fatalf("expected fix command before integration, got:\n%s", out)
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
	if strings.Contains(out, "fix: ackchyually integrate codex") {
		t.Fatalf("did not expect fix command after integration, got:\n%s", out)
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
	if !strings.Contains(out, "fix: ackchyually integrate codex") {
		t.Fatalf("expected fix command after undo, got:\n%s", out)
	}
}

func TestIntegrate_Claude_DryRunDoesNotWriteSettings(t *testing.T) {
	setTempHomeAndCWD(t)
	tmp := t.TempDir()
	writeExec(
		t,
		tmp,
		"claude",
		"#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"claude 0.0.0\"; exit 0; fi\necho ok\n",
		"@echo off\r\nif \"%1\"==\"--version\" (\r\necho claude 0.0.0\r\nexit /b 0\r\n)\r\necho ok\r\n",
	)
	t.Setenv("PATH", tmp)

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
	tmp := t.TempDir()
	writeExec(
		t,
		tmp,
		"claude",
		"#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"claude 0.0.0\"; exit 0; fi\necho ok\n",
		"@echo off\r\nif \"%1\"==\"--version\" (\r\necho claude 0.0.0\r\nexit /b 0\r\n)\r\necho ok\r\n",
	)
	t.Setenv("PATH", tmp)

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
	if !strings.Contains(out, "fix: ackchyually integrate claude") {
		t.Fatalf("expected fix command before integration, got:\n%s", out)
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
	if strings.Contains(out, "fix: ackchyually integrate claude") {
		t.Fatalf("did not expect fix command after integration, got:\n%s", out)
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
	if !strings.Contains(out, "fix: ackchyually integrate claude") {
		t.Fatalf("expected fix command after undo, got:\n%s", out)
	}
}

func TestIntegrate_Status_ScanLogs(t *testing.T) {
	setTempHomeAndCWD(t)

	tmp := t.TempDir()
	writeExec(
		t,
		tmp,
		"codex",
		"#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"codex 0.0.0\"; exit 0; fi\necho ok\n",
		"@echo off\r\nif \"%1\"==\"--version\" (\r\necho codex 0.0.0\r\nexit /b 0\r\n)\r\necho ok\r\n",
	)
	t.Setenv("PATH", tmp)

	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	shimDir := execx.ShimDir()

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatalf("mkdir codex dir: %v", err)
	}
	history := filepath.Join(codexDir, "history.jsonl")
	logContent := "DO_NOT_PRINT /usr/bin/git status\n" + filepath.Join(shimDir, "git") + "\n"
	if err := os.WriteFile(history, []byte(logContent), 0o600); err != nil { //nolint:gosec
		t.Fatalf("write history: %v", err)
	}

	code, out, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "status", "--scan-logs", "--scan-max-files", "10", "--scan-max-bytes", "1024"})
	})
	if code != 0 {
		t.Fatalf("integrate status --scan-logs returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	if strings.Contains(out, "DO_NOT_PRINT") {
		t.Fatalf("expected log contents to be suppressed, got:\n%s", out)
	}
	if !strings.Contains(out, "log scan") || !strings.Contains(out, "scan: codex:") {
		t.Fatalf("expected scan summary output, got:\n%s", out)
	}
	if !strings.Contains(out, "shim_refs=yes") || !strings.Contains(out, "abs_git_refs=yes") {
		t.Fatalf("expected shim and abs refs to be reported, got:\n%s", out)
	}
	if !strings.Contains(out, "~/.codex/history.jsonl") {
		t.Fatalf("expected home-relativized filename, got:\n%s", out)
	}
}

func TestIntegrate_Copilot_StatusApplyUndo(t *testing.T) {
	setTempHomeAndCWD(t)

	tmp := t.TempDir()
	writeExec(
		t,
		tmp,
		"copilot",
		"#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"copilot 0.0.0\"; exit 0; fi\necho ok\n",
		"@echo off\r\nif \"%1\"==\"--version\" (\r\necho copilot 0.0.0\r\nexit /b 0\r\n)\r\necho ok\r\n",
	)
	t.Setenv("PATH", tmp)

	code, out, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "status"})
	})
	if code != 0 {
		t.Fatalf("integrate status returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	copilotLine, ok := lineWithPrefix(out, "copilot:")
	if !ok {
		t.Fatalf("expected copilot status line, got:\n%s", out)
	}
	if !strings.Contains(copilotLine, "integrated=no") {
		t.Fatalf("expected integrated=no before integration, got:\n%s", copilotLine)
	}
	if !strings.Contains(out, "fix: ackchyually integrate copilot") {
		t.Fatalf("expected fix command before integration, got:\n%s", out)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "copilot"})
	})
	if code != 0 {
		t.Fatalf("integrate copilot returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "status"})
	})
	if code != 0 {
		t.Fatalf("integrate status returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	copilotLine, ok = lineWithPrefix(out, "copilot:")
	if !ok {
		t.Fatalf("expected copilot status line, got:\n%s", out)
	}
	if !strings.Contains(copilotLine, "integrated=yes") {
		t.Fatalf("expected integrated=yes after integration, got:\n%s", copilotLine)
	}
	if strings.Contains(out, "fix: ackchyually integrate copilot") {
		t.Fatalf("did not expect fix command after integration, got:\n%s", out)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "copilot", "--undo"})
	})
	if code != 0 {
		t.Fatalf("integrate copilot --undo returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}

	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"integrate", "status"})
	})
	if code != 0 {
		t.Fatalf("integrate status returned %d, want 0\nSTDOUT:\n%s\nSTDERR:\n%s", code, out, errOut)
	}
	copilotLine, ok = lineWithPrefix(out, "copilot:")
	if !ok {
		t.Fatalf("expected copilot status line, got:\n%s", out)
	}
	if !strings.Contains(copilotLine, "integrated=no") {
		t.Fatalf("expected integrated=no after undo, got:\n%s", copilotLine)
	}
	if !strings.Contains(out, "fix: ackchyually integrate copilot") {
		t.Fatalf("expected fix command after undo, got:\n%s", out)
	}
}
