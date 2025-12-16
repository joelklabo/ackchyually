package app

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"

	"github.com/joelklabo/ackchyually/internal/ui"
)

func shimEnable(args []string) int {
	fs := flag.NewFlagSet("shim enable", flag.ContinueOnError)
	shell := fs.String("shell", "", "shell name (zsh|bash|fish); default: $SHELL")
	rcFile := fs.String("file", "", "rc file to edit (optional)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: ackchyually shim enable [--shell zsh|bash|fish] [--file <path>]")
		return 2
	}

	u := ui.New(os.Stdout)

	shimDir := shimDir()

	shellName := strings.TrimSpace(*shell)
	if shellName == "" {
		shellName = filepath.Base(strings.TrimSpace(os.Getenv("SHELL")))
	}

	path := strings.TrimSpace(*rcFile)
	if path == "" {
		var ok bool
		path, ok = defaultRCFile(shellName)
		if !ok {
			fmt.Fprintln(os.Stderr, "ackchyually: unsupported shell (set $SHELL or pass --shell):", shellName)
			fmt.Fprintf(os.Stderr, "ackchyually: add this to your shell rc file:\n  export PATH=\"%s:$PATH\"\n", shimDir)
			return 2
		}
	}
	if expanded, err := homedir.Expand(path); err == nil {
		path = expanded
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "ackchyually: cannot determine rc file path")
		return 1
	}

	snippet := enableSnippet(shellName, shimDir)
	if snippet == "" {
		fmt.Fprintln(os.Stderr, "ackchyually: unsupported shell:", shellName)
		return 2
	}

	existed := true
	content, mode, err := readFileWithMode(path)
	if err != nil {
		if os.IsNotExist(err) {
			existed = false
			content = ""
			mode = 0o644
		} else {
			fmt.Fprintln(os.Stderr, "ackchyually:", err)
			return 1
		}
	}

	if strings.Contains(content, "ackchyually/shims") || strings.Contains(content, "# ackchyually shims") {
		fmt.Printf("%s: already enabled in: %s\n", u.OK("OK"), path)
		return 0
	}

	newContent := content
	if newContent != "" && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	if existed {
		newContent += "\n"
	}
	newContent += snippet

	if err := writeFileAtomic(path, []byte(newContent), mode); err != nil {
		fmt.Fprintln(os.Stderr, "ackchyually:", err)
		return 1
	}

	fmt.Printf("%s: enabled shims in: %s\n", u.OK("OK"), path)
	fmt.Println()
	fmt.Println(u.Bold("Next:"))
	fmt.Printf("  source %s\n", path)
	fmt.Println("  hash -r 2>/dev/null || true")
	fmt.Println("  which <tool>")
	fmt.Printf("  # %s%c<tool>\n", shimDir, os.PathSeparator)
	return 0
}

func defaultRCFile(shellName string) (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	if home == "" {
		return "", false
	}

	switch shellName {
	case "zsh":
		return filepath.Join(home, ".zshrc"), true
	case "bash":
		return filepath.Join(home, ".bashrc"), true
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), true
	default:
		return "", false
	}
}

func enableSnippet(shellName, shimDir string) string {
	switch shellName {
	case "zsh", "bash":
		return fmt.Sprintf(`# ackchyually shims
export PATH="%s:$PATH"
# ackchyually shims end
`, shimDir)
	case "fish":
		return fmt.Sprintf(`# ackchyually shims
set -gx PATH "%s" $PATH
# ackchyually shims end
`, shimDir)
	default:
		return ""
	}
}

func readFileWithMode(path string) (content string, mode os.FileMode, err error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	return string(b), st.Mode() & os.ModePerm, nil
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".ackchyually-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
