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
