package helpcount

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIsHelpInvocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		argv []string
		want bool
	}{
		{
			name: "subcommand_help_counts",
			argv: []string{"git", "help", "log"},
			want: true,
		},
		{
			name: "flag_help_counts",
			argv: []string{"git", "log", "-h"},
			want: true,
		},
		{
			name: "flag_double_dash_help_counts",
			argv: []string{"git", "log", "--help"},
			want: true,
		},
		{
			name: "help_word_in_arg_does_not_count",
			argv: []string{"git", "commit", "-m", "help"},
			want: false,
		},
		{
			name: "help_word_not_first_subcommand_does_not_count",
			argv: []string{"git", "commit", "help"},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isHelpInvocation(tt.argv); got != tt.want {
				t.Fatalf("isHelpInvocation(%q)=%v want %v", tt.argv, got, tt.want)
			}
		})
	}
}

func TestMatchesHelpInvocation(t *testing.T) {
	t.Parallel()

	tool := "git"
	helpArgs := []string{"log", "-h"}

	if !matchesHelpInvocation(tool, helpArgs, []string{"git", "log", "-h"}) {
		t.Fatalf("expected exact help argv to match")
	}
	if matchesHelpInvocation(tool, helpArgs, []string{"git", "help", "log"}) {
		t.Fatalf("did not expect alternative help form to match when helpArgs are specified")
	}
	if !matchesHelpInvocation(tool, nil, []string{"git", "help", "log"}) {
		t.Fatalf("expected any-help matcher to match git help")
	}
}

func TestFilterScenarios(t *testing.T) {
	t.Parallel()

	s := []Scenario{
		{Name: "Alpha", Description: "first", Tool: "git"},
		{Name: "beta", Description: "second", Tool: "curl"},
	}

	all := filterScenarios(s, "")
	if len(all) != len(s) {
		t.Fatalf("expected all scenarios when filter empty, got %d", len(all))
	}

	got := filterScenarios(s, "GIT")
	if len(got) != 1 || got[0].Tool != "git" {
		t.Fatalf("expected git scenario, got %#v", got)
	}

	got = filterScenarios(s, "second")
	if len(got) != 1 || got[0].Description != "second" {
		t.Fatalf("expected description match, got %#v", got)
	}
}

func TestExtractSuggestion(t *testing.T) {
	t.Parallel()

	cmd, ok := ExtractSuggestion("ackchyually: suggestion\n  git status\n")
	if !ok || cmd != "git status" {
		t.Fatalf("expected suggestion, got %q ok=%v", cmd, ok)
	}

	cmd, ok = ExtractSuggestion("ackchyually: suggestion\n\n(no suggestion)\n")
	if !ok || cmd != "(no suggestion)" {
		t.Fatalf("expected placeholder suggestion, got %q ok=%v", cmd, ok)
	}

	cmd, ok = ExtractSuggestion("no suggestion here")
	if ok || cmd != "" {
		t.Fatalf("expected no suggestion, got %q ok=%v", cmd, ok)
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	if exitCode(nil) != 0 {
		t.Fatalf("nil error should yield exit 0")
	}

	if runtime.GOOS == "windows" {
		t.Skip("sh not available on windows in CI")
	}
	err := exec.Command("sh", "-c", "exit 9").Run()
	if got := exitCode(err); got != 9 {
		t.Fatalf("exitCode=%d want 9", got)
	}
}

func TestEnvHelpers(t *testing.T) {
	t.Parallel()

	env := []string{"PATH=/usr/bin", "HOME=/home/user"}
	env = upsertEnv(env, "HOME", "/tmp/home")
	env = upsertEnv(env, "NEWVAR", "x")
	if got := getEnv(env, "HOME"); got != "/tmp/home" {
		t.Fatalf("HOME=%q", got)
	}
	if got := getEnv(env, "NEWVAR"); got != "x" {
		t.Fatalf("NEWVAR=%q", got)
	}
	env = deleteEnv(env, "HOME")
	if got := getEnv(env, "HOME"); got != "" {
		t.Fatalf("expected HOME removed, got %q", got)
	}
}

func TestStripAckchyuallyShimDirs(t *testing.T) {
	t.Parallel()

	shim := filepath.Join(os.TempDir(), ".local", "share", "ackchyually", "shims")
	path := strings.Join([]string{"/usr/bin", shim, "/bin"}, string(os.PathListSeparator))
	got := stripAckchyuallyShimDirs(path)
	if strings.Contains(got, "ackchyually") {
		t.Fatalf("expected shim dir removed, got %q", got)
	}
	if !strings.Contains(got, "/usr/bin") || !strings.Contains(got, "/bin") {
		t.Fatalf("expected other entries preserved, got %q", got)
	}
}

func TestIsAckchyuallyShimDir(t *testing.T) {
	t.Parallel()

	shim := filepath.Join("/home/user", ".local", "share", "ackchyually", "shims")
	if !isAckchyuallyShimDir(shim) {
		t.Fatalf("expected %q to be shim dir", shim)
	}
	if isAckchyuallyShimDir("/home/user/.local/share/ackchyually") {
		t.Fatalf("unexpected shim dir match")
	}
}

func TestLookPathEnv(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bin := filepath.Join(dir, "tool")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write file: %v", err)
	}

	path, err := lookPathEnv("tool", []string{"PATH=" + dir})
	if err != nil {
		t.Fatalf("lookPathEnv error: %v", err)
	}
	if path != bin {
		t.Fatalf("got %q want %q", path, bin)
	}

	if _, err := lookPathEnv("missing", []string{"PATH=" + dir}); err == nil {
		t.Fatalf("expected error for missing executable")
	}
}

func TestSlicesEqual(t *testing.T) {
	t.Parallel()

	if !slicesEqual([]string{"a", "b"}, []string{"a", "b"}) {
		t.Fatalf("expected equal slices")
	}
	if slicesEqual([]string{"a"}, []string{"a", "b"}) {
		t.Fatalf("expected unequal slices (len)")
	}
	if slicesEqual([]string{"a", "b"}, []string{"a", "c"}) {
		t.Fatalf("expected unequal slices (content)")
	}
}
