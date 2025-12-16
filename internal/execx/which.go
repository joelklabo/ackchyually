package execx

import (
	"errors"
	"os"
	"path/filepath"
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
	parts := filepath.SplitList(pathEnv) // Use SplitList for cross-platform

	shim := filepath.Clean(ShimDir())
	for _, dir := range parts {
		if dir == "" {
			dir = "."
		}
		dir = filepath.Clean(dir)
		if dir == shim {
			continue
		}
		if found, ok := findExecutable(dir, tool); ok {
			return found, nil
		}
	}
	return "", errors.New(tool + " not found in PATH (excluding shims)")
}
