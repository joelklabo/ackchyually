package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepro_NestedShims(t *testing.T) {
	// Setup:
	// toolA calls toolB.
	// Both are shimmed.
	// We want to see if toolB is called with the shim dir in PATH or not.

	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	shimDir := filepath.Join(tmp, "shims")
	if err := os.Mkdir(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(shimDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create real toolB
	// It prints "PATH=<path>" so we can inspect it.
	toolB := filepath.Join(binDir, "toolB")
	if err := os.WriteFile(toolB, []byte("#!/bin/sh\necho PATH=$PATH\n"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create real toolA
	// It calls toolB
	toolA := filepath.Join(binDir, "toolA")
	if err := os.WriteFile(toolA, []byte("#!/bin/sh\ntoolB\n"), 0755); err != nil {
		t.Fatal(err)
	}

	// Setup shims (we simulate shims by just setting up the environment and calling RunShim)
	// In a real scenario, the shim binary would be executed. Here we call RunShim directly for toolA.
	// But toolA calls toolB. For toolB to be intercepted, it must be looked up in PATH.
	// So we need 'toolB' in 'shims/' to be a symlink to 'ackchyually' (or in this test context, we can't easily execute the test binary as the shim).

	// Issue: RunShim executes the REAL toolA. Real toolA executes 'toolB'.
	// If PATH has shims/ first, it finds shims/toolB.
	// shims/toolB should point to the ackchyually binary.
	// Since we are running tests, we can't easily make shims/toolB point to "this test binary running RunShim".

	// However, we can simply assert that the PATH seen by toolB DOES NOT contain shimDir.
	// If it contains shimDir, then the fix is not applied.
	// If it doesn't contain shimDir, the fix is applied.
	// This relies on the fact that if shimDir IS in path, toolB *would* be shimmed if we had the shim set up.
	// Even without the actual shim binary there, checking the PATH variable in toolB is enough to verify if we are stripping it.

	// So we don't need actual shim binaries. We just need to check the PATH environment variable inside toolB.

	ctxKey := setTempHomeAndCWD(t)
	_ = ctxKey

	// Mock ShimDir logic:
	// The ShimDir() function in execx uses HOME/.local/share/ackchyually/shims.
	// We can't easily mock that return value without changing global state or exploring internal/execx.
	// execx.ShimDir() depends on UserHomeDir. We can change HOME env var.

	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(filepath.Join(home, ".local", "share", "ackchyually", "shims"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	realShimDir := filepath.Join(home, ".local", "share", "ackchyually", "shims")

	targetPath := realShimDir + string(os.PathListSeparator) + binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("PATH", targetPath)

	// RunShim("toolA") -> finds bin/toolA -> runs it.
	// bin/toolA runs toolB.
	// toolB prints PATH.

	code, out, _ := captureStdoutStderr(t, func() int {
		return RunShim("toolA", nil)
	})

	if code != 0 {
		t.Fatalf("toolA failed with code %d", code)
	}

	if !strings.Contains(out, "PATH=") {
		t.Fatalf("toolB output not found in:\n%s", out)
	}

	// Check if shimDir is in the output PATH
	// We expect this to fail currently because we haven't implemented the fix yet.
	if strings.Contains(out, realShimDir) {
		t.Fatalf("Shim dir FOUND in PATH (nested shimming possible). Expected it to be stripped.")
	}
}

// Helpers copied from other tests if needed, but we can reuse if in same package.
// We are in 'app', existing tests are in 'app'. We can use them if they are in non-_test.go files or if we put this in _test.go
// setTempHomeAndCWD is in cli_test.go. captureStdoutStderr is in cli_test.go.
// So we should be fine.
