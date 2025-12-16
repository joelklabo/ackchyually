package app

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestPrintVersion(t *testing.T) {
	// We can't easily mock the build info, but we can verify it prints something reasonable.
	// And we can temporarily set the global variables if we are in the same package.

	oldVer := buildVersion
	oldCommit := buildCommit
	oldDate := buildDate
	defer func() {
		buildVersion = oldVer
		buildCommit = oldCommit
		buildDate = oldDate
	}()

	buildVersion = "1.2.3"
	buildCommit = "abcdef1234567890"
	buildDate = "2023-01-01"

	code, out, _ := captureStdoutStderr(t, func() int {
		printVersion()
		return 0
	})
	if code != 0 {
		t.Fatalf("printVersion returned %d want 0", code)
	}

	if !strings.Contains(out, "version: v1.2.3") {
		t.Errorf("output missing version, got:\n%s", out)
	}
	if !strings.Contains(out, "commit:  abcdef123456") {
		t.Errorf("output missing commit, got:\n%s", out)
	}
	if !strings.Contains(out, "built:   2023-01-01") {
		t.Errorf("output missing date, got:\n%s", out)
	}
}

func TestGetVersionInfo_Error(t *testing.T) {
	// This is hard to test because it relies on runtime/debug.ReadBuildInfo
	// which usually succeeds in tests.
	// However, we can verify that normalizeVersion handles empty strings correctly.
	if v := normalizeVersion(""); v != "" {
		t.Errorf("normalizeVersion(\"\") = %q, want \"\"", v)
	}
	if v := normalizeVersion("v1.2.3"); v != "v1.2.3" {
		t.Errorf("normalizeVersion(\"v1.2.3\") = %q, want \"v1.2.3\"", v)
	}
	if v := normalizeVersion("1.2.3"); v != "v1.2.3" {
		t.Errorf("normalizeVersion(\"1.2.3\") = %q, want \"v1.2.3\"", v)
	}
}

func TestGetVersionInfo_Defaults(t *testing.T) {
	oldVer := buildVersion
	oldCommit := buildCommit
	oldDate := buildDate
	defer func() {
		buildVersion = oldVer
		buildCommit = oldCommit
		buildDate = oldDate
	}()

	buildVersion = ""
	buildCommit = ""
	buildDate = ""

	v := getVersionInfo()
	// In a test environment, ReadBuildInfo might return (devel) or something.
	// But we can at least assert that it doesn't crash and returns something sane.
	if v.Version == "" {
		t.Error("getVersionInfo returned empty version")
	}
}

func TestGetVersionInfo_BuildInfo(t *testing.T) {
	mockReadBuildInfo := func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Version: "v1.2.3",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef123456"},
				{Key: "vcs.time", Value: "2023-01-01T00:00:00Z"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true
	}

	v := getVersionInfoWithReader(mockReadBuildInfo)

	if v.Version != "v1.2.3" {
		t.Errorf("expected version v1.2.3, got %s", v.Version)
	}
	if v.Commit != "abcdef123456" {
		t.Errorf("expected commit abcdef123456, got %s", v.Commit)
	}
	if v.Date != "2023-01-01T00:00:00Z" {
		t.Errorf("expected date 2023-01-01T00:00:00Z, got %s", v.Date)
	}
	if !v.Modified {
		t.Errorf("expected modified true, got false")
	}
}
