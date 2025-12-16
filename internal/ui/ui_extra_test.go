package ui

import (
	"os"
	"testing"
)

func TestUI_AllMethods(t *testing.T) {
	// Force lipgloss to render colors even if not TTY
	t.Setenv("CLICOLOR_FORCE", "1")

	// Mock enabled UI
	u := New(os.Stdout)
	u.enabled = true

	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"Bold", u.Bold},
		{"Dim", u.Dim},
		{"OK", u.OK},
		{"Warn", u.Warn},
		{"Error", u.Error},
		{"Label", u.Label},
	}

	for _, tt := range tests {
		got := tt.fn("text")
		if got == "text" {
			t.Errorf("%s() returned plain text when enabled", tt.name)
		}
	}
}

func TestUI_Disabled(t *testing.T) {
	u := New(os.Stdout)
	u.enabled = false

	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"Bold", u.Bold},
		{"Dim", u.Dim},
		{"OK", u.OK},
		{"Warn", u.Warn},
		{"Error", u.Error},
		{"Label", u.Label},
	}

	for _, tt := range tests {
		got := tt.fn("text")
		if got != "text" {
			t.Errorf("%s() returned styled text when disabled", tt.name)
		}
	}
}

func TestShouldStyle(t *testing.T) {
	if shouldStyle(nil) {
		t.Error("shouldStyle(nil) = true, want false")
	}

	// We can't easily mock IsTerminal on real FDs without PTYs,
	// but we can test NO_COLOR and TERM=dumb if we could mock the Fd check.
	// However, shouldStyle checks Fd first.
	// We can trust standard library checks mostly.
}
