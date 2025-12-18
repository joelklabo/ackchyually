package execx

import (
	"os"

	"golang.org/x/term"
)

func IsTTY() bool {
	// Check if both stdin and stdout are terminals AND we can get their state.
	// This helps avoid "device not configured" errors when terminals are in an invalid state.
	stdinFd := int(os.Stdin.Fd())
	stdoutFd := int(os.Stdout.Fd())
	
	if !term.IsTerminal(stdinFd) || !term.IsTerminal(stdoutFd) {
		return false
	}
	
	// Verify we can actually get the terminal state before attempting PTY operations.
	// If GetState fails, the terminal might be in an invalid state (closed/detached).
	if _, err := term.GetState(stdinFd); err != nil {
		return false
	}
	
	return true
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
