package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joelklabo/ackchyually/internal/contextkey"
	"github.com/joelklabo/ackchyually/internal/execx"
	"github.com/joelklabo/ackchyually/internal/redact"
	"github.com/joelklabo/ackchyually/internal/store"
	"github.com/joelklabo/ackchyually/internal/toolid"
)

func RunShim(tool string, args []string) int {
	return runShim(tool, args, true)
}

func runShim(tool string, args []string, allowAutoExec bool) int {
	exe, err := execx.WhichSkippingShims(tool)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ackchyually:", err)
		return 127
	}

	ctxKey := contextkey.Detect()
	ti, err := toolid.Identify(exe)
	if err != nil {
		ti = toolid.ToolIdentity{}
	}

	start := time.Now()
	res, err := execx.Run(exe, args)
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			fmt.Fprintln(os.Stderr, "ackchyually:", err)
		}
	}
	dur := time.Since(start)

	// redact argv before writing
	r := redact.Default()
	argvSafe := r.RedactArgs(append([]string{tool}, args...))
	stdoutTailSafe := r.RedactText(res.StdoutTail)
	stderrTailSafe := r.RedactText(res.StderrTail)
	combinedTailSafe := r.RedactText(res.CombinedTail)

	// best-effort logging
	if err := store.WithDB(func(db *store.DB) error {
		return db.InsertInvocation(store.Invocation{
			At:           start,
			DurationMS:   dur.Milliseconds(),
			ContextKey:   ctxKey,
			Tool:         tool,
			ExePath:      exe,
			ToolID:       ti.ID,
			ArgvJSON:     store.MustJSON(argvSafe),
			ExitCode:     res.ExitCode,
			Mode:         res.Mode,
			StdoutTail:   stdoutTailSafe,
			StderrTail:   stderrTailSafe,
			CombinedTail: combinedTailSafe,
		})
	}); err != nil {
		_ = err // best-effort
	}

	if isUsageish(res.ExitCode, res) {
		if allowAutoExec && autoExecKnownSuccessEnabled() && execx.IsTTY() {
			if code, ok := autoExecKnownSuccess(tool, ctxKey, argvSafe); ok {
				return code
			}
		}
		suggestKnownGood(tool, ctxKey)
	}

	return res.ExitCode
}

func isUsageish(code int, res execx.Result) bool {
	if code == 64 {
		return true
	}
	t := res.StderrTail + res.CombinedTail
	return execx.ContainsFold(t, "usage:") ||
		execx.ContainsFold(t, "unknown option") ||
		execx.ContainsFold(t, "unrecognized option") ||
		execx.ContainsFold(t, "unrecognized argument") ||
		execx.ContainsFold(t, "invalid option") ||
		execx.ContainsFold(t, "missing required")
}

func suggestKnownGood(tool, ctxKey string) {
	if err := store.WithDB(func(db *store.DB) error {
		cmds, err := db.ListSuccessful(tool, ctxKey, 1)
		if err != nil {
			return err
		}
		if len(cmds) == 0 {
			return nil
		}
		fmt.Fprintln(os.Stderr, "ackchyually: this worked before here:")
		fmt.Fprintln(os.Stderr, "  "+execx.ShellJoin(cmds[0]))
		return nil
	}); err != nil {
		_ = err // best-effort
	}
}

func autoExecKnownSuccessEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("ACKCHYUALLY_AUTO_EXEC")))
	return v == "known_success"
}

func autoExecKnownSuccess(tool, ctxKey string, argvSafe []string) (int, bool) {
	var cmd []string
	if err := store.WithDB(func(db *store.DB) error {
		cmds, err := db.ListSuccessful(tool, ctxKey, 1)
		if err != nil {
			return err
		}
		if len(cmds) == 0 {
			return nil
		}
		cmd = cmds[0]
		return nil
	}); err != nil {
		return 0, false
	}

	if len(cmd) == 0 {
		return 0, false
	}
	if containsRedacted(cmd) {
		return 0, false
	}
	if slicesEqual(cmd, argvSafe) {
		return 0, false
	}

	fmt.Fprintln(os.Stderr, "ackchyually: auto-exec (known_success):")
	fmt.Fprintln(os.Stderr, "  "+execx.ShellJoin(cmd))
	return runShim(cmd[0], cmd[1:], false), true
}

func containsRedacted(argv []string) bool {
	for _, a := range argv {
		if strings.Contains(a, "<redacted>") {
			return true
		}
	}
	return false
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
