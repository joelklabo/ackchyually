package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func installShimWindows(shimDir string, tool string, exe string) error {
	dst := filepath.Join(shimDir, tool+".exe")
	_ = os.Remove(dst)

	// Try hardlink first (fast, works on same volume)
	err := os.Link(exe, dst)
	if err == nil {
		return nil
	}

	// Fallback to copy
	input, err := os.ReadFile(exe)
	if err != nil {
		return fmt.Errorf("read executable: %w", err)
	}
	if err := os.WriteFile(dst, input, 0o755); err != nil { //nolint:gosec
		return fmt.Errorf("write shim: %w", err)
	}
	return nil
}

func installShimUnix(shimDir string, tool string, exe string) error {
	dst := filepath.Join(shimDir, tool)
	_ = os.Remove(dst)
	if err := os.Symlink(exe, dst); err != nil {
		return fmt.Errorf("symlink failed: %w", err)
	}
	return nil
}

func installShim(shimDir, tool, exe string) error {
	if runtime.GOOS == "windows" {
		return installShimWindows(shimDir, tool, exe)
	}
	return installShimUnix(shimDir, tool, exe)
}
