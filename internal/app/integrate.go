package app

import (
	"flag"
	"fmt"
	"os"
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
	_ = fs.Bool("json", false, "output JSON (not yet implemented)")
	if err := parseFlags(fs, args); err != nil {
		return 2
	}

	fmt.Fprintln(os.Stderr, "integrate status: not implemented yet")
	return 2
}

func integrateTool(tool string, args []string) int {
	fs := flag.NewFlagSet("integrate "+tool, flag.ContinueOnError)
	_ = fs.Bool("dry-run", false, "print planned changes without writing")
	_ = fs.Bool("undo", false, "undo ackchyually-managed integration changes")
	if err := parseFlags(fs, args); err != nil {
		return 2
	}

	fmt.Fprintf(os.Stderr, "integrate %s: not implemented yet\n", tool)
	return 2
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
