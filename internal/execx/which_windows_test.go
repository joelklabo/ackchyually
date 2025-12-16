//go:build windows

package execx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindExecutable_Windows(t *testing.T) {
	tmp := t.TempDir()

	// 1. Exact match with extension provided
	bat := filepath.Join(tmp, "script.bat")
	if err := os.WriteFile(bat, []byte(""), 0o700); err != nil {
		t.Fatalf("write bat: %v", err)
	}

	if got, ok := findExecutable(tmp, "script.bat"); !ok || !strings.EqualFold(got, bat) {
		t.Errorf("findExecutable(script.bat) = %q, %v; want %q, true", got, ok, bat)
	}

	// 2. Extension inference via PATHEXT
	// Default PATHEXT usually includes .BAT, but we can set it explicitly to be safe
	t.Setenv("PATHEXT", ".EXE;.BAT")

	if got, ok := findExecutable(tmp, "script"); !ok || !strings.EqualFold(got, bat) {
		t.Errorf("findExecutable(script) [implicit .bat] = %q, %v; want %q, true", got, ok, bat)
	}

	// 3. .EXE priority?
	// If both script.bat and script.exe exist, which one is picked depends on PATHEXT order.
	exe := filepath.Join(tmp, "script.exe")
	if err := os.WriteFile(exe, []byte(""), 0o700); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	// With PATHEXT=.EXE;.BAT, .EXE should win
	if got, ok := findExecutable(tmp, "script"); !ok || !strings.EqualFold(got, exe) {
		t.Errorf("findExecutable(script) [implicit .exe priority] = %q, %v; want %q, true", got, ok, exe)
	}

	// 4. Missing file
	if _, ok := findExecutable(tmp, "missing"); ok {
		t.Error("findExecutable(missing) = true; want false")
	}
}
