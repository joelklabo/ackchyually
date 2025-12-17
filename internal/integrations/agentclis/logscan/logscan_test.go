package logscan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScanCodex_TailReadAndFindings(t *testing.T) {
	home := t.TempDir()
	shimDir := filepath.Join(home, ".local", "share", "ackchyually", "shims")

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	history := filepath.Join(codexDir, "history.jsonl")
	content := strings.Repeat("x", 4096) + "\n" + "/usr/bin/git status\n" + filepath.Join(shimDir, "git") + "\n"
	if err := os.WriteFile(history, []byte(content), 0o600); err != nil { //nolint:gosec
		t.Fatalf("write history: %v", err)
	}

	sum := ScanCodex(home, shimDir, Options{MaxFiles: 10, MaxBytes: 256})
	if sum.FilesScanned != 1 {
		t.Fatalf("FilesScanned=%d want 1", sum.FilesScanned)
	}
	if sum.FilesWithShim != 1 {
		t.Fatalf("FilesWithShim=%d want 1", sum.FilesWithShim)
	}
	if sum.FilesWithAbs != 1 {
		t.Fatalf("FilesWithAbs=%d want 1", sum.FilesWithAbs)
	}
	if sum.BytesRead <= 0 || sum.BytesRead > 256 {
		t.Fatalf("BytesRead=%d want (0,256]", sum.BytesRead)
	}
	if len(sum.FileNames) != 1 || sum.FileNames[0] != "~/.codex/history.jsonl" {
		t.Fatalf("FileNames=%v want [~/.codex/history.jsonl]", sum.FileNames)
	}
}

func TestScanCodex_SessionsMostRecentFirst(t *testing.T) {
	home := t.TempDir()
	shimDir := filepath.Join(home, ".local", "share", "ackchyually", "shims")
	sessionsDir := filepath.Join(home, ".codex", "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	old := filepath.Join(sessionsDir, "old.jsonl")
	if err := os.WriteFile(old, []byte("/usr/bin/git status\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("write old: %v", err)
	}
	newer := filepath.Join(sessionsDir, "new.jsonl")
	if err := os.WriteFile(newer, []byte(filepath.Join(shimDir, "git")+"\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("write newer: %v", err)
	}

	// Ensure deterministic ordering by mtime.
	if err := os.Chtimes(old, time.Unix(1, 0), time.Unix(1, 0)); err != nil {
		t.Fatalf("chtimes old: %v", err)
	}
	if err := os.Chtimes(newer, time.Unix(2, 0), time.Unix(2, 0)); err != nil {
		t.Fatalf("chtimes newer: %v", err)
	}

	sum := ScanCodex(home, shimDir, Options{MaxFiles: 1, MaxBytes: 1024})
	// MaxFiles=1 should include only the most recent file from sessions (no history present).
	if sum.FilesScanned != 1 {
		t.Fatalf("FilesScanned=%d want 1", sum.FilesScanned)
	}
	if sum.FilesWithShim != 1 {
		t.Fatalf("FilesWithShim=%d want 1", sum.FilesWithShim)
	}
	if sum.FilesWithAbs != 0 {
		t.Fatalf("FilesWithAbs=%d want 0", sum.FilesWithAbs)
	}
	if len(sum.FileNames) != 1 || sum.FileNames[0] != "~/.codex/sessions/new.jsonl" {
		t.Fatalf("FileNames=%v want [~/.codex/sessions/new.jsonl]", sum.FileNames)
	}
}

func TestScanCopilot_LogsDir(t *testing.T) {
	home := t.TempDir()
	shimDir := filepath.Join(home, ".local", "share", "ackchyually", "shims")

	logsDir := filepath.Join(home, ".copilot", "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logFile := filepath.Join(logsDir, "copilot.log")
	if err := os.WriteFile(logFile, []byte("/usr/bin/git status\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("write log: %v", err)
	}

	sum := ScanCopilot(home, shimDir, Options{MaxFiles: 10, MaxBytes: 1024})
	if sum.FilesScanned != 1 {
		t.Fatalf("FilesScanned=%d want 1", sum.FilesScanned)
	}
	if sum.FilesWithAbs != 1 {
		t.Fatalf("FilesWithAbs=%d want 1", sum.FilesWithAbs)
	}
	if sum.FilesWithShim != 0 {
		t.Fatalf("FilesWithShim=%d want 0", sum.FilesWithShim)
	}
}
