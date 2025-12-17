package execx

import (
	"os"
	"path/filepath"
	"strings"
)

// PrependToPATH returns a PATH string with dir inserted at the beginning.
// If dir already exists in the PATH (anywhere), it is de-duplicated.
func PrependToPATH(dir, pathVal string) string {
	dir = cleanPath(dir)
	if dir == "" {
		return pathVal
	}

	parts := filepath.SplitList(pathVal)
	out := make([]string, 0, len(parts)+1)
	out = append(out, dir)
	for _, p := range parts {
		p = cleanPath(p)
		if p == "" {
			continue
		}
		if samePath(p, dir) {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, string(os.PathListSeparator))
}

func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	return filepath.Clean(p)
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if os.PathSeparator == '\\' {
		return strings.EqualFold(a, b)
	}
	return a == b
}
