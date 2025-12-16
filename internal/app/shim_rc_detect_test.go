package app

import (
	"testing"
)

func TestShellRCCandidates(t *testing.T) {
	home := "/home/user"
	t.Setenv("HOME", home)

	tests := []struct {
		shell string
		want  []string
	}{
		{"zsh", []string{"/home/user/.zshrc", "/home/user/.zprofile", "/home/user/.zshenv"}},
		{"bash", []string{"/home/user/.bashrc", "/home/user/.bash_profile", "/home/user/.bash_login", "/home/user/.profile"}},
		{"fish", []string{"/home/user/.config/fish/config.fish", "/home/user/.config/fish/conf.d/ackchyually.fish"}},
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
