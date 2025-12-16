package helpcount

import (
	"os"
	"path/filepath"
	"strings"
)

func structOutJSONScenario() Scenario {
	return Scenario{
		Name:        "structout_json_stdout_contains_usage_strings",
		Description: "Seed outputs JSON containing \"Usage:\"/\"unknown flag\" strings but is still a real success.",
		Tool:        "structout",
		Setup: func(env *Env) error {
			script := `#!/bin/sh
set -eu

case "${1:-}" in
  list)
    case "${2:-}" in
      --help|-h|help)
        echo "Usage: structout list [--json|--yaml]"
        exit 0
        ;;
      --json)
        echo '{"format":"json","notes":"Usage: not help","example":"unknown flag: --jsn"}'
        exit 0
        ;;
      --yaml)
        cat <<'EOF'
usage: this is a yaml key, not a help banner
format: yaml
items:
  - id: 1
    notes: "unknown flag: --jsn"
EOF
        exit 0
        ;;
      "")
        echo "format: default"
        exit 0
        ;;
      --jsn|--yml)
        echo "Error: unknown flag: ${2}" 1>&2
        echo "Usage: structout list [--json|--yaml]" 1>&2
        exit 2
        ;;
      *)
        echo "Error: unknown argument: ${2}" 1>&2
        echo "Usage: structout list [--json|--yaml]" 1>&2
        exit 2
        ;;
    esac
    ;;
  logs)
    case "${2:-}" in
      --help|-h|help)
        echo "Usage: structout logs --filter=<value>"
        exit 0
        ;;
      --filter=ok)
        echo "INFO: starting"
        echo "ERROR: previous run failed"
        echo "filter=ok"
        exit 0
        ;;
      --filter=noise)
        echo "INFO: starting"
        echo "filter=noise"
        exit 0
        ;;
      --flter=ok)
        echo "error: unknown option --flter (did you mean --filter?)" 1>&2
        echo "usage: structout logs --filter=<value>" 1>&2
        exit 2
        ;;
      *)
        echo "error: missing required --filter" 1>&2
        echo "usage: structout logs --filter=<value>" 1>&2
        exit 2
        ;;
    esac
    ;;
  *)
    echo "Usage: structout <list|logs> ..." 1>&2
    exit 2
    ;;
esac
`
			return writeToolScript(env, "structout", script)
		},
		Seed:  Command{Args: []string{"list", "--json"}},
		Noise: []Command{{Args: []string{"list"}}},
		Bad:   Command{Args: []string{"list", "--jsn"}}, //nolint:misspell // intentional typo scenario
		Help:  Command{Args: []string{"list", "--help"}},
		Expect: Expectation{
			FinalExitCode:           0,
			FinalStdoutContain:      `"format":"json"`,
			BaselineHelpInvocations: intPtr(1),
			MemoryHelpInvocations:   intPtr(0),
			BaselineSuggestionUsed:  boolPtr(false),
			MemorySuggestionUsed:    boolPtr(true),
		},
	}
}

func structOutYAMLScenario() Scenario {
	return Scenario{
		Name:        "structout_yaml_stdout_begins_with_usage_key",
		Description: "Seed outputs YAML with a leading \"usage:\" key but is still a real success.",
		Tool:        "structout",
		Setup: func(env *Env) error {
			// Reuse the same script as structOutJSONScenario.
			return structOutJSONScenario().Setup(env)
		},
		Seed:  Command{Args: []string{"list", "--yaml"}},
		Noise: []Command{{Args: []string{"list"}}},
		Bad:   Command{Args: []string{"list", "--yml"}}, //nolint:misspell // intentional typo scenario
		Help:  Command{Args: []string{"list", "--help"}},
		Expect: Expectation{
			FinalExitCode:           0,
			FinalStdoutContain:      "format: yaml",
			BaselineHelpInvocations: intPtr(1),
			MemoryHelpInvocations:   intPtr(0),
			BaselineSuggestionUsed:  boolPtr(false),
			MemorySuggestionUsed:    boolPtr(true),
		},
	}
}

func structOutLogsScenario() Scenario {
	return Scenario{
		Name:        "structout_logs_stdout_includes_error_prefix",
		Description: "Seed logs include an \"ERROR:\" line on stdout but should still count as a success.",
		Tool:        "structout",
		Setup: func(env *Env) error {
			// Reuse the same script as structOutJSONScenario.
			return structOutJSONScenario().Setup(env)
		},
		Seed:  Command{Args: []string{"logs", "--filter=ok"}},
		Noise: []Command{{Args: []string{"logs", "--filter=noise"}}},
		Bad:   Command{Args: []string{"logs", "--flter=ok"}}, //nolint:misspell // intentional typo scenario
		Help:  Command{Args: []string{"logs", "--help"}},
		Expect: Expectation{
			FinalExitCode:           0,
			FinalStdoutContain:      "filter=ok",
			BaselineHelpInvocations: intPtr(1),
			MemoryHelpInvocations:   intPtr(0),
			BaselineSuggestionUsed:  boolPtr(false),
			MemorySuggestionUsed:    boolPtr(true),
		},
	}
}

func ansiExit0Scenario() Scenario {
	return Scenario{
		Name:        "ansi_exit0_error_prefixes",
		Description: "Bad command prints ANSI-colored Error/Usage but exits 0; should still suggest known-good.",
		Tool:        "ansitool",
		Setup: func(env *Env) error {
			script := `#!/bin/sh
set -eu

case "${1:-}" in
  do)
    case "${2:-}" in
      --help|-h|help)
        echo "Usage: ansitool do --ok"
        exit 0
        ;;
      --ok)
        echo "OK ok=true"
        exit 0
        ;;
      --badd)
        printf '\033[31mError:\033[0m unknown flag: --badd\n' 1>&2
        printf '\033[1mUsage:\033[0m ansitool do --ok\n' 1>&2
        exit 0
        ;;
      *)
        echo "Error: unknown flag: ${2}" 1>&2
        echo "Usage: ansitool do --ok" 1>&2
        exit 2
        ;;
    esac
    ;;
  *)
    echo "Usage: ansitool do --ok" 1>&2
    exit 2
    ;;
esac
`
			return writeToolScript(env, "ansitool", script)
		},
		Seed: Command{Args: []string{"do", "--ok"}},
		Bad:  Command{Args: []string{"do", "--badd"}}, //nolint:misspell // intentional typo scenario
		Help: Command{Args: []string{"do", "--help"}},
		Expect: Expectation{
			BadExitCode:             intPtr(0),
			BadOutputContain:        "Usage:",
			FinalExitCode:           0,
			FinalStdoutContain:      "OK ok=true",
			BaselineHelpInvocations: intPtr(1),
			MemoryHelpInvocations:   intPtr(0),
			BaselineSuggestionUsed:  boolPtr(false),
			MemorySuggestionUsed:    boolPtr(true),
		},
	}
}

func clapUnexpectedArgScenario() Scenario {
	return Scenario{
		Name:        "clap_unexpected_argument_no_usage",
		Description: "Bad command prints clap-style \"unexpected argument\" without Usage banner; should still suggest known-good.",
		Tool:        "claptool",
		Setup: func(env *Env) error {
			script := `#!/bin/sh
set -eu

case "${1:-}" in
  run)
    case "${2:-}" in
      --help|-h|help)
        echo "Usage: claptool run [--json]"
        exit 0
        ;;
      --json)
        echo "OK format=json"
        exit 0
        ;;
      --jsn)
        echo "error: unexpected argument '--jsn' found" 1>&2
        exit 2
        ;;
      *)
        echo "error: unexpected argument '${2}' found" 1>&2
        exit 2
        ;;
    esac
    ;;
  *)
    echo "Usage: claptool run [--json]" 1>&2
    exit 2
    ;;
esac
`
			return writeToolScript(env, "claptool", script)
		},
		Seed: Command{Args: []string{"run", "--json"}},
		Bad:  Command{Args: []string{"run", "--jsn"}}, //nolint:misspell // intentional typo scenario
		Help: Command{Args: []string{"run", "--help"}},
		Expect: Expectation{
			FinalExitCode:           0,
			FinalStdoutContain:      "OK format=json",
			BaselineHelpInvocations: intPtr(1),
			MemoryHelpInvocations:   intPtr(0),
			BaselineSuggestionUsed:  boolPtr(false),
			MemorySuggestionUsed:    boolPtr(true),
		},
	}
}

func attachedShortFlagScenario() Scenario {
	return Scenario{
		Name:        "attached_short_flag_value_matches",
		Description: "Prefer the known-good command that used an attached -n1 value over a more recent `run` without -n.",
		Tool:        "attachttool",
		Setup: func(env *Env) error {
			script := `#!/bin/sh
set -eu

case "${1:-}" in
  run)
    if [ "${2:-}" = "--help" ] || [ "${2:-}" = "-h" ] || [ "${2:-}" = "help" ]; then
      echo "Usage: attachttool run [-nN]"
      exit 0
    fi

    if [ "${2:-}" = "-n1" ]; then
      echo "OK n=1"
      exit 0
    fi

    if [ "${2:-}" = "-n" ] && [ "${3:-}" = "1" ]; then
      if [ "${4:-}" = "--badflag" ]; then
        echo "error: unknown flag: --badflag" 1>&2
        echo "usage: attachttool run [-nN]" 1>&2
        exit 2
      fi
      echo "OK n=1"
      exit 0
    fi

    echo "OK n=default"
    exit 0
    ;;
  *)
    echo "Usage: attachttool run [-nN]" 1>&2
    exit 2
    ;;
esac
`
			return writeToolScript(env, "attachttool", script)
		},
		Seed:  Command{Args: []string{"run", "-n1"}},
		Noise: []Command{{Args: []string{"run"}}},
		Bad:   Command{Args: []string{"run", "-n", "1", "--badflag"}},
		Help:  Command{Args: []string{"run", "--help"}},
		Expect: Expectation{
			FinalExitCode:           0,
			FinalStdoutContain:      "OK n=1",
			BaselineHelpInvocations: intPtr(1),
			MemoryHelpInvocations:   intPtr(0),
			BaselineSuggestionUsed:  boolPtr(false),
			MemorySuggestionUsed:    boolPtr(true),
		},
	}
}

func flagTypoPrefersSpecificSuccessScenario() Scenario {
	return Scenario{
		Name:        "flag_typo_prefers_specific_success",
		Description: "When both plain and --json worked, a later --jsn typo should suggest the --json variant (even if last success was plain).",
		Tool:        "ranktool",
		Setup: func(env *Env) error {
			script := `#!/bin/sh
set -eu

case "${1:-}" in
  list)
    case "${2:-}" in
      --help|-h|help)
        echo "Usage: ranktool list [--json]"
        exit 0
        ;;
      --json)
        echo "OK format=json"
        exit 0
        ;;
      "")
        echo "OK format=plain"
        exit 0
        ;;
      --jsn)
        echo "Error: unknown flag: --jsn" 1>&2
        echo "Usage: ranktool list [--json]" 1>&2
        exit 2
        ;;
      *)
        echo "Error: unknown argument: ${2}" 1>&2
        echo "Usage: ranktool list [--json]" 1>&2
        exit 2
        ;;
    esac
    ;;
  --version|version|-v|-V)
    echo "ranktool 1.0.0"
    exit 0
    ;;
  *)
    echo "Usage: ranktool list [--json]" 1>&2
    exit 2
    ;;
esac
`
			return writeToolScript(env, "ranktool", script)
		},
		Seed:  Command{Args: []string{"list", "--json"}},
		Noise: []Command{{Args: []string{"list"}}},
		Bad:   Command{Args: []string{"list", "--jsn"}}, //nolint:misspell // intentional typo scenario
		Help:  Command{Args: []string{"list", "--help"}},
		Expect: Expectation{
			FinalExitCode:           0,
			FinalStdoutContain:      "OK format=json",
			BaselineHelpInvocations: intPtr(1),
			MemoryHelpInvocations:   intPtr(0),
			BaselineSuggestionUsed:  boolPtr(false),
			MemorySuggestionUsed:    boolPtr(true),
		},
	}
}

func writeToolScript(env *Env, tool, script string) error {
	bin := filepath.Join(env.WorkDir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil { //nolint:gosec
		return err
	}
	toolPath := filepath.Join(bin, tool)
	if err := os.WriteFile(toolPath, []byte(script), 0o755); err != nil { //nolint:gosec
		return err
	}

	env.basePath = strings.Join([]string{bin, env.basePath}, string(os.PathListSeparator))
	env.directEnv = upsertEnv(env.directEnv, "PATH", env.basePath)
	env.shimmedEnv = upsertEnv(env.shimmedEnv, "PATH", strings.Join([]string{env.ShimDir, env.basePath}, string(os.PathListSeparator)))
	return nil
}

func boolPtr(b bool) *bool { return &b }
