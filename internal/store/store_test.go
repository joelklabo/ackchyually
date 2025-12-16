package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setTempHome(t *testing.T) {
	t.Helper()

	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // For Windows

	// Guardrail: avoid writing to the developer's real home directory.
	if d := dataDir(); !strings.HasPrefix(d, home+string(os.PathSeparator)) {
		t.Fatalf("dataDir=%q does not respect HOME=%q", d, home)
	}
}

func openTestDB(t *testing.T) *DB {
	t.Helper()

	setTempHome(t)

	db, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestMustJSON_UnsupportedTypeReturnsNull(t *testing.T) {
	if got := MustJSON(make(chan int)); got != "null" {
		t.Fatalf("MustJSON(chan int) = %q, want %q", got, "null")
	}
	if got := MustJSON([]string{"a", "b"}); got != `["a","b"]` {
		t.Fatalf("MustJSON([]string) = %q, want %q", got, `["a","b"]`)
	}
}

func TestOpen_CreatesSchemaTables(t *testing.T) {
	db := openTestDB(t)

	var n int
	if err := db.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM sqlite_master
WHERE type = 'table'
  AND name IN ('tool_identities', 'tool_path_cache', 'invocations', 'tags')`).Scan(&n); err != nil {
		t.Fatalf("query schema: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected 4 tables, got %d", n)
	}
}

func TestOpen_FailsWhenHomeIsFile(t *testing.T) {
	tmp := t.TempDir()
	homeFile := filepath.Join(tmp, "homefile")
	if err := os.WriteFile(homeFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write homefile: %v", err)
	}
	t.Setenv("HOME", homeFile)

	_, err := Open()
	if err == nil {
		t.Fatalf("expected Open to fail when HOME is a file")
	}
}

func TestWithDB_CallsFn(t *testing.T) {
	setTempHome(t)

	called := false
	if err := WithDB(func(db *DB) error {
		called = true
		var one int
		return db.QueryRowContext(context.Background(), `SELECT 1`).Scan(&one)
	}); err != nil {
		t.Fatalf("WithDB: %v", err)
	}
	if !called {
		t.Fatalf("expected fn to be called")
	}
}

func TestInsertInvocation_AndListSuccessful(t *testing.T) {
	db := openTestDB(t)

	ctxKey := "cwd:/tmp/repo"
	base := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

	mustInsert := func(inv Invocation) {
		t.Helper()
		if err := db.InsertInvocation(inv); err != nil {
			t.Fatalf("InsertInvocation: %v", err)
		}
	}

	mustInsert(Invocation{
		At:         base.Add(1 * time.Second),
		DurationMS: 10,
		ContextKey: ctxKey,
		Tool:       "git",
		ExePath:    "/usr/bin/git",
		ArgvJSON:   MustJSON([]string{"git", "status"}),
		ExitCode:   0,
		Mode:       "pipes",
	})
	mustInsert(Invocation{
		At:         base.Add(2 * time.Second),
		DurationMS: 20,
		ContextKey: ctxKey,
		Tool:       "git",
		ExePath:    "/usr/bin/git",
		ArgvJSON:   MustJSON([]string{"git", "commit", "-m", "msg"}),
		ExitCode:   0,
		Mode:       "pipes",
	})
	mustInsert(Invocation{
		At:         base.Add(3 * time.Second),
		DurationMS: 30,
		ContextKey: ctxKey,
		Tool:       "git",
		ExePath:    "/usr/bin/git",
		ArgvJSON:   MustJSON([]string{"git", "log", "--badflag"}),
		ExitCode:   1,
		Mode:       "pipes",
	})
	mustInsert(Invocation{
		At:         base.Add(4 * time.Second),
		DurationMS: 40,
		ContextKey: ctxKey,
		Tool:       "gh",
		ExePath:    "/opt/homebrew/bin/gh",
		ArgvJSON:   MustJSON([]string{"gh", "auth", "status"}),
		ExitCode:   0,
		Mode:       "pipes",
	})

	cmds, err := db.ListSuccessful("git", ctxKey, 10)
	if err != nil {
		t.Fatalf("ListSuccessful: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 successful git commands, got %d: %#v", len(cmds), cmds)
	}
	if got, want := cmds[0], []string{"git", "commit", "-m", "msg"}; !equalStringSlices(got, want) {
		t.Fatalf("cmds[0]=%#v, want %#v", got, want)
	}
	if got, want := cmds[1], []string{"git", "status"}; !equalStringSlices(got, want) {
		t.Fatalf("cmds[1]=%#v, want %#v", got, want)
	}

	cmds1, err := db.ListSuccessful("git", ctxKey, 1)
	if err != nil {
		t.Fatalf("ListSuccessful(limit=1): %v", err)
	}
	if len(cmds1) != 1 || !equalStringSlices(cmds1[0], []string{"git", "commit", "-m", "msg"}) {
		t.Fatalf("ListSuccessful(limit=1)=%#v", cmds1)
	}

	none, err := db.ListSuccessful("missingtool", ctxKey, 10)
	if err != nil {
		t.Fatalf("ListSuccessful(missingtool): %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected empty for missingtool, got %#v", none)
	}

	// Insert invalid JSON to test robust unmarshalling
	mustInsert(Invocation{
		At:         base.Add(5 * time.Second),
		DurationMS: 50,
		ContextKey: ctxKey,
		Tool:       "git",
		ExePath:    "/usr/bin/git",
		ArgvJSON:   "{invalid-json",
		ExitCode:   0,
		Mode:       "pipes",
	})
	cmdsInvalid, err := db.ListSuccessful("git", ctxKey, 10)
	if err != nil {
		t.Fatalf("ListSuccessful: %v", err)
	}
	// Should still be 2, ignoring the invalid one
	if len(cmdsInvalid) != 2 {
		t.Fatalf("expected 2 successful commands (ignoring invalid JSON), got %d", len(cmdsInvalid))
	}
}

func TestNullIfZero(t *testing.T) {
	if got := nullIfZero(0); got != nil {
		t.Errorf("nullIfZero(0) = %v; want nil", got)
	}
	if got := nullIfZero(123); got != int64(123) {
		t.Errorf("nullIfZero(123) = %v; want 123", got)
	}
}

func TestListSuccessCandidates_GroupsCountsAndOrders(t *testing.T) {
	db := openTestDB(t)

	ctxKey := "git:/tmp/repo"
	base := time.Date(2025, 2, 3, 4, 5, 6, 0, time.UTC)

	mustInsert := func(at time.Time, argvJSON string) {
		t.Helper()
		if err := db.InsertInvocation(Invocation{
			At:         at,
			DurationMS: 1,
			ContextKey: ctxKey,
			Tool:       "git",
			ExePath:    "/usr/bin/git",
			ArgvJSON:   argvJSON,
			ExitCode:   0,
			Mode:       "pipes",
		}); err != nil {
			t.Fatalf("InsertInvocation: %v", err)
		}
	}

	mustInsert(base.Add(1*time.Second), MustJSON([]string{"git", "status"}))
	mustInsert(base.Add(2*time.Second), MustJSON([]string{"git", "status"}))
	mustInsert(base.Add(3*time.Second), MustJSON([]string{"git", "log", "-1"}))

	// corrupt JSON should be skipped
	mustInsert(base.Add(4*time.Second), "not json")

	cands, err := db.ListSuccessCandidates("git", ctxKey, 10)
	if err != nil {
		t.Fatalf("ListSuccessCandidates: %v", err)
	}
	if len(cands) != 2 {
		t.Fatalf("expected 2 candidates, got %d: %#v", len(cands), cands)
	}

	if got, want := cands[0].Argv, []string{"git", "log", "-1"}; !equalStringSlices(got, want) {
		t.Fatalf("cands[0].Argv=%#v, want %#v", got, want)
	}
	if got, want := cands[0].Count, 1; got != want {
		t.Fatalf("cands[0].Count=%d, want %d", got, want)
	}
	if got, want := cands[0].Last.UTC(), base.Add(3*time.Second); !got.Equal(want) {
		t.Fatalf("cands[0].Last=%v, want %v", got, want)
	}

	if got, want := cands[1].Argv, []string{"git", "status"}; !equalStringSlices(got, want) {
		t.Fatalf("cands[1].Argv=%#v, want %#v", got, want)
	}
	if got, want := cands[1].Count, 2; got != want {
		t.Fatalf("cands[1].Count=%d, want %d", got, want)
	}
	if got, want := cands[1].Last.UTC(), base.Add(2*time.Second); !got.Equal(want) {
		t.Fatalf("cands[1].Last=%v, want %v", got, want)
	}
}

func TestParseDBTime_HandlesCommonLayouts(t *testing.T) {
	if got := parseDBTime(""); !got.IsZero() {
		t.Fatalf("expected zero time, got %v", got)
	}

	t1 := time.Date(2025, 12, 15, 1, 2, 3, 123456789, time.UTC)

	for _, s := range []string{
		t1.Format(time.RFC3339Nano),
		t1.Format(time.RFC3339),
		"2025-12-15 01:02:03.123456789",
		"2025-12-15 01:02:03",
	} {
		got := parseDBTime(s)
		if got.IsZero() {
			t.Fatalf("parseDBTime(%q) returned zero", s)
		}
	}

	now := time.Now()
	parsed := parseDBTime(now.String())
	if parsed.IsZero() {
		t.Fatalf("expected parseDBTime(time.Now().String()) to parse, got zero")
	}
	if !parsed.Equal(now) {
		t.Fatalf("parseDBTime(now.String())=%v, want %v", parsed, now)
	}
}

func TestTags_UpsertGetList(t *testing.T) {
	db := openTestDB(t)

	ctxKey := "cwd:/tmp/repo"
	if err := db.UpsertTag(Tag{
		ContextKey: ctxKey,
		Tag:        "build",
		Tool:       "go",
		ArgvJSON:   MustJSON([]string{"go", "build", "./..."}),
	}); err != nil {
		t.Fatalf("UpsertTag(build): %v", err)
	}

	if err := db.UpsertTag(Tag{
		ContextKey: ctxKey,
		Tag:        "test",
		Tool:       "go",
		ArgvJSON:   MustJSON([]string{"go", "test", "./..."}),
	}); err != nil {
		t.Fatalf("UpsertTag(test): %v", err)
	}

	// Update existing tag.
	if err := db.UpsertTag(Tag{
		ContextKey: ctxKey,
		Tag:        "build",
		Tool:       "just",
		ArgvJSON:   MustJSON([]string{"just", "build"}),
	}); err != nil {
		t.Fatalf("UpsertTag(build update): %v", err)
	}

	got, err := db.GetTag(ctxKey, "build")
	if err != nil {
		t.Fatalf("GetTag: %v", err)
	}
	if got.Tool != "just" {
		t.Fatalf("GetTag(build).Tool=%q, want %q", got.Tool, "just")
	}

	all, err := db.ListTags(ctxKey, "")
	if err != nil {
		t.Fatalf("ListTags(all): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 tags, got %d: %#v", len(all), all)
	}
	if all[0].Tag != "build" || all[1].Tag != "test" {
		t.Fatalf("expected tags sorted by tag asc (build, test), got: %#v", []string{all[0].Tag, all[1].Tag})
	}

	justOnly, err := db.ListTags(ctxKey, "just")
	if err != nil {
		t.Fatalf("ListTags(tool=just): %v", err)
	}
	if len(justOnly) != 1 || justOnly[0].Tag != "build" {
		t.Fatalf("expected only build for tool=just, got: %#v", justOnly)
	}
}

func TestToolIdentity_UpsertAndGet(t *testing.T) {
	db := openTestDB(t)

	id, err := db.UpsertTool(ToolIdentity{
		ExePath:    "/usr/bin/git",
		SHA256:     "deadbeef",
		VersionStr: "git version 2.0.0",
	})
	if err != nil {
		t.Fatalf("UpsertTool: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero id")
	}

	id2, err := db.UpsertTool(ToolIdentity{
		ExePath:    "/usr/local/bin/git",
		SHA256:     "deadbeef",
		VersionStr: "git version 3.0.0",
	})
	if err != nil {
		t.Fatalf("UpsertTool(second): %v", err)
	}
	if id2 != id {
		t.Fatalf("expected stable id for same sha, got %d then %d", id, id2)
	}

	got, err := db.GetToolBySHA("deadbeef")
	if err != nil {
		t.Fatalf("GetToolBySHA: %v", err)
	}
	if got.ID != id {
		t.Fatalf("GetToolBySHA.ID=%d, want %d", got.ID, id)
	}
	if got.ExePath != "/usr/bin/git" {
		t.Fatalf("GetToolBySHA.ExePath=%q, want %q", got.ExePath, "/usr/bin/git")
	}
}

func TestToolPathCache_UpsertAndGet(t *testing.T) {
	db := openTestDB(t)

	in := ToolPathCache{
		ExePath:     "/usr/bin/git",
		FileSize:    123,
		FileMtimeNS: 456,
		SHA256:      "cafebabe",
	}
	if err := db.UpsertToolPathCache(in); err != nil {
		t.Fatalf("UpsertToolPathCache: %v", err)
	}

	got, err := db.GetToolPathCache(in.ExePath)
	if err != nil {
		t.Fatalf("GetToolPathCache: %v", err)
	}
	if got != in {
		t.Fatalf("GetToolPathCache=%#v, want %#v", got, in)
	}
}

func equalStringSlices(a, b []string) bool {
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
