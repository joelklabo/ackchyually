package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const brewAnalyticsURL = "https://formulae.brew.sh/api/analytics/install-on-request/30d.json"

type brewAnalytics struct {
	Items []struct {
		Formula string `json:"formula"`
	} `json:"items"`
}

type cmdSpec struct {
	Name    string
	Formula string
	Prefix  string
}

type toolResult struct {
	Name    string        `json:"name"`
	Formula string        `json:"formula"`
	Prefix  string        `json:"prefix"`
	Args    []string      `json:"args,omitempty"`
	OK      bool          `json:"ok"`
	Skipped bool          `json:"skipped,omitempty"`
	Reason  string        `json:"reason,omitempty"`
	Elapsed time.Duration `json:"elapsed"`
}

type report struct {
	StartedAt time.Time     `json:"started_at"`
	EndedAt   time.Time     `json:"ended_at"`
	Count     int           `json:"count"`
	Results   []toolResult  `json:"results"`
	Pass      int           `json:"pass"`
	Fail      int           `json:"fail"`
	Skip      int           `json:"skip"`
	Elapsed   time.Duration `json:"elapsed"`
}

func main() {
	os.Exit(run())
}

func run() int {
	var (
		count     = flag.Int("count", 250, "number of distinct executables to smoke-test")
		install   = flag.Bool("install", false, "brew install missing formulae (slow/expensive)")
		timeout   = flag.Duration("timeout", 5*time.Second, "per-command timeout")
		dryRun    = flag.Bool("dry-run", false, "print selected executables and exit")
		jsonOut   = flag.Bool("json", false, "output JSON report")
		skipFile  = flag.String("skip-file", "", "optional newline-delimited formula skip list")
		scenario  = flag.String("only", "", "only run executables whose name contains this substring")
		maxItems  = flag.Int("max-formulae", 500, "max formulae to consider from analytics")
		userAgent = flag.String("user-agent", "ackchyually-toolsmoke", "HTTP User-Agent")
	)
	flag.Parse()

	if *count <= 0 {
		fmt.Fprintln(os.Stderr, "count must be > 0")
		return 2
	}
	if _, err := exec.LookPath("brew"); err != nil {
		fmt.Fprintln(os.Stderr, "brew not found in PATH (required for this eval)")
		return 2
	}

	ctx := context.Background()
	formulae, err := fetchTopFormulae(ctx, *maxItems, *userAgent)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	skip := map[string]struct{}{}
	if *skipFile != "" {
		s, err := os.ReadFile(*skipFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		for _, line := range strings.Split(string(s), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			skip[line] = struct{}{}
		}
	}

	home, err := os.MkdirTemp("", "ackchyually-toolsmoke-home-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer func() { _ = os.RemoveAll(home) }()

	repoRoot, err := repoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	tmp, err := os.MkdirTemp("", "ackchyually-toolsmoke-bin-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	ackPath := filepath.Join(tmp, "ackchyually")
	if err := buildBinary(repoRoot, "./cmd/ackchyually", ackPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	shimDir := filepath.Join(home, ".local", "share", "ackchyually", "shims")
	baseEnv := baseEnvWithHome(home)

	specs := selectExecutables(formulae, *count, *install, skip, *scenario)
	if *dryRun {
		for _, s := range specs {
			fmt.Printf("%s\t%s\n", s.Name, s.Formula)
		}
		return 0
	}

	// Install shims in batches to reduce overhead.
	byPrefix := map[string][]cmdSpec{}
	for _, s := range specs {
		byPrefix[s.Prefix] = append(byPrefix[s.Prefix], s)
	}
	prefixes := make([]string, 0, len(byPrefix))
	for p := range byPrefix {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)

	for _, prefix := range prefixes {
		var tools []string
		for _, s := range byPrefix[prefix] {
			tools = append(tools, s.Name)
		}
		if err := runAckchyually(ctx, ackPath, baseEnv, append([]string{"shim", "install"}, tools...), 30*time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "shim install failed for prefix %s: %v\n", prefix, err)
			return 2
		}
	}

	r := report{StartedAt: time.Now(), Count: *count}
	for _, s := range specs {
		res := smokeOne(ctx, shimDir, baseEnv, s, *timeout)
		r.Results = append(r.Results, res)
		switch {
		case res.Skipped:
			r.Skip++
		case res.OK:
			r.Pass++
		default:
			r.Fail++
		}
	}
	r.EndedAt = time.Now()
	r.Elapsed = r.EndedAt.Sub(r.StartedAt)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(r); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		return 0
	}

	fmt.Printf("ackchyually eval (toptools smoke)\n")
	fmt.Printf("tools:   %d\n", len(r.Results))
	fmt.Printf("pass:    %d\n", r.Pass)
	fmt.Printf("fail:    %d\n", r.Fail)
	fmt.Printf("skipped: %d\n", r.Skip)
	fmt.Printf("elapsed: %s\n", r.Elapsed.Truncate(time.Millisecond))

	if r.Fail > 0 {
		fmt.Println()
		fmt.Println("failures:")
		n := 0
		for _, tr := range r.Results {
			if tr.OK || tr.Skipped {
				continue
			}
			fmt.Printf("- %s (%s): %s\n", tr.Name, tr.Formula, tr.Reason)
			n++
			if n >= 25 {
				fmt.Println("  ...")
				break
			}
		}
		return 1
	}
	return 0
}

func fetchTopFormulae(ctx context.Context, maxItems int, userAgent string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, brewAnalyticsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, readErr := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		msg := strings.TrimSpace(string(b))
		if readErr != nil && msg == "" {
			msg = readErr.Error()
		}
		return nil, fmt.Errorf("fetch brew analytics: HTTP %d: %s", resp.StatusCode, msg)
	}
	var a brewAnalytics
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return nil, err
	}
	if maxItems > 0 && len(a.Items) > maxItems {
		a.Items = a.Items[:maxItems]
	}
	out := make([]string, 0, len(a.Items))
	for _, it := range a.Items {
		if strings.TrimSpace(it.Formula) == "" {
			continue
		}
		out = append(out, it.Formula)
	}
	return out, nil
}

func selectExecutables(formulae []string, count int, install bool, skip map[string]struct{}, onlySubstr string) []cmdSpec {
	onlySubstr = strings.ToLower(strings.TrimSpace(onlySubstr))

	seen := map[string]struct{}{}
	var out []cmdSpec
	for _, f := range formulae {
		if _, ok := skip[f]; ok {
			continue
		}
		if count > 0 && len(out) >= count {
			break
		}

		if !install {
			ok, err := isBrewInstalled(f)
			if err != nil {
				continue
			}
			if !ok {
				continue
			}
		} else {
			ok, err := isBrewInstalled(f)
			if err != nil {
				continue
			}
			if !ok {
				if err := runBrew("install", f); err != nil {
					// Keep going; large runs will have failures.
					continue
				}
			}
		}

		prefix, err := brewPrefix(f)
		if err != nil || strings.TrimSpace(prefix) == "" {
			continue
		}
		exes := listExecutables(prefix)
		if len(exes) == 0 {
			continue
		}

		for _, name := range exes {
			if count > 0 && len(out) >= count {
				break
			}
			if _, ok := seen[name]; ok {
				continue
			}
			if onlySubstr != "" && !strings.Contains(strings.ToLower(name), onlySubstr) {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, cmdSpec{Name: name, Formula: f, Prefix: prefix})
		}
	}
	return out
}

func listExecutables(prefix string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, dir := range []string{filepath.Join(prefix, "bin"), filepath.Join(prefix, "sbin")} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if name == "" || name[0] == '.' {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0o111 == 0 {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func smokeOne(ctx context.Context, shimDir string, baseEnv []string, s cmdSpec, timeout time.Duration) toolResult {
	start := time.Now()
	tr := toolResult{Name: s.Name, Formula: s.Formula, Prefix: s.Prefix}

	toolEnv := prependPath(baseEnv, filepath.Join(s.Prefix, "bin"), filepath.Join(s.Prefix, "sbin"))
	directEnv := toolEnv
	shimEnv := prependPath(toolEnv, shimDir)

	probes := [][]string{
		{"--version"},
		{"version"},
		{"-V"},
		{"-v"},
		{"--help"},
		{"-h"},
		{"help"},
	}

	var chosen []string
	var directOut string
	var directExit int
	for _, p := range probes {
		out, code, err := runCmd(ctx, s.Name, p, directEnv, timeout)
		if err != nil {
			continue
		}
		if code == 0 {
			chosen = p
			directOut = out
			directExit = code
			break
		}
	}

	if chosen == nil {
		tr.Skipped = true
		tr.Reason = "no probe returned exit 0 (skipping)"
		tr.Elapsed = time.Since(start)
		return tr
	}

	tr.Args = chosen

	shimOut, shimExit, err := runCmd(ctx, s.Name, chosen, shimEnv, timeout)
	if err != nil {
		tr.Reason = err.Error()
		tr.Elapsed = time.Since(start)
		return tr
	}
	if shimExit != directExit {
		tr.Reason = fmt.Sprintf("exit %d via shim (want %d)", shimExit, directExit)
		tr.Elapsed = time.Since(start)
		return tr
	}
	if strings.Contains(shimOut, "ackchyually: suggestion") || strings.Contains(shimOut, "ackchyually: this worked before here:") {
		tr.Reason = "unexpected ackchyually suggestion on exit=0 probe"
		tr.Elapsed = time.Since(start)
		return tr
	}
	if shimOut != directOut {
		// Don't fail; many tools include paths/usernames in help output.
		tr.OK = true
		tr.Reason = "pass (output differs)"
		tr.Elapsed = time.Since(start)
		return tr
	}

	tr.OK = true
	tr.Reason = "pass"
	tr.Elapsed = time.Since(start)
	return tr
}

func runCmd(ctx context.Context, name string, args []string, env []string, timeout time.Duration) (string, int, error) {
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	path, err := lookPathEnv(name, env)
	if err != nil {
		return "", 127, err
	}
	cmd := exec.CommandContext(ctx2, path, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	code := exitCode(err)
	if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
		return string(out), code, fmt.Errorf("%s %s: timed out after %s", name, strings.Join(args, " "), timeout)
	}
	return string(out), code, nil
}

func runAckchyually(ctx context.Context, ackPath string, env []string, args []string, timeout time.Duration) error {
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx2, ackPath, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("ackchyually %s: timed out after %s", strings.Join(args, " "), timeout)
	}
	return fmt.Errorf("ackchyually %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
}

func isBrewInstalled(formula string) (bool, error) {
	cmd := exec.Command("brew", "list", "--versions", formula)
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func brewPrefix(formula string) (string, error) {
	cmd := exec.Command("brew", "--prefix", formula)
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func runBrew(args ...string) error {
	cmd := exec.Command("brew", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func buildBinary(repoRoot, pkg, out string) error {
	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Dir = repoRoot
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build %s: %w\n%s", pkg, err, string(b))
	}
	return nil
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root (go.mod)")
		}
		dir = parent
	}
}

func baseEnvWithHome(home string) []string {
	env := append([]string{}, os.Environ()...)
	env = upsertEnv(env, "HOME", home)
	env = deleteEnv(env, "ACKCHYUALLY_AUTO_EXEC")
	return env
}

func prependPath(env []string, dirs ...string) []string {
	pathEnv := getEnv(env, "PATH")
	if pathEnv == "" {
		pathEnv = os.Getenv("PATH")
	}
	parts := make([]string, 0, len(dirs)+1)
	for _, d := range dirs {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		parts = append(parts, d)
	}
	if pathEnv != "" {
		parts = append(parts, pathEnv)
	}
	return upsertEnv(env, "PATH", strings.Join(parts, string(os.PathListSeparator)))
}

func getEnv(env []string, key string) string {
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return strings.TrimPrefix(kv, prefix)
		}
	}
	return ""
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
		candidate := filepath.Join(dir, file)
		if isExecutableFile(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s not found in PATH", file)
}

func isExecutableFile(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	if st.IsDir() {
		return false
	}
	return st.Mode()&0o111 != 0
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
