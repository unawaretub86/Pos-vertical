package repository

// US-088 contract §4, T-1 (AC-1, AC-8): voiding a sale writes no register
// movement, and the rest of the void (status, voided_by, stock, audit) stays
// as it was before the fix.

import (
	"context"
	"testing"
)

func TestSaleRepo_T1_VoidWritesNoRegisterMovement(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()

	// Register 1 open with $100,000; product 1 with stock 9 after the sale of
	// 1 unit; cash sale #1 of $50,000 and its 'sale' movement.
	metricsExec(t, db, `INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 100000, 'open')`)
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (1, 'Arroz', 'A', 50000, 30000, 'unit', 1, 9)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, register_id, subtotal, total, status, payment_method, reference) VALUES (1, 1, 1, 50000, 50000, 'completed', 'cash', '')`)
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 1, 50000, 50000)`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'cash', 50000)`)
	metricsExec(t, db, `INSERT INTO register_movements (register_id, type, amount, description) VALUES (1, 'sale', 50000, 'Venta #1')`)

	if _, err := NewSaleRepository(db.DB).Void(ctx, 1, 1); err != nil {
		t.Fatalf("Void: %v", err)
	}

	var total, cashOut int
	if err := db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1`).Scan(&total); err != nil {
		t.Fatalf("count movements: %v", err)
	}
	if err := db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1 AND type = 'cash_out'`).Scan(&cashOut); err != nil {
		t.Fatalf("count cash_out: %v", err)
	}
	if total != 1 {
		t.Errorf("register 1 movements after void = %d, want 1 (only the original 'sale' row)", total)
	}
	if cashOut != 0 {
		t.Errorf("register 1 cash_out movements after void = %d, want 0", cashOut)
	}

	// Guard (AC-8): the rest of the void is unchanged.
	var status string
	var voidedBy int64
	if err := db.DB.QueryRowContext(ctx, `SELECT status, COALESCE(voided_by, 0) FROM sales WHERE id = 1`).Scan(&status, &voidedBy); err != nil {
		t.Fatalf("read sale: %v", err)
	}
	if status != "voided" || voidedBy != 1 {
		t.Errorf("sale 1 status/voided_by = %q/%d, want voided/1", status, voidedBy)
	}

	var stock float64
	if err := db.DB.QueryRowContext(ctx, `SELECT stock FROM products WHERE id = 1`).Scan(&stock); err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if stock != 10 {
		t.Errorf("product 1 stock after void = %.0f, want 10", stock)
	}

	var auditCount int
	if err := db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log WHERE action = 'void' AND entity = 'sale' AND entity_id = 1`).Scan(&auditCount); err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("audit_log void/sale/1 rows = %d, want 1", auditCount)
	}
	var details string
	if err := db.DB.QueryRowContext(ctx, `SELECT details FROM audit_log WHERE action = 'void' AND entity = 'sale' AND entity_id = 1`).Scan(&details); err != nil {
		t.Fatalf("read audit details: %v", err)
	}
	if details != `{"total":50000.00}` {
		t.Errorf("audit details = %s, want {\"total\":50000.00}", details)
	}
}
