package helpcount

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func BuiltinScenarios() []Scenario {
	return []Scenario{
		fakeExitZeroUsageScenario(),
		gitLogSubjectScenario(),
		gitLogSubjectNoiseScenario(),
		gitLogSubjectConfusingNoiseScenario(),
		gitLogInvalidCountScenario(),
		gitLogUnknownDateFormatScenario(),
		gitLogDateFormatMissingColonScenario(),
		gitLogInvalidDecorateOptionScenario(),
		gitLogInvalidColorValueScenario(),
		gitLogInvalidPrettyFormatScenario(),
		gitLogInvalidDiffFilterScenario(),
		gitLogFollowRequiresPathspecScenario(),
		gitResetSoftWithPathsScenario(),
		gitSwitchConflictingCreateFlagsScenario(),
		gitSwitchOnlyOneReferenceExpectedScenario(),
		gitCheckoutDetachConflictsWithCreateScenario(),
		gitCheckoutNeedsPathsScenario(),
		gitRevParseMissingRevisionScenario(),
		gitShowUnknownRevisionScenario(),
		gitConfigWrongNumberOfArgsScenario(),
		gitConfigInvalidTypeScenario(),
		gitAddPathspecScenario(),
		gitBranchNameRequiredScenario(),
		gitStatusTypoScenario(),
		gitCommitMissingValueScenario(),
		gitDiffNameOnlyScenario(),
		gitDiffInvalidDiffAlgorithmScenario(),
		gitDiffInvalidStatValueScenario(),
		curlUnknownOptionScenario(),
		curlUnknownWriteOutVarScenario(),
		curlMissingOptionValueScenario(),
		curlInvalidURLScenario(),
		goEnvWriteKeyValueScenario(),
		goEnvWriteUnknownVarScenario(),
		goEnvWriteGopathRelativeScenario(),
		goTestCountScenario(),
		goTestUnknownFlagScenario(),
		goTestUnknownFlagNoiseScenario(),
		goTestInvalidRunRegexScenario(),
		goUnknownCommandScenario(),
		ghAliasSetMissingExpansionScenario(),
		ghConfigSetInvalidValueScenario(),
		ghConfigGetUnknownKeyScenario(),
		ghVersionTypoScenario(),
	}
}

func fakeExitZeroUsageScenario() Scenario {
	return Scenario{
		Name:        "fake_exit0_usage",
		Description: "A tool prints usage but exits 0 (seeded vs unseeded).",
		Tool:        "exit0tool",
		Setup: func(env *Env) error {
			bin := filepath.Join(env.WorkDir, "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				return err
			}

			toolPath := filepath.Join(bin, "exit0tool")
			script := `#!/bin/sh
set -eu

for a in "$@"; do
  case "$a" in
    -h|--help|-help|help)
      echo "usage: exit0tool log -1 --pretty=%s"
      exit 0
      ;;
  esac
done

case "${1:-}" in
  log)
    if [ "${2:-}" != "-1" ]; then
      echo "usage: exit0tool log -1 --pretty=%s" >&2
      exit 0
    fi

    case "${3:-}" in
      --pretty=%s)
        echo "OK"
        exit 0
        ;;
      --prety=%s)
        echo "error: unknown option prety=%s" >&2
        echo "usage: exit0tool log -1 --pretty=%s" >&2
        exit 0
        ;;
      *)
        echo "usage: exit0tool log -1 --pretty=%s" >&2
        exit 0
        ;;
    esac
    ;;
  --version|version|-v|-V)
    echo "exit0tool 1.0.0"
    exit 0
    ;;
  *)
    echo "usage: exit0tool log -1 --pretty=%s" >&2
    exit 0
    ;;
esac
`
			if err := os.WriteFile(toolPath, []byte(script), 0o755); err != nil {
				return err
			}

			env.basePath = strings.Join([]string{bin, env.basePath}, string(os.PathListSeparator))
			env.directEnv = upsertEnv(env.directEnv, "PATH", env.basePath)
			env.shimmedEnv = upsertEnv(env.shimmedEnv, "PATH", strings.Join([]string{env.ShimDir, env.basePath}, string(os.PathListSeparator)))
			return nil
		},
		Seed: Command{Args: []string{"log", "-1", "--pretty=%s"}},
		Bad:  Command{Args: []string{"log", "-1", "--prety=%s"}}, //nolint:misspell // intentional typo scenario
		Help: Command{Args: []string{"log", "-h"}},
		Expect: Expectation{
			BadExitCode:        intPtr(0),
			BadOutputContain:   "usage:",
			FinalExitCode:      0,
			FinalStdoutContain: "OK",
		},
	}
}

func gitLogSubjectScenario() Scenario {
	return Scenario{
		Name:        "git_log_subject",
		Description: "Get last commit subject (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.email", "eval@example.com"); err != nil {
				return fmt.Errorf("git config user.email: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.name", "Eval"); err != nil {
				return fmt.Errorf("git config user.name: %w", err)
			}

			if err := os.WriteFile(filepath.Join(repo, "hello.txt"), []byte("hello\n"), 0o644); err != nil {
				return err
			}
			if _, err := env.RunDirect("git", "add", "."); err != nil {
				return fmt.Errorf("git add: %w", err)
			}
			if _, err := env.RunDirect("git", "commit", "-m", "hello from eval", "-q"); err != nil {
				return fmt.Errorf("git commit: %w", err)
			}
			return nil
		},
		Seed: Command{Args: []string{"log", "-1", "--pretty=%s"}},
		Bad:  Command{Args: []string{"log", "-1", "--prety=%s"}},
		Help: Command{Args: []string{"log", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "hello from eval",
		},
	}
}

func gitLogSubjectNoiseScenario() Scenario {
	s := gitLogSubjectScenario()
	s.Name = "git_log_subject_noise"
	s.Description = "Get last commit subject with intervening noise (seeded vs unseeded)."
	s.Noise = []Command{
		{Args: []string{"status"}},
	}
	return s
}

func gitLogSubjectConfusingNoiseScenario() Scenario {
	s := gitLogSubjectScenario()
	s.Name = "git_log_subject_confusing_noise"
	s.Description = "Get last commit subject with a more similar intervening success (seeded vs unseeded)."
	s.Noise = []Command{
		{Args: []string{"log", "-1", "--pretty=%h"}},
	}
	return s
}

func gitLogInvalidCountScenario() Scenario {
	s := gitLogSubjectScenario()
	s.Name = "git_log_invalid_count"
	s.Description = "Run git log with a non-integer -n value (seeded vs unseeded)."
	s.Bad = Command{Args: []string{"log", "-n", "abc", "-1", "--pretty=%s"}}
	s.Help = Command{Args: []string{"log", "-h"}}
	return s
}

func gitLogUnknownDateFormatScenario() Scenario {
	s := gitLogSubjectScenario()
	s.Name = "git_log_unknown_date_format"
	s.Description = "Run git log with an unknown --date format value (seeded vs unseeded)."
	s.Seed = Command{Args: []string{"log", "-1", "--date=short", "--pretty=%s"}}
	s.Bad = Command{Args: []string{"log", "-1", "--date=bananas", "--pretty=%s"}}
	s.Help = Command{Args: []string{"log", "-h"}}
	return s
}

func gitLogDateFormatMissingColonScenario() Scenario {
	s := gitLogSubjectScenario()
	s.Name = "git_log_date_format_missing_colon"
	s.Description = "Run git log with --date=format missing a colon separator (seeded vs unseeded)."
	s.Seed = Command{Args: []string{"log", "-1", "--date=format:%Y-%m-%d", "--pretty=%s"}}
	s.Bad = Command{Args: []string{"log", "-1", "--date=format", "--pretty=%s"}}
	s.Help = Command{Args: []string{"log", "-h"}}
	return s
}

func gitLogInvalidDecorateOptionScenario() Scenario {
	s := gitLogSubjectScenario()
	s.Name = "git_log_invalid_decorate_option"
	s.Description = "Run git log with an invalid --decorate option value (seeded vs unseeded)."
	s.Seed = Command{Args: []string{"log", "-1", "--decorate=short", "--pretty=%s"}}
	s.Bad = Command{Args: []string{"log", "-1", "--decorate=bananas", "--pretty=%s"}}
	s.Help = Command{Args: []string{"log", "-h"}}
	return s
}

func gitLogInvalidColorValueScenario() Scenario {
	s := gitLogSubjectScenario()
	s.Name = "git_log_invalid_color_value"
	s.Description = "Run git log with an invalid --color value (seeded vs unseeded)."
	s.Seed = Command{Args: []string{"log", "-1", "--color=always", "--pretty=%s"}}
	s.Bad = Command{Args: []string{"log", "-1", "--color=banana", "--pretty=%s"}}
	s.Help = Command{Args: []string{"log", "-h"}}
	return s
}

func gitLogInvalidPrettyFormatScenario() Scenario {
	s := gitLogSubjectScenario()
	s.Name = "git_log_invalid_pretty_format"
	s.Description = "Run git log with an invalid --pretty format value (seeded vs unseeded)."
	s.Seed = Command{Args: []string{"log", "-1", "--pretty=%s"}}
	s.Bad = Command{Args: []string{"log", "-1", "--pretty=foo"}}
	s.Help = Command{Args: []string{"log", "-h"}}
	return s
}

func gitLogInvalidDiffFilterScenario() Scenario {
	s := gitLogSubjectScenario()
	s.Name = "git_log_invalid_diff_filter"
	s.Description = "Run git log with an invalid --diff-filter value (seeded vs unseeded)."
	s.Seed = Command{Args: []string{"log", "-1", "--diff-filter=A", "--pretty=%s"}}
	s.Bad = Command{Args: []string{"log", "-1", "--diff-filter=Z", "--pretty=%s"}}
	s.Help = Command{Args: []string{"log", "-h"}}
	return s
}

func gitLogFollowRequiresPathspecScenario() Scenario {
	s := gitLogSubjectScenario()
	s.Name = "git_log_follow_requires_pathspec"
	s.Description = "Run git log --follow without a pathspec (seeded vs unseeded)."
	s.Seed = Command{Args: []string{"log", "-1", "--follow", "--pretty=%s", "--", "hello.txt"}}
	s.Bad = Command{Args: []string{"log", "-1", "--follow", "--pretty=%s"}}
	s.Help = Command{Args: []string{"log", "-h"}}
	return s
}

func gitResetSoftWithPathsScenario() Scenario {
	s := gitLogSubjectScenario()
	s.Name = "git_reset_soft_with_paths"
	s.Description = "Run git reset --soft with a pathspec (seeded vs unseeded)."
	s.Seed = Command{Args: []string{"reset", "--soft", "HEAD"}}
	s.Bad = Command{Args: []string{"reset", "--soft", "HEAD", "--", "hello.txt"}}
	s.Help = Command{Args: []string{"reset", "-h"}}
	s.Expect = Expectation{FinalExitCode: 0}
	return s
}

func gitSwitchConflictingCreateFlagsScenario() Scenario {
	return Scenario{
		Name:        "git_switch_conflicting_create_flags",
		Description: "Run git switch with conflicting create flags (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			return nil
		},
		Seed: Command{Args: []string{"switch", "-C", "foo"}},
		Bad:  Command{Args: []string{"switch", "-c", "foo", "-C", "bar"}},
		Help: Command{Args: []string{"switch", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "Switched to a new branch 'foo'",
		},
	}
}

func gitSwitchOnlyOneReferenceExpectedScenario() Scenario {
	return Scenario{
		Name:        "git_switch_only_one_reference_expected",
		Description: "Run git switch with too many references (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			return nil
		},
		Seed: Command{Args: []string{"switch", "-C", "foo"}},
		Bad:  Command{Args: []string{"switch", "foo", "bar"}},
		Help: Command{Args: []string{"switch", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "branch 'foo'",
		},
	}
}

func gitCheckoutDetachConflictsWithCreateScenario() Scenario {
	return Scenario{
		Name:        "git_checkout_detach_conflicts_with_create",
		Description: "Run git checkout with --detach and -b/-B flags together (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			return nil
		},
		Seed: Command{Args: []string{"checkout", "-B", "foo"}},
		Bad:  Command{Args: []string{"checkout", "-B", "foo", "--detach"}},
		Help: Command{Args: []string{"checkout", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "Switched to a new branch 'foo'",
		},
	}
}

func gitCheckoutNeedsPathsScenario() Scenario {
	return Scenario{
		Name:        "git_checkout_needs_paths",
		Description: "Run git checkout --ours/--theirs without paths (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q", "-b", "main"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.email", "eval@example.com"); err != nil {
				return fmt.Errorf("git config user.email: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.name", "Eval"); err != nil {
				return fmt.Errorf("git config user.name: %w", err)
			}

			if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("base\n"), 0o644); err != nil {
				return err
			}
			if _, err := env.RunDirect("git", "add", "a.txt"); err != nil {
				return fmt.Errorf("git add: %w", err)
			}
			if _, err := env.RunDirect("git", "commit", "-m", "base", "-q"); err != nil {
				return fmt.Errorf("git commit: %w", err)
			}

			if _, err := env.RunDirect("git", "checkout", "-b", "feature", "-q"); err != nil {
				return fmt.Errorf("git checkout -b feature: %w", err)
			}
			if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("feature\n"), 0o644); err != nil {
				return err
			}
			if _, err := env.RunDirect("git", "commit", "-am", "feature", "-q"); err != nil {
				return fmt.Errorf("git commit feature: %w", err)
			}

			if _, err := env.RunDirect("git", "checkout", "main", "-q"); err != nil {
				return fmt.Errorf("git checkout main: %w", err)
			}
			if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("main\n"), 0o644); err != nil {
				return err
			}
			if _, err := env.RunDirect("git", "commit", "-am", "main", "-q"); err != nil {
				return fmt.Errorf("git commit main: %w", err)
			}

			mergeOut, mergeErr := env.RunDirect("git", "merge", "feature")
			if mergeErr == nil {
				return fmt.Errorf("expected merge conflict, but merge succeeded:\n%s", mergeOut)
			}

			statusOut, statusErr := env.RunDirect("git", "status", "--porcelain")
			if statusErr != nil {
				return fmt.Errorf("git status: %w", statusErr)
			}
			if !strings.Contains(statusOut, "UU a.txt") {
				return fmt.Errorf("expected merge conflict, but status missing 'UU a.txt':\n%s\nmerge output:\n%s", statusOut, mergeOut)
			}
			return nil
		},
		Seed:   Command{Args: []string{"checkout", "--ours", "--", "a.txt"}},
		Bad:    Command{Args: []string{"checkout", "--ours", "--theirs"}},
		Help:   Command{Args: []string{"checkout", "-h"}},
		Expect: Expectation{FinalExitCode: 0},
	}
}

func gitRevParseMissingRevisionScenario() Scenario {
	return Scenario{
		Name:        "git_rev_parse_missing_revision",
		Description: "Run git rev-parse --verify without a revision (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q", "-b", "main"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.email", "eval@example.com"); err != nil {
				return fmt.Errorf("git config user.email: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.name", "Eval"); err != nil {
				return fmt.Errorf("git config user.name: %w", err)
			}

			if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
				return err
			}
			if _, err := env.RunDirect("git", "add", "."); err != nil {
				return fmt.Errorf("git add: %w", err)
			}
			if _, err := env.RunDirect("git", "commit", "-m", "seed", "-q"); err != nil {
				return fmt.Errorf("git commit: %w", err)
			}
			return nil
		},
		Seed: Command{Args: []string{"rev-parse", "--verify", "--symbolic-full-name", "HEAD"}},
		Bad:  Command{Args: []string{"rev-parse", "--verify", "--symbolic-full-name"}},
		Help: Command{Args: []string{"rev-parse", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "refs/heads/main",
		},
	}
}

func gitShowUnknownRevisionScenario() Scenario {
	return Scenario{
		Name:        "git_show_unknown_revision",
		Description: "Run git show with an unknown revision (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q", "-b", "main"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.email", "eval@example.com"); err != nil {
				return fmt.Errorf("git config user.email: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.name", "Eval"); err != nil {
				return fmt.Errorf("git config user.name: %w", err)
			}

			if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
				return err
			}
			if _, err := env.RunDirect("git", "add", "."); err != nil {
				return fmt.Errorf("git add: %w", err)
			}
			if _, err := env.RunDirect("git", "commit", "-m", "seed", "-q"); err != nil {
				return fmt.Errorf("git commit: %w", err)
			}
			return nil
		},
		Seed: Command{Args: []string{"show", "-s", "--pretty=%s", "HEAD"}},
		Bad:  Command{Args: []string{"show", "-s", "--pretty=%s", "HEA"}},
		Help: Command{Args: []string{"show", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "seed",
		},
	}
}

func curlUnknownOptionScenario() Scenario {
	return Scenario{
		Name:        "curl_unknown_option",
		Description: "Run curl with an unknown option (seeded vs unseeded).",
		Tool:        "curl",
		Setup:       func(_ *Env) error { return nil },
		Seed:        Command{Args: []string{"-fsSL", "-o", "/dev/null", "-w", "ok", "file:///etc/hosts"}},
		Bad:         Command{Args: []string{"--fial", "-fsSL", "-o", "/dev/null", "-w", "ok", "file:///etc/hosts"}}, //nolint:misspell // intentional typo scenario
		Help:        Command{Args: []string{"--help"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "ok",
		},
	}
}

func curlUnknownWriteOutVarScenario() Scenario {
	return Scenario{
		Name:        "curl_unknown_write_out_var",
		Description: "Run curl with an unknown --write-out variable (curl exits 0; seeded vs unseeded).",
		Tool:        "curl",
		Setup:       func(_ *Env) error { return nil },
		Seed:        Command{Args: []string{"-fsSL", "-o", "/dev/null", "-w", "ok", "file:///etc/hosts"}},
		Bad:         Command{Args: []string{"-fsSL", "-o", "/dev/null", "-w", "%{fial}", "file:///etc/hosts"}}, //nolint:misspell // intentional typo scenario
		Help:        Command{Args: []string{"--help"}},
		Expect: Expectation{
			BadExitCode:        intPtr(0),
			BadOutputContain:   "unknown --write-out variable",
			FinalExitCode:      0,
			FinalStdoutContain: "ok",
		},
	}
}

func curlMissingOptionValueScenario() Scenario {
	return Scenario{
		Name:        "curl_missing_option_value",
		Description: "Run curl with a missing option value (seeded vs unseeded).",
		Tool:        "curl",
		Setup:       func(_ *Env) error { return nil },
		Seed:        Command{Args: []string{"-fsSL", "-o", "/dev/null", "-w", "ok", "file:///etc/hosts"}},
		Bad:         Command{Args: []string{"-fsSL", "-o", "/dev/null", "-w", "ok", "file:///etc/hosts", "-X"}},
		Help:        Command{Args: []string{"--help"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "ok",
		},
	}
}

func intPtr(v int) *int { return &v }

func curlInvalidURLScenario() Scenario {
	return Scenario{
		Name:        "curl_invalid_url",
		Description: "Run curl with a malformed URL (seeded vs unseeded).",
		Tool:        "curl",
		Setup:       func(_ *Env) error { return nil },
		Seed:        Command{Args: []string{"-fsSL", "-o", "/dev/null", "-w", "ok", "file:///etc/hosts"}},
		Bad:         Command{Args: []string{"-fsSL", "-o", "/dev/null", "-w", "ok", "http://"}},
		Help:        Command{Args: []string{"--help"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "ok",
		},
	}
}

func gitDiffNameOnlyScenario() Scenario {
	return Scenario{
		Name:        "git_diff_name_only",
		Description: "List modified files (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.email", "eval@example.com"); err != nil {
				return fmt.Errorf("git config user.email: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.name", "Eval"); err != nil {
				return fmt.Errorf("git config user.name: %w", err)
			}

			if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
				return err
			}
			if _, err := env.RunDirect("git", "add", "."); err != nil {
				return fmt.Errorf("git add: %w", err)
			}
			if _, err := env.RunDirect("git", "commit", "-m", "seed", "-q"); err != nil {
				return fmt.Errorf("git commit: %w", err)
			}

			if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a2\n"), 0o644); err != nil {
				return err
			}
			return nil
		},
		Seed: Command{Args: []string{"diff", "--name-only"}},
		Bad:  Command{Args: []string{"diff", "--name-onlyy"}},
		Help: Command{Args: []string{"diff", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "a.txt",
		},
	}
}

func gitDiffInvalidDiffAlgorithmScenario() Scenario {
	s := gitDiffNameOnlyScenario()
	s.Name = "git_diff_invalid_diff_algorithm"
	s.Description = "Run git diff with an invalid --diff-algorithm value (seeded vs unseeded)."
	s.Seed = Command{Args: []string{"diff", "--diff-algorithm=myers", "--name-only"}}
	s.Bad = Command{Args: []string{"diff", "--diff-algorithm=banana", "--name-only"}}
	s.Help = Command{Args: []string{"diff", "-h"}}
	return s
}

func gitDiffInvalidStatValueScenario() Scenario {
	s := gitDiffNameOnlyScenario()
	s.Name = "git_diff_invalid_stat_value"
	s.Description = "Run git diff with an invalid --stat value (seeded vs unseeded)."
	s.Seed = Command{Args: []string{"diff", "--stat"}}
	s.Bad = Command{Args: []string{"diff", "--stat=foo"}}
	s.Help = Command{Args: []string{"diff", "-h"}}
	return s
}

func gitCommitMissingValueScenario() Scenario {
	return Scenario{
		Name:        "git_commit_missing_value",
		Description: "Commit with a missing -m value (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.email", "eval@example.com"); err != nil {
				return fmt.Errorf("git config user.email: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.name", "Eval"); err != nil {
				return fmt.Errorf("git config user.name: %w", err)
			}
			return nil
		},
		Seed: Command{Args: []string{"commit", "--allow-empty", "-m", "seed from eval"}},
		Bad:  Command{Args: []string{"commit", "-m"}},
		Help: Command{Args: []string{"commit", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "seed from eval",
		},
	}
}

func gitConfigWrongNumberOfArgsScenario() Scenario {
	return Scenario{
		Name:        "git_config_wrong_number_of_args",
		Description: "Run git config with a missing argument (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q", "-b", "main"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.email", "eval@example.com"); err != nil {
				return fmt.Errorf("git config user.email: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.name", "Eval"); err != nil {
				return fmt.Errorf("git config user.name: %w", err)
			}
			return nil
		},
		Seed: Command{Args: []string{"config", "user.name"}},
		Bad:  Command{Args: []string{"config", "--add", "user.name"}},
		Help: Command{Args: []string{"config", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "Eval",
		},
	}
}

func gitConfigInvalidTypeScenario() Scenario {
	s := gitConfigWrongNumberOfArgsScenario()
	s.Name = "git_config_invalid_type"
	s.Description = "Run git config with an invalid --type value (seeded vs unseeded)."
	s.Bad = Command{Args: []string{"config", "--type=banana", "user.name"}}
	s.Help = Command{Args: []string{"config", "-h"}}
	return s
}

func gitAddPathspecScenario() Scenario {
	return Scenario{
		Name:        "git_add_pathspec",
		Description: "Run git add with a pathspec typo (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q", "-b", "main"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
				return err
			}
			return nil
		},
		Seed: Command{Args: []string{"add", "-n", "a.txt"}},
		Bad:  Command{Args: []string{"add", "-n", "a.tx"}},
		Help: Command{Args: []string{"add", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "add 'a.txt'",
		},
	}
}

func gitBranchNameRequiredScenario() Scenario {
	return Scenario{
		Name:        "git_branch_name_required",
		Description: "Run git branch -D with a missing branch name (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q", "-b", "main"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.email", "eval@example.com"); err != nil {
				return fmt.Errorf("git config user.email: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.name", "Eval"); err != nil {
				return fmt.Errorf("git config user.name: %w", err)
			}

			if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
				return err
			}
			if _, err := env.RunDirect("git", "add", "."); err != nil {
				return fmt.Errorf("git add: %w", err)
			}
			if _, err := env.RunDirect("git", "commit", "-m", "seed", "-q"); err != nil {
				return fmt.Errorf("git commit: %w", err)
			}
			if _, err := env.RunDirect("git", "branch", "foo"); err != nil {
				return fmt.Errorf("git branch foo: %w", err)
			}
			return nil
		},
		Seed: Command{Args: []string{"branch", "-D", "foo"}},
		Noise: []Command{
			{Args: []string{"branch", "foo"}},
		},
		Bad:  Command{Args: []string{"branch", "-D"}},
		Help: Command{Args: []string{"branch", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "Deleted branch foo",
		},
	}
}

func gitStatusTypoScenario() Scenario {
	return Scenario{
		Name:        "git_status_typo",
		Description: "Run git status with a subcommand typo (seeded vs unseeded).",
		Tool:        "git",
		Setup: func(env *Env) error {
			repo := filepath.Join(env.WorkDir, "repo")
			if err := os.MkdirAll(repo, 0o755); err != nil {
				return err
			}
			env.WorkDir = repo

			if _, err := env.RunDirect("git", "init", "-q", "-b", "main"); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.email", "eval@example.com"); err != nil {
				return fmt.Errorf("git config user.email: %w", err)
			}
			if _, err := env.RunDirect("git", "config", "user.name", "Eval"); err != nil {
				return fmt.Errorf("git config user.name: %w", err)
			}

			if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
				return err
			}
			if _, err := env.RunDirect("git", "add", "."); err != nil {
				return fmt.Errorf("git add: %w", err)
			}
			if _, err := env.RunDirect("git", "commit", "-m", "seed", "-q"); err != nil {
				return fmt.Errorf("git commit: %w", err)
			}
			return nil
		},
		Seed: Command{Args: []string{"status"}},
		Bad:  Command{Args: []string{"stauts"}},
		Help: Command{Args: []string{"status", "-h"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "working tree clean",
		},
	}
}

func goTestCountScenario() Scenario {
	return Scenario{
		Name:        "go_test_count_flag",
		Description: "Run go test with a common flag parsing mistake (seeded vs unseeded).",
		Tool:        "go",
		Setup:       setupGoTestModule,
		Seed:        Command{Args: []string{"test", "./...", "-count=1"}},
		Bad:         Command{Args: []string{"test", "./...", "-count", "one"}},
		Help:        Command{Args: []string{"help", "testflag"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "example.com/ackchyually-eval",
		},
	}
}

func goEnvWriteKeyValueScenario() Scenario {
	return Scenario{
		Name:        "go_env_write_key_value",
		Description: "Run go env -w with a missing KEY=VALUE (seeded vs unseeded).",
		Tool:        "go",
		Setup:       func(_ *Env) error { return nil },
		Seed:        Command{Args: []string{"env", "-w", "GOPATH=/tmp/ackchyually-eval-gopath"}},
		Bad:         Command{Args: []string{"env", "-w", "GOPATH"}},
		Help:        Command{Args: []string{"help", "env"}},
		Expect: Expectation{
			FinalExitCode: 0,
		},
	}
}

func goEnvWriteUnknownVarScenario() Scenario {
	return Scenario{
		Name:        "go_env_write_unknown_var",
		Description: "Run go env -w with an unknown go command variable (seeded vs unseeded).",
		Tool:        "go",
		Setup:       func(_ *Env) error { return nil },
		Seed:        Command{Args: []string{"env", "-w", "GOPATH=/tmp/ackchyually-eval-gopath"}},
		Bad:         Command{Args: []string{"env", "-w", "FOO=bar"}},
		Help:        Command{Args: []string{"help", "env"}},
		Expect: Expectation{
			FinalExitCode: 0,
		},
	}
}

func goEnvWriteGopathRelativeScenario() Scenario {
	return Scenario{
		Name:        "go_env_write_gopath_relative",
		Description: "Run go env -w GOPATH with a relative path value (seeded vs unseeded).",
		Tool:        "go",
		Setup:       func(_ *Env) error { return nil },
		Seed:        Command{Args: []string{"env", "-w", "GOPATH=/tmp/ackchyually-eval-gopath"}},
		Bad:         Command{Args: []string{"env", "-w", "GOPATH=."}},
		Help:        Command{Args: []string{"help", "env"}},
		Expect: Expectation{
			FinalExitCode: 0,
		},
	}
}

func goTestUnknownFlagScenario() Scenario {
	return Scenario{
		Name:        "go_test_unknown_flag",
		Description: "Run go test with an unknown flag (usage printed to stdout).",
		Tool:        "go",
		Setup:       setupGoTestModule,
		Seed:        Command{Args: []string{"test", "./...", "-count=1"}},
		Bad:         Command{Args: []string{"test", "./...", "-cunt=1"}},
		Help:        Command{Args: []string{"help", "testflag"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "example.com/ackchyually-eval",
		},
	}
}

func goTestUnknownFlagNoiseScenario() Scenario {
	s := goTestUnknownFlagScenario()
	s.Name = "go_test_unknown_flag_noise"
	s.Description = "Run go test with an unknown flag and intervening noise (seeded vs unseeded)."
	s.Noise = []Command{
		{Args: []string{"env", "GOPATH"}},
	}
	return s
}

func goTestInvalidRunRegexScenario() Scenario {
	return Scenario{
		Name:        "go_test_invalid_run_regex",
		Description: "Run go test with an invalid -run regexp (seeded vs unseeded).",
		Tool:        "go",
		Setup:       setupGoTestModule,
		Seed:        Command{Args: []string{"test", "./...", "-run", "TestOK"}},
		Bad:         Command{Args: []string{"test", "./...", "-run", "["}},
		Help:        Command{Args: []string{"help", "testflag"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "example.com/ackchyually-eval",
		},
	}
}

func goUnknownCommandScenario() Scenario {
	return Scenario{
		Name:        "go_unknown_command",
		Description: "Run go with an unknown subcommand (seeded vs unseeded).",
		Tool:        "go",
		Setup:       setupGoTestModule,
		Seed:        Command{Args: []string{"test", "./...", "-count=1"}},
		Bad:         Command{Args: []string{"tset"}},
		Help:        Command{Args: []string{"help"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "example.com/ackchyually-eval",
		},
	}
}

func setupGoTestModule(env *Env) error {
	mod := filepath.Join(env.WorkDir, "gomod")
	if err := os.MkdirAll(mod, 0o755); err != nil {
		return err
	}
	env.WorkDir = mod

	if err := os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module example.com/ackchyually-eval\ngo 1.24.0\n"), 0o644); err != nil {
		return err
	}
	const testSrc = `package main

import "testing"

func TestOK(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(mod, "ok_test.go"), []byte(testSrc), 0o644); err != nil {
		return err
	}
	return nil
}

func ghVersionTypoScenario() Scenario {
	return Scenario{
		Name:        "gh_version_typo",
		Description: "Run gh version with a subcommand typo (seeded vs unseeded).",
		Tool:        "gh",
		Setup: func(_ *Env) error {
			return nil
		},
		Seed: Command{Args: []string{"version"}},
		Bad:  Command{Args: []string{"verison"}}, //nolint:misspell // intentional typo scenario
		Help: Command{Args: []string{"--help"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "gh version",
		},
	}
}

func ghAliasSetMissingExpansionScenario() Scenario {
	return Scenario{
		Name:        "gh_alias_set_missing_expansion",
		Description: "Run gh alias set with a missing expansion argument (seeded vs unseeded).",
		Tool:        "gh",
		Setup:       func(_ *Env) error { return nil },
		Seed:        Command{Args: []string{"alias", "set", "--clobber", "vv", "version"}},
		Bad:         Command{Args: []string{"alias", "set", "vv"}},
		Help:        Command{Args: []string{"alias", "set", "--help"}},
		Expect: Expectation{
			FinalExitCode: 0,
		},
	}
}

func ghConfigSetInvalidValueScenario() Scenario {
	return Scenario{
		Name:        "gh_config_set_invalid_value",
		Description: "Run gh config set with an invalid value (seeded vs unseeded).",
		Tool:        "gh",
		Setup:       func(_ *Env) error { return nil },
		Seed:        Command{Args: []string{"config", "set", "git_protocol", "ssh"}},
		Bad:         Command{Args: []string{"config", "set", "git_protocol", "bananas"}},
		Help:        Command{Args: []string{"config", "set", "--help"}},
		Expect: Expectation{
			FinalExitCode: 0,
		},
	}
}

func ghConfigGetUnknownKeyScenario() Scenario {
	return Scenario{
		Name:        "gh_config_get_unknown_key",
		Description: "Run gh config get with an unknown key (seeded vs unseeded).",
		Tool:        "gh",
		Setup: func(env *Env) error {
			_, err := env.RunDirect("gh", "config", "set", "git_protocol", "ssh")
			return err
		},
		Seed: Command{Args: []string{"config", "get", "git_protocol"}},
		Bad:  Command{Args: []string{"config", "get", "git_protocl"}}, //nolint:misspell // intentional typo scenario
		Help: Command{Args: []string{"config", "get", "--help"}},
		Expect: Expectation{
			FinalExitCode:      0,
			FinalStdoutContain: "ssh",
		},
	}
}
