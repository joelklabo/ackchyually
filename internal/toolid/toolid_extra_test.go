package toolid

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIdentify_Sha256FileError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping chmod 000 on windows")
	}

	tmp := t.TempDir()
	f := filepath.Join(tmp, "unreadable")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	// Readable by stat, unreadable by open
	if err := os.Chmod(f, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	_, err := Identify(f)
	if err == nil {
		t.Error("Identify: expected error for unreadable file")
	}
}

func TestIdentify_DetectVersionTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping slow script on windows")
	}

	tmp := t.TempDir()
	exe := filepath.Join(tmp, "slow.sh")
	// Sleeps 1 second, longer than 800ms timeout
	//nolint:gosec
	err := os.WriteFile(exe, []byte(`#!/bin/sh
sleep 1
echo "v1"
`), 0o755)
	if err != nil {
		t.Fatalf("write slow script: %v", err)
	}

	// Should fallback to (version unknown) because it timed out
	ti, err := Identify(exe)
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if ti.VersionStr != "slow.sh (version unknown)" {
		t.Errorf("expected unknown version on timeout, got %q", ti.VersionStr)
	}
}

func TestLooksVersionish_ExtraCases(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"v1.2.", false}, // trailing dot
		{"1.a", false},   // "1." matches number loop, "a" stops it.
		{"V1", true},
		{"\n", false},
		{"v", false},
	}
	for _, tc := range tests {
		if got := looksVersionish(tc.in); got != tc.want {
			t.Errorf("looksVersionish(%q)=%v, want %v", tc.in, got, tc.want)
		}
	}
}
