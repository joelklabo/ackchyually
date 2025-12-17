package app

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joelklabo/ackchyually/internal/execx"
	"github.com/joelklabo/ackchyually/internal/integrations/agentclis"
	"github.com/joelklabo/ackchyually/internal/integrations/claude"
	"github.com/joelklabo/ackchyually/internal/integrations/codex"
	"github.com/joelklabo/ackchyually/internal/integrations/copilot"
)

func integrateCmd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "integrate: missing subcommand")
		return 2
	}

	switch args[0] {
	case "status":
		return integrateStatus(args[1:])
	case "codex", "claude", "copilot", "all":
		return integrateTool(args[0], args[1:])
	case "verify":
		return integrateVerify(args[1:])
	default:
		printUnknownSubcommand("integrate", args[0], []string{"status", "codex", "claude", "copilot", "all", "verify"})
		return 2
	}
}

func integrateStatus(args []string) int {
	fs := flag.NewFlagSet("integrate status", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output JSON (not yet implemented)")
	if err := parseFlags(fs, args); err != nil {
		return 2
	}

	if *jsonOut {
		fmt.Fprintln(os.Stderr, "integrate status: --json not implemented yet")
		return 2
	}

	manifest, err := agentclis.LoadManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "integrate status: supported versions manifest: %v\n", err)
		return 1
	}

	shimDir := execx.ShimDir()

	codexSt, err := codex.DetectStatus(context.Background(), "", shimDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integrate status: codex: %v\n", err)
		return 1
	}

	claudeSt, err := claude.DetectStatus(context.Background(), "", shimDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integrate status: claude: %v\n", err)
		return 1
	}

	copilotSt, err := copilot.DetectStatus(context.Background(), shimDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integrate status: copilot: %v\n", err)
		return 1
	}

	printToolStatus(manifest, toolStatus{
		ID:            "codex",
		Installed:     codexSt.Installed,
		VersionRaw:    codexSt.Version,
		Integrated:    codexSt.Integrated,
		LocationLabel: "config",
		LocationValue: pathWithMissingSuffix(codexSt.ConfigPath, codexSt.ConfigExists),
		FixCommand:    maybeFixCommand("codex", codexSt.Installed, codexSt.Integrated),
	})
	printToolStatus(manifest, toolStatus{
		ID:            "claude",
		Installed:     claudeSt.Installed,
		VersionRaw:    claudeSt.Version,
		Integrated:    claudeSt.Integrated,
		LocationLabel: "settings",
		LocationValue: pathWithMissingSuffix(claudeSt.SettingsPath, claudeSt.SettingsExists),
		FixCommand:    maybeFixCommand("claude", claudeSt.Installed, claudeSt.Integrated),
	})
	printToolStatus(manifest, toolStatus{
		ID:            "copilot",
		Installed:     copilotSt.Installed,
		VersionRaw:    copilotSt.Version,
		Integrated:    copilotSt.Integrated,
		LocationLabel: "wrapper",
		LocationValue: fallbackDash(copilotSt.WrapperPath),
		FixCommand:    maybeFixCommand("copilot", copilotSt.Installed, copilotSt.Integrated),
	})
	return 0
}

func integrateTool(tool string, args []string) int {
	fs := flag.NewFlagSet("integrate "+tool, flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "print planned changes without writing")
	undo := fs.Bool("undo", false, "undo ackchyually-managed integration changes")
	if err := parseFlags(fs, args); err != nil {
		return 2
	}

	switch tool {
	case "codex":
		return integrateCodex(*dryRun, *undo)
	case "claude":
		return integrateClaude(*dryRun, *undo)
	case "copilot":
		return integrateCopilot(*dryRun, *undo)
	default:
		fmt.Fprintf(os.Stderr, "integrate %s: not implemented yet\n", tool)
		return 2
	}
}

func integrateVerify(args []string) int {
	fs := flag.NewFlagSet("integrate verify", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output JSON (not yet implemented)")
	if err := parseFlags(fs, args); err != nil {
		return 2
	}

	if *jsonOut {
		fmt.Fprintln(os.Stderr, "integrate verify: --json not implemented yet")
		return 2
	}

	target := "all"
	if fs.NArg() > 0 {
		target = fs.Arg(0)
	}
	return integrateVerifyImpl(target)
}

func integrateCodex(dryRun, undo bool) int {
	if undo {
		plan, err := codex.PlanUndo("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "integrate codex: %v\n", err)
			return 1
		}
		if !plan.Changed {
			fmt.Println("codex: nothing to undo")
			return 0
		}
		if dryRun {
			fmt.Printf("codex: would undo changes in %s\n", plan.Path)
			return 0
		}
		if err := codex.Apply(plan); err != nil {
			fmt.Fprintf(os.Stderr, "integrate codex: %v\n", err)
			return 1
		}
		fmt.Printf("codex: undo applied to %s\n", plan.Path)
		return 0
	}

	shimDir := execx.ShimDir()
	plan, err := codex.PlanIntegrate("", shimDir, os.Getenv("PATH"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "integrate codex: %v\n", err)
		return 1
	}
	if !plan.Changed {
		fmt.Println("codex: already integrated")
		return 0
	}
	if dryRun {
		fmt.Printf("codex: would update %s\n", plan.Path)
		return 0
	}
	if err := codex.Apply(plan); err != nil {
		fmt.Fprintf(os.Stderr, "integrate codex: %v\n", err)
		return 1
	}
	fmt.Printf("codex: integrated (wrote %s)\n", plan.Path)
	return 0
}

func integrateClaude(dryRun, undo bool) int {
	if undo {
		plan, err := claude.PlanUndo("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "integrate claude: %v\n", err)
			return 1
		}
		if !plan.Changed {
			fmt.Println("claude: nothing to undo")
			return 0
		}
		if dryRun {
			fmt.Printf("claude: would undo changes in %s\n", plan.Path)
			return 0
		}
		if err := claude.Apply(plan); err != nil {
			fmt.Fprintf(os.Stderr, "integrate claude: %v\n", err)
			return 1
		}
		fmt.Printf("claude: undo applied to %s\n", plan.Path)
		return 0
	}

	shimDir := execx.ShimDir()
	plan, err := claude.PlanIntegrate("", shimDir, os.Getenv("PATH"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "integrate claude: %v\n", err)
		return 1
	}
	if !plan.Changed {
		fmt.Println("claude: already integrated")
		return 0
	}
	if dryRun {
		fmt.Printf("claude: would update %s\n", plan.Path)
		return 0
	}
	if err := claude.Apply(plan); err != nil {
		fmt.Fprintf(os.Stderr, "integrate claude: %v\n", err)
		return 1
	}
	fmt.Printf("claude: integrated (wrote %s)\n", plan.Path)
	return 0
}

func integrateCopilot(dryRun, undo bool) int {
	shimDir := execx.ShimDir()

	var (
		plan copilot.Plan
		err  error
	)
	if undo {
		plan, err = copilot.PlanUndo("")
	} else {
		plan, err = copilot.PlanInstall("", shimDir)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "integrate copilot: %v\n", err)
		return 1
	}

	if len(plan.Actions) == 0 {
		if undo {
			fmt.Println("copilot: nothing to undo")
		} else {
			fmt.Println("copilot: already integrated")
		}
		return 0
	}

	if dryRun {
		if undo {
			fmt.Println("copilot: would undo wrapper install")
		} else {
			fmt.Println("copilot: would install wrapper")
		}
		return 0
	}

	if err := copilot.Apply(plan); err != nil {
		fmt.Fprintf(os.Stderr, "integrate copilot: %v\n", err)
		return 1
	}

	if undo {
		fmt.Println("copilot: wrapper removed")
	} else {
		fmt.Println("copilot: wrapper installed")
	}
	return 0
}

type toolStatus struct {
	ID            string
	Installed     bool
	VersionRaw    string
	Integrated    bool
	LocationLabel string
	LocationValue string
	FixCommand    string
}

func printToolStatus(manifest agentclis.Manifest, st toolStatus) {
	installed := "no"
	if st.Installed {
		installed = "yes"
	}
	version := displayVersion(st.Installed, st.VersionRaw)

	integrated := "no"
	if st.Integrated {
		integrated = "yes"
	}

	supported := "-"
	supportedRange := ""
	docsURL := ""
	tool, ok := manifest.ToolByID(st.ID)
	if ok {
		supportedRange = formatSupportedRange(tool.SupportedRange)
		docsURL = strings.TrimSpace(tool.DocsURL)
	}

	if st.Installed && ok {
		res, err := tool.CheckInstalledVersion(st.VersionRaw)
		if err == nil {
			if res.Parseable {
				if res.WithinRange {
					supported = "yes"
				} else {
					supported = "no"
				}
			} else {
				supported = "?"
			}
		} else {
			supported = "?"
		}
	}

	fmt.Printf("%s: installed=%s version=%s supported=%s integrated=%s %s=%s\n", st.ID, installed, version, supported, integrated, st.LocationLabel, st.LocationValue)

	if st.FixCommand != "" {
		fmt.Printf("  fix: %s\n", st.FixCommand)
	}

	if supported == "no" {
		msg := "installed version is outside the supported range"
		if supportedRange != "" {
			msg += " (" + supportedRange + ")"
		}
		fmt.Printf("  warning: %s\n", msg)
		if docsURL != "" {
			fmt.Printf("  docs: %s\n", docsURL)
		}
		fmt.Printf("  docs: docs/agent_cli_supported_versions.md\n")
	}
}

func displayVersion(installed bool, raw string) string {
	raw = strings.TrimSpace(raw)
	switch {
	case !installed:
		return "-"
	case raw == "":
		return "?"
	default:
		return raw
	}
}

func maybeFixCommand(tool string, installed, integrated bool) string {
	if !installed || integrated {
		return ""
	}
	return "ackchyually integrate " + tool
}

func pathWithMissingSuffix(path string, exists bool) string {
	if path == "" {
		return "-"
	}
	if exists {
		return path
	}
	return path + " (missing)"
}

func fallbackDash(path string) string {
	if strings.TrimSpace(path) == "" {
		return "-"
	}
	return path
}

func formatSupportedRange(r agentclis.Range) string {
	minInclusive := strings.TrimSpace(r.MinInclusive)
	maxExclusive := strings.TrimSpace(r.MaxExclusive)
	switch {
	case minInclusive == "" && maxExclusive == "":
		return ""
	case minInclusive != "" && maxExclusive == "":
		return ">=" + minInclusive
	case minInclusive == "" && maxExclusive != "":
		return "<" + maxExclusive
	default:
		return ">=" + minInclusive + " <" + maxExclusive
	}
}
