package app

import (
	"strings"
	"testing"
	"time"
)

func TestRunCLI_Best_EdgeCases(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	now := time.Now()
	// Seed some data
	seedInvocation(t, ctxKey, "git", []string{"git", "status"}, now.Add(-time.Minute), 0)
	seedInvocation(t, ctxKey, "git", []string{"git", "status", "-s"}, now.Add(-2*time.Minute), 0)
	seedInvocation(t, ctxKey, "git", []string{"git", "commit", "--amend"}, now.Add(-3*time.Minute), 0)
	seedInvocation(t, ctxKey, "go", []string{"go", "build"}, now.Add(-time.Minute), 0)
	seedInvocation(t, ctxKey, "--help", []string{"--help", "me"}, now.Add(-time.Minute), 0)

	tests := []struct {
		name           string
		args           []string
		wantCode       int
		wantOutContain string
		wantErrContain string
	}{
		{
			name:           "Flag Value Masquerading: --tool --help",
			args:           []string{"best", "--tool", "--help", "me"},
			wantCode:       0,
			wantOutContain: "--help me",
		},
		{
			name:           "Repeated Flags: --tool git --tool go",
			args:           []string{"best", "--tool", "git", "--tool", "go", "build"},
			wantCode:       0,
			wantOutContain: "go build",
		},
		{
			name:           "Multiple Positional Args: status -s (without quotes)",
			args:           []string{"best", "--tool", "git", "status", "-s"},
			wantCode:       0,
			wantOutContain: "git status -s",
		},
		{
			name:           "Unknown Flag as Positional: --unknown",
			args:           []string{"best", "--tool", "git", "commit", "--unknown"},
			wantCode:       0,
			wantOutContain: "git commit --amend", // Should match 'commit' and maybe '--unknown' if it was part of query?
			// Actually if query is "commit --unknown", it might not match "commit --amend" well if tokens are strict.
			// But "commit" should match.
			// The key is that it shouldn't fail with "flag provided but not defined".
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errOut := captureStdoutStderr(t, func() int {
				return RunCLI(tt.args)
			})

			if code != tt.wantCode {
				t.Errorf("RunCLI code = %d, want %d. Stderr: %s", code, tt.wantCode, errOut)
			}
			if tt.wantOutContain != "" && !strings.Contains(out, tt.wantOutContain) {
				t.Errorf("RunCLI output missing %q, got:\n%s", tt.wantOutContain, out)
			}
			if tt.wantErrContain != "" && !strings.Contains(errOut, tt.wantErrContain) {
				t.Errorf("RunCLI stderr missing %q, got:\n%s", tt.wantErrContain, errOut)
			}
		})
	}
}
