package app

import (
	"strings"
	"testing"
	"time"
)

func TestRunCLI_Best_FlagOrdering(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	now := time.Now()
	seedInvocation(t, ctxKey, "git", []string{"git", "status"}, now.Add(-time.Minute), 0)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "standard order",
			args:    []string{"best", "--tool", "git", "status"},
			wantErr: false,
		},
		{
			name:    "flag after positional",
			args:    []string{"best", "status", "--tool", "git"},
			wantErr: false,
		},
		{
			name:    "flag interleaved (not applicable for best as it takes one arg string, but let's try)",
			args:    []string{"best", "--tool", "git", "status"}, // same as standard
			wantErr: false,
		},
		{
			name:    "missing tool flag",
			args:    []string{"best", "status"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errOut := captureStdoutStderr(t, func() int {
				return RunCLI(tt.args)
			})

			if tt.wantErr {
				if code != 2 {
					t.Errorf("RunCLI(%v) code = %d, want 2", tt.args, code)
				}
				if !strings.Contains(errOut, "best: --tool is required") {
					t.Errorf("RunCLI(%v) stderr missing error message, got: %s", tt.args, errOut)
				}
			} else {
				if code != 0 {
					t.Errorf("RunCLI(%v) code = %d, want 0. Stderr: %s", tt.args, code, errOut)
				}
				if !strings.Contains(out, "git status") {
					t.Errorf("RunCLI(%v) output missing 'git status', got: %s", tt.args, out)
				}
			}
		})
	}
}

func TestRunCLI_Export_FlagOrdering(t *testing.T) {
	setTempHomeAndCWD(t)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "standard order",
			args:    []string{"export", "--format", "json"},
			wantErr: false,
		},
		{
			name:    "multiple flags standard",
			args:    []string{"export", "--format", "json", "--tool", "git"},
			wantErr: false,
		},
		{
			name:    "multiple flags swapped",
			args:    []string{"export", "--tool", "git", "--format", "json"},
			wantErr: false,
		},
		// Export doesn't really take positional args, so "flag after positional"
		// might mean "export <ignored> --format json".
		// If we support flags anywhere, we should probably ignore non-flag args if the command doesn't use them,
		// OR we should ensure flags are parsed even if there are junk args.
		// But let's stick to what's useful.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, errOut := captureStdoutStderr(t, func() int {
				return RunCLI(tt.args)
			})

			if tt.wantErr {
				if code != 2 {
					t.Errorf("RunCLI(%v) code = %d, want 2", tt.args, code)
				}
			} else {
				if code != 0 {
					t.Errorf("RunCLI(%v) code = %d, want 0. Stderr: %s", tt.args, code, errOut)
				}
			}
		})
	}
}
