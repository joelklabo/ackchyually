package app

import (
	"flag"
	"fmt"
	"os"
)

func RunCLI(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "shim":
		return shimCmd(args[1:])
	case "best":
		return bestCmd(args[1:])
	case "tag":
		return tagCmd(args[1:])
	case "export":
		return exportCmd(args[1:])
	case "version":
		fmt.Println("ackchyually dev")
		return 0
	default:
		printUnknownCommand(args[0], []string{"shim", "best", "tag", "export", "version"})
		return 2
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `ackchyually

Commands:
  shim install <tool...>
  shim list
  shim enable
  shim uninstall <tool...>
  shim doctor
  best --tool <tool> "<query>"
  tag add "<tag>" -- <command...>
  tag run "<tag>"
  export --format md|json [--tool <tool>]

Non-negotiable: PTY-first for interactive shells.
`)
}

func shimCmd(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "install":
		return shimInstall(args[1:])
	case "list":
		return shimList(args[1:])
	case "enable":
		return shimEnable(args[1:])
	case "uninstall":
		return shimUninstall(args[1:])
	case "doctor":
		return shimDoctor()
	default:
		printUnknownSubcommand("shim", args[0], []string{"install", "list", "enable", "uninstall", "doctor"})
		return 2
	}
}

func bestCmd(args []string) int {
	fs := flag.NewFlagSet("best", flag.ContinueOnError)
	tool := fs.String("tool", "", "tool name (required)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	q := ""
	if fs.NArg() > 0 {
		q = fs.Arg(0)
	}
	if *tool == "" {
		fmt.Fprintln(os.Stderr, "best: --tool is required")
		return 2
	}
	return bestImpl(*tool, q)
}

func exportCmd(args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	format := fs.String("format", "md", "md|json")
	tool := fs.String("tool", "", "tool name (optional)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	return exportImpl(*format, *tool)
}

func tagCmd(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "add":
		return tagAdd(args[1:])
	case "run":
		return tagRun(args[1:])
	default:
		printUnknownSubcommand("tag", args[0], []string{"add", "run"})
		return 2
	}
}
