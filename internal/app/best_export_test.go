package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joelklabo/ackchyually/internal/store"
)

func TestExport_JSON(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)

	// Seed a tag
	err := store.WithDB(func(db *store.DB) error {
		return db.UpsertTag(store.Tag{
			ContextKey: ctxKey,
			Tag:        "mytag",
			Tool:       "git",
			ArgvJSON:   store.MustJSON([]string{"git", "status"}),
		})
	})
	if err != nil {
		t.Fatalf("seed tag: %v", err)
	}

	code, out, _ := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"export", "--format", "json"})
	})
	if code != 0 {
		t.Fatalf("export json code = %d, want 0", code)
	}
	if !strings.Contains(out, `"tool": "git"`) {
		t.Errorf("json output missing tool field, got:\n%s", out)
	}
	if !strings.Contains(out, `"argv": [`) {
		t.Errorf("json output missing argv field, got:\n%s", out)
	}
}

func TestExport_RepoRoot(t *testing.T) {
	// Create a fake git repo
	tmp := t.TempDir()
	tmp, err := filepath.EvalSymlinks(tmp)
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	repo := filepath.Join(tmp, "repo")
	mkdirAll(t, repo)

	// Initialize git repo
	cmd := exec.CommandContext(context.Background(), "git", "init")
	cmd.Dir = repo
	if err := cmd.Run(); err != nil {
		t.Skip("git init failed, skipping repo root test")
	}

	cwd := filepath.Join(repo, "subdir")
	mkdirAll(t, cwd)

	t.Setenv("HOME", tmp)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("chdir back: %v", err)
		}
	}()
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// We need to seed a tag that matches the current context.
	// Since we are in a git repo, contextkey.Detect() should return "git:/path/to/repo".
	// We can't easily predict the exact path (symlinks etc), so we rely on Detect() to match itself.

	// But we need to seed the DB with that context key.
	// So we call Detect() first.
	// But Detect() is internal/contextkey.
	// We can't call it from here easily?
	// Wait, we are in package app.
	// contextkey is in internal/contextkey.
	// We can import it.

	// But we need to ensure the DB is initialized in the right place.
	// setTempHomeAndCWD does that. But we manually set HOME here.
	// So we need to ensure DB dir exists.
	mkdirAll(t, filepath.Join(tmp, ".local", "share", "ackchyually"))

	// We need to get the context key.
	// We can use a helper or just assume it works if we use the same logic.
	// But better: use `tag add` CLI command!
	// It uses Detect() internally.

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"tag", "add", "mytag", "--", "git", "add", filepath.Join(repo, "file.txt")})
	})
	if code != 0 {
		t.Fatalf("tag add failed: %s", errOut)
	}

	// Now run export
	code, out, _ := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"export", "--format", "json"})
	})
	if code != 0 {
		t.Fatalf("export code = %d", code)
	}

	// We expect the path to be relativized to repo root.
	// repo/file.txt -> ./file.txt
	// Or if we are in subdir, and repo root is parent.
	// exportNormalizeArgv uses repoRoot.
	// If repoRoot is `repo`.
	// And arg is `repo/file.txt`.
	// It becomes `./file.txt`.

	if !strings.Contains(out, `"./file.txt"`) {
		t.Errorf("json output missing relative path, got:\n%s", out)
	}
}

func TestExport_MD(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	now := time.Now()
	seedInvocation(t, ctxKey, "git", []string{"git", "status"}, now, 0)

	// Seed a tag too
	err := store.WithDB(func(db *store.DB) error {
		return db.UpsertTag(store.Tag{
			ContextKey: ctxKey,
			Tag:        "mytag",
			Tool:       "git",
			ArgvJSON:   store.MustJSON([]string{"git", "status"}),
		})
	})
	if err != nil {
		t.Fatalf("seed tag: %v", err)
	}

	code, out, _ := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"export", "--format", "md", "--tool", "git"})
	})
	if code != 0 {
		t.Fatalf("export md code = %d, want 0", code)
	}
	if !strings.Contains(out, "## ackchyually export") {
		t.Errorf("md output missing header, got:\n%s", out)
	}
	if !strings.Contains(out, "- **mytag**: `git status`") {
		t.Errorf("md output missing tag, got:\n%s", out)
	}
	if !strings.Contains(out, "### Recent successful commands") {
		t.Errorf("md output missing recent commands header, got:\n%s", out)
	}
	if !strings.Contains(out, "- `git status`") {
		t.Errorf("md output missing recent command, got:\n%s", out)
	}
}

func TestExportDecodeArgv_Error(t *testing.T) {
	// exportDecodeArgv returns nil on error
	if got := exportDecodeArgv("invalid json"); got != nil {
		t.Errorf("exportDecodeArgv(\"invalid json\") = %v, want nil", got)
	}
}

func TestExportRepoRoot_EdgeCases(t *testing.T) {
	tests := []struct {
		ctxKey string
		want   string
	}{
		{"", ""},
		{"git:/path/to/repo", "/path/to/repo"},
		{"git:", "."}, // empty path -> . after clean? No, filepath.Clean("") is "."
		{"other:/path", ""},
		{"git", ""}, // missing colon
	}
	for _, tt := range tests {
		got := exportRepoRoot(tt.ctxKey)
		// filepath.Clean might behave differently on windows, but for simple paths it should be fine.
		// For "git:", path is empty string. filepath.Clean("") is ".".
		want := tt.want
		if want != "" && want != "." {
			want = filepath.Clean(want)
		}
		if got != want {
			t.Errorf("exportRepoRoot(%q) = %q, want %q", tt.ctxKey, got, want)
		}
	}
}

func TestExportNormalizeArgv_Empty(t *testing.T) {
	if got := exportNormalizeArgv(nil, "", ""); got != nil {
		t.Errorf("exportNormalizeArgv(nil) = %v, want nil", got)
	}
	if got := exportNormalizeArgv([]string{}, "", ""); len(got) != 0 {
		t.Errorf("exportNormalizeArgv([]) = %v, want []", got)
	}
}

func TestExportNormalizeContextKey_EdgeCases(t *testing.T) {
	tests := []struct {
		ctxKey string
		home   string
		want   string
	}{
		{"", "", ""},
		{"git:/path/to/repo", "", "git:/path/to/repo"},
		{"git:/home/user/repo", "/home/user", "git:~/repo"},
		{"other:/path", "", "other:/path"},
		{"git", "", "git"}, // missing colon
	}
	for _, tt := range tests {
		got := exportNormalizeContextKey(tt.ctxKey, tt.home)
		if got != tt.want {
			t.Errorf("exportNormalizeContextKey(%q, %q) = %q, want %q", tt.ctxKey, tt.home, got, tt.want)
		}
	}
}
