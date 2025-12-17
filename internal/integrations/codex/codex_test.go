package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestIntegrate_EmptyConfig(t *testing.T) {
	out, err := integrateBytes(nil, "/shim:/usr/bin")
	if err != nil {
		t.Fatalf("integrateBytes: %v", err)
	}

	var parsed map[string]any
	if err := toml.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("re-parse: %v\nOUTPUT:\n%s", err, string(out))
	}

	if got := getNestedString(t, parsed, "shell_environment_policy", "set", "PATH"); got != "/shim:/usr/bin" {
		t.Fatalf("got %q, want %q", got, "/shim:/usr/bin")
	}
	if got := getNestedString(t, parsed, "ackchyually", "codex", "managed_set_path"); got != "/shim:/usr/bin" {
		t.Fatalf("got %q, want %q", got, "/shim:/usr/bin")
	}
	if _, ok := getNested(t, parsed, "ackchyually", "codex", "previous_set_path"); ok {
		t.Fatalf("expected no previous_set_path")
	}
}

func TestIntegrate_RecordsPreviousPATH_AndUndoRestores(t *testing.T) {
	in := mustReadTestdata(t, "existing_policy_inline.toml")

	after, err := integrateBytes(in, "/shim:/usr/bin")
	if err != nil {
		t.Fatalf("integrateBytes: %v", err)
	}

	parsed := mustParse(t, after)
	if got := getNestedString(t, parsed, "ackchyually", "codex", "previous_set_path"); got != "/usr/bin" {
		t.Fatalf("got %q, want %q", got, "/usr/bin")
	}
	if got := getNestedString(t, parsed, "shell_environment_policy", "set", "PATH"); got != "/shim:/usr/bin" {
		t.Fatalf("got %q, want %q", got, "/shim:/usr/bin")
	}

	undone, err := undoBytes(after)
	if err != nil {
		t.Fatalf("undoBytes: %v", err)
	}
	parsed2 := mustParse(t, undone)
	if got := getNestedString(t, parsed2, "shell_environment_policy", "set", "PATH"); got != "/usr/bin" {
		t.Fatalf("got %q, want %q", got, "/usr/bin")
	}
	if _, ok := getNested(t, parsed2, "ackchyually", "codex"); ok {
		t.Fatalf("expected ackchyually.codex table removed")
	}
}

func TestIntegrate_IncludeOnly_AppendsPathAndHome_AndUndoRestores(t *testing.T) {
	in := mustReadTestdata(t, "include_only_missing_path_home.toml")

	after, err := integrateBytes(in, "/shim:/usr/bin")
	if err != nil {
		t.Fatalf("integrateBytes: %v", err)
	}

	parsed := mustParse(t, after)
	prev := getNestedStringSlice(t, parsed, "ackchyually", "codex", "previous_include_only")
	if len(prev) != 1 || prev[0] != "LANG" {
		t.Fatalf("unexpected previous_include_only: %#v", prev)
	}
	includeOnly := getNestedStringSlice(t, parsed, "shell_environment_policy", "include_only")
	assertContains(t, includeOnly, "LANG")
	assertContains(t, includeOnly, "PATH")
	assertContains(t, includeOnly, "HOME")

	undone, err := undoBytes(after)
	if err != nil {
		t.Fatalf("undoBytes: %v", err)
	}
	parsed2 := mustParse(t, undone)
	includeOnly2 := getNestedStringSlice(t, parsed2, "shell_environment_policy", "include_only")
	if len(includeOnly2) != 1 || includeOnly2[0] != "LANG" {
		t.Fatalf("unexpected include_only after undo: %#v", includeOnly2)
	}
}

func TestIntegratedPATHFromConfig(t *testing.T) {
	in := mustReadTestdata(t, "existing_policy_inline.toml")
	after, err := integrateBytes(in, "/shim:/usr/bin")
	if err != nil {
		t.Fatalf("integrateBytes: %v", err)
	}
	p, ok, err := IntegratedPATHFromConfig(after, "/shim")
	if err != nil {
		t.Fatalf("IntegratedPATHFromConfig: %v", err)
	}
	if !ok {
		t.Fatalf("expected integrated=true, got false (PATH=%q)", p)
	}
}

func mustParse(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var parsed map[string]any
	if err := toml.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("toml.Unmarshal: %v\nOUTPUT:\n%s", err, string(b))
	}
	return parsed
}

func getNested(t *testing.T, m map[string]any, path ...string) (any, bool) {
	t.Helper()
	var cur any = m
	for _, k := range path {
		nextMap, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := nextMap[k]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

func getNestedString(t *testing.T, m map[string]any, path ...string) string {
	t.Helper()
	v, ok := getNested(t, m, path...)
	if !ok {
		t.Fatalf("missing %v", path)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("expected string at %v, got %T", path, v)
	}
	return s
}

func getNestedStringSlice(t *testing.T, m map[string]any, path ...string) []string {
	t.Helper()
	v, ok := getNested(t, m, path...)
	if !ok {
		t.Fatalf("missing %v", path)
	}
	switch s := v.(type) {
	case []string:
		return append([]string{}, s...)
	case []any:
		out := make([]string, 0, len(s))
		for _, e := range s {
			es, ok := e.(string)
			if !ok {
				t.Fatalf("expected string slice at %v, got %T element", path, e)
			}
			out = append(out, es)
		}
		return out
	default:
		t.Fatalf("expected string slice at %v, got %T", path, v)
		return nil
	}
}

func assertContains(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, s := range haystack {
		if s == needle {
			return
		}
	}
	t.Fatalf("expected %#v to contain %q", haystack, needle)
}

func mustReadTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}
	return b
}
