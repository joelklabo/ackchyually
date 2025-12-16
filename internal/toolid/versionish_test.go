package toolid

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLooksVersionish(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{in: "", want: false},
		{in: "git version 2.39.2", want: true},
		{in: "v1", want: true},
		{in: "1", want: true},
		{in: "v1.2.3-alpha", want: true},
		{in: "v1.", want: false},
		{in: "PROMPTLY_START", want: false},
		{in: "usage: tool [flags]", want: false},
	} {
		if got := looksVersionish(tc.in); got != tc.want {
			t.Fatalf("looksVersionish(%q)=%v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestDetectVersion_TriesFallbackWhenOutputNotVersionish(t *testing.T) {
	tmp := t.TempDir()

	exe := filepath.Join(tmp, "tool.sh")
	//nolint:gosec
	if err := os.WriteFile(exe, []byte(`#!/bin/sh
if [ "${1:-}" = "--version" ]; then
  echo "usage: tool [flags]"
  exit 2
fi
if [ "${1:-}" = "version" ]; then
  echo "v2"
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatalf("write tool: %v", err)
	}

	got := detectVersion(exe)
	want := "tool.sh v2"
	if got != want {
		t.Fatalf("detectVersion=%q, want %q", got, want)
	}
}
