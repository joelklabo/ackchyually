package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultRCFile(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("skipping defaultRCFile test: no user home dir")
	}

	tests := []struct {
		shell string
		want  string
		ok    bool
	}{
		{"zsh", filepath.Join(home, ".zshrc"), true},
		{"bash", filepath.Join(home, ".bashrc"), true},
		{"fish", filepath.Join(home, ".config", "fish", "config.fish"), true},
		{"unknown", "", false},
	}

	for _, tt := range tests {
		got, ok := defaultRCFile(tt.shell)
		if ok != tt.ok {
			t.Errorf("defaultRCFile(%q) ok = %v, want %v", tt.shell, ok, tt.ok)
		}
		if got != tt.want {
			t.Errorf("defaultRCFile(%q) = %q, want %q", tt.shell, got, tt.want)
		}
	}
}

func TestEnableSnippet(t *testing.T) {
	dir := "/tmp/shims"

	tests := []struct {
		shell string
		want  string
	}{
		{"zsh", "export PATH=\"/tmp/shims:$PATH\"\n"},
		{"bash", "export PATH=\"/tmp/shims:$PATH\"\n"},
		{"fish", "set -gx PATH \"/tmp/shims\" $PATH\n"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		got := enableSnippet(tt.shell, dir)
		if tt.want == "" {
			if got != "" {
				t.Errorf("enableSnippet(%q) = %q, want empty", tt.shell, got)
			}
		} else {
			if !strings.Contains(got, tt.want) {
				t.Errorf("enableSnippet(%q) = %q, want substring %q", tt.shell, got, tt.want)
			}
			if !strings.Contains(got, "# ackchyually shims") {
				t.Errorf("enableSnippet(%q) missing header", tt.shell)
			}
		}
	}
}
