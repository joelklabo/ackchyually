package agentclis

import (
	"testing"
)

func TestLoadManifest_Valid(t *testing.T) {
	m, err := LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m.SchemaVersion != 1 {
		t.Fatalf("schema_version=%d, want 1", m.SchemaVersion)
	}

	seen := map[string]bool{}
	for _, tool := range m.Tools {
		if tool.ID == "" {
			t.Fatalf("empty tool id")
		}
		if seen[tool.ID] {
			t.Fatalf("duplicate tool id: %s", tool.ID)
		}
		seen[tool.ID] = true

		if _, err := normalizeManifestVersion(tool.SupportedRange.MinInclusive); err != nil {
			t.Fatalf("%s min_inclusive invalid: %v", tool.ID, err)
		}
		if _, err := normalizeManifestVersion(tool.SupportedRange.MaxExclusive); err != nil {
			t.Fatalf("%s max_exclusive invalid: %v", tool.ID, err)
		}
	}
}

func TestNormalizeInstalledVersion(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"v1.2.3", "v1.2.3", true},
		{"1.2.3", "v1.2.3", true},
		{"codex 1.2.3", "v1.2.3", true},
		{"copilot/1.2.3+build.1", "v1.2.3+build.1", true},
		{"nope", "", false},
	}

	for _, tt := range tests {
		got, ok := NormalizeInstalledVersion(tt.in)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("NormalizeInstalledVersion(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestCheckInstalledVersion_Range(t *testing.T) {
	tool := Tool{
		ID:             "codex",
		SupportedRange: Range{MinInclusive: "1.0.0", MaxExclusive: "2.0.0"},
	}

	{
		res, err := tool.CheckInstalledVersion("1.5.0")
		if err != nil {
			t.Fatalf("CheckInstalledVersion: %v", err)
		}
		if !res.Parseable || !res.WithinRange {
			t.Fatalf("expected within range, got %#v", res)
		}
	}
	{
		res, err := tool.CheckInstalledVersion("0.9.0")
		if err != nil {
			t.Fatalf("CheckInstalledVersion: %v", err)
		}
		if !res.Parseable || res.WithinRange {
			t.Fatalf("expected out of range, got %#v", res)
		}
	}
	{
		res, err := tool.CheckInstalledVersion("2.0.0")
		if err != nil {
			t.Fatalf("CheckInstalledVersion: %v", err)
		}
		if !res.Parseable || res.WithinRange {
			t.Fatalf("expected out of range, got %#v", res)
		}
	}
	{
		res, err := tool.CheckInstalledVersion("not-a-version")
		if err != nil {
			t.Fatalf("CheckInstalledVersion: %v", err)
		}
		if res.Parseable || res.WithinRange {
			t.Fatalf("expected unparseable/out of range, got %#v", res)
		}
	}
}
