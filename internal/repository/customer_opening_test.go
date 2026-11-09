package repository

// US-092 contract §4, T-2, T-3 and T-12: the opening fiado balance (deuda
// anterior) in every balance query and in the debt age.
//
// Opening balances are seeded with literal INSERTs into
// customer_opening_balances (migration 22). debt_date is stored as text
// "YYYY-MM-DD HH:MM:SS" in UTC for 00:00 server-local time of the picked date
// (contract §3, shared terms).

import (
	"context"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/database"
)

// openingLocalDate returns 00:00 server-local time of today plus days.
func openingLocalDate(days int) time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+days, 0, 0, 0, 0, time.Local)
}

// openingStoredDate is the stored layout of a debt date: UTC text of 00:00
// server-local time of that date.
func openingStoredDate(d time.Time) string {
	return d.UTC().Format("2006-01-02 15:04:05")
}

// seedOpeningCustomers seeds customers 7 Dora, 8 Beto (active) and 9 Eva
// (active = 0). Admin id 1 comes from migration 1.
func seedOpeningCustomers(t *testing.T, db *database.DB) {
	t.Helper()
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (7, 'Dora', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (8, 'Beto', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address, active) VALUES (9, 'Eva', '', '', 0)`)
}

func insertOpening(t *testing.T, db *database.DB, customerID int64, amount float64, debtDate time.Time) {
	t.Helper()
	metricsExec(t, db,
		`INSERT INTO customer_opening_balances (customer_id, amount, debt_date, note, created_by) VALUES (?, ?, ?, '', 1)`,
		customerID, amount, openingStoredDate(debtDate))
}

// insertCreditSale inserts a credit sale paid fully on credit, created at
// datetime('now', modifier) (UTC text, as CURRENT_TIMESTAMP writes it).
func insertCreditSale(t *testing.T, db *database.DB, saleID, customerID int64, amount float64, status, modifier string) {
	t.Helper()
	metricsExec(t, db,
		`INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (?, 1, ?, ?, ?, ?, 'credit', datetime('now', ?))`,
		saleID, customerID, amount, amount, status, modifier)
	metricsExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (?, 'credit', ?)`, saleID, amount)
}

func insertAbono(t *testing.T, db *database.DB, customerID int64, amount float64) {
	t.Helper()
	metricsExec(t, db, `INSERT INTO customer_payments (customer_id, amount, method) VALUES (?, ?, 'cash')`, customerID, amount)
}

func listBalances(t *testing.T, repo CustomerRepository) map[int64]float64 {
	t.Helper()
	rows, err := repo.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	out := map[int64]float64{}
	for _, c := range rows {
		out[c.ID] = c.Balance
	}
	return out
}

// ─── T-2: AC-8 ───────────────────────────────────────────────────────────────

func TestCustomerOpening_T2_EveryBalanceQueryCarriesOpening(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	seedOpeningCustomers(t, db)

	// Dora: opening 150,000 (D45) + completed credit 40,000 (today) + voided
	// credit 25,000 + abono 30,000 → 150,000 + 40,000 − 30,000 = 160,000.
	insertOpening(t, db, 7, 150000, openingLocalDate(-45))
	insertCreditSale(t, db, 1, 7, 40000, "completed", "+0 seconds")
	insertCreditSale(t, db, 2, 7, 25000, "voided", "+0 seconds")
	insertAbono(t, db, 7, 30000)
	// Beto: opening 20,000 + abono 50,000 → −30,000.
	insertOpening(t, db, 8, 20000, openingLocalDate(-45))
	insertAbono(t, db, 8, 50000)

	customers := NewCustomerRepository(db.DB)
	metrics := NewMetricsRepository(db.DB)

	bal, err := customers.Balance(ctx, 7)
	if err != nil {
		t.Fatalf("Balance(7): %v", err)
	}
	if bal != 160000 {
		t.Errorf("Balance(7) = %.0f, want 160000", bal)
	}

	list := listBalances(t, customers)
	if got, ok := list[7]; !ok || got != 160000 {
		t.Errorf("List Dora balance = %.0f (present %v), want 160000", got, ok)
	}
	if got, ok := list[8]; !ok || got != -30000 {
		t.Errorf("List Beto balance = %.0f (present %v), want -30000", got, ok)
	}

	total, err := customers.TotalPortfolioBalance(ctx)
	if err != nil {
		t.Fatalf("TotalPortfolioBalance: %v", err)
	}
	if total != 160000 {
		t.Errorf("TotalPortfolioBalance = %.0f, want 160000 (Beto excluded)", total)
	}

	debtors, err := metrics.Debtors(ctx, 0)
	if err != nil {
		t.Fatalf("Debtors: %v", err)
	}
	if len(debtors) != 1 || debtors[0].ID != 7 || debtors[0].Balance != 160000 {
		t.Errorf("Debtors = %+v, want exactly [Dora 160000]", debtors)
	}

	aging, err := metrics.AgingReceivables(ctx, 10)
	if err != nil {
		t.Fatalf("AgingReceivables: %v", err)
	}
	if len(aging) != 1 || aging[0].CustomerID != 7 || aging[0].Balance != 160000 {
		t.Errorf("AgingReceivables = %+v, want exactly [Dora 160000]", aging)
	}

	summary, err := customers.GetPOSSummary(ctx, 7)
	if err != nil {
		t.Fatalf("GetPOSSummary(7): %v", err)
	}
	if summary.Balance != 160000 {
		t.Errorf("GetPOSSummary(7).Balance = %.0f, want 160000", summary.Balance)
	}
}

// ─── T-3: AC-9 ───────────────────────────────────────────────────────────────

func daysOverdue(t *testing.T, repo CustomerRepository, id int64) *int {
	t.Helper()
	s, err := repo.GetPOSSummary(context.Background(), id)
	if err != nil {
		t.Fatalf("GetPOSSummary(%d): %v", id, err)
	}
	return s.DaysOverdue
}

func agingDays(t *testing.T, metrics MetricsRepository, id int64) (int, bool) {
	t.Helper()
	items, err := metrics.AgingReceivables(context.Background(), 10)
	if err != nil {
		t.Fatalf("AgingReceivables: %v", err)
	}
	for _, it := range items {
		if it.CustomerID == id {
			return it.DaysOverdue, true
		}
	}
	return 0, false
}

func TestCustomerOpening_T3_DebtAge(t *testing.T) {
	t.Run("a_opening_only", func(t *testing.T) {
		db := openMetricsDB(t)
		seedOpeningCustomers(t, db)
		d45 := openingLocalDate(-45)
		insertOpening(t, db, 7, 150000, d45)
		customers := NewCustomerRepository(db.DB)
		metrics := NewMetricsRepository(db.DB)

		if d := daysOverdue(t, customers, 7); d == nil || *d != 45 {
			t.Errorf("pos-summary days_overdue = %v, want 45", deref(d))
		}
		rows, err := customers.List(context.Background(), "")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		found := false
		for _, c := range rows {
			if c.ID != 7 {
				continue
			}
			found = true
			if c.OldestDebtAt == nil || !c.OldestDebtAt.Equal(d45) {
				t.Errorf("List Dora oldest_debt_at = %v, want %v (D45 00:00 local)", c.OldestDebtAt, d45)
			}
		}
		if !found {
			t.Errorf("List has no row for Dora")
		}
		if days, ok := agingDays(t, metrics, 7); !ok || days != 45 {
			t.Errorf("AgingReceivables Dora days_overdue = %d (present %v), want 45", days, ok)
		}
	})

	t.Run("b_older_credit_sale_wins", func(t *testing.T) {
		db := openMetricsDB(t)
		seedOpeningCustomers(t, db)
		insertOpening(t, db, 7, 150000, openingLocalDate(-45))
		insertCreditSale(t, db, 1, 7, 10000, "completed", "-60 days")
		customers := NewCustomerRepository(db.DB)
		metrics := NewMetricsRepository(db.DB)

		if d := daysOverdue(t, customers, 7); d == nil || *d != 60 {
			t.Errorf("pos-summary days_overdue = %v, want 60 (the sale is older)", deref(d))
		}
		if days, ok := agingDays(t, metrics, 7); !ok || days != 60 {
			t.Errorf("AgingReceivables Dora days_overdue = %d (present %v), want 60", days, ok)
		}
	})

	t.Run("c_older_opening_wins", func(t *testing.T) {
		db := openMetricsDB(t)
		seedOpeningCustomers(t, db)
		insertCreditSale(t, db, 1, 8, 10000, "completed", "-10 days")
		insertOpening(t, db, 8, 20000, openingLocalDate(-45))
		customers := NewCustomerRepository(db.DB)
		metrics := NewMetricsRepository(db.DB)

		if d := daysOverdue(t, customers, 8); d == nil || *d != 45 {
			t.Errorf("Beto pos-summary days_overdue = %v, want 45 (the opening is older)", deref(d))
		}
		if days, ok := agingDays(t, metrics, 8); !ok || days != 45 {
			t.Errorf("AgingReceivables Beto days_overdue = %d (present %v), want 45", days, ok)
		}
	})

	t.Run("d_opening_fully_paid", func(t *testing.T) {
		db := openMetricsDB(t)
		seedOpeningCustomers(t, db)
		insertOpening(t, db, 7, 150000, openingLocalDate(-45))
		insertAbono(t, db, 7, 150000)
		customers := NewCustomerRepository(db.DB)
		metrics := NewMetricsRepository(db.DB)

		s, err := customers.GetPOSSummary(context.Background(), 7)
		if err != nil {
			t.Fatalf("GetPOSSummary(7): %v", err)
		}
		if s.Balance != 0 {
			t.Errorf("balance = %.0f, want 0", s.Balance)
		}
		if s.DaysOverdue != nil {
			t.Errorf("days_overdue = %d, want nil", *s.DaysOverdue)
		}
		if days, ok := agingDays(t, metrics, 7); ok {
			t.Errorf("AgingReceivables has Dora with days_overdue %d, want absent", days)
		}
		bal, err := customers.Balance(context.Background(), 7)
		if err != nil {
			t.Fatalf("Balance(7): %v", err)
		}
		if bal != 0 {
			t.Errorf("Balance(7) = %.0f, want 0", bal)
		}
	})
}

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// ─── T-12: AC-11 ─────────────────────────────────────────────────────────────

func TestCustomerOpening_T12_InvariantWithAndWithoutOpening(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	seedOpeningCustomers(t, db)
	// Credit 50,000; abonos 20,000 + 10,000 + 5,000 → 15,000, the number
	// certified by US-091 (TestShiftCert_T1_FullShift).
	insertCreditSale(t, db, 1, 7, 50000, "completed", "-1 days")
	insertAbono(t, db, 7, 20000)
	insertAbono(t, db, 7, 10000)
	insertAbono(t, db, 7, 5000)
	customers := NewCustomerRepository(db.DB)

	bal, err := customers.Balance(ctx, 7)
	if err != nil {
		t.Fatalf("Balance(7): %v", err)
	}
	if bal != 15000 {
		t.Fatalf("Balance(7) with no opening = %.0f, want 15000", bal)
	}
	if got := listBalances(t, customers)[7]; got != 15000 {
		t.Errorf("List Dora with no opening = %.0f, want 15000", got)
	}

	// Same customer + opening 100,000 → 100,000 + 50,000 − 35,000 = 115,000.
	insertOpening(t, db, 7, 100000, openingLocalDate(-30))
	bal, err = customers.Balance(ctx, 7)
	if err != nil {
		t.Fatalf("Balance(7): %v", err)
	}
	if bal != 115000 {
		t.Errorf("Balance(7) with opening 100000 = %.0f, want 115000", bal)
	}
	if got := listBalances(t, customers)[7]; got != 115000 {
		t.Errorf("List Dora with opening 100000 = %.0f, want 115000", got)
	}
}
