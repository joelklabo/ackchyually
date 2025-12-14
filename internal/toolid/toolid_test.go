package toolid

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joelklabo/ackchyually/internal/store"
)

func TestIdentify_UsesCachedSHAWhenFileUnreadable(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	exe := filepath.Join(tmp, "tool.sh")
	if err := os.WriteFile(exe, []byte(`#!/bin/sh
if [ "${1:-}" = "--version" ]; then
  echo "v1"
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}

	first, err := Identify(exe)
	if err != nil {
		t.Fatalf("Identify (first): %v", err)
	}
	if first.SHA256 == "" || first.ID == 0 {
		t.Fatalf("expected sha/id, got: %#v", first)
	}

	// Make unreadable: if Identify tries to hash again, it should fail.
	if err := os.Chmod(exe, 0o111); err != nil {
		t.Fatal(err)
	}

	second, err := Identify(exe)
	if err != nil {
		t.Fatalf("Identify (second): %v", err)
	}
	if second.SHA256 != first.SHA256 || second.ID != first.ID {
		t.Fatalf("expected same identity; first=%#v second=%#v", first, second)
	}
}

func TestIdentify_RehashesWhenFileChanges(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	exe := filepath.Join(tmp, "tool.sh")
	if err := os.WriteFile(exe, []byte(`#!/bin/sh
if [ "${1:-}" = "--version" ]; then
  echo "v1"
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}

	first, err := Identify(exe)
	if err != nil {
		t.Fatalf("Identify (first): %v", err)
	}

	// Verify cache row exists.
	if err := store.WithDB(func(db *store.DB) error {
		c, err := db.GetToolPathCache(exe)
		if err != nil {
			t.Fatalf("GetToolPathCache: %v", err)
		}
		if c.SHA256 != first.SHA256 {
			t.Fatalf("expected cache sha %q, got %q", first.SHA256, c.SHA256)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Modify the file (changes size + mtime), so we should rehash and get a new sha.
	if err := os.WriteFile(exe, []byte(`#!/bin/sh
if [ "${1:-}" = "--version" ]; then
  echo "v2"
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}

	second, err := Identify(exe)
	if err != nil {
		t.Fatalf("Identify (second): %v", err)
	}
	if second.SHA256 == first.SHA256 {
		t.Fatalf("expected sha to change; got %q", second.SHA256)
	}
}
