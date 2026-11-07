package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/model"
)

// seedPortfolioT1 seeds the fixture shared by T-1, T-2 (case 2), T-3, T-5:
//
//	Dora: fiado $40.000, abono $10.000  -> saldo  $30.000  (en cartera)
//	Eli:  fiado $20.000, abono  $5.000  -> saldo  $15.000  (en cartera)
//	Fer:  fiado $20.000, abono $20.000  -> saldo      $0   (fuera de cartera)
//	Gus:  fiado $10.000, abono $30.000  -> saldo -$20.000  (saldo a favor, fuera de cartera)
//
// TotalPortfolioBalance correcto = 30.000 + 15.000 = 45.000
// Valor con el bug (verificado, US-064 contract §2/§4) = 25.000
func seedPortfolioT1(t *testing.T, db *database.DB) {
	t.Helper()
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Eli', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (3, 'Fer', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (4, 'Gus', '', '')`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 40000, 40000, 'completed', 'credit', datetime('now','-5 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 40000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 10000, datetime('now','-1 days'))`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2, 20000, 20000, 'completed', 'credit', datetime('now','-4 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 20000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 5000, datetime('now','-1 days'))`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 3, 20000, 20000, 'completed', 'credit', datetime('now','-3 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 20000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (3, 20000, datetime('now','-1 days'))`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (4, 1, 4, 10000, 10000, 'completed', 'credit', datetime('now','-2 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (4, 'credit', 10000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (4, 30000, datetime('now','-1 days'))`)
}

// seedPortfolioFerGus seeds only Fer and Gus of the T-1 fixture (both with
// balance <= 0), for T-2 case B.
func seedPortfolioFerGus(t *testing.T, db *database.DB) {
	t.Helper()
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (3, 'Fer', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (4, 'Gus', '', '')`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 3, 20000, 20000, 'completed', 'credit', datetime('now','-3 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 20000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (3, 20000, datetime('now','-1 days'))`)

	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (4, 1, 4, 10000, 10000, 'completed', 'credit', datetime('now','-2 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (4, 'credit', 10000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (4, 30000, datetime('now','-1 days'))`)
}

// ─── T-1: AC-1, AC-2, AC-3 ───────────────────────────────────────────────────

func TestCustomerRepo_T1_TotalPortfolioBalance_ExcludesZeroAndNegative(t *testing.T) {
	db := openMetricsDB(t)
	seedPortfolioT1(t, db)
	repo := NewCustomerRepository(db.DB)

	got, err := repo.TotalPortfolioBalance(context.Background())
	if err != nil {
		t.Fatalf("TotalPortfolioBalance: %v", err)
	}

	const bugValue = 25000.0 // (40000+20000+20000+10000) - (10000+5000+20000+30000), the bug this US fixes
	if got == bugValue {
		t.Fatalf("TotalPortfolioBalance = %v, still the buggy net value (R-44)", got)
	}
	const want = 45000.0 // 30.000 (Dora) + 15.000 (Eli)
	if got != want {
		t.Errorf("TotalPortfolioBalance = %v, want %v", got, want)
	}
}

// ─── T-2: AC-4 ───────────────────────────────────────────────────────────────

func TestCustomerRepo_T2_TotalPortfolioBalance_NoDebtorsIsZero(t *testing.T) {
	t.Run("sin clientes", func(t *testing.T) {
		db := openMetricsDB(t)
		repo := NewCustomerRepository(db.DB)

		got, err := repo.TotalPortfolioBalance(context.Background())
		if err != nil {
			t.Fatalf("TotalPortfolioBalance: %v", err)
		}
		if got != 0 {
			t.Errorf("TotalPortfolioBalance = %v, want 0", got)
		}
	})

	t.Run("solo clientes con saldo <= 0", func(t *testing.T) {
		db := openMetricsDB(t)
		seedPortfolioFerGus(t, db)
		repo := NewCustomerRepository(db.DB)

		got, err := repo.TotalPortfolioBalance(context.Background())
		if err != nil {
			t.Fatalf("TotalPortfolioBalance: %v", err)
		}
		if got != 0 {
			t.Errorf("TotalPortfolioBalance = %v, want 0", got)
		}
	})
}

// ─── T-3: AC-1 (equivalence with Balance) ───────────────────────────────────

func TestCustomerRepo_T3_TotalPortfolioBalance_MatchesSumOfBalance(t *testing.T) {
	db := openMetricsDB(t)
	seedPortfolioT1(t, db)
	repo := NewCustomerRepository(db.DB)
	ctx := context.Background()

	doraBalance, err := repo.Balance(ctx, 1)
	if err != nil {
		t.Fatalf("Balance(Dora): %v", err)
	}
	eliBalance, err := repo.Balance(ctx, 2)
	if err != nil {
		t.Fatalf("Balance(Eli): %v", err)
	}

	want := doraBalance + eliBalance
	got, err := repo.TotalPortfolioBalance(ctx)
	if err != nil {
		t.Fatalf("TotalPortfolioBalance: %v", err)
	}
	if got != want {
		t.Errorf("TotalPortfolioBalance = %v, want %v (Balance(Dora)=%v + Balance(Eli)=%v)", got, want, doraBalance, eliBalance)
	}
}

// ─── T-4: AC-6 ───────────────────────────────────────────────────────────────

func TestCustomerRepo_T4_TotalPortfolioBalance_ClosedDBWrapsError(t *testing.T) {
	db := openMetricsDB(t)
	repo := NewCustomerRepository(db.DB)
	db.Close()

	_, err := repo.TotalPortfolioBalance(context.Background())
	if err == nil {
		t.Fatal("TotalPortfolioBalance: err = nil, want error on closed DB")
	}
	if errors.Unwrap(err) == nil {
		t.Errorf("TotalPortfolioBalance: error %q is not wrapped with %%w", err)
	}
}

// ─── T-5 (US-065 T-39): AC-1, AC-2 ───────────────────────────────────────────

// TestCustomerRepo_T5_ListSearchFiltersByNameOrPhone covers List after it stops
// having its own textual copy of the balance SQL and reuses
// customerBalanceExpr/customerBalanceFrom (US-065): search by name, search by
// phone and the no-search path must keep returning the right balances.
func TestCustomerRepo_T5_ListSearchFiltersByNameOrPhone(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()

	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Ana', '3001111111', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Beto', '3002222222', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (3, 'Carla', '3003333333', '')`)

	// Ana: fiado $20.000, sin abono -> saldo 20.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 20000, 20000, 'completed', 'credit', datetime('now','-1 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 20000)`)
	// Beto: fiado $30.000, abono $10.000 -> saldo 20.000.
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2, 30000, 30000, 'completed', 'credit', datetime('now','-1 days'))`)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 30000)`)
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 10000, datetime('now'))`)
	// Carla: sin fiado.

	repo := NewCustomerRepository(db.DB)

	t.Run("por nombre", func(t *testing.T) {
		got, err := repo.List(ctx, "Ana")
		if err != nil {
			t.Fatalf("List(Ana): %v", err)
		}
		if len(got) != 1 || got[0].Name != "Ana" || got[0].Balance != 20000 {
			t.Errorf("List(Ana) = %+v, want 1 resultado Ana con saldo 20000", got)
		}
	})

	t.Run("por telefono", func(t *testing.T) {
		got, err := repo.List(ctx, "300222")
		if err != nil {
			t.Fatalf("List(300222): %v", err)
		}
		if len(got) != 1 || got[0].Name != "Beto" || got[0].Balance != 20000 {
			t.Errorf("List(300222) = %+v, want 1 resultado Beto con saldo 20000", got)
		}
	})

	t.Run("sin busqueda", func(t *testing.T) {
		got, err := repo.List(ctx, "")
		if err != nil {
			t.Fatalf("List(\"\"): %v", err)
		}
		if len(got) != 3 {
			t.Errorf("List(\"\") len = %d, want 3 (Ana, Beto, Carla)", len(got))
		}
	})
}

// ─── US-082 contract §4, T-11 (AC-8) ─────────────────────────────────────────

func TestCustomerRepo_T11_CreatePaymentStoresMethodAndRegister(t *testing.T) {
	db := openMetricsDB(t)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
	metricsExec(t, db, `INSERT INTO registers (id, user_id, opening_amount) VALUES (1, 1, 100000)`)

	repo := NewCustomerRepository(db.DB)
	ctx := context.Background()
	regID := int64(1)

	created, err := repo.CreatePayment(ctx, &model.CustomerPayment{
		CustomerID: 1,
		Amount:     15000,
		Method:     "transfer",
		RegisterID: &regID,
		Note:       "Nequi",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if created.Method != "transfer" {
		t.Errorf("created method = %q, want transfer", created.Method)
	}
	if created.RegisterID == nil || *created.RegisterID != 1 {
		t.Errorf("created register_id = %v, want 1", created.RegisterID)
	}

	payments, err := repo.ListPayments(ctx, 1)
	if err != nil {
		t.Fatalf("ListPayments: %v", err)
	}
	if len(payments) != 1 {
		t.Fatalf("ListPayments len = %d, want 1", len(payments))
	}
	p := payments[0]
	if p.Amount != 15000 || p.Method != "transfer" {
		t.Errorf("listed payment = amount %.0f method %q, want 15000 transfer", p.Amount, p.Method)
	}
	if p.RegisterID == nil || *p.RegisterID != 1 {
		t.Errorf("listed register_id = %v, want 1", p.RegisterID)
	}

	var storedMethod string
	var storedReg sql.NullInt64
	if err := db.DB.QueryRow(`SELECT method, register_id FROM customer_payments WHERE id = ?`, created.ID).Scan(&storedMethod, &storedReg); err != nil {
		t.Fatalf("read stored row: %v", err)
	}
	if storedMethod != "transfer" || !storedReg.Valid || storedReg.Int64 != 1 {
		t.Errorf("stored row = %q/%v, want transfer/1", storedMethod, storedReg)
	}
}

// ─── US-043: GetPOSSummary (contract US-043 §4, T-16..T-18) ─────────────────

// seedTopProductsUS043 seeds customer 1 with 5 completed cash sales over 8
// distinct active products, plus 1 voided sale:
//
//	Arroz  in 4 sales (s1..s4)
//	Leche  in 3 sales (s1..s3)
//	Pan    in 2 sales (s1, s2)
//	Huevos, Café, Azúcar, Sal, Fríjol in 1 sale each
//	s6 (voided): Café x3
//
// If the voided sale were counted, Café would tie Pan at 2 and win on name
// ("Café" < "Pan"), so the voided-sale rule is observable in the top 3.
// withInactive adds Gaseosa (active = 0) to all 5 completed sales.
func seedTopProductsUS043(t *testing.T, db *database.DB, withInactive bool) {
	t.Helper()
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Pedro', '', '')`)
	products := []struct {
		id     int
		name   string
		active int
	}{
		{1, "Arroz", 1}, {2, "Leche", 1}, {3, "Pan", 1}, {4, "Huevos", 1},
		{5, "Café", 1}, {6, "Azúcar", 1}, {7, "Sal", 1}, {8, "Fríjol", 1},
		{9, "Gaseosa", 0},
	}
	for _, p := range products {
		metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (?, ?, ?, 1000, 0, 'unit', ?)`, p.id, p.name, p.name, p.active)
	}
	sales := []struct {
		id    int
		items []int
	}{
		{1, []int{1, 2, 3, 4}},
		{2, []int{1, 2, 3, 5}},
		{3, []int{1, 2, 6}},
		{4, []int{1, 7}},
		{5, []int{8}},
	}
	for _, s := range sales {
		metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (?, 1, 1, 1000, 1000, 'completed', 'cash', datetime('now', '-' || ? || ' days'))`, s.id, s.id+1)
		items := s.items
		if withInactive {
			items = append(items, 9)
		}
		for _, pid := range items {
			metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (?, ?, 1, 1000, 1000)`, s.id, pid)
		}
	}
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (6, 1, 1, 3000, 3000, 'voided', 'cash', datetime('now','-1 days'))`)
	metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (6, 5, 3, 1000, 3000)`)
}

type topProductWant struct {
	name  string
	times int
}

func assertTopProducts(t *testing.T, got []model.CustomerTopProduct, want []topProductWant) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("top_products = %+v, want %d entries %+v", got, len(want), want)
	}
	for i, w := range want {
		if got[i].Name != w.name || got[i].TimesBought != w.times {
			t.Errorf("top_products[%d] = %s(%d), want %s(%d)", i, got[i].Name, got[i].TimesBought, w.name, w.times)
		}
	}
}

// ─── T-16: AC-21 ─────────────────────────────────────────────────────────────

func TestPOSSummaryRepo_T16_TopProducts(t *testing.T) {
	ctx := context.Background()
	wantTop3 := []topProductWant{{"Arroz", 4}, {"Leche", 3}, {"Pan", 2}}

	t.Run("top 3 by distinct completed sales, voided sale ignored", func(t *testing.T) {
		db := openMetricsDB(t)
		seedTopProductsUS043(t, db, false)
		s, err := NewCustomerRepository(db.DB).GetPOSSummary(ctx, 1)
		if err != nil {
			t.Fatalf("GetPOSSummary: %v", err)
		}
		assertTopProducts(t, s.TopProducts, wantTop3)
	})

	t.Run("tie broken by name ascending", func(t *testing.T) {
		db := openMetricsDB(t)
		metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Pedro', '', '')`)
		// Panela gets the lower id and is inserted first, so id or insertion
		// order would put it first; name order must put Arepa first.
		metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Panela', 'PNL', 3000, 0, 'unit', 1)`)
		metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (2, 'Arepa', 'ARP', 2000, 0, 'unit', 1)`)
		metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 3000, 3000, 'completed', 'cash', datetime('now','-2 days'))`)
		metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 1, 3000, 3000)`)
		metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 1, 2000, 2000, 'completed', 'cash', datetime('now','-1 days'))`)
		metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (2, 2, 1, 2000, 2000)`)
		s, err := NewCustomerRepository(db.DB).GetPOSSummary(ctx, 1)
		if err != nil {
			t.Fatalf("GetPOSSummary: %v", err)
		}
		assertTopProducts(t, s.TopProducts, []topProductWant{{"Arepa", 1}, {"Panela", 1}})
	})

	t.Run("inactive product never returned", func(t *testing.T) {
		db := openMetricsDB(t)
		seedTopProductsUS043(t, db, true) // Gaseosa (active = 0) in all 5 completed sales
		s, err := NewCustomerRepository(db.DB).GetPOSSummary(ctx, 1)
		if err != nil {
			t.Fatalf("GetPOSSummary: %v", err)
		}
		for _, p := range s.TopProducts {
			if p.Name == "Gaseosa" {
				t.Errorf("inactive product Gaseosa returned with times_bought %d", p.TimesBought)
			}
		}
		assertTopProducts(t, s.TopProducts, wantTop3)
	})
}

// ─── T-17: AC-20 ─────────────────────────────────────────────────────────────

func TestPOSSummaryRepo_T17_LastVisitIgnoresVoidedSale(t *testing.T) {
	db := openMetricsDB(t)
	now := time.Now().UTC()
	cashAt := now.Add(-72 * time.Hour).Format("2006-01-02 15:04:05")   // completed cash sale, 3 days ago
	voidedAt := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05") // voided sale, 1 day ago
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Pedro', '', '')`)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 12000, 12000, 'completed', 'cash', ?)`, cashAt)
	metricsExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 1, 8000, 8000, 'voided', 'cash', ?)`, voidedAt)

	s, err := NewCustomerRepository(db.DB).GetPOSSummary(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetPOSSummary: %v", err)
	}
	if s.LastVisitAt == nil {
		t.Fatalf("last_visit_at = nil, want %q", cashAt)
	}
	if *s.LastVisitAt != cashAt {
		t.Errorf("last_visit_at = %q, want %q (the completed sale; the voided one is %q)", *s.LastVisitAt, cashAt, voidedAt)
	}
}

// ─── T-18: AC-22 ─────────────────────────────────────────────────────────────

func TestPOSSummaryRepo_T18_LastVisitQueryErrorIsReturned(t *testing.T) {
	ctx := context.Background()

	// Only the last-visit query fails: sales is replaced by a view whose
	// created_at raises "malformed JSON" when evaluated. The balance query
	// evaluates created_at only for rows joined to credit sale_payments (none
	// here) and the top-products query only for rows joined to sale_items
	// (none here); the last-visit query evaluates MAX(created_at) over the
	// customer's one completed sale.
	t.Run("only the last-visit query fails", func(t *testing.T) {
		db := openMetricsDB(t)
		metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Pedro', '', '')`)
		metricsExec(t, db, `DROP TABLE sales`)
		metricsExec(t, db, `CREATE TABLE sales_src (id INTEGER PRIMARY KEY, customer_id INTEGER, status TEXT)`)
		metricsExec(t, db, `INSERT INTO sales_src (id, customer_id, status) VALUES (1, 1, 'completed')`)
		metricsExec(t, db, `CREATE VIEW sales AS SELECT id, customer_id, status, json_extract('not-json-' || id, '$') AS created_at FROM sales_src`)

		// Precondition: the fixture really makes the last-visit query fail.
		var lv sql.NullString
		if err := db.DB.QueryRowContext(ctx, `SELECT MAX(created_at) FROM sales WHERE customer_id = 1 AND status = 'completed'`).Scan(&lv); err == nil {
			t.Fatalf("fixture broken: last-visit query succeeded (%v)", lv)
		}

		s, err := NewCustomerRepository(db.DB).GetPOSSummary(ctx, 1)
		if err == nil {
			t.Fatalf("GetPOSSummary err = nil, last_visit_at = %v; want the last-visit query error", s.LastVisitAt)
		}
	})
}
