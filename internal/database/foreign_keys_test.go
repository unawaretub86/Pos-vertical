package database

import (
	"path/filepath"
	"strings"
	"testing"
)

// US-078 AC-2: deleting a register that has sales must fail with an FK error.
func TestForeignKeys_DeleteRegisterWithSalesFails(t *testing.T) {
	db, err := New(filepath.Join(t.TempDir(), "fk.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	mustExec := func(q string) {
		t.Helper()
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("exec %q: %v", q, err)
		}
	}
	mustExec(`INSERT INTO users (username, password_hash, role) VALUES ('fk-user', 'h', 'admin')`)
	mustExec(`INSERT INTO registers (id, user_id, opening_amount) VALUES (1, (SELECT id FROM users WHERE username = 'fk-user'), 0)`)
	mustExec(`INSERT INTO sales (id, user_id, register_id, subtotal, total, payment_method) VALUES (1, (SELECT id FROM users WHERE username = 'fk-user'), 1, 100, 100, 'cash')`)

	_, err = db.Exec(`DELETE FROM registers WHERE id = 1`)
	if err == nil {
		t.Fatal("expected FOREIGN KEY constraint error, got nil")
	}
	if !strings.Contains(strings.ToUpper(err.Error()), "FOREIGN KEY") {
		t.Fatalf("expected FOREIGN KEY error, got: %v", err)
	}
}
