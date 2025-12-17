package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/joelklabo/ackchyually/internal/execx"
	"github.com/joelklabo/ackchyually/internal/ui"
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
		if t == "ackchyually" || t == "ackchyually.exe" {
			continue
		}
		if err := installShim(shimDir, t, exe); err != nil {
			fmt.Fprintln(os.Stderr, "ackchyually:", err)
			return 1
		}
	}

	u := ui.New(os.Stdout)

	fmt.Printf("%s shims in:\n  %s\n", u.OK("Installed"), shimDir)
	fmt.Println()

	pathEnv := os.Getenv("PATH")
	parts := strings.Split(pathEnv, string(os.PathListSeparator))
	want := filepath.Clean(shimDir)
	found := -1
	for i, p := range parts {
		if p == "" {
			p = "."
		}
		if filepath.Clean(p) == want {
			found = i
			break
		}
	}

	switch {
	case found == 0:
		fmt.Printf("%s: shim dir is first in PATH\n", u.OK("OK"))
		if rcPath, ok := detectPersistedShims(shimDir); ok {
			fmt.Printf("%s: shims are enabled in: %s\n", u.OK("OK"), rcPath)
		} else {
			fmt.Printf("%s: persist this with: ackchyually shim enable\n", u.Dim("Tip"))
		}
	case found == -1:
		fmt.Printf("%s: put shim dir first in PATH\n", u.Warn("Required"))
		printPathInstructions(shimDir)
	case found > 0:
		fmt.Printf("%s: shim dir must be first in PATH (currently index=%d)\n", u.Warn("Required"), found)
		printPathInstructions(shimDir)
	}

	fmt.Println()
	fmt.Printf("%s:\n", u.Bold("Refresh"))
	if runtime.GOOS == "windows" {
		fmt.Println("  # Restart your shell")
	} else {
		fmt.Println("  hash -r 2>/dev/null || true")
	}
	fmt.Println()
	fmt.Printf("%s:\n", u.Bold("Verify"))
	fmt.Printf("  which %s\n", tools[0])                            //nolint:gosec
	fmt.Printf("  # %s%c%s\n", shimDir, os.PathSeparator, tools[0]) //nolint:gosec
	fmt.Println()
	fmt.Println(u.Dim(fmt.Sprintf("If it prints something like /opt/homebrew/bin/%s, your shim dir isn't taking effect.", tools[0]))) //nolint:gosec

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

func shimDoctor() int { //nolint:gocyclo
	shimDir := shimDir()
	u := ui.New(os.Stdout)

	ackExe := "(unknown)"
	if p, err := os.Executable(); err == nil {
		ackExe = p
	}
	dbPath := filepath.Join(filepath.Dir(shimDir), "ackchyually.sqlite")

	fmt.Println(u.Bold("ackchyually shim doctor"))
	fmt.Printf("binary:   %s\n", ackExe)
	fmt.Printf("shim dir: %s\n", shimDir)
	fmt.Printf("db:       %s\n", dbPath)
	fmt.Println()

	exitCode := 0

	entries, err := os.ReadDir(shimDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("installed shims: (none)")
		} else {
			fmt.Fprintln(os.Stderr, "ackchyually:", err)
			exitCode = 1
		}
	} else {
		hasAckchyuallyShim := false
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			name := e.Name()
			if name == "" || name[0] == '.' {
				continue
			}
			if name == "ackchyually" || name == "ackchyually.exe" {
				hasAckchyuallyShim = true
				continue
			}
			names = append(names, name)
		}
		sort.Strings(names)

		if len(names) == 0 {
			fmt.Println("installed shims: (none)")
		} else {
			ok := true
			var details []string
			var inactive []string
			var broken []string
			var missingReal []string

			if hasAckchyuallyShim {
				fmt.Println("note:    shim dir contains 'ackchyually' (not needed)")
				fmt.Println("         remove: ackchyually shim uninstall ackchyually")
				fmt.Println()
			}

			fmt.Printf("installed shims: %s\n", strings.Join(names, ", "))

			for _, name := range names {
				dst := filepath.Join(shimDir, name)

				info, err := os.Lstat(dst)
				if err != nil {
					ok = false
					broken = append(broken, name)
					details = append(details, fmt.Sprintf("%s: stat failed", name))
					continue
				}
				if info.Mode()&os.ModeSymlink == 0 {
					ok = false
					broken = append(broken, name)
					details = append(details, fmt.Sprintf("%s: not a symlink (expected symlink to ackchyually)", name))
					continue
				}

				target, err := os.Readlink(dst)
				if err != nil {
					ok = false
					broken = append(broken, name)
					details = append(details, fmt.Sprintf("%s: readlink failed", name))
					continue
				}
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(dst), target)
				}
				target = filepath.Clean(target)

				if st, err := os.Stat(target); err != nil || st.IsDir() || st.Mode()&0o111 == 0 {
					ok = false
					broken = append(broken, name)
					details = append(details, fmt.Sprintf("%s: broken shim target", name))
					continue
				}

				if _, err := execx.WhichSkippingShims(name); err != nil {
					ok = false
					missingReal = append(missingReal, name)
					details = append(details, fmt.Sprintf("%s: real tool not found in PATH (excluding shims)", name))
					continue
				}

				got, err := exec.LookPath(name)
				active := err == nil && filepath.Clean(got) == filepath.Clean(dst)
				if !active {
					ok = false
					inactive = append(inactive, name)
					if err != nil {
						details = append(details, fmt.Sprintf("%s: not in PATH (expected %s)", name, dst))
					} else {
						details = append(details, fmt.Sprintf("%s: PATH=%s (expected %s)", name, got, dst))
					}
					continue
				}
			}

			if ok {
				fmt.Printf("status:   %s (shims are active)\n", u.OK("ok"))
			} else {
				fmt.Println()
				fmt.Printf("status:   %s\n", u.Warn("warn"))
				if len(inactive) > 0 {
					fmt.Printf("inactive: %s\n", strings.Join(inactive, ", "))
				}
				if len(broken) > 0 {
					fmt.Printf("broken:   %s\n", strings.Join(broken, ", "))
				}
				if len(missingReal) > 0 {
					fmt.Printf("missing:  %s\n", strings.Join(missingReal, ", "))
				}

				if len(details) > 0 {
					fmt.Println()
					fmt.Println(u.Bold("details:"))
					for _, d := range details {
						fmt.Printf("  - %s\n", d)
					}
				}

				fmt.Println()
				fmt.Println(u.Bold("fix:"))
				if len(inactive) > 0 {
					fmt.Printf("  export PATH=\"%s%c$PATH\"\n", shimDir, os.PathListSeparator)
					fmt.Println("  hash -r 2>/dev/null || true")
					fmt.Println("  # or: ackchyually shim enable")
				}
				if len(broken) > 0 {
					fmt.Printf("  ackchyually shim install %s\n", strings.Join(broken, " "))
				}
				if len(missingReal) > 0 {
					fmt.Printf("  # install missing tools, or: ackchyually shim uninstall %s\n", strings.Join(missingReal, " "))
				}
				exitCode = 1
			}
		}
	}

	fmt.Println()
	fmt.Println(u.Bold("Agent CLIs"))
	_ = integrateStatus(nil) // best-effort

	return exitCode
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
