package app

import (
	"strings"
	"testing"
	"time"
)

func TestRunCLI_SuggestCommand(t *testing.T) {
	setTempHomeAndCWD(t)

	tests := []struct {
		args     []string
		wantOut  string
		wantCode int
	}{
		{
			args:     []string{"shmi"},
			wantOut:  "try: ackchyually shim",
			wantCode: 2,
		},
		{
			args:     []string{"bst"},
			wantOut:  "try: ackchyually best",
			wantCode: 2,
		},
		{
			args:     []string{"xyz"}, // no close match
			wantOut:  "unknown command: xyz",
			wantCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.args[0], func(t *testing.T) {
			code, _, errOut := captureStdoutStderr(t, func() int {
				return RunCLI(tt.args)
			})
			if code != tt.wantCode {
				t.Errorf("RunCLI(%v) code = %d, want %d", tt.args, code, tt.wantCode)
			}
			if !strings.Contains(errOut, tt.wantOut) {
				t.Errorf("RunCLI(%v) stderr missing %q, got:\n%s", tt.args, tt.wantOut, errOut)
			}
		})
	}
}

func TestRunCLI_SuggestSubcommand(t *testing.T) {
	setTempHomeAndCWD(t)

	tests := []struct {
		args     []string
		wantOut  string
		wantCode int
	}{
		{
			args:     []string{"shim", "instal"}, // typo
			wantOut:  "try: ackchyually shim install",
			wantCode: 2,
		},
		{
			args:     []string{"tag", "ad"},
			wantOut:  "try: ackchyually tag add",
			wantCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			code, _, errOut := captureStdoutStderr(t, func() int {
				return RunCLI(tt.args)
			})
			if code != tt.wantCode {
				t.Errorf("RunCLI(%v) code = %d, want %d", tt.args, code, tt.wantCode)
			}
			if !strings.Contains(errOut, tt.wantOut) {
				t.Errorf("RunCLI(%v) stderr missing %q, got:\n%s", tt.args, tt.wantOut, errOut)
			}
		})
	}
}

func TestRunCLI_SuggestLastSuccessful(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	now := time.Now()

	// Seed successful ackchyually invocation
	seedInvocation(t, ctxKey, "ackchyually", []string{"ackchyually", "shim", "list"}, now, 0)

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"xyz"})
	})
	if code != 2 {
		t.Fatalf("RunCLI code = %d", code)
	}
	if !strings.Contains(errOut, "last success here: ackchyually shim list") {
		t.Errorf("stderr missing last successful, got:\n%s", errOut)
	}
}

func TestRunCLI_SuggestSubcommand_NoMatch(t *testing.T) {
	setTempHomeAndCWD(t)

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"shim", "xyz"})
	})
	if code != 2 {
		t.Fatalf("RunCLI code = %d", code)
	}
	if !strings.Contains(errOut, "unknown shim subcommand: xyz") {
		t.Errorf("stderr missing unknown subcommand, got:\n%s", errOut)
	}
	// Should list available
	if !strings.Contains(errOut, "available: doctor, enable, install, list, uninstall") {
		t.Errorf("stderr missing available subcommands, got:\n%s", errOut)
	}
}
