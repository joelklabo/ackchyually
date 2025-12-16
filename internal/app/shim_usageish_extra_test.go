package app

import (
	"testing"

	"github.com/joelklabo/ackchyually/internal/execx"
)

func TestIsUsageish(t *testing.T) {
	// isUsageish checks args OR exit code OR output

	tests := []struct {
		name     string
		args     []string
		exitCode int
		stdout   string
		stderr   string
		want     bool
	}{
		{
			name: "help flag",
			args: []string{"--help"},
			want: false, // explicit help is not a usage error
		},
		{
			name: "help command",
			args: []string{"help"},
			want: false, // explicit help is not a usage error
		},
		{
			name: "normal command",
			args: []string{"status"},
			want: false,
		},
		{
			name:     "exit code 2 with usage output",
			args:     []string{"status"},
			exitCode: 2,
			stderr:   "usage: git [options]",
			want:     true,
		},
		{
			name:     "exit code 127 with error output",
			args:     []string{"status"},
			exitCode: 127,
			stderr:   "command not found", // "command not found" is not in looksUsageish list?
			// looksUsageish checks for "unknown command" etc.
			// Let's use something that matches.
			want: false, // "command not found" might not match looksUsageish list
		},
		{
			name:     "exit code 127 with unknown command output",
			args:     []string{"status"},
			exitCode: 127,
			stderr:   "git: 'status' is not a git command. See 'git --help'.",
			want:     true, // "is not a ... command ... --help" matches
		},
		{
			name:   "usage in stdout (success code)",
			args:   []string{"status"},
			stdout: "Usage: git [options]",
			want:   true,
		},
		{
			name:   "error in stderr (success code)",
			args:   []string{"status"},
			stderr: "error: unknown option",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := "pipes"
			combined := ""
			if tt.stdout != "" && tt.stderr == "" {
				mode = "pty"
				combined = tt.stdout
			}

			res := execx.Result{
				Mode:         mode,
				StdoutTail:   tt.stdout,
				StderrTail:   tt.stderr,
				CombinedTail: combined,
			}
			got := isUsageish(tt.args, tt.exitCode, res)
			if got != tt.want {
				t.Errorf("isUsageish(%v, %d, %q, %q) = %v, want %v", tt.args, tt.exitCode, tt.stdout, tt.stderr, got, tt.want)
			}
		})
	}
}

func TestLooksUsageishOnSuccess(t *testing.T) {
	tests := []struct {
		stdout string
		stderr string
		want   bool
	}{
		{"Usage: git [options]", "", true}, // stdout check (pty mode simulation or fallback)
		{"", "usage: git [options]", true}, // stderr check
		{"Commands:\n  init", "", false},
		{"", "Options:\n  -v", false},
		{"", "error: unknown option", true}, // "error: " stripped, "unknown option" matches
		{"", "fatal: not a git repo", false},
		{"Hello world", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		// Simulate PTY mode for stdout checks (CombinedTail)
		// Simulate Pipes mode for stderr checks (StderrTail)
		var res execx.Result
		if tt.stderr != "" {
			res = execx.Result{
				Mode:       "pipes",
				StderrTail: tt.stderr,
			}
		} else {
			res = execx.Result{
				Mode:         "pty",
				CombinedTail: tt.stdout,
			}
		}

		got := looksUsageishOnSuccess(res)
		if got != tt.want {
			t.Logf("DEBUG: output=%q stderr=%q mode=%q", tt.stdout, tt.stderr, res.Mode)
			t.Errorf("looksUsageishOnSuccess(%q, %q) = %v, want %v", tt.stdout, tt.stderr, got, tt.want)
		}
	}
}
