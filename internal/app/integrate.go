package app

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/joelklabo/ackchyually/internal/execx"
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

	printCodexStatus(codexSt)
	printClaudeStatus(claudeSt)
	printCopilotStatus(copilotSt)
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
	_ = fs.Bool("json", false, "output JSON (not yet implemented)")
	if err := parseFlags(fs, args); err != nil {
		return 2
	}

	target := "all"
	if fs.NArg() > 0 {
		target = fs.Arg(0)
	}
	switch target {
	case "all", "codex", "claude", "copilot":
	default:
		fmt.Fprintf(os.Stderr, "integrate verify: unknown target %q\n", target)
		return 2
	}

	fmt.Fprintf(os.Stderr, "integrate verify %s: not implemented yet\n", target)
	return 2
}

func printCodexStatus(st codex.Status) {
	installed := "no"
	if st.Installed {
		installed = "yes"
	}
	version := st.Version
	switch {
	case !st.Installed:
		version = "-"
	case version == "":
		version = "?"
	}

	integrated := "no"
	if st.Integrated {
		integrated = "yes"
	}
	config := st.ConfigPath
	if !st.ConfigExists {
		config += " (missing)"
	}

	fmt.Printf("codex: installed=%s version=%s integrated=%s config=%s\n", installed, version, integrated, config)
}

func printClaudeStatus(st claude.Status) {
	installed := "no"
	if st.Installed {
		installed = "yes"
	}
	version := st.Version
	switch {
	case !st.Installed:
		version = "-"
	case version == "":
		version = "?"
	}

	integrated := "no"
	if st.Integrated {
		integrated = "yes"
	}
	settings := st.SettingsPath
	if !st.SettingsExists {
		settings += " (missing)"
	}

	fmt.Printf("claude: installed=%s version=%s integrated=%s settings=%s\n", installed, version, integrated, settings)
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

func printCopilotStatus(st copilot.Status) {
	installed := "no"
	if st.Installed {
		installed = "yes"
	}
	version := st.Version
	switch {
	case !st.Installed:
		version = "-"
	case version == "":
		version = "?"
	}

	integrated := "no"
	if st.Integrated {
		integrated = "yes"
	}

	wrapper := st.WrapperPath
	if wrapper == "" {
		wrapper = "-"
	}

	fmt.Printf("copilot: installed=%s version=%s integrated=%s wrapper=%s\n", installed, version, integrated, wrapper)
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
