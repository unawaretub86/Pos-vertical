package repository

// Contrato US-056 T-14 (AC-15): Deactivate con pedido open.

import (
	"context"
	"errors"
	"testing"
)

func TestDiningTable_T14_DeactivateWithOpenOrder(t *testing.T) {
	db := openFileDB(t)
	ctx := context.Background()
	orderExec(t, db, `INSERT OR IGNORE INTO users (id, username, password_hash, role) VALUES (1, 'u1', 'x', 'admin')`)
	orderExec(t, db, `INSERT INTO dining_tables (id, name) VALUES (1, 'Mesa 1')`)
	orderExec(t, db, `INSERT INTO orders (id, type, table_id, status, user_id) VALUES (1, 'table', 1, 'open', 1)`)
	repo := NewDiningTableRepository(db.DB)

	if err := repo.Deactivate(ctx, 1); !errors.Is(err, ErrTableHasOpenOrder) {
		t.Fatalf("Deactivate with open order: err = %v, want ErrTableHasOpenOrder", err)
	}
	var active int
	if err := db.QueryRow(`SELECT active FROM dining_tables WHERE id = 1`).Scan(&active); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if active != 1 {
		t.Errorf("active = %d, want 1", active)
	}

	orderExec(t, db, `UPDATE orders SET status = 'cancelled' WHERE id = 1`)
	if err := repo.Deactivate(ctx, 1); err != nil {
		t.Fatalf("Deactivate after cancel: %v", err)
	}
	if err := db.QueryRow(`SELECT active FROM dining_tables WHERE id = 1`).Scan(&active); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if active != 0 {
		t.Errorf("active = %d, want 0", active)
	}

	if err := repo.Deactivate(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("Deactivate 999: err = %v, want ErrNotFound", err)
	}
}
