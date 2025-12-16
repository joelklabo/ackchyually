package execx

import (
	"os"

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