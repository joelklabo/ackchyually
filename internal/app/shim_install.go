package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/term"

	"github.com/joelklabo/ackchyually/internal/execx"
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

	bold, dim, green, yellow, reset := ansiStyles()

	fmt.Printf("%s%sInstalled%s shims in:\n  %s\n", green, bold, reset, shimDir)
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
		fmt.Printf("%sOK%s: shim dir is first in PATH\n", green, reset)
	case found == -1:
		fmt.Printf("%s%sRequired%s: add shim dir to PATH\n", yellow, bold, reset)
		fmt.Printf("  export PATH=\"%s%c$PATH\"\n", shimDir, os.PathListSeparator)
	case found > 0:
		fmt.Printf("%s%sRecommended%s: put shim dir first in PATH (currently index=%d)\n", yellow, bold, reset, found)
		fmt.Printf("  export PATH=\"%s%c$PATH\"\n", shimDir, os.PathListSeparator)
	}
	fmt.Println("  # for future shells, add that line to your ~/.zshrc or ~/.bashrc")
	fmt.Println("  hash -r 2>/dev/null || true")

	fmt.Println()
	fmt.Printf("%sVerify%s:\n", bold, reset)
	fmt.Printf("  which %s\n", tools[0])
	fmt.Printf("  # %s%c%s\n", shimDir, os.PathSeparator, tools[0])
	fmt.Println()
	fmt.Printf("%sIf it prints something like /opt/homebrew/bin/%s, your shim dir isn't taking effect.%s\n", dim, tools[0], reset)

	return 0
}

func ansiStyles() (bold, dim, green, yellow, reset string) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return "", "", "", "", ""
	}
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return "", "", "", "", ""
	}
	return "\033[1m", "\033[2m", "\033[32m", "\033[33m", "\033[0m"
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
	ackExe := "(unknown)"
	if p, err := os.Executable(); err == nil {
		ackExe = p
	}
	dbPath := filepath.Join(filepath.Dir(shimDir), "ackchyually.sqlite")

	fmt.Println("ackchyually shim doctor")
	fmt.Println()
	fmt.Println("ackchyually:", ackExe)
	fmt.Println("shim dir:", shimDir)
	fmt.Println("db:", dbPath)
	fmt.Println()

	ok := true

	if st, err := os.Stat(shimDir); err != nil {
		ok = false
		fmt.Println("WARN: shim dir missing (run: ackchyually shim install <tool...>)")
	} else if !st.IsDir() {
		ok = false
		fmt.Println("WARN: shim dir exists but is not a directory")
	}

	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		ok = false
		fmt.Println("WARN: PATH is empty")
	} else {
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
		case found == -1:
			ok = false
			fmt.Println("WARN: shim dir is not present in PATH")
			fmt.Printf("Fix: export PATH=\"%s%c$PATH\"\n", shimDir, os.PathListSeparator)
		case found != 0:
			ok = false
			fmt.Printf("WARN: shim dir is in PATH but not first (index=%d)\n", found)
			fmt.Printf("Fix: export PATH=\"%s%c$PATH\"\n", shimDir, os.PathListSeparator)
		default:
			fmt.Println("OK: shim dir is first in PATH")
		}
	}

	if _, err := os.Stat(dbPath); err != nil {
		fmt.Println("INFO: db not found yet (created on first invocation)")
	} else {
		fmt.Println("OK: db file exists")
	}

	entries, err := os.ReadDir(shimDir)
	switch {
	case err != nil:
		ok = false
		fmt.Println("WARN: cannot read shim dir:", err)
	case len(entries) == 0:
		fmt.Println("INFO: no shims installed")
	default:
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		sort.Strings(names)

		fmt.Println()
		fmt.Printf("Installed shims (%d):\n", len(names))
		for _, name := range names {
			dst := filepath.Join(shimDir, name)

			info, err := os.Lstat(dst)
			if err != nil {
				ok = false
				fmt.Printf("- %s: WARN: stat failed: %v\n", name, err)
				continue
			}
			if info.Mode()&os.ModeSymlink == 0 {
				ok = false
				fmt.Printf("- %s: WARN: not a symlink (expected symlink to ackchyually)\n", name)
				continue
			}

			target, err := os.Readlink(dst)
			if err != nil {
				ok = false
				fmt.Printf("- %s: WARN: readlink failed: %v\n", name, err)
				continue
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(dst), target)
			}
			target = filepath.Clean(target)

			if st, err := os.Stat(target); err != nil || st.IsDir() || st.Mode()&0o111 == 0 {
				ok = false
				fmt.Printf("- %s: WARN: broken shim target: %s\n", name, target)
				continue
			}

			realPath, err := execx.WhichSkippingShims(name)
			if err != nil {
				ok = false
				fmt.Printf("- %s: WARN: real tool not found in PATH (excluding shims)\n", name)
				continue
			}
			fmt.Printf("- %s: OK (real: %s)\n", name, realPath)
		}
	}

	if !ok {
		return 1
	}
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
