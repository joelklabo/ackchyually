package helpcount

import (
	"fmt"
	"os"
	"path/filepath"
)

func BuiltinScenarios() []Scenario {
	return []Scenario{
		gitLogSubjectScenario(),
		gitLogSubjectNoiseScenario(),
		gitLogSubjectConfusingNoiseScenario(),
		gitLogInvalidCountScenario(),
		gitLogUnknownDateFormatScenario(),
		gitLogInvalidDecorateOptionScenario(),
		gitLogInvalidColorValueScenario(),
		gitRevParseMissingRevisionScenario(),
		gitShowUnknownRevisionScenario(),
		gitConfigWrongNumberOfArgsScenario(),
		gitAddPathspecScenario(),
		gitBranchNameRequiredScenario(),
		gitStatusTypoScenario(),
		gitCommitMissingValueScenario(),
		gitDiffNameOnlyScenario(),
		curlUnknownOptionScenario(),
		curlMissingOptionValueScenario(),
		goTestCountScenario(),
		goTestUnknownFlagScenario(),
		goTestUnknownFlagNoiseScenario(),
		goTestInvalidRunRegexScenario(),
		goUnknownCommandScenario(),
		ghVersionTypoScenario(),
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
