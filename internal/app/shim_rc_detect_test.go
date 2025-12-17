package app

import (
	"path/filepath"
	"testing"
)

func TestShellRCCandidates(t *testing.T) {
	join := filepath.Join
	home := join("home", "user")

	t.Setenv("HOME", home)

	tests := []struct {
		shell string
		want  []string
	}{
		{"zsh", []string{join(home, ".zshrc"), join(home, ".zprofile"), join(home, ".zshenv")}},
		{"bash", []string{join(home, ".bashrc"), join(home, ".bash_profile"), join(home, ".bash_login"), join(home, ".profile")}},
		{"fish", []string{join(home, ".config", "fish", "config.fish"), join(home, ".config", "fish", "conf.d", "ackchyually.fish")}},
	}

	for _, tt := range tests {
		cands := shellRCCandidates(tt.shell, home)
		for _, want := range tt.want {
			found := false
			for _, got := range cands {
				if got == want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("shell=%q: expected candidate %q not found in %v", tt.shell, want, cands)
			}
		}
	}
}

func TestShellRCCandidates_Unknown(t *testing.T) {
	cands := shellRCCandidates("unknown", "/home")
	if len(cands) != 0 {
		t.Errorf("expected empty candidates, got %v", cands)
	}
}

func TestDetectPersistedShims_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	// Should not panic
	_, ok := detectPersistedShims("/tmp/shims")
	if ok {
		t.Error("expected false when no home")
	}
}
