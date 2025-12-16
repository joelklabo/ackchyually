package execx

import (
	"strings"

	"github.com/kballard/go-shellquote"
)

func ShellJoin(argv []string) string {
	return shellquote.Join(argv...)
}

func ContainsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
