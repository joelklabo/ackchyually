package copilot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/joelklabo/ackchyually/internal/execx"
)

const (
	backupSuffix = ".ackchyually-orig"
)

type Status struct {
	Installed    bool
	Version      string
	WrapperPath  string
	BackupPath   string
	Integrated   bool
	ShimDir      string
	ShimFirstVia string
}

func DetectInstalledVersion(ctx context.Context) (bool, string) {
	path, err := exec.LookPath("copilot")
	if err != nil {
		return false, ""
	}
	cmd := exec.CommandContext(ctx, path, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return true, ""
	}
	return true, strings.TrimSpace(string(out))
}

func DetectStatus(ctx context.Context, shimDir string) (Status, error) {
	var st Status
	st.Installed, st.Version = DetectInstalledVersion(ctx)
	if !st.Installed {
		return st, nil
	}

	path, err := exec.LookPath("copilot")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return st, nil
		}
		return Status{}, err
	}
	st.WrapperPath = path
	st.BackupPath = backupPath(path)
	st.ShimDir = shimDir
	st.ShimFirstVia = execx.PrependToPATH(shimDir, os.Getenv("PATH"))

	st.Integrated = isWrapperInstalled(path, st.BackupPath)
	return st, nil
}

type Action struct {
	Op      string // rename | write | remove
	Src     string
	Dst     string
	Content []byte
	Mode    fs.FileMode
}

type Plan struct {
	Actions []Action
}

func PlanInstall(wrapperPath, shimDir string) (Plan, error) {
	if wrapperPath == "" {
		p, err := exec.LookPath("copilot")
		if err != nil {
			return Plan{}, errors.New("copilot not found in PATH")
		}
		wrapperPath = p
	}
	if shimDir == "" {
		shimDir = execx.ShimDir()
	}

	st, err := os.Lstat(wrapperPath)
	if err != nil {
		return Plan{}, err
	}
	if st.Mode().IsDir() {
		return Plan{}, fmt.Errorf("copilot path is a directory: %s", wrapperPath)
	}

	bak := backupPath(wrapperPath)

	// If already installed, just rewrite wrapper content (idempotent).
	if isWrapperInstalled(wrapperPath, bak) {
		return Plan{
			Actions: []Action{
				{Op: "write", Dst: wrapperPath, Content: wrapperScript(wrapperPath, bak, shimDir), Mode: st.Mode() | 0o111},
			},
		}, nil
	}

	if _, err := os.Lstat(bak); err == nil {
		return Plan{}, fmt.Errorf("backup path already exists: %s", bak)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Plan{}, err
	}

	return Plan{
		Actions: []Action{
			{Op: "rename", Src: wrapperPath, Dst: bak},
			{Op: "write", Dst: wrapperPath, Content: wrapperScript(wrapperPath, bak, shimDir), Mode: 0o755},
		},
	}, nil
}

func PlanUndo(wrapperPath string) (Plan, error) {
	if wrapperPath == "" {
		p, err := exec.LookPath("copilot")
		if err != nil {
			return Plan{}, errors.New("copilot not found in PATH")
		}
		wrapperPath = p
	}

	bak := backupPath(wrapperPath)
	if !isWrapperInstalled(wrapperPath, bak) {
		return Plan{Actions: nil}, nil
	}

	return Plan{
		Actions: []Action{
			{Op: "remove", Src: wrapperPath},
			{Op: "rename", Src: bak, Dst: wrapperPath},
		},
	}, nil
}

func Apply(plan Plan) error {
	for _, a := range plan.Actions {
		switch a.Op {
		case "rename":
			if err := os.Rename(a.Src, a.Dst); err != nil {
				return err
			}
		case "write":
			if err := os.WriteFile(a.Dst, a.Content, a.Mode); err != nil {
				return err
			}
		case "remove":
			if err := os.Remove(a.Src); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown action op: %q", a.Op)
		}
	}
	return nil
}

func backupPath(wrapperPath string) string {
	return wrapperPath + backupSuffix
}

func isWrapperInstalled(wrapperPath, backupPath string) bool {
	if _, err := os.Lstat(backupPath); err != nil {
		return false
	}
	b, err := os.ReadFile(wrapperPath)
	if err != nil {
		return false
	}
	return bytes.Contains(b, []byte("ackchyually copilot wrapper"))
}

func wrapperScript(wrapperPath, backupPath, shimDir string) []byte {
	// If the copilot entrypoint is a *.cmd file (common on Windows npm installs),
	// generate a cmd wrapper; otherwise generate a POSIX sh wrapper.
	if runtime.GOOS == "windows" || strings.HasSuffix(strings.ToLower(wrapperPath), ".cmd") {
		return []byte(fmt.Sprintf(`@echo off
rem ackchyually copilot wrapper
set "ACKCHYUALLY_SHIM_DIR=%s"
set "PATH=%s;%%PATH%%"
"%s" %%*
`, shimDir, shimDir, backupPath))
	}

	return []byte(fmt.Sprintf(`#!/bin/sh
# ackchyually copilot wrapper
ACKCHYUALLY_SHIM_DIR=%q
export PATH="%s:$PATH"
exec %q "$@"
`, shimDir, shimDir, backupPath))
}
