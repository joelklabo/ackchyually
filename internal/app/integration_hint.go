package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/joelklabo/ackchyually/internal/execx"
	"github.com/joelklabo/ackchyually/internal/integrations/agentclis"
	"github.com/joelklabo/ackchyually/internal/integrations/claude"
	"github.com/joelklabo/ackchyually/internal/integrations/codex"
	"github.com/joelklabo/ackchyually/internal/integrations/copilot"
)

const agentCLIHintMinInterval = 24 * time.Hour

func maybePrintAgentCLIHint(now time.Time) {
	if !isAgentCLIHintTTY() {
		return
	}
	maybePrintAgentCLIHintImpl(context.Background(), os.Stderr, now)
}

func isAgentCLIHintTTY() bool {
	return execx.IsTTY() && term.IsTerminal(int(os.Stderr.Fd()))
}

func agentCLIHintStatePath() string {
	// Keep this adjacent to the DB path (same base dir as shims).
	// Example: ~/.local/share/ackchyually/agent_cli_hint_last_shown
	return filepath.Join(filepath.Dir(execx.ShimDir()), "agent_cli_hint_last_shown")
}

func maybePrintAgentCLIHintImpl(ctx context.Context, w io.Writer, now time.Time) {
	statePath := agentCLIHintStatePath()
	if !shouldCheckAgentCLIHint(statePath, now) {
		return
	}

	needs, err := agentCLIIntegrationNeeds(ctx, execx.ShimDir())
	if err != nil || len(needs) == 0 {
		return
	}

	if err := writeAgentCLIHintState(statePath, now); err != nil {
		return
	}

	fmt.Fprintf(w, "ackchyually: tip: integrate %s so they use shims: ackchyually integrate all\n", strings.Join(needs, ", "))
}

func shouldCheckAgentCLIHint(statePath string, now time.Time) bool {
	b, err := os.ReadFile(statePath)
	if err != nil {
		return os.IsNotExist(err)
	}

	lastUnix, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return true
	}
	last := time.Unix(lastUnix, 0)
	return now.Sub(last) >= agentCLIHintMinInterval
}

func writeAgentCLIHintState(statePath string, now time.Time) error {
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(statePath, []byte(strconv.FormatInt(now.Unix(), 10)), 0o600)
}

func agentCLIIntegrationNeeds(ctx context.Context, shimDir string) ([]string, error) {
	manifest, err := agentclis.LoadManifest()
	if err != nil {
		return nil, err
	}

	codexSt, err := codex.DetectStatus(ctx, "", shimDir)
	if err != nil {
		return nil, err
	}

	claudeSt, err := claude.DetectStatus(ctx, "", shimDir)
	if err != nil {
		return nil, err
	}

	copilotSt, err := copilot.DetectStatus(ctx, shimDir)
	if err != nil {
		return nil, err
	}

	var needs []string

	if needsAgentCLIIntegration(manifest, "codex", codexSt.Installed, codexSt.Version, codexSt.Integrated) {
		needs = append(needs, "codex")
	}
	if needsAgentCLIIntegration(manifest, "claude", claudeSt.Installed, claudeSt.Version, claudeSt.Integrated) {
		needs = append(needs, "claude")
	}
	if needsAgentCLIIntegration(manifest, "copilot", copilotSt.Installed, copilotSt.Version, copilotSt.Integrated) {
		needs = append(needs, "copilot")
	}

	return needs, nil
}

func needsAgentCLIIntegration(manifest agentclis.Manifest, toolID string, installed bool, version string, integrated bool) bool {
	if !installed {
		return false
	}
	if !integrated {
		return true
	}
	tool, ok := manifest.ToolByID(toolID)
	if !ok {
		return false
	}
	res, err := tool.CheckInstalledVersion(version)
	if err != nil {
		return false
	}
	return res.Parseable && !res.WithinRange
}
