package repository

// Contrato US-056 T-3, T-15. Firma asumida (el contrato no la fija):
//   NewOrderRepository(db *sql.DB) OrderRepository
//   Open(ctx, tableID, userID int64) (*model.Order, error)
// Errores: ErrTableHasOpenOrder, ErrBusy (nuevo), y algun error para mesa inactiva.

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

func openFileDB(t *testing.T) *database.DB {
	t.Helper()
	// :memory: no sirve: cada conexion del pool seria una DB vacia.
	db, err := database.New(filepath.Join(t.TempDir(), "pos.db"))
	if err != nil {
		t.Fatalf("open file db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func orderExec(t *testing.T, db *database.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("seed exec: %v\nSQL: %s", err, q)
	}
}

func TestOrder_T3_ConcurrentOpenSingleWinner(t *testing.T) {
	db := openFileDB(t)
	orderExec(t, db, `INSERT OR IGNORE INTO users (id, username, password_hash, role) VALUES (1, 'u1', 'x', 'admin')`)
	orderExec(t, db, `INSERT INTO dining_tables (id, name) VALUES (1, 'Mesa 1')`)
	repo := NewOrderRepository(db.DB)

	const n = 8
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = repo.Open(context.Background(), 1, 1)
		}(i)
	}
	close(start)
	wg.Wait()

	ok := 0
	for i, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrTableHasOpenOrder), errors.Is(err, ErrBusy):
		default:
			t.Errorf("goroutine %d: unexpected error %v", i, err)
		}
	}
	if ok != 1 {
		t.Errorf("successes = %d, want exactly 1", ok)
	}
	var open int
	if err := db.QueryRow(`SELECT COUNT(*) FROM orders WHERE table_id = 1 AND status = 'open'`).Scan(&open); err != nil {
		t.Fatalf("count: %v", err)
	}
	if open != 1 {
		t.Errorf("open orders = %d, want 1", open)
	}
}

func TestOrder_T15_InactiveTableAndOpenThenDeactivate(t *testing.T) {
	db := openFileDB(t)
	ctx := context.Background()
	orderExec(t, db, `INSERT OR IGNORE INTO users (id, username, password_hash, role) VALUES (1, 'u1', 'x', 'admin')`)
	orderExec(t, db, `INSERT INTO dining_tables (id, name, active) VALUES (1, 'Inactiva', 0)`)
	orderExec(t, db, `INSERT INTO dining_tables (id, name, active) VALUES (2, 'Mesa 2', 1)`)
	repo := NewOrderRepository(db.DB)

	if _, err := repo.Open(ctx, 1, 1); err == nil {
		t.Errorf("Open on inactive table: err = nil, want error")
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("orders after Open on inactive table = %d, want 0", n)
	}

	if _, err := repo.Open(ctx, 2, 1); err != nil {
		t.Fatalf("Open on active table: %v", err)
	}
	tables := NewDiningTableRepository(db.DB)
	if err := tables.Deactivate(ctx, 2); !errors.Is(err, ErrTableHasOpenOrder) {
		t.Errorf("Deactivate with open order: err = %v, want ErrTableHasOpenOrder", err)
	}
}
