package app

import (
	"testing"
)

func TestRunCLI_ShimDispatch(t *testing.T) {
	setTempHomeAndCWD(t)

	// We just want to ensure dispatch works, so we expect exit codes consistent with empty/invalid args
	// or successful execution if no args needed.

	tests := []struct {
		args []string
		code int
	}{
		{[]string{"shim", "install"}, 2},   // missing args
		{[]string{"shim", "uninstall"}, 2}, // missing args
		{[]string{"shim", "list"}, 0},      // ok
		{[]string{"shim", "doctor"}, 0},    // ok
		{[]string{"shim", "enable"}, 0},    // ok (idempotent/safe)
		{[]string{"version"}, 0},           // ok
	}

	for _, tt := range tests {
		name := tt.args[0]
		if len(tt.args) > 1 {
			name = tt.args[1]
		}
		t.Run(name, func(t *testing.T) {
			code, _, _ := captureStdoutStderr(t, func() int {
				return RunCLI(tt.args)
			})
			if code != tt.code {
				t.Errorf("RunCLI(%v) code = %d, want %d", tt.args, code, tt.code)
			}
		})
	}
}
