package app

import "testing"

func TestExportSanitizeArg_FlagValuePathNormalizesHome(t *testing.T) {
	got := exportSanitizeArg("--config=/Users/alice/.config/tool/config.json", "/Users/alice", "")
	want := "--config=~/.config/tool/config.json"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExportSanitizeArg_UserPathAnonymizesOtherUsers(t *testing.T) {
	got := exportSanitizeArg("/Users/bob/secrets.txt", "/Users/alice", "")
	want := "/Users/<user>/secrets.txt"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExportSanitizeArg_SensitiveEnvAssignmentsRedactValue(t *testing.T) {
	got := exportSanitizeArg("GITHUB_TOKEN=ghp_12345678901234567890", "/Users/alice", "")
	want := "GITHUB_TOKEN=<redacted>"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExportSanitizeArg_PATHAssignmentSanitizesSegments(t *testing.T) {
	got := exportSanitizeArg("PATH=/usr/bin:/Users/alice/bin:/Users/bob/bin", "/Users/alice", "")
	want := "PATH=/usr/bin:~/bin:/Users/<user>/bin"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
