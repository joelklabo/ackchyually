package app

import (
	"strings"
	"testing"
	"time"

	"github.com/joelklabo/ackchyually/internal/store"
)

func TestAutoExecKnownSuccess_Executes(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_AUTO_EXEC", "known_success")

	// Seed a successful invocation of "echo hello"
	err := store.WithDB(func(db *store.DB) error {
		return db.InsertInvocation(store.Invocation{
			At:         time.Now(),
			DurationMS: 1,
			ContextKey: ctxKey,
			Tool:       "echo",
			ExePath:    "/bin/echo",
			ArgvJSON:   store.MustJSON([]string{"echo", "hello"}),
			ExitCode:   0,
			Mode:       "pipes",
		})
	})
	if err != nil {
		t.Fatalf("seed invocation: %v", err)
	}

	// Call with a typo "echo helo"
	// Wait, autoExecKnownSuccess is called when we *have* a candidate.
	// It takes the *original* tool and argv? No, it takes the *candidate* argv?
	// Let's check the signature: func autoExecKnownSuccess(tool, ctxKey string, argv []string) (int, bool)
	// It seems `argv` is the *current* invocation args.
	// But `autoExecKnownSuccess` logic is:
	// 1. Check if enabled.
	// 2. Look for *one* candidate that is "good enough".
	// 3. If found, exec it.

	// Actually, looking at shim.go:
	// It calls `db.ListSuccessCandidates`.
	// If there is exactly one candidate, and it matches the fuzzy logic...

	// So I need to pass the *typo* args to `autoExecKnownSuccess`?
	// No, `autoExecKnownSuccess` is called *inside* `runShim`?
	// Or is it a helper called by `runShim`?

	// Let's look at `shim.go` again.
	// It seems `autoExecKnownSuccess` is called with the *typo* args.
	// And it searches for the *correct* args.

	// Wait, `autoExecKnownSuccess` signature in `shim.go`:
	// func autoExecKnownSuccess(tool, ctxKey string, argv []string) (int, bool)

	// If I pass "echo helo", and "echo hello" is in DB.

	code, out, errOut := captureStdoutStderr(t, func() int {
		c, ok := autoExecKnownSuccess("echo", ctxKey, []string{"echo", "helo"})
		if !ok {
			return -1
		}
		return c
	})

	if code != 0 {
		t.Fatalf("autoExecKnownSuccess returned code %d, want 0 (ok=true). Stderr: %s", code, errOut)
	}
	if !strings.Contains(errOut, "ackchyually: auto-exec (known_success):") {
		t.Errorf("stderr missing auto-exec message, got:\n%s", errOut)
	}
	if !strings.Contains(errOut, "echo hello") {
		t.Errorf("stderr missing echoed command, got:\n%s", errOut)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("stdout missing 'hello', got:\n%s", out)
	}
}

func TestAutoExecKnownSuccess_NoMatch(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_AUTO_EXEC", "known_success")

	// No seed data

	code, _, _ := captureStdoutStderr(t, func() int {
		c, ok := autoExecKnownSuccess("git", ctxKey, []string{"git", "st"})
		if ok {
			return c
		}
		return -1
	})
	if code != -1 {
		t.Errorf("autoExecKnownSuccess returned %d want -1 (ok=false)", code)
	}
}

func TestAutoExecKnownSuccess_Ambiguous(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_AUTO_EXEC", "known_success")
	now := time.Now()

	seedInvocation(t, ctxKey, "git", []string{"git", "status"}, now, 0)
	seedInvocation(t, ctxKey, "git", []string{"git", "stash"}, now, 0)

	code, _, _ := captureStdoutStderr(t, func() int {
		c, ok := autoExecKnownSuccess("git", ctxKey, []string{"git", "st"})
		if ok {
			return c
		}
		return -1
	})
	if code != -1 {
		t.Errorf("autoExecKnownSuccess returned %d want -1 (ok=false)", code)
	}
}
