package helpcount

import (
	"fmt"
	"os"
	"path/filepath"
)

func BuiltinScenarios() []Scenario {
	return []Scenario{
		gitLogSubjectScenario(),
		gitDiffNameOnlyScenario(),
		goTestCountScenario(),
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

func goTestCountScenario() Scenario {
	return Scenario{
		Name:        "go_test_count_flag",
		Description: "Run go test with a common flag parsing mistake (seeded vs unseeded).",
		Tool:        "go",
		Setup: func(env *Env) error {
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
		},
		Seed: Command{Args: []string{"test", "./...", "-count=1"}},
		Bad:  Command{Args: []string{"test", "./...", "-count", "one"}},
		Help: Command{Args: []string{"help", "testflag"}},
		Expect: Expectation{
			FinalExitCode: 0,
		},
	}
}
