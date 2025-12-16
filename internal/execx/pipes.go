package execx

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
)

func runPipes(exe string, args []string) (Result, error) {
	cmd := exec.CommandContext(context.Background(), exe, args...)
	cmd.Stdin = os.Stdin

	outTail := NewTail(64 * 1024)
	errTail := NewTail(64 * 1024)

	cmd.Stdout = io.MultiWriter(os.Stdout, outTail)
	cmd.Stderr = io.MultiWriter(os.Stderr, errTail)

	err := cmd.Run()
	code := exitCode(err)

	return Result{
		ExitCode:   code,
		Mode:       "pipes",
		StdoutTail: outTail.String(),
		StderrTail: errTail.String(),
	}, err
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok2 := ee.Sys().(syscall.WaitStatus); ok2 {
			if ws.Signaled() {
				return 128 + int(ws.Signal())
			}
			return ws.ExitStatus()
		}
		return ee.ExitCode()
	}
	return 1
}
