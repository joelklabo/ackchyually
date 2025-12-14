package contextkey

import (
	"os"
	"path/filepath"
)

func Detect() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	if root := findGitRoot(cwd); root != "" {
		return "git:" + root
	}
	return "cwd:" + cwd
}

func findGitRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
