//go:build !windows

package execx

import (
	"errors"
	"os/exec"
	"syscall"
)

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
