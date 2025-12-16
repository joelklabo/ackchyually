//go:build windows

package execx

import (
	"os"
	"path/filepath"
	"strings"
)

func findExecutable(dir, tool string) (string, bool) {
	if filepath.Ext(tool) != "" {
		if isExec(filepath.Join(dir, tool)) {
			return filepath.Join(dir, tool), true
		}
		return "", false
	}

	pathext := os.Getenv("PATHEXT")
	if pathext == "" {
		pathext = ".COM;.EXE;.BAT;.CMD"
	}

	for _, ext := range strings.Split(pathext, ";") {
		if ext == "" {
			continue
		}
		path := filepath.Join(dir, tool+ext)
		if isExec(path) {
			return path, true
		}
	}
	return "", false
}

func isExec(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
