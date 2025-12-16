package execx

import (
	"context"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func IsTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

type Result struct {
	ExitCode     int
	Mode         string // "pty" or "pipes"
	StdoutTail   string
	StderrTail   string
	CombinedTail string
}

func Run(exe string, args []string) (Result, error) {
	if IsTTY() {
		return runPTY(exe, args)
	}
	return runPipes(exe, args)
}

func runPTY(exe string, args []string) (Result, error) {
	cmd := exec.CommandContext(context.Background(), exe, args...)

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

	if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
		_ = err // best-effort
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	defer signal.Stop(ch)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				_ = err // best-effort
			}
		}
	}()

	combined := NewTail(64 * 1024)

	outputDone := make(chan struct{})
	go func() {
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
