package app

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func RunCLI(args []string) int {
	start := time.Now()
	code := runCLI(args)
	logCLIInvocation(start, time.Since(start), args, code)
	return code
}

func runCLI(args []string) int {
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
	case "integrate":
		return integrateCmd(args[1:])
	case "version":
		printVersion()
		return 0
	default:
		printUnknownCommand(args[0], []string{"shim", "best", "tag", "export", "integrate", "version"})
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
  integrate status
  integrate codex|claude|copilot|all [--dry-run] [--undo]
  integrate verify [codex|claude|copilot|all]

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
	if err := parseFlags(fs, args); err != nil {
		return 2
	}
	q := ""
	if fs.NArg() > 0 {
		q = strings.Join(fs.Args(), " ")
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
	if err := parseFlags(fs, args); err != nil {
		return 2
	}
	return exportImpl(*format, *tool)
}

func parseFlags(fs *flag.FlagSet, args []string) error {
	var flagArgs []string
	var posArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			posArgs = append(posArgs, args[i:]...)
			break
		}
		if strings.HasPrefix(arg, "-") {
			name := strings.TrimLeft(arg, "-")
			if idx := strings.Index(name, "="); idx != -1 {
				name = name[:idx]
			}

			f := fs.Lookup(name)
			if f == nil {
				// Unknown flag, treat as positional
				posArgs = append(posArgs, arg)
				continue
			}

			flagArgs = append(flagArgs, arg)

			// Check if it's a bool flag
			isBool := false
			if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
				isBool = bf.IsBoolFlag()
			}

			if !isBool && !strings.Contains(arg, "=") {
				if i+1 < len(args) {
					flagArgs = append(flagArgs, args[i+1])
					i++
				}
			}
		} else {
			posArgs = append(posArgs, arg)
		}
	}

	return fs.Parse(append(flagArgs, posArgs...))
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
