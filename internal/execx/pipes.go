package execx

import (
	"context"
	"io"
	"os"
	"os/exec"
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