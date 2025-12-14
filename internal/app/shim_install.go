package app

import (
	"fmt"
	"os"
	"path/filepath"
)

func shimInstall(tools []string) int {
	if len(tools) == 0 {
		fmt.Fprintln(os.Stderr, "shim install: specify tools")
		return 2
	}
	shimDir := shimDir()
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "ackchyually:", err)
		return 1
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ackchyually:", err)
		return 1
	}

	for _, t := range tools {
		dst := filepath.Join(shimDir, t)
		_ = os.Remove(dst)
		if err := os.Symlink(exe, dst); err != nil {
			fmt.Fprintln(os.Stderr, "ackchyually: symlink failed:", err)
			return 1
		}
	}
	fmt.Println("Installed shims in:", shimDir)
	fmt.Println("Ensure PATH begins with:", shimDir)
	return 0
}

func shimUninstall(tools []string) int {
	shimDir := shimDir()
	if len(tools) == 0 {
		fmt.Fprintln(os.Stderr, "shim uninstall: specify tools")
		return 2
	}
	for _, t := range tools {
		_ = os.Remove(filepath.Join(shimDir, t))
	}
	return 0
}

func shimDoctor() int {
	shimDir := shimDir()
	fmt.Println("Shim dir:", shimDir)
	fmt.Println("PATH should start with shim dir.")
	return 0
}

func shimDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "ackchyually", "shims")
}
