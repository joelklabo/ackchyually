package app

import (
	"path/filepath"
	"regexp"
	"strings"
)

var exportEnvKeyRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func exportSanitizeArg(s, home, repoRoot string) string {
	if k, v, ok := strings.Cut(s, "="); ok && exportEnvKeyRE.MatchString(k) && !strings.HasPrefix(k, "-") {
		return exportSanitizeEnvAssignment(k, v, home, repoRoot)
	}

	if k, v, ok := strings.Cut(s, "="); ok && strings.HasPrefix(k, "-") {
		return k + "=" + exportSanitizeValue(v, home, repoRoot)
	}

	return exportSanitizeValue(s, home, repoRoot)
}

func exportSanitizeEnvAssignment(key, value, home, repoRoot string) string {
	if exportLooksSensitiveEnvKey(key) {
		return key + "=<redacted>"
	}

	upper := strings.ToUpper(key)
	if strings.HasSuffix(upper, "PATH") {
		parts := strings.Split(value, string(filepath.ListSeparator))
		for i := range parts {
			parts[i] = exportSanitizeValue(parts[i], home, repoRoot)
		}
		return key + "=" + strings.Join(parts, string(filepath.ListSeparator))
	}

	return key + "=" + exportSanitizeValue(value, home, repoRoot)
}

func exportSanitizeValue(s, home, repoRoot string) string {
	s = exportNormalizePath(s, home, repoRoot)
	if filepath.IsAbs(s) {
		s = exportAnonymizeUserPath(s)
	}
	return s
}

func exportAnonymizeUserPath(p string) string {
	switch {
	case strings.HasPrefix(p, "/Users/"):
		return exportReplaceSecondSegment(p)
	case strings.HasPrefix(p, "/home/"):
		return exportReplaceSecondSegment(p)
	default:
		return p
	}
}

func exportReplaceSecondSegment(p string) string {
	parts := strings.Split(p, "/")
	// e.g. "", "Users", "alice", "proj"
	if len(parts) >= 3 && parts[2] != "" {
		parts[2] = "<user>"
		return strings.Join(parts, "/")
	}
	return p
}

func exportLooksSensitiveEnvKey(key string) bool {
	u := strings.ToUpper(key)
	return strings.Contains(u, "TOKEN") ||
		strings.Contains(u, "SECRET") ||
		strings.Contains(u, "PASSWORD") ||
		strings.Contains(u, "SESSION") ||
		strings.Contains(u, "COOKIE") ||
		strings.Contains(u, "AUTH") ||
		strings.Contains(u, "BEARER") ||
		strings.Contains(u, "API_KEY") ||
		strings.Contains(u, "APIKEY") ||
		strings.HasSuffix(u, "_KEY")
}
