package app

import (
	"testing"

	"github.com/joelklabo/ackchyually/internal/execx"
)

func TestIsUsageish_Exit0HelpInvocationIgnored(t *testing.T) {
	got := isUsageish([]string{"--help"}, 0, execx.Result{CombinedTail: "Usage: tool [flags]\n"})
	if got {
		t.Fatalf("got true want false")
	}
}

func TestIsUsageish_Exit0ErrorOutputStillCounts(t *testing.T) {
	got := isUsageish(
		[]string{"-fsSL", "-o", "/dev/null", "-w", "%{fial}", "file:///etc/hosts"},
		0,
		execx.Result{StderrTail: "curl: unknown --write-out variable: 'fial'\n"},
	)
	if !got {
		t.Fatalf("got false want true")
	}
}
