package service

import (
	"context"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/repository"
)

type expenseTestEnv struct {
	svc        *ExpenseService
	registers  *RegisterService
	registerID int64
}

func newExpenseTestEnv(t *testing.T, openRegister bool) *expenseTestEnv {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	regRepo := repository.NewRegisterRepository(db.DB)
	expenseRepo := repository.NewExpenseRepository(db.DB)
	registers := NewRegisterService(regRepo, nil, nil)

	env := &expenseTestEnv{
		svc:       NewExpenseService(expenseRepo, registers),
		registers: registers,
	}

	if openRegister {
		reg, err := registers.Open(context.Background(), 1, 0)
		if err != nil {
			t.Fatalf("open register: %v", err)
		}
		env.registerID = reg.ID
	}
	return env
}

// Doc test: gasto de $80.000 (Servicios) con caja abierta → aparece en
// register_movements como cash_out y por lo tanto en expected_close del cierre.
func TestExpenseService_Create_WithOpenRegister_RecordsCashOut(t *testing.T) {
	env := newExpenseTestEnv(t, true)

	expense, err := env.svc.Create(context.Background(), 1, "servicios", 80000, "Luz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expense.RegisterID == nil || *expense.RegisterID != env.registerID {
		t.Fatalf("expected expense associated with open register %d, got %v", env.registerID, expense.RegisterID)
	}

	summary, err := env.registers.Summary(context.Background(), env.registerID)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.CashOut != 80000 {
		t.Errorf("expected cash_out 80000, got %.0f", summary.CashOut)
	}
	if summary.Expected != -80000 {
		t.Errorf("expected expected_close -80000, got %.0f", summary.Expected)
	}
}

// Doc test (Scope): si no hay caja abierta, el gasto se guarda igual
// (register_id = null), sin bloquear el registro.
func TestExpenseService_Create_NoOpenRegister_StillSaves(t *testing.T) {
	env := newExpenseTestEnv(t, false)

	expense, err := env.svc.Create(context.Background(), 1, "arriendo", 500000, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expense.RegisterID != nil {
		t.Errorf("expected register_id nil, got %v", *expense.RegisterID)
	}
}

// Doc test: gasto con monto 0 o negativo → rechazado con 400 (service-level
// error the handler maps to 400).
func TestExpenseService_Create_InvalidAmount_Rejected(t *testing.T) {
	env := newExpenseTestEnv(t, false)

	if _, err := env.svc.Create(context.Background(), 1, "otro", 0, ""); err != ErrInvalidExpenseAmount {
		t.Errorf("expected ErrInvalidExpenseAmount for amount=0, got %v", err)
	}
	if _, err := env.svc.Create(context.Background(), 1, "otro", -100, ""); err != ErrInvalidExpenseAmount {
		t.Errorf("expected ErrInvalidExpenseAmount for amount=-100, got %v", err)
	}
}

func TestExpenseService_Create_InvalidCategory_Rejected(t *testing.T) {
	env := newExpenseTestEnv(t, false)

	if _, err := env.svc.Create(context.Background(), 1, "bogus", 1000, ""); err != ErrInvalidExpenseCategory {
		t.Errorf("expected ErrInvalidExpenseCategory, got %v", err)
	}
}

// Doc test: ganancia bruta $200.000 hoy, gastos $80.000 + $40.000 hoy →
// net_profit = 80.000 (verified at the SumRange level the dashboard handler uses).
func TestExpenseService_SumRange_AddsTodayExpenses(t *testing.T) {
	env := newExpenseTestEnv(t, false)

	if _, err := env.svc.Create(context.Background(), 1, "servicios", 80000, ""); err != nil {
		t.Fatalf("create expense 1: %v", err)
	}
	if _, err := env.svc.Create(context.Background(), 1, "nomina", 40000, ""); err != nil {
		t.Fatalf("create expense 2: %v", err)
	}

	// created_at is written via SQLite's CURRENT_TIMESTAMP, which is always UTC —
	// the range must be computed in UTC to match, regardless of the test
	// machine's local timezone.
	grossProfit := 200000.0
	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)
	sum, err := env.svc.SumRange(context.Background(), from, to)
	if err != nil {
		t.Fatalf("sum range: %v", err)
	}
	if sum != 120000 {
		t.Fatalf("expected expenses sum 120000, got %.0f", sum)
	}
	if netProfit := grossProfit - sum; netProfit != 80000 {
		t.Errorf("expected net_profit 80000, got %.0f", netProfit)
	}
}
