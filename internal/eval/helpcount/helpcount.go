package helpcount

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // register sqlite driver
)

type Mode string

const (
	ModeBaseline Mode = "baseline"
	ModeMemory   Mode = "memory"
	ModeCompare  Mode = "compare"
)

type Options struct {
	Mode           Mode
	ScenarioFilter string // substring match
	JSON           bool
}

type Scenario struct {
	Name        string
	Description string
	Tool        string
	Setup       func(*Env) error

	Seed Command
	// Noise runs after Seed (memory mode only) and before Bad.
	// This simulates realistic sessions where unrelated successes happen between
	// a known-good invocation and a later failure.
	Noise []Command
	Bad   Command
	Help  Command

	Expect Expectation
}

type Command struct {
	Args []string
}

type Expectation struct {
	// BadExitCode, when set, is the expected exit code for the "bad" command.
	// When unset, the harness expects the bad command to exit non-zero.
	BadExitCode *int
	// BadOutputContain, when set, must appear in the "bad" command output.
	BadOutputContain string

	FinalExitCode      int
	FinalStdoutContain string

	// Optional expectations for baseline vs memory runs.
	// Useful for eval regression tests (to ensure we actually reduced help usage).
	BaselineHelpInvocations *int
	MemoryHelpInvocations   *int
	BaselineSuggestionUsed  *bool
	MemorySuggestionUsed    *bool
}

type Report struct {
	StartedAt time.Time
	EndedAt   time.Time
	Results   []ScenarioResult
}

type ScenarioResult struct {
	Name        string
	Description string

	Baseline *RunResult
	Memory   *RunResult
}

type RunResult struct {
	Mode              Mode
	Success           bool
	Steps             int
	HelpInvocations   int
	SuggestionPrinted bool
	SuggestionUsed    bool
	Error             string
}

type Runner struct {
	RepoRoot string
	ackPath  string
}

func NewRunner(repoRoot string) (*Runner, error) {
	tmp, err := os.MkdirTemp("", "ackchyually-eval-*")
	if err != nil {
		return nil, err
	}
	ackPath := filepath.Join(tmp, "ackchyually")
	if err := buildBinary(repoRoot, "./cmd/ackchyually", ackPath); err != nil {
		_ = os.RemoveAll(tmp)
		return nil, err
	}
	return &Runner{RepoRoot: repoRoot, ackPath: ackPath}, nil
}

func (r *Runner) Close() error {
	return os.RemoveAll(filepath.Dir(r.ackPath))
}

func (r *Runner) Run(opts Options) (report Report, err error) {
	report.StartedAt = time.Now()
	defer func() { report.EndedAt = time.Now() }()

	scenarios := BuiltinScenarios()
	if opts.ScenarioFilter != "" {
		scenarios = filterScenarios(scenarios, opts.ScenarioFilter)
	}
	if len(scenarios) == 0 {
		return report, fmt.Errorf("no scenarios match filter %q", opts.ScenarioFilter)
	}

	for _, s := range scenarios {
		res := ScenarioResult{
			Name:        s.Name,
			Description: s.Description,
		}

		switch opts.Mode {
		case ModeBaseline:
			runRes := r.runOne(s, ModeBaseline)
			res.Baseline = &runRes
		case ModeMemory:
			runRes := r.runOne(s, ModeMemory)
			res.Memory = &runRes
		case "", ModeCompare:
			b := r.runOne(s, ModeBaseline)
			m := r.runOne(s, ModeMemory)
			res.Baseline = &b
			res.Memory = &m
		default:
			return report, fmt.Errorf("unknown mode: %q (want baseline|memory|compare)", opts.Mode)
		}

		report.Results = append(report.Results, res)
	}

	return report, nil
}

func filterScenarios(in []Scenario, substr string) []Scenario {
	substr = strings.ToLower(strings.TrimSpace(substr))
	if substr == "" {
		return in
	}
	out := make([]Scenario, 0, len(in))
	for _, s := range in {
		if strings.Contains(strings.ToLower(s.Name), substr) ||
			strings.Contains(strings.ToLower(s.Description), substr) ||
			strings.Contains(strings.ToLower(s.Tool), substr) {
			out = append(out, s)
		}
	}
	return out
}

func (r *Runner) runOne(s Scenario, mode Mode) RunResult { //nolint:gocyclo
	env, err := NewEnv(r.ackPath)
	if err != nil {
		return RunResult{Mode: mode, Error: err.Error()}
	}
	defer env.Close()

	if err := env.InstallShim(s.Tool); err != nil {
		return RunResult{Mode: mode, Error: err.Error()}
	}
	if err := s.Setup(env); err != nil {
		return RunResult{Mode: mode, Error: err.Error()}
	}

	steps := 0
	if mode == ModeMemory {
		if _, err := env.RunShim(s.Tool, s.Seed.Args...); err != nil {
			return RunResult{Mode: mode, Error: fmt.Sprintf("seed failed: %v", err)}
		}
		steps++

		for _, c := range s.Noise {
			if _, err := env.RunShim(s.Tool, c.Args...); err != nil {
				return RunResult{Mode: mode, Error: fmt.Sprintf("noise failed: %v", err)}
			}
			steps++
		}
	}

	badOut, badErr := env.RunShim(s.Tool, s.Bad.Args...)
	steps++

	badExit := exitCode(badErr)
	badOK := true
	switch {
	case s.Expect.BadExitCode == nil:
		if badExit == 0 {
			badOK = false
		}
	case badExit != *s.Expect.BadExitCode:
		badOK = false
	}
	if badOK && s.Expect.BadOutputContain != "" && !strings.Contains(badOut, s.Expect.BadOutputContain) {
		badOK = false
	}

	suggested, suggestionPrinted := ExtractSuggestion(badOut)

	finalOut := ""
	finalExit := 0
	suggestionUsed := false

	if suggested != "" {
		out, err := env.RunShellShim(suggested)
		steps++
		finalOut = out
		finalExit = exitCode(err)
		suggestionUsed = true
	} else {
		// No suggestion: simulate an agent reaching for --help/-h.
		if _, err := env.RunShim(s.Tool, s.Help.Args...); err != nil {
			_ = err // best-effort
		}
		steps++

		out, err := env.RunShim(s.Tool, s.Seed.Args...)
		steps++
		finalOut = out
		finalExit = exitCode(err)
	}

	helpInv, err := env.CountHelpInvocations(s.Tool, nil)
	if err != nil {
		return RunResult{Mode: mode, Error: err.Error()}
	}

	success := badOK && finalExit == s.Expect.FinalExitCode
	if success && s.Expect.FinalStdoutContain != "" && !strings.Contains(finalOut, s.Expect.FinalStdoutContain) {
		success = false
	}

	errMsg := ""
	switch {
	case !badOK && s.Expect.BadExitCode == nil && badExit == 0:
		errMsg = "bad command unexpectedly exited 0"
	case !badOK && s.Expect.BadExitCode != nil && badExit != *s.Expect.BadExitCode:
		errMsg = fmt.Sprintf("bad exit code %d (want %d)", badExit, *s.Expect.BadExitCode)
	case !badOK && s.Expect.BadOutputContain != "" && !strings.Contains(badOut, s.Expect.BadOutputContain):
		errMsg = fmt.Sprintf("bad output missing %q", s.Expect.BadOutputContain)
	case finalExit != s.Expect.FinalExitCode:
		errMsg = fmt.Sprintf("final exit code %d (want %d)", finalExit, s.Expect.FinalExitCode)
	case s.Expect.FinalStdoutContain != "" && !strings.Contains(finalOut, s.Expect.FinalStdoutContain):
		errMsg = fmt.Sprintf("final output missing %q", s.Expect.FinalStdoutContain)
	case suggestionPrinted && suggested == "":
		errMsg = "ackchyually printed suggestion header but no command line"
	}

	if success {
		wantHelpInv := (*int)(nil)
		wantSuggested := (*bool)(nil)
		switch mode {
		case ModeBaseline:
			wantHelpInv = s.Expect.BaselineHelpInvocations
			wantSuggested = s.Expect.BaselineSuggestionUsed
		case ModeMemory:
			wantHelpInv = s.Expect.MemoryHelpInvocations
			wantSuggested = s.Expect.MemorySuggestionUsed
		}

		switch {
		case wantHelpInv != nil && helpInv != *wantHelpInv:
			success = false
			errMsg = fmt.Sprintf("help invocations=%d (want %d)", helpInv, *wantHelpInv)
		case wantSuggested != nil && suggestionUsed != *wantSuggested:
			success = false
			if *wantSuggested {
				errMsg = "expected suggestion to be used"
			} else {
				errMsg = "expected no suggestion to be used"
			}
		}
	}

	return RunResult{
		Mode:              mode,
		Success:           success,
		Steps:             steps,
		HelpInvocations:   helpInv,
		SuggestionPrinted: suggestionPrinted,
		SuggestionUsed:    suggestionUsed,
		Error:             errMsg,
	}
}

type Env struct {
	BaseDir string
	Home    string
	WorkDir string
	ShimDir string

	ackPath    string
	basePath   string
	shimmedEnv []string
	directEnv  []string
}

func NewEnv(ackPath string) (*Env, error) {
	base, err := os.MkdirTemp("", "ackchyually-eval-run-*")
	if err != nil {
		return nil, err
	}
	home := filepath.Join(base, "home")
	work := filepath.Join(base, "work")
	shimDir := filepath.Join(home, ".local", "share", "ackchyually", "shims")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		_ = os.RemoveAll(base)
		return nil, err
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		_ = os.RemoveAll(base)
		return nil, err
	}

	basePath := os.Getenv("PATH")
	if basePath == "" {
		basePath = "/usr/bin:/bin"
	}
	basePath = stripAckchyuallyShimDirs(basePath)

	directEnv := append([]string{}, os.Environ()...)
	directEnv = upsertEnv(directEnv, "HOME", home)
	directEnv = upsertEnv(directEnv, "PATH", basePath)
	directEnv = deleteEnv(directEnv, "ACKCHYUALLY_AUTO_EXEC")

	shimmedEnv := append([]string{}, directEnv...)
	shimmedEnv = upsertEnv(shimmedEnv, "PATH", strings.Join([]string{shimDir, basePath}, string(os.PathListSeparator)))

	return &Env{
		BaseDir: base,
		Home:    home,
		WorkDir: work,
		ShimDir: shimDir,
		ackPath: ackPath,

		basePath:   basePath,
		shimmedEnv: shimmedEnv,
		directEnv:  directEnv,
	}, nil
}

func (e *Env) Close() { _ = os.RemoveAll(e.BaseDir) }

func (e *Env) InstallShim(tool string) error {
	path := filepath.Join(e.ShimDir, tool)
	if _, err := os.Lstat(path); err == nil {
		return nil
	}
	return os.Symlink(e.ackPath, path)
}

func (e *Env) RunDirect(name string, args ...string) (string, error) {
	return e.run(name, args, e.directEnv)
}

func (e *Env) RunShim(tool string, args ...string) (string, error) {
	return e.run(tool, args, e.shimmedEnv)
}

func (e *Env) RunShellShim(cmd string) (string, error) {
	return e.run("sh", []string{"-c", cmd}, e.shimmedEnv)
}

func (e *Env) run(name string, args []string, env []string) (string, error) {
	path, err := lookPathEnv(name, env)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(context.Background(), path, args...)
	cmd.Dir = e.WorkDir
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.String() + stderr.String(), err
}

func (e *Env) dbPath() string {
	return filepath.Join(e.Home, ".local", "share", "ackchyually", "ackchyually.sqlite")
}

func (e *Env) CountHelpInvocations(tool string, helpArgs []string) (int, error) {
	dbPath := e.dbPath()
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	rows, err := db.QueryContext(context.Background(), `SELECT argv_json FROM invocations WHERE tool = ? ORDER BY created_at ASC`, tool)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var argvJSON string
		if err := rows.Scan(&argvJSON); err != nil {
			return 0, err
		}
		var argv []string
		if err := json.Unmarshal([]byte(argvJSON), &argv); err != nil {
			continue
		}
		if matchesHelpInvocation(tool, helpArgs, argv) {
			n++
		}
	}
	return n, rows.Err()
}

func matchesHelpInvocation(tool string, helpArgs []string, argv []string) bool {
	if len(helpArgs) > 0 {
		want := append([]string{tool}, helpArgs...)
		return slicesEqual(argv, want)
	}
	return isHelpInvocation(argv)
}

func isHelpInvocation(argv []string) bool {
	if len(argv) > 1 && argv[1] == "help" {
		return true
	}
	for _, a := range argv[1:] {
		switch a {
		case "-h", "--help", "-help":
			return true
		}
	}
	return false
}

func ExtractSuggestion(output string) (string, bool) {
	lines := strings.Split(output, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], "ackchyually: suggestion") ||
			strings.Contains(lines[i], "ackchyually: this worked before here:") {
			for j := i + 1; j < len(lines); j++ {
				l := strings.TrimSpace(lines[j])
				if l == "" {
					continue
				}
				return l, true
			}
			return "", true
		}
	}
	return "", false
}

func buildBinary(repoRoot, pkg, out string) error {
	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", out, pkg)
	cmd.Dir = repoRoot
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build %s: %w\n%s", pkg, err, string(b))
	}
	return nil
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 1
}

func upsertEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, prefix+value)
	return out
}

func deleteEnv(env []string, key string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			continue
		}
		out = append(out, kv)
	}
	return out
}

func stripAckchyuallyShimDirs(pathEnv string) string {
	parts := filepath.SplitList(pathEnv)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if isAckchyuallyShimDir(p) {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, string(os.PathListSeparator))
}

func isAckchyuallyShimDir(dir string) bool {
	if strings.TrimSpace(dir) == "" {
		return false
	}
	clean := filepath.Clean(dir)
	suffix := filepath.Join(".local", "share", "ackchyually", "shims")
	return strings.HasSuffix(clean, suffix)
}

func lookPathEnv(file string, env []string) (string, error) {
	if file == "" {
		return "", fmt.Errorf("empty executable name")
	}
	if strings.ContainsRune(file, os.PathSeparator) {
		return file, nil
	}
	pathEnv := getEnv(env, "PATH")
	if pathEnv == "" {
		pathEnv = os.Getenv("PATH")
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			dir = "."
		}
		full := filepath.Join(dir, file)
		fi, err := os.Stat(full)
		if err != nil || fi.IsDir() {
			continue
		}
		if fi.Mode()&0o111 == 0 {
			continue
		}
		return full, nil
	}
	return "", fmt.Errorf("exec: %q: executable file not found in $PATH", file)
}

func getEnv(env []string, key string) string {
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return kv[len(prefix):]
		}
	}
	return ""
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
