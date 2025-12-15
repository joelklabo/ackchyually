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

func TestIsUsageish_Exit0_JSONOutputContainingErrorStringsIgnored(t *testing.T) {
	got := isUsageish(
		[]string{"list", "--json"},
		0,
		execx.Result{
			Mode: "pty",
			CombinedTail: `{"id":"x","notes":"unknown flag: --jsn","desc":"Usage: not help"}\n` +
				`{"id":"y","notes":"fatal: also not help"}\n`,
		},
	)
	if got {
		t.Fatalf("got true want false")
	}
}

func TestIsUsageish_Exit0_YAMLOutputWithUsageKeyIgnored(t *testing.T) {
	got := isUsageish(
		[]string{"list", "--yaml"},
		0,
		execx.Result{
			Mode:         "pty",
			CombinedTail: "usage: this is a yaml key, not a help banner\nformat: yaml\n",
		},
	)
	if got {
		t.Fatalf("got true want false")
	}
}

func TestIsUsageish_Exit0_LogOutputWithErrorPrefixIgnored(t *testing.T) {
	got := isUsageish(
		[]string{"logs"},
		0,
		execx.Result{
			Mode:         "pty",
			CombinedTail: "INFO: starting\nERROR: previous run failed\n",
		},
	)
	if got {
		t.Fatalf("got true want false")
	}
}

func TestIsUsageish_Exit0_ANSIPrefixedErrorDetected(t *testing.T) {
	got := isUsageish(
		[]string{"do", "--badd"},
		0,
		execx.Result{
			Mode:       "pipes",
			StderrTail: "\x1b[31mError:\x1b[0m unknown flag: --badd\n\x1b[1mUsage:\x1b[0m tool do --ok\n",
		},
	)
	if !got {
		t.Fatalf("got false want true")
	}
}

func TestIsUsageish_Exit2_UnexpectedArgumentDetected(t *testing.T) {
	got := isUsageish(
		[]string{"run", "--jsn"},
		2,
		execx.Result{
			StderrTail: "error: unexpected argument '--jsn' found\n",
		},
	)
	if !got {
		t.Fatalf("got false want true")
	}
}
