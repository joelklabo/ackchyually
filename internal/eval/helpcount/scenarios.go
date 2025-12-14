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
		gitStatusTypoScenario(),
		gitCommitMissingValueScenario(),
		gitDiffNameOnlyScenario(),
		goTestCountScenario(),
		goTestUnknownFlagScenario(),
		goTestUnknownFlagNoiseScenario(),
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
