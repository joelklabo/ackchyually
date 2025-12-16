//go:build !windows

package execx

import (
	"os"
	"path/filepath"
)

func findExecutable(dir, tool string) (string, bool) {
	path := filepath.Join(dir, tool)
	st, err := os.Stat(path)
	if err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
		return path, true
	}
	return "", false
}
