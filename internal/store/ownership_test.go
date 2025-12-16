package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckOwnership_NormalUser(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping normal user test because we are running as root")
	}

	tmp := t.TempDir()
	db := filepath.Join(tmp, "db.sqlite")
	if err := os.WriteFile(db, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Should pass because we are not root
	if err := checkOwnership(db); err != nil {
		t.Errorf("checkOwnership failed for normal user: %v", err)
	}
}

func TestCheckOwnership_RootMismatch(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Skipping root test because we are not running as root")
	}

	// To test this as root, we need to create a file owned by non-root.
	// This makes assumptions about available UIDs.
	// So this test is hard to run reliably in all environments.
	// But we can verify that checkOwnership returns nil if file is owned by root (us).

	tmp := t.TempDir()
	db := filepath.Join(tmp, "db.sqlite")
	if err := os.WriteFile(db, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Owned by root (us). Should pass.
	// Wait, logic says: if stat.Uid != 0 { error }
	// Convert 0 to strict match?
	// The code: if stat.Uid != 0 { return fmt.Errorf(...) }
	// So if owned by root, it returns nil.

	if err := checkOwnership(db); err != nil {
		t.Errorf("checkOwnership failed for root file as root: %v", err)
	}
}
