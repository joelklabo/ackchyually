package codex

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joelklabo/ackchyually/internal/execx"
	"github.com/joelklabo/ackchyually/internal/integrations/tomledit"
)

const (
	codexDirName    = ".codex"
	codexConfigName = "config.toml"
)

func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	if home == "" {
		return "", errors.New("codex: unable to determine home directory")
	}
	return filepath.Join(home, codexDirName, codexConfigName), nil
}

type Status struct {
	Installed      bool
	Version        string
	ConfigPath     string
	ConfigExists   bool
	Integrated     bool
	IntegratedPATH string
}

func DetectStatus(ctx context.Context, configPath, shimDir string) (Status, error) {
	var st Status

	st.Installed, st.Version = DetectInstalledVersion(ctx)

	if configPath == "" {
		var err error
		configPath, err = DefaultConfigPath()
		if err != nil {
			return Status{}, err
		}
	}
	st.ConfigPath = configPath

	b, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			st.ConfigExists = false
			return st, nil
		}
		return Status{}, err
	}
	st.ConfigExists = true

	p, ok, err := IntegratedPATHFromConfig(b, shimDir)
	if err != nil {
		return Status{}, err
	}
	st.Integrated = ok
	st.IntegratedPATH = p
	return st, nil
}

func DetectInstalledVersion(ctx context.Context) (bool, string) {
	path, err := exec.LookPath("codex")
	if err != nil {
		return false, ""
	}

	cmd := exec.CommandContext(ctx, path, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Treat as installed but version unknown.
		return true, ""
	}
	return true, strings.TrimSpace(string(out))
}

func IntegratedPATHFromConfig(b []byte, shimDir string) (string, bool, error) {
	doc, err := tomledit.Parse(b)
	if err != nil {
		return "", false, err
	}
	v, ok := doc.Get("shell_environment_policy", "set", "PATH")
	s, okStr := v.(string)
	if !ok || !okStr || s == "" {
		return "", false, nil
	}

	parts := filepath.SplitList(s)
	if len(parts) == 0 {
		return s, false, nil
	}
	return s, samePath(parts[0], shimDir), nil
}

type Plan struct {
	Path    string
	Before  []byte
	After   []byte
	Changed bool
}

func PlanIntegrate(configPath, shimDir, currentPATH string) (Plan, error) {
	if configPath == "" {
		var err error
		configPath, err = DefaultConfigPath()
		if err != nil {
			return Plan{}, err
		}
	}

	before, err := os.ReadFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Plan{}, err
	}

	shimFirst := execx.PrependToPATH(shimDir, currentPATH)
	after, err := integrateBytes(before, shimFirst)
	if err != nil {
		return Plan{}, err
	}

	changed := !bytes.Equal(before, after)
	return Plan{Path: configPath, Before: before, After: after, Changed: changed}, nil
}

func Apply(plan Plan) error {
	if !plan.Changed {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(plan.Path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(plan.Path, plan.After, 0o600)
}

func PlanUndo(configPath string) (Plan, error) {
	if configPath == "" {
		var err error
		configPath, err = DefaultConfigPath()
		if err != nil {
			return Plan{}, err
		}
	}

	before, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Plan{Path: configPath, Before: nil, After: nil, Changed: false}, nil
		}
		return Plan{}, err
	}

	after, err := undoBytes(before)
	if err != nil {
		return Plan{}, err
	}
	changed := !bytes.Equal(before, after)
	return Plan{Path: configPath, Before: before, After: after, Changed: changed}, nil
}

func integrateBytes(before []byte, shimFirstPath string) ([]byte, error) {
	doc, err := tomledit.Parse(before)
	if err != nil {
		return nil, err
	}

	if err := maybeRecordPrevious(doc, "ackchyually", "codex", "previous_set_path", "shell_environment_policy", "set", "PATH"); err != nil {
		return nil, err
	}

	if err := doc.Set(shimFirstPath, "shell_environment_policy", "set", "PATH"); err != nil {
		return nil, err
	}
	if err := doc.Set(shimFirstPath, "ackchyually", "codex", "managed_set_path"); err != nil {
		return nil, err
	}

	if err := maybeEnsureIncludeOnlyHasPathAndHome(doc); err != nil {
		return nil, err
	}

	return doc.Bytes()
}

func undoBytes(before []byte) ([]byte, error) {
	doc, err := tomledit.Parse(before)
	if err != nil {
		return nil, err
	}

	if _, ok := doc.Get("ackchyually", "codex", "managed_set_path"); !ok {
		return before, nil
	}

	if v, ok := doc.Get("ackchyually", "codex", "previous_set_path"); ok {
		if prev, ok := v.(string); ok && prev != "" {
			if err := doc.Set(prev, "shell_environment_policy", "set", "PATH"); err != nil {
				return nil, err
			}
		} else {
			doc.Delete("shell_environment_policy", "set", "PATH")
		}
		doc.Delete("ackchyually", "codex", "previous_set_path")
	} else {
		doc.Delete("shell_environment_policy", "set", "PATH")
	}

	if v, ok := doc.Get("ackchyually", "codex", "previous_include_only"); ok {
		if prev, ok := coerceStringSlice(v); ok {
			if err := doc.Set(prev, "shell_environment_policy", "include_only"); err != nil {
				return nil, err
			}
		}
		doc.Delete("ackchyually", "codex", "previous_include_only")
	}

	doc.Delete("ackchyually", "codex", "managed_set_path")
	deleteEmptyTable(doc, "ackchyually", "codex")
	deleteEmptyTable(doc, "ackchyually")

	deleteEmptyTable(doc, "shell_environment_policy", "set")
	deleteEmptyTable(doc, "shell_environment_policy")

	return doc.Bytes()
}

func maybeRecordPrevious(doc *tomledit.Document, recordPath ...string) error {
	// recordPath is: ackKey1, ackKey2, ackKey3, sourceKey1, sourceKey2, sourceKey3...
	if len(recordPath) < 4 {
		return errors.New("codex: invalid maybeRecordPrevious args")
	}

	ackPath := recordPath[:3]
	srcPath := recordPath[3:]

	if _, ok := doc.Get(ackPath...); ok {
		return nil
	}
	v, ok := doc.Get(srcPath...)
	if !ok {
		return nil
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	return doc.Set(s, ackPath...)
}

func maybeEnsureIncludeOnlyHasPathAndHome(doc *tomledit.Document) error {
	v, ok := doc.Get("shell_environment_policy", "include_only")
	if !ok {
		return nil
	}
	old, ok := coerceStringSlice(v)
	if !ok {
		return nil
	}

	newVal := ensureContains(old, "PATH", "HOME")
	if slicesEqual(old, newVal) {
		return nil
	}

	if _, ok := doc.Get("ackchyually", "codex", "previous_include_only"); !ok {
		if err := doc.Set(old, "ackchyually", "codex", "previous_include_only"); err != nil {
			return err
		}
	}
	return doc.Set(newVal, "shell_environment_policy", "include_only")
}

func ensureContains(list []string, vals ...string) []string {
	out := append([]string{}, list...)
	for _, want := range vals {
		var found bool
		for _, got := range out {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			out = append(out, want)
		}
	}
	return out
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

func coerceStringSlice(v any) ([]string, bool) {
	switch t := v.(type) {
	case []string:
		return append([]string{}, t...), true
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}

func deleteEmptyTable(doc *tomledit.Document, path ...string) {
	v, ok := doc.Get(path...)
	if !ok {
		return
	}
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	if len(m) != 0 {
		return
	}
	doc.Delete(path...)
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if os.PathSeparator == '\\' {
		return strings.EqualFold(a, b)
	}
	return a == b
}
