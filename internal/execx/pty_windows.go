//go:build windows

package execx

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func runPTY(exe string, args []string) (Result, error) {
	cmd := exec.CommandContext(context.Background(), exe, args...)

	// pty.Start on Windows uses ConPTY if available.
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return Result{ExitCode: 1, Mode: "pty"}, err
	}
	defer func() { _ = ptmx.Close() }()

	if oldState, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
		defer func() {
			if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
				_ = err // best-effort
			}
		}()
	}

	// Windows doesn't use SIGWINCH for resize events in the same way.
	// Initial resize.
	if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
		_ = err // best-effort
	}

	// TODO: Monitor console resize events on Windows?
	// For now, no dynamic resize loop.

	combined := NewTail(64 * 1024)

	outputDone := make(chan struct{})
	go func() {
		// On Windows, ptmx is the ConPTY pipe.
		if _, err := io.Copy(io.MultiWriter(os.Stdout, combined), ptmx); err != nil {
			_ = err // best-effort
		}
		close(outputDone)
	}()
	go func() {
		if _, err := io.Copy(ptmx, os.Stdin); err != nil {
			_ = err // best-effort
		}
	}()

	err = cmd.Wait()
	<-outputDone

	code := exitCode(err)
	return Result{
		ExitCode:     code,
		Mode:         "pty",
		CombinedTail: combined.String(),
	}, err
}
