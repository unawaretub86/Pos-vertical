package database

// US-081 contract §4, T-1 (AC-1): every BeginTx must take the SQLite write
// lock at BEGIN (BEGIN IMMEDIATE), so lock waits go through busy_timeout
// instead of failing later on a read-to-write upgrade.

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

// isSQLiteBusy reports whether err carries SQLite primary result code 5
// (SQLITE_BUSY), which also covers extended codes such as 517.
func isSQLiteBusy(err error) bool {
	var coder interface{ Code() int }
	if errors.As(err, &coder) {
		return coder.Code()&0xff == 5
	}
	return false
}

func TestTxLock_T1_BeginTxTakesWriteLockAtBegin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.db")
	db, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()

	// tx1 runs no statement at all: only BEGIN.
	tx1, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	tx1Open := true
	defer func() {
		if tx1Open {
			tx1.Rollback()
		}
	}()

	// A second, independent handle on the same file that never waits.
	raw, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(0)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("open second handle: %v", err)
	}
	defer raw.Close()
	conn, err := raw.Conn(ctx)
	if err != nil {
		t.Fatalf("second handle conn: %v", err)
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, `BEGIN IMMEDIATE`)
	if err == nil {
		conn.ExecContext(ctx, `ROLLBACK`)
		t.Fatalf("second BEGIN IMMEDIATE succeeded while tx1 was open: tx1 did not take the write lock at BEGIN (deferred transaction)")
	}
	if !isSQLiteBusy(err) {
		t.Fatalf("second BEGIN IMMEDIATE error = %v, want SQLITE_BUSY (5)", err)
	}

	if err := tx1.Rollback(); err != nil {
		t.Fatalf("tx1 rollback: %v", err)
	}
	tx1Open = false

	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		t.Fatalf("BEGIN IMMEDIATE after tx1 rollback: %v, want success", err)
	}
	if _, err := conn.ExecContext(ctx, `ROLLBACK`); err != nil {
		t.Fatalf("rollback second handle: %v", err)
	}
}
