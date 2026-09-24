package repository

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/database"
)

// ─── helpers (US-059) ────────────────────────────────────────────────────────

// openMetricsDB follows the inline pattern of report_test.go:14.
func openMetricsDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func metricsExec(t *testing.T, db *database.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.DB.ExecContext(context.Background(), q, args...); err != nil {
		t.Fatalf("seed exec: %v\nSQL: %s", err, q)
	}
}

// metricsToday0 returns 00:00 local time of today.
func metricsToday0() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func approxEq(a, b float64) bool { return math.Abs(a-b) < 0.01 }

// seedProfitT4 is the seed shared by T-4, T-5 and T-8.
// Prod A (10.000 / cost 5.000), Prod B (6.000 / cost 0), Prod C (12.000 / cost 8.000);
// one sale today with one of each ($28.000); expense today $3.000, yesterday $2.000.
func seedProfitT4(t *testing.T, db *database.DB, hoy0 time.Time) {
	t.Helper()
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Prod A', 'A', 10000, 5000, 'unit', 1)`)
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (2, 'Prod B', 'B', 6000, 0, 'unit', 1)`)
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (3, 'Prod C', 'C', 12000, 8000, 'unit', 1)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 28000, 28000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 1, 10000, 10000)`)
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 2, 1, 6000, 6000)`)
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 3, 1, 12000, 12000)`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'cash', 28000)`)
	metricsExec(t, db, `INSERT INTO expenses (user_id, category, amount, created_at) VALUES (1, 'otro', 3000, ?)`, hoy0.Add(2*time.Hour))
	metricsExec(t, db, `INSERT INTO expenses (user_id, category, amount, created_at) VALUES (1, 'otro', 2000, ?)`, hoy0.Add(-12*time.Hour))
}

// seedCashT6 is the seed of T-6 (reused by T-7).
func seedCashT6(t *testing.T, db *database.DB, hoy0 time.Time) {
	t.Helper()
	metricsExec(t, db, `INSERT INTO registers (id, user_id, opening_amount) VALUES (1, 1, 0)`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Ana', '', '')`)
	// Mixed sale $50.000: cash 30.000 + credit 20.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 50000, 50000, 'completed', 'mixed', ?)`, hoy0.Add(time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'cash', 30000)`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 20000)`)
	// Cash sale $10.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 10000, 10000, 'completed', 'cash', ?)`, hoy0.Add(2*time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'cash', 10000)`)
	// Movements.
	metricsExec(t, db, `INSERT INTO register_movements (register_id, type, amount, created_at) VALUES (1, 'cash_out', 5000, ?)`, hoy0.Add(3*time.Hour))
	metricsExec(t, db, `INSERT INTO register_movements (register_id, type, amount, created_at) VALUES (1, 'cash_out', 4000, ?)`, hoy0.Add(-12*time.Hour))
	metricsExec(t, db, `INSERT INTO register_movements (register_id, type, amount, created_at) VALUES (1, 'cash_in', 8000, ?)`, hoy0.Add(4*time.Hour))
}

// seedDebtorsT12 is the seed of T-12 (reused by T-13).
// Ana: credit $150.000. Beto: 2 credits of $50.000 + payment $20.000 → $80.000.
// Carla: credit $30.000 + payment $30.000 → $0.
func seedDebtorsT12(t *testing.T, db *database.DB, hoy0 time.Time) {
	t.Helper()
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Ana', '3001111111', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Beto', '3002222222', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (3, 'Carla', '', '')`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 150000, 150000, 'completed', 'credit', ?)`, hoy0.Add(-72*time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 150000)`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2, 50000, 50000, 'completed', 'credit', ?)`, hoy0.Add(-48*time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 50000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 2, 50000, 50000, 'completed', 'credit', ?)`, hoy0.Add(-24*time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 50000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 20000, ?)`, hoy0.Add(time.Hour))

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (4, 1, 3, 30000, 30000, 'completed', 'credit', ?)`, hoy0.Add(-24*time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (4, 'credit', 30000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (3, 30000, ?)`, hoy0.Add(time.Hour))
}

// ─── T-1 .. T-3: Sold ────────────────────────────────────────────────────────

func TestMetricsRepo_T1_SoldTodayAndYesterday(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()

	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 28000, 28000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 12000, 12000, 'completed', 'cash', ?)`, hoy0.Add(2*time.Hour))
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 50000, 50000, 'completed', 'cash', ?)`, hoy0.Add(-12*time.Hour))

	repo := NewMetricsRepository(db.DB)

	total, count, err := repo.Sold(ctx, hoy0, hoy0.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Sold(today): %v", err)
	}
	if total != 40000 || count != 2 {
		t.Errorf("Sold(today) = %.0f / %d, want 40000 / 2", total, count)
	}

	total, count, err = repo.Sold(ctx, hoy0.Add(-24*time.Hour), hoy0)
	if err != nil {
		t.Fatalf("Sold(yesterday): %v", err)
	}
	if total != 50000 || count != 1 {
		t.Errorf("Sold(yesterday) = %.0f / %d, want 50000 / 1", total, count)
	}
}

func TestMetricsRepo_T2_SoldHalfOpenMidnight(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()
	manana0 := hoy0.Add(24 * time.Hour)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 10000, 10000, 'completed', 'cash', ?)`, manana0.Add(-time.Second))
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 7000, 7000, 'completed', 'cash', ?)`, manana0)

	repo := NewMetricsRepository(db.DB)

	total, count, err := repo.Sold(ctx, hoy0, manana0)
	if err != nil {
		t.Fatalf("Sold(today): %v", err)
	}
	if total != 10000 || count != 1 {
		t.Errorf("Sold(today) = %.0f / %d, want 10000 / 1 (midnight sale belongs to tomorrow)", total, count)
	}

	total, count, err = repo.Sold(ctx, manana0, manana0.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Sold(tomorrow): %v", err)
	}
	if total != 7000 || count != 1 {
		t.Errorf("Sold(tomorrow) = %.0f / %d, want 7000 / 1", total, count)
	}
}

func TestMetricsRepo_T3_SoldIgnoresVoided(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()

	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 15000, 15000, 'voided', 'cash', ?)`, hoy0.Add(time.Hour))
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 20000, 20000, 'completed', 'cash', ?)`, hoy0.Add(2*time.Hour))

	total, count, err := NewMetricsRepository(db.DB).Sold(ctx, hoy0, hoy0.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Sold: %v", err)
	}
	if total != 20000 || count != 1 {
		t.Errorf("Sold(today) = %.0f / %d, want 20000 / 1", total, count)
	}
}

// ─── T-4, T-5: Profit ────────────────────────────────────────────────────────

func TestMetricsRepo_T4_Profit(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()
	seedProfitT4(t, db, hoy0)

	gross, expenses, net, unconfigured, err := NewMetricsRepository(db.DB).Profit(ctx, hoy0, hoy0.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Profit: %v", err)
	}
	// (10.000-5.000)*1 + (12.000-8.000)*1 = 9.000; Prod B (cost 0) excluded and counted.
	if gross != 9000 {
		t.Errorf("gross = %.0f, want 9000", gross)
	}
	if expenses != 3000 {
		t.Errorf("expenses = %.0f, want 3000", expenses)
	}
	if net != 6000 {
		t.Errorf("net = %.0f, want 6000", net)
	}
	if unconfigured != 1 {
		t.Errorf("unconfiguredCost = %d, want 1", unconfigured)
	}
}

func TestMetricsRepo_T5_ProfitExpensesMatchSumRange(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()
	seedProfitT4(t, db, hoy0)
	from, to := hoy0, hoy0.Add(24*time.Hour)

	_, expenses, _, _, err := NewMetricsRepository(db.DB).Profit(ctx, from, to)
	if err != nil {
		t.Fatalf("Profit: %v", err)
	}
	sum, err := NewExpenseRepository(db.DB).SumRange(ctx, from, to)
	if err != nil {
		t.Fatalf("SumRange: %v", err)
	}
	if expenses != sum || expenses != 3000 {
		t.Errorf("Profit.expenses = %.0f, SumRange = %.0f, want both 3000", expenses, sum)
	}
}

// ─── T-6, T-7: CashFlow / CreditFlow ─────────────────────────────────────────

func TestMetricsRepo_T6_CashFlow(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()
	seedCashT6(t, db, hoy0)

	cashSales, cashOut, err := NewMetricsRepository(db.DB).CashFlow(ctx, hoy0, hoy0.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("CashFlow: %v", err)
	}
	// 30.000 (cash part of mixed) + 10.000 = 40.000; cash_in 8.000 and yesterday's cash_out ignored.
	if cashSales != 40000 {
		t.Errorf("cashSales = %.0f, want 40000", cashSales)
	}
	if cashOut != 5000 {
		t.Errorf("cashOut = %.0f, want 5000", cashOut)
	}
}

func TestMetricsRepo_T7_CreditFlow(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()
	seedCashT6(t, db, hoy0)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 1, 15000, 15000, 'completed', 'credit', ?)`, hoy0.Add(-12*time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 15000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 12000, ?)`, hoy0.Add(5*time.Hour))
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 6000, ?)`, hoy0.Add(-6*time.Hour))

	repo := NewMetricsRepository(db.DB)

	newCredit, collected, err := repo.CreditFlow(ctx, hoy0, hoy0.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("CreditFlow(today): %v", err)
	}
	if newCredit != 20000 || collected != 12000 {
		t.Errorf("CreditFlow(today) = %.0f / %.0f, want 20000 / 12000", newCredit, collected)
	}

	newCredit, collected, err = repo.CreditFlow(ctx, hoy0.Add(-24*time.Hour), hoy0)
	if err != nil {
		t.Fatalf("CreditFlow(yesterday): %v", err)
	}
	if newCredit != 15000 || collected != 6000 {
		t.Errorf("CreditFlow(yesterday) = %.0f / %.0f, want 15000 / 6000", newCredit, collected)
	}
}

// ─── T-8, T-9: TopByMargin ───────────────────────────────────────────────────

func TestMetricsRepo_T8_TopByMargin(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()
	seedProfitT4(t, db, hoy0)
	repo := NewMetricsRepository(db.DB)
	from, to := hoy0, hoy0.Add(24*time.Hour)

	items, err := repo.TopByMargin(ctx, from, to, 5)
	if err != nil {
		t.Fatalf("TopByMargin(5): %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("TopByMargin(5) len = %d, want 2 (Prod B without cost excluded): %+v", len(items), items)
	}
	if items[0].Name != "Prod A" || !approxEq(items[0].MarginPct, 50.00) || items[0].UnitsSold != 1 {
		t.Errorf("items[0] = %+v, want Prod A 50.00%% / 1 u.", items[0])
	}
	if items[1].Name != "Prod C" || !approxEq(items[1].MarginPct, 33.33) || items[1].UnitsSold != 1 {
		t.Errorf("items[1] = %+v, want Prod C 33.33%% / 1 u.", items[1])
	}
	for _, it := range items {
		if it.Name == "Prod B" {
			t.Errorf("Prod B (cost 0) must not appear in TopByMargin")
		}
	}

	items, err = repo.TopByMargin(ctx, from, to, 1)
	if err != nil {
		t.Fatalf("TopByMargin(1): %v", err)
	}
	if len(items) != 1 || items[0].Name != "Prod A" {
		t.Errorf("TopByMargin(1) = %+v, want [Prod A]", items)
	}
}

func TestMetricsRepo_T9_TopByMarginSingleItem(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()

	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Prod A', 'A', 10000, 5000, 'unit', 1)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 10000, 10000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 1, 10000, 10000)`)

	items, err := NewMetricsRepository(db.DB).TopByMargin(ctx, hoy0, hoy0.Add(24*time.Hour), 5)
	if err != nil {
		t.Fatalf("TopByMargin: %v", err)
	}
	// The ">= 2 items" rule belongs to the handler, not the repository.
	if len(items) != 1 || items[0].Name != "Prod A" || !approxEq(items[0].MarginPct, 50.00) {
		t.Errorf("TopByMargin = %+v, want exactly [Prod A 50.00%%]", items)
	}
}

// ─── T-10: TopSeller ─────────────────────────────────────────────────────────

func TestMetricsRepo_T10_TopSeller(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()

	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Arroz', 'ARZ', 3000, 2500, 'unit', 1)`)
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (2, 'Aceite', 'ACE', 12000, 8000, 'unit', 1)`)
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (3, 'Gaseosa', 'GAS', 4000, 0, 'unit', 1)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 56000, 56000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 4, 3000, 12000)`)
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 2, 3, 12000, 36000)`)
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 3, 2, 4000, 8000)`)

	name, qty, err := NewMetricsRepository(db.DB).TopSeller(ctx, hoy0, hoy0.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("TopSeller: %v", err)
	}
	if name != "Arroz" || qty != 4 {
		t.Errorf("TopSeller = %q / %.0f, want Arroz / 4", name, qty)
	}
}

// ─── T-11: LowStock ──────────────────────────────────────────────────────────

func TestMetricsRepo_T11_LowStock(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()

	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock, min_stock) VALUES (1, 'X', 'X', 1000, 500, 'unit', 1, 2, 10)`)
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock, min_stock) VALUES (2, 'Y', 'Y', 1000, 500, 'unit', 1, 4, 5)`)
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock, min_stock) VALUES (3, 'Z', 'Z', 1000, 500, 'unit', 1, 0, 0)`)
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock, min_stock) VALUES (4, 'W', 'W', 1000, 500, 'unit', 0, 1, 20)`)

	repo := NewMetricsRepository(db.DB)

	items, err := repo.LowStock(ctx, 5)
	if err != nil {
		t.Fatalf("LowStock(5): %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("LowStock(5) len = %d, want 2 (X, Y): %+v", len(items), items)
	}
	if items[0].ProductID != 1 || items[0].Name != "X" || items[0].Stock != 2 || items[0].MinStock != 10 {
		t.Errorf("items[0] = %+v, want X (id 1) 2/10", items[0])
	}
	if items[1].ProductID != 2 || items[1].Name != "Y" || items[1].Stock != 4 || items[1].MinStock != 5 {
		t.Errorf("items[1] = %+v, want Y (id 2) 4/5", items[1])
	}

	items, err = repo.LowStock(ctx, 1)
	if err != nil {
		t.Fatalf("LowStock(1): %v", err)
	}
	if len(items) != 1 || items[0].Name != "X" {
		t.Errorf("LowStock(1) = %+v, want [X]", items)
	}
}

// ─── T-12 .. T-15: Debtors ───────────────────────────────────────────────────

func TestMetricsRepo_T12_Debtors(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	seedDebtorsT12(t, db, metricsToday0())

	debtors, err := NewMetricsRepository(db.DB).Debtors(ctx, 5)
	if err != nil {
		t.Fatalf("Debtors(5): %v", err)
	}
	if len(debtors) != 2 {
		t.Fatalf("Debtors(5) len = %d, want 2 (Carla at $0 excluded): %+v", len(debtors), debtors)
	}
	if debtors[0].ID != 1 || debtors[0].Name != "Ana" || debtors[0].Phone != "3001111111" || debtors[0].Balance != 150000 {
		t.Errorf("debtors[0] = %+v, want Ana (id 1, 3001111111) 150000", debtors[0])
	}
	// Beto: 50.000 + 50.000 − 20.000 = 80.000. If payments were multiplied by the
	// number of credit payments (D-4 formula) it would show 60.000.
	if debtors[1].ID != 2 || debtors[1].Name != "Beto" || debtors[1].Phone != "3002222222" || debtors[1].Balance != 80000 {
		t.Errorf("debtors[1] = %+v, want Beto (id 2, 3002222222) 80000", debtors[1])
	}
}

func TestMetricsRepo_T13_DebtorsLimit(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	seedDebtorsT12(t, db, metricsToday0())
	repo := NewMetricsRepository(db.DB)

	debtors, err := repo.Debtors(ctx, 1)
	if err != nil {
		t.Fatalf("Debtors(1): %v", err)
	}
	if len(debtors) != 1 || debtors[0].Name != "Ana" {
		t.Errorf("Debtors(1) = %+v, want [Ana]", debtors)
	}

	debtors, err = repo.Debtors(ctx, 0)
	if err != nil {
		t.Fatalf("Debtors(0): %v", err)
	}
	if len(debtors) != 2 || debtors[0].Name != "Ana" || debtors[1].Name != "Beto" {
		t.Errorf("Debtors(0) = %+v, want [Ana, Beto] (no limit)", debtors)
	}
}

func TestMetricsRepo_T14_DebtorsTieByName(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()

	// Dora is inserted before Beto (lower id) so the tie is broken by name, not by id.
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '3004444444', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Beto', '3002222222', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (3, 'Ana', '3001111111', '')`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 80000, 80000, 'completed', 'credit', ?)`, hoy0.Add(-24*time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 80000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2, 80000, 80000, 'completed', 'credit', ?)`, hoy0.Add(-24*time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 80000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 3, 150000, 150000, 'completed', 'credit', ?)`, hoy0.Add(-24*time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 150000)`)

	debtors, err := NewMetricsRepository(db.DB).Debtors(ctx, 0)
	if err != nil {
		t.Fatalf("Debtors(0): %v", err)
	}
	want := []struct {
		name    string
		balance float64
	}{{"Ana", 150000}, {"Beto", 80000}, {"Dora", 80000}}
	if len(debtors) != len(want) {
		t.Fatalf("Debtors(0) len = %d, want 3: %+v", len(debtors), debtors)
	}
	for i, w := range want {
		if debtors[i].Name != w.name || debtors[i].Balance != w.balance {
			t.Errorf("debtors[%d] = %s %.0f, want %s %.0f", i, debtors[i].Name, debtors[i].Balance, w.name, w.balance)
		}
	}
}

func TestMetricsRepo_T15_DebtorsIgnoresVoidedCredit(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()

	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Eva', '3005555555', '')`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 40000, 40000, 'voided', 'credit', ?)`, hoy0.Add(time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 40000)`)
	// Control debtor so an implementation that returns nothing cannot pass.
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Fer', '3006666666', '')`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2, 10000, 10000, 'completed', 'credit', ?)`, hoy0.Add(2*time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 10000)`)

	debtors, err := NewMetricsRepository(db.DB).Debtors(ctx, 0)
	if err != nil {
		t.Fatalf("Debtors(0): %v", err)
	}
	for _, d := range debtors {
		if d.Name == "Eva" {
			t.Errorf("Eva (only a voided credit sale of 40000) must not appear, got balance %.0f", d.Balance)
		}
	}
	if len(debtors) != 1 || debtors[0].Name != "Fer" || debtors[0].Balance != 10000 {
		t.Errorf("Debtors(0) = %+v, want only [Fer 10000]", debtors)
	}
}

// ─── T-16, T-17: AgingReceivables ────────────────────────────────────────────

func TestMetricsRepo_T16_AgingReceivables(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()

	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Ana', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Beto', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (3, 'Carla', '', '')`)

	// Beto: credit $30.000 ten days ago, payment $10.000 → 20.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 2, 30000, 30000, 'completed', 'credit', datetime('now','-10 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 30000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 10000, datetime('now','-1 days'))`)
	// Ana: credit $20.000 three days ago, payment $5.000 → 15.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 1, 20000, 20000, 'completed', 'credit', datetime('now','-3 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 20000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 5000, datetime('now','-1 days'))`)
	// Carla: credit $10.000 five days ago, payment $10.000 → 0.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 3, 10000, 10000, 'completed', 'credit', datetime('now','-5 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 10000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (3, 10000, datetime('now','-1 days'))`)

	items, err := NewMetricsRepository(db.DB).AgingReceivables(ctx, 5)
	if err != nil {
		t.Fatalf("AgingReceivables(5): %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("AgingReceivables(5) len = %d, want 2 (Carla at 0 excluded): %+v", len(items), items)
	}
	if items[0].CustomerID != 2 || items[0].Name != "Beto" || items[0].Balance != 20000 || items[0].DaysOverdue != 10 {
		t.Errorf("items[0] = %+v, want Beto (id 2) 20000 / 10 days", items[0])
	}
	if items[1].CustomerID != 1 || items[1].Name != "Ana" || items[1].Balance != 15000 || items[1].DaysOverdue != 3 {
		t.Errorf("items[1] = %+v, want Ana (id 1) 15000 / 3 days", items[1])
	}
}

func TestMetricsRepo_T17_AgingReceivablesPaymentsCountedOnce(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()

	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 25000, 25000, 'completed', 'credit', datetime('now','-7 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 25000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 1, 25000, 25000, 'completed', 'credit', datetime('now','-6 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 25000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 10000, datetime('now','-1 days'))`)

	items, err := NewMetricsRepository(db.DB).AgingReceivables(ctx, 5)
	if err != nil {
		t.Fatalf("AgingReceivables(5): %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("AgingReceivables(5) len = %d, want 1: %+v", len(items), items)
	}
	if items[0].CustomerID != 1 || items[0].Name != "Dora" {
		t.Errorf("items[0] = %+v, want Dora (id 1)", items[0])
	}
	// 25.000 + 25.000 − 10.000 = 40.000: the payment is subtracted once, not once per credit sale.
	if items[0].Balance != 40000 {
		t.Errorf("Dora balance = %.0f, want 40000", items[0].Balance)
	}
	if items[0].DaysOverdue != 7 {
		t.Errorf("Dora days_overdue = %v, want 7", items[0].DaysOverdue)
	}
}

// ─── T-40, T-41 (US-065): AgingReceivables tie-break ─────────────────────────

// seedAgingTieBreak seeds Zeta (id 1, saldo $10.000, days_overdue 5), Alfa
// (id 2, saldo $50.000, days_overdue 5) and Beto (id 3, saldo $50.000,
// days_overdue 5). Insertion order (Zeta, Alfa, Beto) deliberately does not
// match the expected output order, so a leftover ORDER BY rowid/id would fail
// both T-40 and T-41.
func seedAgingTieBreak(t *testing.T, db *database.DB) {
	t.Helper()
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Zeta', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Alfa', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (3, 'Beto', '', '')`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 10000, 10000, 'completed', 'credit', datetime('now','-5 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 10000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2, 50000, 50000, 'completed', 'credit', datetime('now','-5 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 50000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 3, 50000, 50000, 'completed', 'credit', datetime('now','-5 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 50000)`)
}

func TestMetricsRepo_T40_AgingTieBreakByBalanceThenName(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	seedAgingTieBreak(t, db)

	// Dora: saldo $5.000, days_overdue 9 (control de no-regresion: days_overdue manda primero).
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (4, 'Dora', '', '')`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (4, 1, 4, 5000, 5000, 'completed', 'credit', datetime('now','-9 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (4, 'credit', 5000)`)

	items, err := NewMetricsRepository(db.DB).AgingReceivables(ctx, 5)
	if err != nil {
		t.Fatalf("AgingReceivables(5): %v", err)
	}
	want := []struct {
		name    string
		balance float64
		days    int
	}{
		{"Dora", 5000, 9},
		{"Alfa", 50000, 5},
		{"Beto", 50000, 5},
		{"Zeta", 10000, 5},
	}
	if len(items) != len(want) {
		t.Fatalf("AgingReceivables(5) len = %d, want %d: %+v", len(items), len(want), items)
	}
	for i, w := range want {
		if items[i].Name != w.name || items[i].Balance != w.balance || items[i].DaysOverdue != w.days {
			t.Errorf("items[%d] = %+v, want %s %.0f / %d dias", i, items[i], w.name, w.balance, w.days)
		}
	}
}

func TestMetricsRepo_T41_AgingTieBreakDecidesLimitCutoff(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	seedAgingTieBreak(t, db)

	items, err := NewMetricsRepository(db.DB).AgingReceivables(ctx, 2)
	if err != nil {
		t.Fatalf("AgingReceivables(2): %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("AgingReceivables(2) len = %d, want 2: %+v", len(items), items)
	}
	want := []struct {
		name    string
		balance float64
	}{{"Alfa", 50000}, {"Beto", 50000}}
	for i, w := range want {
		if items[i].Name != w.name || items[i].Balance != w.balance {
			t.Errorf("items[%d] = %+v, want %s %.0f", i, items[i], w.name, w.balance)
		}
	}
	for _, it := range items {
		if it.Name == "Zeta" {
			t.Errorf("Zeta (saldo mas bajo, $10.000) no debe entrar en el corte LIMIT 2, got %+v", items)
		}
	}
}

// ─── T-18: empty range ───────────────────────────────────────────────────────

func TestMetricsRepo_T18_EmptyRange(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	manana0 := metricsToday0().Add(24 * time.Hour)
	from, to := manana0, manana0.Add(24*time.Hour)
	repo := NewMetricsRepository(db.DB)

	total, count, err := repo.Sold(ctx, from, to)
	if err != nil || total != 0 || count != 0 {
		t.Errorf("Sold = %.0f / %d / %v, want 0 / 0 / nil", total, count, err)
	}

	gross, expenses, net, unconfigured, err := repo.Profit(ctx, from, to)
	if err != nil || gross != 0 || expenses != 0 || net != 0 || unconfigured != 0 {
		t.Errorf("Profit = %.0f / %.0f / %.0f / %d / %v, want 0/0/0/0/nil", gross, expenses, net, unconfigured, err)
	}

	cashSales, cashOut, err := repo.CashFlow(ctx, from, to)
	if err != nil || cashSales != 0 || cashOut != 0 {
		t.Errorf("CashFlow = %.0f / %.0f / %v, want 0 / 0 / nil", cashSales, cashOut, err)
	}

	newCredit, collected, err := repo.CreditFlow(ctx, from, to)
	if err != nil || newCredit != 0 || collected != 0 {
		t.Errorf("CreditFlow = %.0f / %.0f / %v, want 0 / 0 / nil", newCredit, collected, err)
	}

	name, qty, err := repo.TopSeller(ctx, from, to)
	if err != nil || name != "" || qty != 0 {
		t.Errorf("TopSeller = %q / %.0f / %v, want \"\" / 0 / nil", name, qty, err)
	}

	margin, err := repo.TopByMargin(ctx, from, to, 5)
	if err != nil || margin == nil || len(margin) != 0 {
		t.Errorf("TopByMargin = %#v / %v, want empty non-nil slice / nil", margin, err)
	}

	low, err := repo.LowStock(ctx, 5)
	if err != nil || low == nil || len(low) != 0 {
		t.Errorf("LowStock = %#v / %v, want empty non-nil slice / nil", low, err)
	}

	debtors, err := repo.Debtors(ctx, 5)
	if err != nil || debtors == nil || len(debtors) != 0 {
		t.Errorf("Debtors = %#v / %v, want empty non-nil slice / nil", debtors, err)
	}

	aging, err := repo.AgingReceivables(ctx, 5)
	if err != nil || aging == nil || len(aging) != 0 {
		t.Errorf("AgingReceivables = %#v / %v, want empty non-nil slice / nil", aging, err)
	}
}

// ─── T-19: errors wrapped with %w ────────────────────────────────────────────

func TestMetricsRepo_T19_ClosedDBWrapsErrors(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()
	from, to := hoy0, hoy0.Add(24*time.Hour)
	repo := NewMetricsRepository(db.DB)
	db.Close()

	check := func(method string, err error) {
		t.Helper()
		if err == nil {
			t.Errorf("%s: err = nil, want error on closed DB", method)
			return
		}
		if errors.Unwrap(err) == nil {
			t.Errorf("%s: error %q is not wrapped with %%w", method, err)
		}
	}

	_, _, err := repo.Sold(ctx, from, to)
	check("Sold", err)
	_, _, _, _, err = repo.Profit(ctx, from, to)
	check("Profit", err)
	_, _, err = repo.CashFlow(ctx, from, to)
	check("CashFlow", err)
	_, _, err = repo.CreditFlow(ctx, from, to)
	check("CreditFlow", err)
	_, err = repo.TopByMargin(ctx, from, to, 5)
	check("TopByMargin", err)
	_, _, err = repo.TopSeller(ctx, from, to)
	check("TopSeller", err)
	_, err = repo.LowStock(ctx, 5)
	check("LowStock", err)
	_, err = repo.Debtors(ctx, 5)
	check("Debtors", err)
	_, err = repo.AgingReceivables(ctx, 5)
	check("AgingReceivables", err)
}

// ─── T-30: aging rows with NULL days_overdue are excluded in SQL ─────────────

func TestMetricsRepo_T30_AgingExcludesNullDaysOverdue(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()

	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Gil', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Beto', '', '')`)

	// Gil: credit $12.000 seeded with time.Time → julianday() is NULL (D-5).
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 12000, 12000, 'completed', 'credit', ?)`, hoy0.Add(-48*time.Hour))
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 12000)`)
	// Beto (T-16): credit $30.000 ten days ago, payment $10.000 → 20.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2, 30000, 30000, 'completed', 'credit', datetime('now','-10 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 30000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 10000, datetime('now','-1 days'))`)

	items, err := NewMetricsRepository(db.DB).AgingReceivables(ctx, 5)
	if err != nil {
		t.Fatalf("AgingReceivables(5): err = %v, want nil (Gil must be excluded in SQL, not by a failed scan)", err)
	}
	if len(items) != 1 {
		t.Fatalf("AgingReceivables(5) len = %d, want 1 (only Beto): %+v", len(items), items)
	}
	if items[0].CustomerID != 2 || items[0].Name != "Beto" || items[0].Balance != 20000 || items[0].DaysOverdue != 10 {
		t.Errorf("items[0] = %+v, want Beto (id 2) 20000 / 10 days", items[0])
	}
}

// ─── T-31..T-35: AgingReceivables uses the single balance definition (US-063) ─

// agingRow is a compact expected row for aging assertions.
type agingRow struct {
	id      int64
	name    string
	balance float64
	days    int
}

func assertAging(t *testing.T, got []AgingReceivable, want []agingRow) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("AgingReceivables len = %d, want %d: got %+v, want %+v", len(got), len(want), got, want)
	}
	for i, w := range want {
		g := got[i]
		if g.CustomerID != w.id || g.Name != w.name || g.Balance != w.balance || g.DaysOverdue != w.days {
			t.Errorf("aging[%d] = {%d %s %.0f %d}, want {%d %s %.0f %d}",
				i, g.CustomerID, g.Name, g.Balance, g.DaysOverdue, w.id, w.name, w.balance, w.days)
		}
	}
}

// seedAgingT31 seeds Beto, Eli and Dora with datetime('now', ...) timestamps.
// Beto (id 1): credit 30.000 (−10 d), payment 10.000 → 20.000 / 10 d.
// Eli (id 2): 3 × credit 10.000 (−9, −8, −7 d), payments 5.000 + 10.000 → 15.000 / 9 d.
// Dora (id 3): credit 25.000 (−7 d) + 25.000 (−6 d), payment 10.000 → 40.000 / 7 d.
func seedAgingT31(t *testing.T, db *database.DB) {
	t.Helper()
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Beto', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Eli', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (3, 'Dora', '', '')`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 30000, 30000, 'completed', 'credit', datetime('now','-10 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 30000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 10000, datetime('now','-1 days'))`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2, 10000, 10000, 'completed', 'credit', datetime('now','-9 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 10000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 2, 10000, 10000, 'completed', 'credit', datetime('now','-8 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 10000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (4, 1, 2, 10000, 10000, 'completed', 'credit', datetime('now','-7 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (4, 'credit', 10000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 5000, datetime('now','-2 days'))`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 10000, datetime('now','-1 days'))`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (5, 1, 3, 25000, 25000, 'completed', 'credit', datetime('now','-7 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (5, 'credit', 25000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (6, 1, 3, 25000, 25000, 'completed', 'credit', datetime('now','-6 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (6, 'credit', 25000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (3, 10000, datetime('now','-1 days'))`)
}

func TestMetricsRepo_T31_AgingSeveralCreditsAndPayments(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	seedAgingT31(t, db)

	items, err := NewMetricsRepository(db.DB).AgingReceivables(ctx, 5)
	if err != nil {
		t.Fatalf("AgingReceivables(5): %v", err)
	}
	// Eli: 30.000 − 15.000 = 15.000 (must appear); Dora: 50.000 − 10.000 = 40.000.
	assertAging(t, items, []agingRow{
		{1, "Beto", 20000, 10},
		{2, "Eli", 15000, 9},
		{3, "Dora", 40000, 7},
	})
}

func TestMetricsRepo_T32_AgingExcludesExactZeroBalance(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()

	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Fer', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Ana', '', '')`)

	// Fer: 2 × 20.000 (−5, −4 d), payment 40.000 → exactly 0.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 20000, 20000, 'completed', 'credit', datetime('now','-5 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 20000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 1, 20000, 20000, 'completed', 'credit', datetime('now','-4 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 20000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 40000, datetime('now','-1 days'))`)
	// Ana (control): 20.000 (−3 d), payment 5.000 → 15.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 2, 20000, 20000, 'completed', 'credit', datetime('now','-3 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 20000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 5000, datetime('now','-1 days'))`)

	items, err := NewMetricsRepository(db.DB).AgingReceivables(ctx, 5)
	if err != nil {
		t.Fatalf("AgingReceivables(5): %v", err)
	}
	assertAging(t, items, []agingRow{{2, "Ana", 15000, 3}})
}

func TestMetricsRepo_T33_AgingExcludesNegativeBalance(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()

	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Gus', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Hana', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (3, 'Beto', '', '')`)

	// Gus: 10.000 (−3 d), payment 25.000 → −15.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 10000, 10000, 'completed', 'credit', datetime('now','-3 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 10000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 25000, datetime('now','-1 days'))`)
	// Hana: 2 × 30.000 (−6, −5 d), payment 70.000 → −10.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2, 30000, 30000, 'completed', 'credit', datetime('now','-6 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 30000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 2, 30000, 30000, 'completed', 'credit', datetime('now','-5 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 30000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 70000, datetime('now','-1 days'))`)
	// Beto (control): 20.000 (−10 d), no payments → 20.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (4, 1, 3, 20000, 20000, 'completed', 'credit', datetime('now','-10 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (4, 'credit', 20000)`)

	items, err := NewMetricsRepository(db.DB).AgingReceivables(ctx, 5)
	if err != nil {
		t.Fatalf("AgingReceivables(5): %v", err)
	}
	assertAging(t, items, []agingRow{{3, "Beto", 20000, 10}})
}

func TestMetricsRepo_T34_AgingMixedAndVoidedSales(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()

	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Iván', '', '')`)
	// Mixed completed sale (−8 d): credit 15.000 + cash 10.000 → contributes 15.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 25000, 25000, 'completed', 'mixed', datetime('now','-8 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 15000)`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'cash', 10000)`)
	// Credit completed sale (−4 d): 20.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 1, 20000, 20000, 'completed', 'credit', datetime('now','-4 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 20000)`)
	// Credit voided sale (−12 d): 50.000 → counts neither for balance nor for days.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 1, 50000, 50000, 'voided', 'credit', datetime('now','-12 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 50000)`)
	// Payment 5.000 → 15.000 + 20.000 − 5.000 = 30.000.
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 5000, datetime('now','-1 days'))`)

	items, err := NewMetricsRepository(db.DB).AgingReceivables(ctx, 5)
	if err != nil {
		t.Fatalf("AgingReceivables(5): %v", err)
	}
	assertAging(t, items, []agingRow{{1, "Iván", 30000, 8}})
}

func TestMetricsRepo_T35_AgingBalanceMatchesDebtorsListAndBalance(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	seedAgingT31(t, db)

	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (4, 'Fer', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (5, 'Gus', '', '')`)
	// Fer: 2 × 20.000 (−5, −4 d), payment 40.000 → 0.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (7, 1, 4, 20000, 20000, 'completed', 'credit', datetime('now','-5 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (7, 'credit', 20000)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (8, 1, 4, 20000, 20000, 'completed', 'credit', datetime('now','-4 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (8, 'credit', 20000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (4, 40000, datetime('now','-1 days'))`)
	// Gus: 10.000 (−3 d), payment 25.000 → −15.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (9, 1, 5, 10000, 10000, 'completed', 'credit', datetime('now','-3 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (9, 'credit', 10000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (5, 25000, datetime('now','-1 days'))`)

	metrics := NewMetricsRepository(db.DB)
	customers := NewCustomerRepository(db.DB)

	aging, err := metrics.AgingReceivables(ctx, 5)
	if err != nil {
		t.Fatalf("AgingReceivables(5): %v", err)
	}
	debtors, err := metrics.Debtors(ctx, 0)
	if err != nil {
		t.Fatalf("Debtors(0): %v", err)
	}
	list, err := customers.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	debtorBal := map[int64]float64{}
	for _, d := range debtors {
		debtorBal[d.ID] = d.Balance
	}
	listBal := map[int64]float64{}
	for _, c := range list {
		listBal[c.ID] = c.Balance
	}

	// Anchors: Debtors(0) is exactly {Beto 1: 20.000, Eli 2: 15.000, Dora 3: 40.000}; List agrees.
	wantBal := map[int64]float64{1: 20000, 2: 15000, 3: 40000}
	if len(debtors) != len(wantBal) {
		t.Errorf("Debtors(0) = %+v, want exactly ids {1,2,3}", debtors)
	}
	for id, want := range wantBal {
		if debtorBal[id] != want || listBal[id] != want {
			t.Errorf("id %d: Debtors %.0f / List %.0f, want %.0f", id, debtorBal[id], listBal[id], want)
		}
	}

	// Invariant: the set of aging ids equals the set of Debtors(0) ids…
	if len(aging) != len(debtors) {
		t.Errorf("aging len = %d, Debtors(0) len = %d, want the same set: aging %+v, debtors %+v", len(aging), len(debtors), aging, debtors)
	}
	seen := map[int64]bool{}
	for _, a := range aging {
		seen[a.CustomerID] = true
		// …and each aging balance equals Debtors, List and Balance() for the same id.
		dBal, ok := debtorBal[a.CustomerID]
		if !ok {
			t.Errorf("aging row %d (%s) not in Debtors(0)", a.CustomerID, a.Name)
		} else if a.Balance != dBal {
			t.Errorf("%s: aging balance %.0f != Debtors balance %.0f", a.Name, a.Balance, dBal)
		}
		if lBal, ok := listBal[a.CustomerID]; !ok || a.Balance != lBal {
			t.Errorf("%s: aging balance %.0f != List balance %.0f (found %v)", a.Name, a.Balance, lBal, ok)
		}
		bal, err := customers.Balance(ctx, a.CustomerID)
		if err != nil {
			t.Fatalf("Balance(%d): %v", a.CustomerID, err)
		}
		if a.Balance != bal {
			t.Errorf("%s: aging balance %.0f != Balance() %.0f", a.Name, a.Balance, bal)
		}
	}
	for id, bal := range debtorBal {
		if !seen[id] {
			t.Errorf("debtor id %d (balance %.0f) missing from aging", id, bal)
		}
	}
}

// ─── T-23 (US-061, AC-21/D-2): TopSellers ranks by units, TopProducts stays by revenue ─

// seedTopSellersT23 seeds A=50 units/$100.000, B=80 units/$40.000, C=10
// units/$200.000 — units and revenue deliberately disagree so an
// implementation that reuses ReportRepository.TopProducts' ORDER BY revenue
// for the assistant would fail this test.
func seedTopSellersT23(t *testing.T, db *database.DB, hoy0 time.Time) {
	t.Helper()
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'A', 'A', 2000, 1000, 'unit', 1)`)
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (2, 'B', 'B', 500, 200, 'unit', 1)`)
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (3, 'C', 'C', 20000, 15000, 'unit', 1)`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 340000, 340000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	// A: 50 units * 2.000 = 100.000
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 50, 2000, 100000)`)
	// B: 80 units * 500 = 40.000
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 2, 80, 500, 40000)`)
	// C: 10 units * 20.000 = 200.000
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 3, 10, 20000, 200000)`)
}

func TestMetricsRepo_T23_TopSellersOrdersByQuantity(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()
	seedTopSellersT23(t, db, hoy0)

	items, err := NewMetricsRepository(db.DB).TopSellers(ctx, hoy0, hoy0.Add(24*time.Hour), 3)
	if err != nil {
		t.Fatalf("TopSellers: %v", err)
	}
	want := []struct {
		name string
		qty  float64
		rev  float64
	}{{"B", 80, 40000}, {"A", 50, 100000}, {"C", 10, 200000}}
	if len(items) != len(want) {
		t.Fatalf("TopSellers len = %d, want %d: %+v", len(items), len(want), items)
	}
	for i, w := range want {
		if items[i].Name != w.name || items[i].Quantity != w.qty || items[i].Revenue != w.rev {
			t.Errorf("items[%d] = %+v, want {%s %.0f %.0f}", i, items[i], w.name, w.qty, w.rev)
		}
	}
}

// TestReportRepo_T23_TopProductsStillOrdersByRevenue is the non-regression
// half of T-23: ReportRepository.TopProducts (used by GET /reports/top-products
// in production) must keep ordering by revenue, unaffected by the new
// MetricsRepository.TopSellers method.
func TestReportRepo_T23_TopProductsStillOrdersByRevenue(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	hoy0 := metricsToday0()
	seedTopSellersT23(t, db, hoy0)

	from, to := hoy0, hoy0.Add(24*time.Hour)
	items, err := NewReportRepository(db.DB).TopProducts(ctx, &from, &to, 3)
	if err != nil {
		t.Fatalf("TopProducts: %v", err)
	}
	want := []struct {
		name string
		rev  float64
	}{{"C", 200000}, {"A", 100000}, {"B", 40000}}
	if len(items) != len(want) {
		t.Fatalf("TopProducts len = %d, want %d: %+v", len(items), len(want), items)
	}
	for i, w := range want {
		if items[i].Name != w.name || items[i].Revenue != w.rev {
			t.Errorf("items[%d] = %+v, want {%s %.0f}", i, items[i], w.name, w.rev)
		}
	}
}
