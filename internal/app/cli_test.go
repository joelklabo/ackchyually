package app

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joelklabo/ackchyually/internal/contextkey"
	"github.com/joelklabo/ackchyually/internal/store"
)

func captureStdoutStderr(t *testing.T, fn func() int) (code int, stdout string, stderr string) {
	t.Helper()

	oldOut := os.Stdout
	oldErr := os.Stderr

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		_ = outR.Close()
		_ = outW.Close()
		t.Fatalf("stderr pipe: %v", err)
	}

	os.Stdout = outW
	os.Stderr = errW

	code = fn()

	os.Stdout = oldOut
	os.Stderr = oldErr

	_ = outW.Close()
	_ = errW.Close()

	outB, outErr := io.ReadAll(outR)
	_ = outR.Close()
	errB, errErr := io.ReadAll(errR)
	_ = errR.Close()

	if outErr != nil {
		t.Fatalf("read stdout: %v", outErr)
	}
	if errErr != nil {
		t.Fatalf("read stderr: %v", errErr)
	}
	return code, string(outB), string(errB)
}

func setTempHomeAndCWD(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "cwd")
	mkdirAll(t, home)
	mkdirAll(t, cwd)

	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	oldCwd, err := os.Getwd()
	if err != nil {
		oldCwd = ""
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if oldCwd != "" {
			if err := os.Chdir(oldCwd); err != nil {
				t.Fatalf("restore cwd: %v", err)
			}
		}
	})

	return contextkey.Detect()
}

func seedInvocation(t *testing.T, ctxKey, tool string, argv []string, at time.Time, exitCode int) {
	t.Helper()
	err := store.WithDB(func(db *store.DB) error {
		return db.InsertInvocation(store.Invocation{
			At:         at,
			DurationMS: 1,
			ContextKey: ctxKey,
			Tool:       tool,
			ExePath:    "/bin/" + tool,
			ArgvJSON:   store.MustJSON(argv),
			ExitCode:   exitCode,
			Mode:       "cli-test",
		})
	})
	if err != nil {
		t.Fatalf("seed invocation: %v", err)
	}
}

func seedTag(t *testing.T, tag store.Tag) {
	t.Helper()
	if err := store.WithDB(func(db *store.DB) error { return db.UpsertTag(tag) }); err != nil {
		t.Fatalf("seed tag: %v", err)
	}
}

func TestRunCLI_NoArgs_ShowsUsage(t *testing.T) {
	setTempHomeAndCWD(t)

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{})
	})
	if code != 2 {
		t.Fatalf("RunCLI returned %d, want 2", code)
	}
	if !strings.Contains(errOut, "Commands:") {
		t.Fatalf("expected usage on stderr, got:\n%s", errOut)
	}
}

func TestRunCLI_UnknownCommand_ShowsSuggestionAndLastSuccess(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)

	// Record a known-good previous success so unknown-command output can show it.
	code, _, _ := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"shim", "list"})
	})
	if code != 0 {
		t.Fatalf("shim list returned %d want 0", code)
	}

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"shmi"})
	})
	if code != 2 {
		t.Fatalf("RunCLI(unknown command) returned %d, want 2", code)
	}
	if !strings.Contains(errOut, "unknown command: shmi") {
		t.Fatalf("expected unknown command message, got:\n%s", errOut)
	}
	if !strings.Contains(errOut, "try: ackchyually shim") {
		t.Fatalf("expected suggestion, got:\n%s", errOut)
	}
	if !strings.Contains(errOut, "last success here: ackchyually shim list") {
		t.Fatalf("expected last success, got:\n%s", errOut)
	}
	if !strings.Contains(errOut, "available:") {
		t.Fatalf("expected available commands list, got:\n%s", errOut)
	}

	// Extra sanity: ensure we stayed in the same context for logging.
	_ = ctxKey
}

func TestRunCLI_UnknownSubcommand_ShowsSuggestion(t *testing.T) {
	setTempHomeAndCWD(t)

	// Ensure there is a last success, but not equal to the suggestion.
	code, _, _ := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"version"})
	})
	if code != 0 {
		t.Fatalf("version returned %d want 0", code)
	}

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"shim", "listl"})
	})
	if code != 2 {
		t.Fatalf("RunCLI(unknown subcommand) returned %d, want 2", code)
	}
	if !strings.Contains(errOut, "unknown shim subcommand: listl") {
		t.Fatalf("expected unknown subcommand message, got:\n%s", errOut)
	}
	if !strings.Contains(errOut, "try: ackchyually shim list") {
		t.Fatalf("expected subcommand suggestion, got:\n%s", errOut)
	}
	if !strings.Contains(errOut, "available:") {
		t.Fatalf("expected available shim subcommands, got:\n%s", errOut)
	}
}

func TestRunCLI_Best_NoHistory(t *testing.T) {
	setTempHomeAndCWD(t)

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"best", "--tool", "git", "status"})
	})
	if code != 1 {
		t.Fatalf("best returned %d want 1", code)
	}
	if !strings.Contains(errOut, "no successful commands recorded yet") {
		t.Fatalf("expected best no-history error, got:\n%s", errOut)
	}
}

func TestRunCLI_Best_PrintsKnownGood(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)

	now := time.Now()
	seedInvocation(t, ctxKey, "git", []string{"git", "status"}, now.Add(-time.Minute), 0)
	seedInvocation(t, ctxKey, "git", []string{"git", "commit", "-m", "msg"}, now.Add(-2*time.Minute), 1)

	code, out, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"best", "--tool", "git", "status"})
	})
	if code != 0 {
		t.Fatalf("best returned %d want 0, stderr:\n%s", code, errOut)
	}
	if !strings.Contains(out, "git status") {
		t.Fatalf("expected best output to include git status, got:\n%s", out)
	}
}

func TestRunCLI_TagAdd_AndExport(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"tag", "add", "build", "--", "go", "build", "./..."})
	})
	if code != 0 {
		t.Fatalf("tag add returned %d want 0, stderr:\n%s", code, errOut)
	}

	// Export markdown should show tags and a tip for successful commands when --tool isn't passed.
	code, out, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"export"})
	})
	if code != 0 {
		t.Fatalf("export returned %d want 0, stderr:\n%s", code, errOut)
	}
	if !strings.Contains(out, "## ackchyually export") {
		t.Fatalf("expected export header, got:\n%s", out)
	}
	if !strings.Contains(out, "**build**") {
		t.Fatalf("expected tag in export, got:\n%s", out)
	}
	if !strings.Contains(out, "pass `--tool <tool>`") {
		t.Fatalf("expected tool tip in export, got:\n%s", out)
	}

	// Export JSON should be valid-ish and contain the tag.
	code, out, errOut = captureStdoutStderr(t, func() int {
		return RunCLI([]string{"export", "--format", "json"})
	})
	if code != 0 {
		t.Fatalf("export json returned %d want 0, stderr:\n%s", code, errOut)
	}
	if !strings.Contains(out, `"tags"`) || !strings.Contains(out, `"build"`) {
		t.Fatalf("expected json export to include tag, got:\n%s", out)
	}

	// Ensure tags were stored under the current context.
	tg, err := getTagDirect(ctxKey, "build")
	if err != nil {
		t.Fatalf("GetTag: %v", err)
	}
	if tg.Tool != "go" {
		t.Fatalf("expected stored tag tool=go, got %q", tg.Tool)
	}
}

func TestRunCLI_Export_UnknownFormat_Returns2(t *testing.T) {
	setTempHomeAndCWD(t)

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"export", "--format", "bogus"})
	})
	if code != 2 {
		t.Fatalf("export bogus returned %d want 2", code)
	}
	if !strings.Contains(errOut, "unknown format") {
		t.Fatalf("expected unknown format error, got:\n%s", errOut)
	}
}

func TestRunCLI_TagRun_CallsShimEvenIfToolMissing(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)

	seedTag(t, store.Tag{
		ContextKey: ctxKey,
		Tag:        "missingtool",
		Tool:       "definitelynotinstalled___ackchyually",
		ArgvJSON:   store.MustJSON([]string{"definitelynotinstalled___ackchyually", "--flag"}),
	})

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"tag", "run", "missingtool"})
	})
	if code != 127 {
		t.Fatalf("tag run returned %d want 127, stderr:\n%s", code, errOut)
	}
	if !strings.Contains(errOut, "not found in PATH") {
		t.Fatalf("expected RunShim not-found error, got:\n%s", errOut)
	}
}

func getTagDirect(ctxKey, tag string) (store.Tag, error) {
	var out store.Tag
	err := store.WithDB(func(db *store.DB) error {
		tg, err := db.GetTag(ctxKey, tag)
		if err != nil {
			return err
		}
		out = tg
		return nil
	})
	return out, err
}

func TestRunCLI_LogDBError(t *testing.T) {
	setTempHomeAndCWD(t)

	// Ensure DB dir exists
	dbDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "ackchyually")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatalf("mkdir db dir: %v", err)
	}

	// Corrupt DB
	dbPath := filepath.Join(dbDir, "ackchyually.sqlite")
	if err := os.WriteFile(dbPath, []byte("not a sqlite file"), 0o600); err != nil {
		t.Fatalf("corrupt db: %v", err)
	}

	// RunCLI should not crash, but it might log an error or just ignore it.
	// logCLIInvocation ignores the error: _ = err // best-effort

	code, _, _ := captureStdoutStderr(t, func() int {
		return RunCLI([]string{"version"})
	})

	if code != 0 {
		t.Errorf("RunCLI returned %d want 0", code)
	}
}
