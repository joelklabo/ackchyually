package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/joelklabo/ackchyually/internal/execx"
	"github.com/joelklabo/ackchyually/internal/integrations/claude"
	"github.com/joelklabo/ackchyually/internal/integrations/codex"
	"github.com/joelklabo/ackchyually/internal/integrations/copilot"
)

func integrateVerifyImpl(target string) int {
	var failed bool
	switch target {
	case "codex":
		if !verifyCodex() {
			failed = true
		}
	case "claude":
		if !verifyClaude() {
			failed = true
		}
	case "copilot":
		if !verifyCopilot() {
			failed = true
		}
	case "all":
		if !verifyCodex() {
			failed = true
		}
		if !verifyClaude() {
			failed = true
		}
		if !verifyCopilot() {
			failed = true
		}
	default:
		fmt.Fprintf(os.Stderr, "integrate verify: unknown target %q\n", target)
		return 2
	}

	if failed {
		return 1
	}
	return 0
}

func verifyCodex() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shimDir := execx.ShimDir()

	st, err := codex.DetectStatus(ctx, "", shimDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "codex: error: %v\n", err)
		return false
	}
	if !st.Installed {
		fmt.Println("codex: not installed")
		return true
	}

	// Ensure the shim exists so the PATH check can be meaningful.
	shimName := "git"
	if runtime.GOOS == "windows" {
		shimName = "git.exe"
	}
	wantShim := filepath.Join(shimDir, shimName)
	if _, err := os.Stat(wantShim); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("codex: FAIL (git shim not installed)")
			fmt.Println("  fix: ackchyually shim install git")
			fmt.Println("  fix: ackchyually integrate codex")
			return false
		}
		fmt.Fprintf(os.Stderr, "codex: error: stat shim: %v\n", err)
		return false
	}

	if !st.Integrated {
		fmt.Println("codex: FAIL (not integrated)")
		fmt.Println("  fix: ackchyually integrate codex")
		fmt.Println("  fix: ackchyually integrate verify codex")
		return false
	}

	fmt.Println("codex: ok")
	return true
}

func verifyClaude() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shimDir := execx.ShimDir()
	st, err := claude.DetectStatus(ctx, "", shimDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claude: error: %v\n", err)
		return false
	}
	if !st.Installed {
		fmt.Println("claude: not installed")
		return true
	}
	if !st.Integrated {
		fmt.Println("claude: FAIL (not integrated)")
		fmt.Println("  fix: ackchyually integrate claude")
		fmt.Println("  fix: ackchyually integrate verify claude")
		return false
	}
	fmt.Println("claude: ok")
	return true
}

func verifyCopilot() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shimDir := execx.ShimDir()
	st, err := copilot.DetectStatus(ctx, shimDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "copilot: error: %v\n", err)
		return false
	}
	if !st.Installed {
		fmt.Println("copilot: not installed")
		return true
	}
	if !st.Integrated {
		fmt.Println("copilot: FAIL (wrapper not installed)")
		fmt.Println("  fix: ackchyually integrate copilot")
		fmt.Println("  fix: ackchyually integrate verify copilot")
		return false
	}
	fmt.Println("copilot: ok")
	return true
}
