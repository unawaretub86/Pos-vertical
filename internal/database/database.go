package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
	path string
}

func New(path string) (*DB, error) {
	// modernc.org/sqlite only honours _pragma=name(value), which it runs on every
	// new pooled connection (busy_timeout first). The mattn-style _journal_mode,
	// _busy_timeout and _foreign_keys keys are silently ignored.
	// _txlock=immediate makes every BeginTx a BEGIN IMMEDIATE: the write lock is
	// taken at BEGIN, where busy_timeout waits for it, instead of failing later
	// with SQLITE_BUSY on a read-to-write upgrade (US-081).
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_time_format=sqlite&_txlock=immediate")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	d := &DB{DB: db, path: path}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	// Foreign keys are enforced, so rows that violate them in an existing
	// database (written while they were not) are reported instead of hidden.
	if n, err := d.foreignKeyViolations(); err != nil {
		log.Printf("[database] foreign_key_check failed: %v", err)
	} else if n > 0 {
		log.Printf("[database] WARNING: %d foreign key violation(s) in existing data; run PRAGMA foreign_key_check", n)
	}
	return d, nil
}

// SchemaVersion returns the version of the last migration this binary knows.
// Tools that must not migrate (cmd/reset-data) compare it with schema_migrations.
func SchemaVersion() int {
	return dbMigrations[len(dbMigrations)-1].version
}

// foreignKeyViolations returns how many rows currently violate a foreign key.
func (d *DB) foreignKeyViolations() (int, error) {
	rows, err := d.Query("PRAGMA foreign_key_check")
	if err != nil {
		return 0, fmt.Errorf("foreign_key_check: %w", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		n++
	}
	return n, rows.Err()
}
