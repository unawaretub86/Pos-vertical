package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

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
