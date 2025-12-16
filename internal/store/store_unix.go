//go:build !windows

package store

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func checkOwnership(path string) error {
	if os.Geteuid() != 0 {
		return nil
	}

	// Check file itself
	info, err := os.Stat(path)
	if err == nil {
		return checkStat(path, info)
	}
	if !os.IsNotExist(err) {
		return nil // checking error is not fatal to the app logic, we proceed
	}

	// File doesn't exist, check directory
	dir := filepath.Dir(path)
	info, err = os.Stat(dir)
	if err == nil {
		return checkStat(dir, info)
	}

	return nil
}

func checkStat(path string, info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	if stat.Uid != 0 {
		return fmt.Errorf("refusing to access non-root path %s as root", path)
	}
	return nil
}
