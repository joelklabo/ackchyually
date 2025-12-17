package app

import (
	"strings"
	"testing"
)

func TestRunCLI_ShimDispatch(t *testing.T) {
	setTempHomeAndCWD(t)

	// We just want to ensure dispatch works, so we expect exit codes consistent with empty/invalid args
	// or successful execution if no args needed.

	tests := []struct {
		args []string
		code int
		want string
	}{
		{[]string{"shim", "install"}, 2, ""},   // missing args
		{[]string{"shim", "uninstall"}, 2, ""}, // missing args
		{[]string{"shim", "list"}, 0, ""},      // ok
		{[]string{"shim", "doctor"}, 0, ""},    // ok
		{[]string{"shim", "enable"}, 0, ""},    // ok (idempotent/safe)
		{[]string{"integrate", "status"}, 0, "codex:"},
		{[]string{"integrate", "codex", "--dry-run"}, 0, "codex:"},
		{[]string{"integrate", "claude", "--dry-run"}, 0, "claude:"},
		{[]string{"integrate", "verify"}, 2, "not implemented"},
		{[]string{"version"}, 0, ""}, // ok
	}

	for _, tt := range tests {
		name := tt.args[0]
		if len(tt.args) > 1 {
			name = tt.args[1]
		}
		t.Run(name, func(t *testing.T) {
			code, out, errOut := captureStdoutStderr(t, func() int {
				return RunCLI(tt.args)
			})
			if code != tt.code {
				t.Errorf("RunCLI(%v) code = %d, want %d", tt.args, code, tt.code)
			}
			if tt.want != "" && !strings.Contains(out+errOut, tt.want) {
				t.Errorf("RunCLI(%v) output missing %q\nSTDOUT:\n%s\nSTDERR:\n%s", tt.args, tt.want, out, errOut)
			}
		})
	}
}
