package app

import (
	"os"
	"time"

	"github.com/joelklabo/ackchyually/internal/contextkey"
	"github.com/joelklabo/ackchyually/internal/redact"
	"github.com/joelklabo/ackchyually/internal/store"
)

func logCLIInvocation(start time.Time, dur time.Duration, args []string, exitCode int) {
	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}

	ctxKey := contextkey.Detect()

	r := redact.Default()
	argvSafe := r.RedactArgs(append([]string{"ackchyually"}, args...))

	if err := store.WithDB(func(db *store.DB) error {
		return db.InsertInvocation(store.Invocation{
			At:         start,
			DurationMS: dur.Milliseconds(),
			ContextKey: ctxKey,
			Tool:       "ackchyually",
			ExePath:    exe,
			ArgvJSON:   store.MustJSON(argvSafe),
			ExitCode:   exitCode,
			Mode:       "cli",
		})
	}); err != nil {
		_ = err // best-effort
	}
}
