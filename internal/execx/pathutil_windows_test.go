//go:build windows

package execx

import (
	"os"
	"strings"
	"testing"
)

func TestPrependToPATH_WindowsCaseInsensitiveDedup(t *testing.T) {
	shim := `C:\Users\me\.local\share\ackchyually\shims`
	in := strings.Join([]string{
		`c:\users\me\.LOCAL\share\ackchyually\shims`,
		`C:\Windows\System32`,
	}, string(os.PathListSeparator))

	want := strings.Join([]string{
		shim,
		`C:\Windows\System32`,
	}, string(os.PathListSeparator))

	if got := PrependToPATH(shim, in); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
