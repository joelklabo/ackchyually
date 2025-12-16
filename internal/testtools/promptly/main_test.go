package main

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMainOutputsMarkers(t *testing.T) {
	t.Parallel()

	origStdin, origStdout, origStderr := os.Stdin, os.Stdout, os.Stderr
	rIn, wIn, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	os.Stdin = rIn
	os.Stdout = wOut
	os.Stderr = wErr
	defer func() {
		os.Stdin, os.Stdout, os.Stderr = origStdin, origStdout, origStderr
	}()

	if _, err := wIn.Write([]byte("y\n")); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	_ = wIn.Close()

	done := make(chan struct{})
	go func() {
		main()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("main timed out")
	}

	_ = wOut.Close()
	_ = wErr.Close()

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	out := string(outBytes) + string(errBytes)

	for _, marker := range []string{"PROMPTLY_START", "PROMPTLY_END", "PROMPT enter y:"} {
		if !strings.Contains(out, marker) {
			t.Fatalf("output missing marker %q:\n%s", marker, out)
		}
	}
}
