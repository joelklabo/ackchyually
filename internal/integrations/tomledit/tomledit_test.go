package tomledit

import (
	"testing"

	"github.com/BurntSushi/toml"
)

func TestSet_RoundTripFromEmpty(t *testing.T) {
	doc, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if err := doc.Set("/shim:/usr/bin", "shell_environment_policy", "set", "PATH"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	out, err := doc.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}

	var parsed map[string]any
	if err := toml.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("re-parse: %v\nOUTPUT:\n%s", err, string(out))
	}
	got := getNestedString(t, parsed, "shell_environment_policy", "set", "PATH")
	if got != "/shim:/usr/bin" {
		t.Fatalf("got %q, want %q", got, "/shim:/usr/bin")
	}
}

func TestSet_UpdatesExistingShellEnvironmentPolicy(t *testing.T) {
	in := []byte(`
[shell_environment_policy]
include_only = ["PATH", "HOME"]
set = { PATH = "/usr/bin", FOO = "bar" }
`)

	doc, err := Parse(in)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if err := doc.Set("/shim:/usr/bin", "shell_environment_policy", "set", "PATH"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	out, err := doc.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}

	var parsed map[string]any
	if err := toml.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("re-parse: %v\nOUTPUT:\n%s", err, string(out))
	}

	got := getNestedString(t, parsed, "shell_environment_policy", "set", "PATH")
	if got != "/shim:/usr/bin" {
		t.Fatalf("got %q, want %q", got, "/shim:/usr/bin")
	}
	gotFoo := getNestedString(t, parsed, "shell_environment_policy", "set", "FOO")
	if gotFoo != "bar" {
		t.Fatalf("got %q, want %q", gotFoo, "bar")
	}
}

func TestSet_PreservesUnknownSections(t *testing.T) {
	in := []byte(`
[tool]
name = "example"

[shell_environment_policy]
set = { PATH = "/usr/bin" }
`)

	doc, err := Parse(in)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if err := doc.Set("/shim:/usr/bin", "shell_environment_policy", "set", "PATH"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	out, err := doc.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}

	var parsed map[string]any
	if err := toml.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("re-parse: %v\nOUTPUT:\n%s", err, string(out))
	}

	gotName := getNestedString(t, parsed, "tool", "name")
	if gotName != "example" {
		t.Fatalf("got %q, want %q", gotName, "example")
	}
}

func TestSet_ParsesFileWithComments(t *testing.T) {
	in := []byte(`
# comment at top

[shell_environment_policy] # inline comment
set = { PATH = "/usr/bin" } # another comment
`)

	doc, err := Parse(in)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if err := doc.Set("/shim:/usr/bin", "shell_environment_policy", "set", "PATH"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	out, err := doc.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if _, err := Parse(out); err != nil {
		t.Fatalf("Parse(roundtrip): %v\nOUTPUT:\n%s", err, string(out))
	}
}

func getNestedString(t *testing.T, m map[string]any, path ...string) string {
	t.Helper()
	var cur any = m
	for _, k := range path {
		nextMap, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("expected map at %q, got %T", k, cur)
		}
		v, ok := nextMap[k]
		if !ok {
			t.Fatalf("missing key %q", k)
		}
		cur = v
	}
	s, ok := cur.(string)
	if !ok {
		t.Fatalf("expected string at %v, got %T", path, cur)
	}
	return s
}
