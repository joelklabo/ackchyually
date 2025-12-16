package execx

import (
	"os"
	"path/filepath"
	"strings"
)

func SanitizedEnv() []string {
	env := os.Environ()
	shimDir := filepath.Clean(ShimDir())

	newEnv := make([]string, 0, len(env))

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			newEnv = append(newEnv, e)
			continue
		}
		k, v := parts[0], parts[1]

		var isPath bool
		if os.PathSeparator == '\\' {
			isPath = strings.EqualFold(k, "PATH")
		} else {
			isPath = k == "PATH"
		}

		if isPath {
			// Filter PATH
			newVal := filterPath(v, shimDir)
			newEnv = append(newEnv, k+"="+newVal)
		} else {
			newEnv = append(newEnv, e)
		}
	}
	return newEnv
}

func filterPath(pathVal, shimDir string) string {
	parts := filepath.SplitList(pathVal)
	newParts := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		if filepath.Clean(p) == shimDir {
			continue
		}
		newParts = append(newParts, p)
	}
	return strings.Join(newParts, string(os.PathListSeparator))
}
