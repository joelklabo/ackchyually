package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joelklabo/ackchyually/internal/execx"
)

const (
	claudeDirName      = ".claude"
	claudeSettingsName = "settings.json"
)

func DefaultSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	if home == "" {
		return "", errors.New("claude: unable to determine home directory")
	}
	return filepath.Join(home, claudeDirName, claudeSettingsName), nil
}

type Status struct {
	Installed      bool
	Version        string
	SettingsPath   string
	SettingsExists bool
	Integrated     bool
	IntegratedPATH string
}

func DetectStatus(ctx context.Context, settingsPath, shimDir string) (Status, error) {
	var st Status
	st.Installed, st.Version = DetectInstalledVersion(ctx)

	if settingsPath == "" {
		var err error
		settingsPath, err = DefaultSettingsPath()
		if err != nil {
			return Status{}, err
		}
	}
	st.SettingsPath = settingsPath

	b, err := os.ReadFile(settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			st.SettingsExists = false
			return st, nil
		}
		return Status{}, err
	}
	st.SettingsExists = true

	p, ok, err := IntegratedPATHFromSettings(b, shimDir)
	if err != nil {
		return Status{}, err
	}
	st.Integrated = ok
	st.IntegratedPATH = p
	return st, nil
}

func DetectInstalledVersion(ctx context.Context) (bool, string) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return false, ""
	}
	cmd := exec.CommandContext(ctx, path, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return true, ""
	}
	return true, strings.TrimSpace(string(out))
}

func IntegratedPATHFromSettings(b []byte, shimDir string) (string, bool, error) {
	m, err := parseSettings(b)
	if err != nil {
		return "", false, err
	}
	env, _ := getMap(m, "env")
	pathVal, ok := env["PATH"].(string)
	if !ok || pathVal == "" {
		return "", false, nil
	}
	parts := filepath.SplitList(pathVal)
	if len(parts) == 0 {
		return pathVal, false, nil
	}
	return pathVal, samePath(parts[0], shimDir), nil
}

type Plan struct {
	Path    string
	Before  []byte
	After   []byte
	Changed bool
}

func PlanIntegrate(settingsPath, shimDir, currentPATH string) (Plan, error) {
	if settingsPath == "" {
		var err error
		settingsPath, err = DefaultSettingsPath()
		if err != nil {
			return Plan{}, err
		}
	}
	before, err := os.ReadFile(settingsPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Plan{}, err
	}

	after, err := integrateBytes(before, shimDir, currentPATH)
	if err != nil {
		return Plan{}, err
	}

	changed := !bytes.Equal(before, after)
	return Plan{Path: settingsPath, Before: before, After: after, Changed: changed}, nil
}

func PlanUndo(settingsPath string) (Plan, error) {
	if settingsPath == "" {
		var err error
		settingsPath, err = DefaultSettingsPath()
		if err != nil {
			return Plan{}, err
		}
	}
	before, err := os.ReadFile(settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Plan{Path: settingsPath, Before: nil, After: nil, Changed: false}, nil
		}
		return Plan{}, err
	}

	after, err := undoBytes(before)
	if err != nil {
		return Plan{}, err
	}
	changed := !bytes.Equal(before, after)
	return Plan{Path: settingsPath, Before: before, After: after, Changed: changed}, nil
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

func integrateBytes(before []byte, shimDir, currentPATH string) ([]byte, error) {
	m, err := parseSettings(before)
	if err != nil {
		return nil, err
	}

	env := ensureMap(m, "env")
	basePath := currentPATH
	if existing, ok := env["PATH"].(string); ok && existing != "" {
		basePath = existing
	}
	shimFirst := execx.PrependToPATH(shimDir, basePath)

	ack := ensureMap(m, "ackchyually")
	ackClaude := ensureMap(ack, "claude")

	if _, ok := ackClaude["previous_env_path"]; !ok {
		if existing, ok := env["PATH"].(string); ok && existing != "" && existing != shimFirst {
			ackClaude["previous_env_path"] = existing
		}
	}

	env["PATH"] = shimFirst
	ackClaude["managed_env_path"] = shimFirst

	return marshalSettings(m)
}

func undoBytes(before []byte) ([]byte, error) {
	m, err := parseSettings(before)
	if err != nil {
		return nil, err
	}

	ack, ok := getMap(m, "ackchyually")
	if !ok {
		return before, nil
	}
	ackClaude, ok := getMap(ack, "claude")
	if !ok {
		return before, nil
	}
	if _, ok := ackClaude["managed_env_path"]; !ok {
		return before, nil
	}

	env, _ := getMap(m, "env")
	if prev, ok := ackClaude["previous_env_path"].(string); ok && prev != "" {
		env["PATH"] = prev
	} else {
		delete(env, "PATH")
	}

	delete(ackClaude, "managed_env_path")
	delete(ackClaude, "previous_env_path")
	deleteEmptyMap(ack, "claude")
	deleteEmptyMap(m, "ackchyually")
	deleteEmptyMap(m, "env")

	return marshalSettings(m)
}

func parseSettings(b []byte) (map[string]any, error) {
	if len(bytes.TrimSpace(b)) == 0 {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func marshalSettings(m map[string]any) ([]byte, error) {
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	out = append(out, '\n')
	return out, nil
}

func getMap(m map[string]any, key string) (map[string]any, bool) {
	v, ok := m[key]
	if !ok {
		return nil, false
	}
	child, ok := v.(map[string]any)
	return child, ok
}

func ensureMap(m map[string]any, key string) map[string]any {
	if child, ok := getMap(m, key); ok {
		return child
	}
	child := map[string]any{}
	m[key] = child
	return child
}

func deleteEmptyMap(m map[string]any, key string) {
	v, ok := m[key]
	if !ok {
		return
	}
	child, ok := v.(map[string]any)
	if !ok {
		return
	}
	if len(child) != 0 {
		return
	}
	delete(m, key)
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if os.PathSeparator == '\\' {
		return strings.EqualFold(a, b)
	}
	return a == b
}
