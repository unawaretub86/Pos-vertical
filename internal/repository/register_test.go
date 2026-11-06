package repository

import (
	"context"
	"testing"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/model"
)

func TestRegisterRepo_Create_AtomicRollback(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := NewRegisterRepository(db.DB)

	// Try to create a register with an invalid movement type that violates CHECK (type IN ('sale', 'cash_in', 'cash_out', 'opening'))
	_, err = repo.Create(context.Background(), &model.Register{
		UserID:        1,
		OpeningAmount: 100,
	}, &model.RegisterMovement{
		Type:        "invalid_movement_type",
		Amount:      100,
		Description: "Should fail",
	})
	
	if err == nil {
		t.Fatal("expected error due to check constraint violation on movement type, got nil")
	}

	// Verify the transaction rolled back and the register wasn't created
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM registers`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count registers: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 registers after rollback, got %d", count)
	}
}

// ─── US-082 contract §4, T-10 (AC-10 cross-register, AC-13, G-1) ─────────────

func TestRegisterRepo_T10_SumCashAbonosPerRegister(t *testing.T) {
	db := openMetricsDB(t)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
	metricsExec(t, db, `INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 100000, 'closed')`)
	metricsExec(t, db, `INSERT INTO registers (id, user_id, opening_amount, status) VALUES (2, 1, 50000, 'closed')`)
	metricsExec(t, db, `INSERT INTO registers (id, user_id, opening_amount, status) VALUES (3, 1, 0, 'open')`)

	// Register 1: cash $20,000 (counts) + transfer $50,000 (must not count).
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, method, register_id) VALUES (1, 20000, 'cash', 1)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, method, register_id) VALUES (1, 50000, 'transfer', 1)`)
	// Register 2: cash $7,000.
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, method, register_id) VALUES (1, 7000, 'cash', 2)`)
	// Unattributed cash $3,000 (register_id NULL): no register counts it.
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, method, register_id) VALUES (1, 3000, 'cash', NULL)`)

	repo := NewRegisterRepository(db.DB)
	ctx := context.Background()

	for _, tc := range []struct {
		registerID int64
		want       float64
	}{
		{1, 20000},
		{2, 7000},
		{3, 0}, // no abonos at all: COALESCE, no NULL scan error
	} {
		got, err := repo.SumCashAbonos(ctx, tc.registerID)
		if err != nil {
			t.Errorf("SumCashAbonos(%d) error: %v", tc.registerID, err)
			continue
		}
		if got != tc.want {
			t.Errorf("SumCashAbonos(%d) = %.0f, want %.0f", tc.registerID, got, tc.want)
		}
	}
}
