package database

import (
	"path/filepath"
	"testing"
)

// A database written while foreign keys were not enforced may hold orphan
// rows; foreignKeyViolations must count them so New can warn.
func TestForeignKeyViolations_CountsOrphans(t *testing.T) {
	db, err := New(filepath.Join(t.TempDir(), "fkcheck.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	if n, err := db.foreignKeyViolations(); err != nil || n != 0 {
		t.Fatalf("fresh db: violations = %d, err = %v, want 0", n, err)
	}

	conn, err := db.Conn(t.Context())
	if err != nil {
		t.Fatalf("Conn: %v", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(t.Context(), `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable fk: %v", err)
	}
	if _, err := conn.ExecContext(t.Context(),
		`INSERT INTO registers (id, user_id, opening_amount) VALUES (99, 12345, 0)`); err != nil {
		t.Fatalf("insert orphan: %v", err)
	}

	if n, err := db.foreignKeyViolations(); err != nil || n != 1 {
		t.Fatalf("with orphan: violations = %d, err = %v, want 1", n, err)
	}
}
