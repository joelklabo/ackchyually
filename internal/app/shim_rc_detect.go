package app

import (
	"os"
	"path/filepath"
	"strings"
)

func detectPersistedShims(shimDir string) (string, bool) {
	shellName := filepath.Base(strings.TrimSpace(os.Getenv("SHELL")))
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	if home == "" {
		return "", false
	}

	candidates := shellRCCandidates(shellName, home)
	for _, path := range candidates {
		content, _, err := readFileWithMode(path)
		if err != nil {
			continue
		}
		if strings.Contains(content, "# ackchyually shims") ||
			strings.Contains(content, "ackchyually/shims") ||
			strings.Contains(content, filepath.Clean(shimDir)) {
			return path, true
		}
	}
	return "", false
}

func shellRCCandidates(shellName, home string) []string {
	home = strings.TrimSpace(home)
	if home == "" {
		return nil
	}

	add := func(out []string, path string) []string {
		if path == "" {
			return out
		}
		path = expandHome(path)
		if path == "" {
			return out
		}
		for _, existing := range out {
			if existing == path {
				return out
			}
		}
		return append(out, path)
	}

	var out []string
	if path, ok := defaultRCFile(shellName); ok {
		out = add(out, path)
	}

	switch shellName {
	case "zsh":
		out = add(out, filepath.Join(home, ".zprofile"))
		out = add(out, filepath.Join(home, ".zshenv"))
	case "bash":
		out = add(out, filepath.Join(home, ".bash_profile"))
		out = add(out, filepath.Join(home, ".bash_login"))
		out = add(out, filepath.Join(home, ".profile"))
	case "fish":
		out = add(out, filepath.Join(home, ".config", "fish", "config.fish"))
		out = add(out, filepath.Join(home, ".config", "fish", "conf.d", "ackchyually.fish"))
	}
	return out
}
