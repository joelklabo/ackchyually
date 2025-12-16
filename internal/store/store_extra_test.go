package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestWithDB_OpenError(t *testing.T) {
	// Make Open fail by making HOME a file
	tmp := t.TempDir()
	homeFile := filepath.Join(tmp, "homefile")
	if err := os.WriteFile(homeFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write homefile: %v", err)
	}
	t.Setenv("HOME", homeFile)

	err := WithDB(func(db *DB) error {
		return nil
	})
	if err == nil {
		t.Error("expected error from WithDB when Open fails")
	}
}

func TestDBMethods_ErrorClosedDB(t *testing.T) {
	db := openTestDB(t)
	db.Close() // Close immediately to force errors

	if _, err := db.UpsertTool(ToolIdentity{SHA256: "x"}); err == nil {
		t.Error("UpsertTool: expected error on closed DB")
	}

	if _, err := db.GetToolPathCache("/bin/git"); err == nil {
		t.Error("GetToolPathCache: expected error on closed DB")
	}

	if err := db.UpsertToolPathCache(ToolPathCache{ExePath: "/bin/git"}); err == nil {
		t.Error("UpsertToolPathCache: expected error on closed DB")
	}
	
	if _, err := db.ListSuccessful("git", "ctx", 10); err == nil {
		t.Error("ListSuccessful: expected error on closed DB")
	}

	if err := db.InsertInvocation(Invocation{}); err == nil {
		t.Error("InsertInvocation: expected error on closed DB")
	}
	
	if err := db.UpsertTag(Tag{}); err == nil {
		t.Error("UpsertTag: expected error on closed DB")
	}
	
	if _, err := db.GetTag("ctx", "tag"); err == nil {
		t.Error("GetTag: expected error on closed DB")
	}
	
	if _, err := db.GetToolBySHA("sha"); err == nil {
		t.Error("GetToolBySHA: expected error on closed DB")
	}
}

func TestGetToolPathCache_NotFound(t *testing.T) {
	db := openTestDB(t)
	_, err := db.GetToolPathCache("/missing")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}
