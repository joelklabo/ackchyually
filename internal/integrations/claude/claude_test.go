package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIntegrate_AddsEnvPATH_AndRecordsPrevious(t *testing.T) {
	in := mustReadTestdata(t, "existing_env.json")

	out, err := integrateBytes(in, "/shim", "/usr/bin")
	if err != nil {
		t.Fatalf("integrateBytes: %v", err)
	}

	m := mustParse(t, out)
	if got := getNestedString(t, m, "env", "PATH"); got != "/shim:/usr/local/bin" {
		t.Fatalf("got %q, want %q", got, "/shim:/usr/local/bin")
	}
	if got := getNestedString(t, m, "ackchyually", "claude", "previous_env_path"); got != "/usr/local/bin" {
		t.Fatalf("got %q, want %q", got, "/usr/local/bin")
	}

	undone, err := undoBytes(out)
	if err != nil {
		t.Fatalf("undoBytes: %v", err)
	}
	m2 := mustParse(t, undone)
	if got := getNestedString(t, m2, "env", "PATH"); got != "/usr/local/bin" {
		t.Fatalf("got %q, want %q", got, "/usr/local/bin")
	}
	if _, ok := getNested(m2, "ackchyually"); ok {
		t.Fatalf("expected ackchyually removed")
	}
}

func TestIntegrate_UsesCurrentPATH_WhenEnvPATHMissing(t *testing.T) {
	out, err := integrateBytes([]byte(`{}`), "/shim", "/usr/bin")
	if err != nil {
		t.Fatalf("integrateBytes: %v", err)
	}
	m := mustParse(t, out)
	if got := getNestedString(t, m, "env", "PATH"); got != "/shim:/usr/bin" {
		t.Fatalf("got %q, want %q", got, "/shim:/usr/bin")
	}
	if _, ok := getNested(m, "ackchyually", "claude", "previous_env_path"); ok {
		t.Fatalf("expected no previous_env_path")
	}
}

func TestIntegrate_MalformedJSON(t *testing.T) {
	if _, err := integrateBytes([]byte(`{not-json`), "/shim", "/usr/bin"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIntegratedPATHFromSettings(t *testing.T) {
	out, err := integrateBytes([]byte(`{}`), "/shim", "/usr/bin")
	if err != nil {
		t.Fatalf("integrateBytes: %v", err)
	}
	p, ok, err := IntegratedPATHFromSettings(out, "/shim")
	if err != nil {
		t.Fatalf("IntegratedPATHFromSettings: %v", err)
	}
	if !ok {
		t.Fatalf("expected integrated=true, got false (PATH=%q)", p)
	}
}

func mustReadTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}
	return b
}

func mustParse(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v\nOUTPUT:\n%s", err, string(b))
	}
	return m
}

func getNested(m map[string]any, path ...string) (any, bool) {
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
	v, ok := getNested(m, path...)
	if !ok {
		t.Fatalf("missing %v", path)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("expected string at %v, got %T", path, v)
	}
	return s
}
