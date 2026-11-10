package repository

import (
	"context"
	"testing"

	"github.com/andreyo/pos/internal/model"
)

// T-6 (AC-5, AC-12): the list filters keep their inclusive "<=" upper bound
// (out of scope), but bind UTC text. A purchase, an audit row, an expense and
// a sale stored at 2026-11-10 01:18:41 UTC (20:18:41 Bogota) are listed for the
// Nov 9 Bogota day and not for a window that starts at Nov 10 00:00 Bogota.
func TestListFilters_T6_EveningRowsBelongToTheLocalDay(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	const at = "2026-11-10 01:18:41"

	metricsExec(t, db, `INSERT INTO suppliers (id, name) VALUES (1, 'Proveedor')`)
	metricsExec(t, db, `INSERT INTO purchases (id, supplier_id, user_id, total, created_at) VALUES (1, 1, 1, 80000, ?)`, at)
	metricsExec(t, db, `INSERT INTO audit_log (user_id, action, entity, entity_id, created_at) VALUES (1, 'login', 'user', 1, ?)`, at)
	metricsExec(t, db, `INSERT INTO expenses (user_id, category, amount, created_at) VALUES (1, 'otro', 7000, ?)`, at)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, reference, created_at) VALUES (1, 1, 50000, 50000, 'completed', 'cash', '', ?)`, at)

	nov9From, nov9To := bogotaMidnight(2026, 11, 9), bogotaMidnight(2026, 11, 10)
	nov10From := bogotaMidnight(2026, 11, 10)

	purchases := NewPurchaseRepository(db.DB)
	audit := NewAuditRepository(db.DB)
	expenses := NewExpenseRepository(db.DB)
	sales := NewSaleRepository(db.DB)

	check := func(name string, inNov9, inNov10 int) {
		t.Helper()
		if inNov9 != 1 {
			t.Errorf("%s listed for Nov 9 (to = Nov 10 00:00 Bogota) = %d rows, want 1", name, inNov9)
		}
		if inNov10 != 0 {
			t.Errorf("%s listed for from = Nov 10 00:00 Bogota ... Nov 11 = %d rows, want 0", name, inNov10)
		}
	}
	nov11To := bogotaMidnight(2026, 11, 11)

	var (
		p9, p10 []*model.Purchase
		err     error
	)
	if p9, err = purchases.List(ctx, PurchaseFilter{From: &nov9From, To: &nov9To}); err != nil {
		t.Fatalf("purchases nov9: %v", err)
	}
	if p10, err = purchases.List(ctx, PurchaseFilter{From: &nov10From, To: &nov11To}); err != nil {
		t.Fatalf("purchases nov10: %v", err)
	}
	check("purchase", len(p9), len(p10))

	a9, err := audit.List(ctx, AuditFilter{From: &nov9From, To: &nov9To})
	if err != nil {
		t.Fatalf("audit nov9: %v", err)
	}
	a10, err := audit.List(ctx, AuditFilter{From: &nov10From, To: &nov11To})
	if err != nil {
		t.Fatalf("audit nov10: %v", err)
	}
	check("audit row", len(a9), len(a10))

	e9, err := expenses.List(ctx, ExpenseFilter{From: &nov9From, To: &nov9To})
	if err != nil {
		t.Fatalf("expenses nov9: %v", err)
	}
	e10, err := expenses.List(ctx, ExpenseFilter{From: &nov10From, To: &nov11To})
	if err != nil {
		t.Fatalf("expenses nov10: %v", err)
	}
	check("expense", len(e9), len(e10))

	s9, err := sales.List(ctx, SaleFilter{From: &nov9From, To: &nov9To})
	if err != nil {
		t.Fatalf("sales nov9: %v", err)
	}
	s10, err := sales.List(ctx, SaleFilter{From: &nov10From, To: &nov11To})
	if err != nil {
		t.Fatalf("sales nov10: %v", err)
	}
	check("sale", len(s9), len(s10))

	// The same instants sent in the browser form (UTC) give the same answer.
	utcFrom, utcTo := nov9From.UTC(), nov9To.UTC()
	pu, err := purchases.List(ctx, PurchaseFilter{From: &utcFrom, To: &utcTo})
	if err != nil {
		t.Fatalf("purchases nov9 (UTC bounds): %v", err)
	}
	if len(pu) != 1 {
		t.Errorf("purchase listed for Nov 9 with UTC bounds = %d rows, want 1", len(pu))
	}
}
