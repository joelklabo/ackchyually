package execx

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func ShimDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "ackchyually", "shims")
}

func WhichSkippingShims(tool string) (string, error) {
	pathEnv := os.Getenv("PATH")
	parts := strings.Split(pathEnv, string(os.PathListSeparator))

	shim := filepath.Clean(ShimDir())
	for _, dir := range parts {
		if dir == "" {
			dir = "."
		}
		dir = filepath.Clean(dir)
		if dir == shim {
			continue
		}
		candidate := filepath.Join(dir, tool)
		if isExecutableFile(candidate) {
			return candidate, nil
		}
	}
	return "", errors.New(tool + " not found in PATH (excluding shims)")
}

func isExecutableFile(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	if st.IsDir() {
		return false
	}
	return st.Mode()&0o111 != 0
}
