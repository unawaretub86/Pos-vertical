package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
)

func openFileDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(filepath.Join(t.TempDir(), "h.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// Hold N connections at once so the pool must create N distinct ones.
func hConns(t *testing.T, db *DB, n int) []*sql.Conn {
	t.Helper()
	var cs []*sql.Conn
	for i := 0; i < n; i++ {
		c, err := db.Conn(context.Background())
		if err != nil {
			t.Fatalf("conn %d: %v", i, err)
		}
		cs = append(cs, c)
	}
	t.Cleanup(func() {
		for _, c := range cs {
			c.Close()
		}
	})
	return cs
}

func TestPragmas_BusyTimeoutOnEveryConnection(t *testing.T) {
	db := openFileDB(t)
	for i, c := range hConns(t, db, 6) {
		var v int
		if err := c.QueryRowContext(context.Background(), `PRAGMA busy_timeout`).Scan(&v); err != nil {
			t.Fatal(err)
		}
		if v != 5000 {
			t.Errorf("conn %d busy_timeout=%d want 5000", i, v)
		}
	}
}

func TestPragmas_WALOnEveryConnection(t *testing.T) {
	db := openFileDB(t)
	for i, c := range hConns(t, db, 6) {
		var v string
		if err := c.QueryRowContext(context.Background(), `PRAGMA journal_mode`).Scan(&v); err != nil {
			t.Fatal(err)
		}
		if v != "wal" {
			t.Errorf("conn %d journal_mode=%q want wal", i, v)
		}
	}
}

func TestPragmas_ForeignKeysStillOnEveryConnection(t *testing.T) {
	db := openFileDB(t)
	for i, c := range hConns(t, db, 6) {
		var v int
		if err := c.QueryRowContext(context.Background(), `PRAGMA foreign_keys`).Scan(&v); err != nil {
			t.Fatal(err)
		}
		if v != 1 {
			t.Errorf("conn %d foreign_keys=%d want 1", i, v)
		}
	}
}

// Concurrent writers must queue on the busy timeout, never fail with SQLITE_BUSY.
func TestPragmas_ConcurrentWritersNoBusy(t *testing.T) {
	db := openFileDB(t)
	if _, err := db.Exec(`CREATE TABLE hp (id INTEGER PRIMARY KEY, v INTEGER)`); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 25; i++ {
				tx, err := db.Begin()
				if err != nil {
					errs <- err
					return
				}
				if _, err := tx.Exec(`INSERT INTO hp (v) VALUES (1)`); err != nil {
					tx.Rollback()
					errs <- err
					return
				}
				if err := tx.Commit(); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("writer failed: %v", err)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM hp`).Scan(&n)
	if n != 200 {
		t.Errorf("rows=%d want 200", n)
	}
}

// Readers must not block while a writer holds a write transaction (WAL).
func TestPragmas_ReaderNotBlockedByWriter(t *testing.T) {
	db := openFileDB(t)
	if _, err := db.Exec(`CREATE TABLE hr (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO hr DEFAULT VALUES`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM hr`).Scan(&n); err != nil {
		t.Fatalf("reader blocked/failed: %v", err)
	}
	if n != 0 {
		t.Errorf("reader saw uncommitted row: %d", n)
	}
}

func TestPragmas_ReopenKeepsWAL(t *testing.T) {
	p := filepath.Join(t.TempDir(), "re.db")
	d1, err := New(p)
	if err != nil {
		t.Fatal(err)
	}
	d1.Close()
	d2, err := New(p)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer d2.Close()
	var v string
	d2.QueryRow(`PRAGMA journal_mode`).Scan(&v)
	if v != "wal" {
		t.Errorf("journal_mode after reopen=%q", v)
	}
}

func TestPragmas_MemoryDBStillWorks(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New(:memory:): %v", err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		t.Fatalf("migrated schema missing: %v", err)
	}
}

func TestPragmas_TimeFormatPreserved(t *testing.T) {
	db := openFileDB(t)
	var s string
	if err := db.QueryRow(`SELECT strftime('%Y', 'now')`).Scan(&s); err != nil || len(s) != 4 {
		t.Fatalf("sanity: %v %q", err, s)
	}
}
