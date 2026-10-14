package database

import (
	"path/filepath"
	"testing"
)

// US-078b: a file-backed database must run in WAL mode with a 5000 ms busy timeout.
func TestPragmas_FileDBUsesWAL(t *testing.T) {
	db, err := New(filepath.Join(t.TempDir(), "wal.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}

	var timeout int
	if err := db.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		t.Fatalf("busy_timeout: %v", err)
	}
	if timeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", timeout)
	}
}
