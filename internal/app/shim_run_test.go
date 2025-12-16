package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// writeExec writes an executable script.
// name is usually "script".
func writeExec(t *testing.T, dir, name, contentUnix, contentWin string) { //nolint:unparam
	t.Helper()
	path := filepath.Join(dir, name)
	content := contentUnix
	if runtime.GOOS == "windows" {
		path += ".bat"
		content = contentWin
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil { //nolint:gosec
		t.Fatalf("write script: %v", err)
	}
}

func TestRunShim_ExecError(t *testing.T) {
	tmp := t.TempDir()
	script := filepath.Join(tmp, "script")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0"), 0o600); err != nil { // not executable
		t.Fatalf("write script: %v", err)
	}

	// We need to call RunShim.
	// RunShim calls execx.Run.
	// execx.Run calls exec.Command.Run.

	code := RunShim(script, nil)
	if code != 127 { // not found / not executable
		t.Errorf("RunShim non-exec returned %d want 127", code)
	}
}

func TestRunShim_AutoExec_NoMatch(t *testing.T) {
	setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_AUTO_EXEC", "known_success")

	tmp := t.TempDir()
	writeExec(t, tmp, "script", "#!/bin/sh\necho running\n", "@echo running\n")

	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	code, out, _ := captureStdoutStderr(t, func() int {
		return RunShim("script", []string{"script"})
	})

	if code != 0 {
		t.Errorf("RunShim returned %d want 0", code)
	}
	if !strings.Contains(out, "running") {
		t.Errorf("output missing 'running', got:\n%s", out)
	}
}

func TestRunShim_Exit1_Usage(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_TEST_FORCE_TTY", "true")
	tmp := t.TempDir()
	writeExec(t, tmp, "script", "#!/bin/sh\necho 'Usage: script'\nexit 1", "@echo Usage: script\r\n@exit /b 1")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Seed known good
	seedInvocation(t, ctxKey, "script", []string{"script", "status"}, time.Now(), 0)

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunShim("script", []string{"script", "statu"}) // typo
	})

	if code != 1 {
		t.Errorf("RunShim returned %d want 1", code)
	}
	// Should suggest known good because it looks like usage
	if !strings.Contains(errOut, "suggestion") {
		t.Errorf("stderr missing suggestion, got:\n%s", errOut)
	}
}

func TestRunShim_Exit1_NoUsage(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	tmp := t.TempDir()
	writeExec(t, tmp, "script", "#!/bin/sh\necho 'Error: failed'\nexit 1", "@echo Error: failed\r\n@exit /b 1")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Seed known good
	seedInvocation(t, ctxKey, "script", []string{"script", "good"}, time.Now(), 0)

	code, _, errOut := captureStdoutStderr(t, func() int {
		return RunShim("script", []string{"script", "bad"})
	})

	if code != 1 {
		t.Errorf("RunShim returned %d want 1", code)
	}
	// Should NOT suggest known good because it doesn't look like usage
	if strings.Contains(errOut, "suggestion") {
		t.Errorf("stderr has unexpected suggestion, got:\n%s", errOut)
	}
}

func TestRunShim_AutoExec_Match(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_AUTO_EXEC", "known_success")

	tmp := t.TempDir()
	writeExec(t, tmp, "script", "#!/bin/sh\necho auto-executed\n", "@echo auto-executed\n")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	seedInvocation(t, ctxKey, "script", []string{"script", "arg"}, time.Now(), 0)

	code, out, _ := captureStdoutStderr(t, func() int {
		return RunShim("script", []string{"script", "arg"})
	})

	if code != 0 {
		t.Errorf("RunShim returned %d want 0", code)
	}
	if !strings.Contains(out, "auto-executed") {
		t.Errorf("output missing 'auto-executed', got:\n%s", out)
	}
}

func TestRunShim_AutoExec_Match_Exit1(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_AUTO_EXEC", "known_success")

	tmp := t.TempDir()
	writeExec(t, tmp, "script", "#!/bin/sh\nexit 1", "@exit /b 1")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	seedInvocation(t, ctxKey, "script", []string{"script", "arg"}, time.Now(), 0)

	code, _, _ := captureStdoutStderr(t, func() int {
		return RunShim("script", []string{"script", "arg"})
	})

	if code != 1 {
		t.Errorf("RunShim returned %d want 1", code)
	}
}

func TestRunShim_AutoExec_Disabled(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_AUTO_EXEC", "false")
	t.Setenv("ACKCHYUALLY_TEST_FORCE_TTY", "true")

	tmp := t.TempDir()
	// Script prints usage and exits 1
	writeExec(t, tmp, "script", "#!/bin/sh\necho 'usage: script <arg>'\nexit 1", "@echo usage: script <arg>\r\n@exit /b 1")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Seed a known success
	seedInvocation(t, ctxKey, "script", []string{"script", "success"}, time.Now(), 0)

	code, _, errOut := captureStdoutStderr(t, func() int {
		// Run with typo that triggers usage
		return RunShim("script", []string{"script", "succes"})
	})

	if code != 1 {
		t.Errorf("RunShim returned %d want 1", code)
	}
	if strings.Contains(errOut, "auto-executed") {
		t.Errorf("should not auto-execute when disabled")
	}
	if !strings.Contains(errOut, "suggestion") {
		t.Errorf("should still suggest, got:\n%s", errOut)
	}
}

func TestRunShim_AutoExec_Redacted(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_AUTO_EXEC", "known_success")

	tmp := t.TempDir()
	writeExec(t, tmp, "script", "#!/bin/sh\necho 'usage: script <arg>'\nexit 1", "@echo usage: script <arg>\r\n@exit /b 1")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Seed a known success that contains redacted info
	// We can't easily seed redacted info via RunShim because it redacts before saving.
	// But we can manually insert into DB.

	// Actually, pickKnownGood filters out redacted args.
	// So if we have a redacted arg in DB, it won't be picked.
	// Let's verify that.

	// We need to access the DB directly or use a helper.
	// Since we are in 'app' package, we can't easily access 'store' internals if they are not exported.
	// But we can use seedInvocation helper if we modify it to allow custom argv.
	// seedInvocation takes argv.

	// If we pass "<redacted>" in argv to seedInvocation, it will be saved as is (because seedInvocation uses db.InsertInvocation directly).
	// Wait, seedInvocation in tests usually calls db.InsertInvocation.

	seedInvocation(t, ctxKey, "script", []string{"script", "<redacted>"}, time.Now(), 0)

	code, out, _ := captureStdoutStderr(t, func() int {
		// Run with typo
		return RunShim("script", []string{"script", "typo"})
	})

	if code != 1 {
		t.Errorf("RunShim returned %d want 1", code)
	}
	if strings.Contains(out, "auto-executed") {
		t.Errorf("should not auto-execute redacted command")
	}
}

func TestRunShim_AutoExec_ExactMatch(t *testing.T) {
	ctxKey := setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_AUTO_EXEC", "known_success")

	tmp := t.TempDir()
	writeExec(t, tmp, "script", "#!/bin/sh\necho 'usage: script <arg>'\nexit 1", "@echo usage: script <arg>\r\n@exit /b 1")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Seed a known success
	seedInvocation(t, ctxKey, "script", []string{"script", "success"}, time.Now(), 0)

	code, out, _ := captureStdoutStderr(t, func() int {
		// Run with exact match (should not auto-exec because it's already what we ran)
		// But wait, if we run "script success" and it fails (e.g. flaky), should we auto-exec "script success"?
		// The logic says: if slicesEqual(cmd, argvSafe) { return 0, false }
		// So it should NOT auto-exec.
		return RunShim("script", []string{"script", "success"})
	})

	if code != 1 {
		t.Errorf("RunShim returned %d want 1", code)
	}
	if strings.Contains(out, "auto-executed") {
		t.Errorf("should not auto-execute exact match")
	}
}

func TestRunShim_AutoExec_DBError(t *testing.T) {
	// We can't easily force DB error without mocking store.WithDB or corrupting DB file.
	// But we can corrupt the DB file.
	setTempHomeAndCWD(t)
	t.Setenv("ACKCHYUALLY_AUTO_EXEC", "known_success")

	tmp := t.TempDir()
	writeExec(t, tmp, "script", "#!/bin/sh\necho 'usage: script <arg>'\nexit 1", "@echo usage: script <arg>\r\n@exit /b 1")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Ensure DB dir exists
	dbDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "ackchyually")
	if err := os.MkdirAll(dbDir, 0o755); err != nil { //nolint:gosec
		t.Fatalf("mkdir db dir: %v", err)
	}

	// Corrupt DB
	dbPath := filepath.Join(dbDir, "ackchyually.sqlite")
	if err := os.WriteFile(dbPath, []byte("not a sqlite file"), 0o600); err != nil {
		t.Fatalf("corrupt db: %v", err)
	}

	code, out, _ := captureStdoutStderr(t, func() int {
		return RunShim("script", []string{"script", "typo"})
	})

	if code != 1 {
		t.Errorf("RunShim returned %d want 1", code)
	}
	if strings.Contains(out, "auto-executed") {
		t.Errorf("should not auto-execute on db error")
	}
}
