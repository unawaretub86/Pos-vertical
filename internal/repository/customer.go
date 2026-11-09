package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andreyo/pos/internal/model"
)

type customerRepo struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) CustomerRepository {
	return &customerRepo{db: db}
}

// List returns all active customers with their on-the-fly credit balance.
// Optional search filters by name or phone (LIKE).
func (r *customerRepo) List(ctx context.Context, search string) ([]*model.CustomerWithBalance, error) {
	query := `
		SELECT
			c.id, c.name, c.phone, c.address, COALESCE(c.notes,''), c.credit_limit, c.created_at,
			` + customerBalanceExpr + ` AS balance,
			` + customerOldestDebtExpr + ` AS oldest_debt_at` + customerBalanceFrom

	query += ` WHERE c.active = 1`
	args := []any{}
	if strings.TrimSpace(search) != "" {
		query += ` AND (c.name LIKE ? OR c.phone LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	query += ` GROUP BY c.id ORDER BY c.name`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("customer list: %w", err)
	}
	defer rows.Close()

	var customers []*model.CustomerWithBalance
	for rows.Next() {
		var cwb model.CustomerWithBalance
		var oldestDebt sql.NullString
		if err := rows.Scan(
			&cwb.ID, &cwb.Name, &cwb.Phone, &cwb.Address, &cwb.Notes, &cwb.CreditLimit, &cwb.CreatedAt,
			&cwb.Balance, &oldestDebt,
		); err != nil {
			return nil, fmt.Errorf("customer scan: %w", err)
		}
		cwb.OldestDebtAt = parseOldestDebt(oldestDebt)
		customers = append(customers, &cwb)
	}
	return customers, rows.Err()
}

func (r *customerRepo) GetByID(ctx context.Context, id int64) (*model.Customer, error) {
	c := &model.Customer{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, COALESCE(phone,''), COALESCE(address,''), COALESCE(notes,''), credit_limit, created_at FROM customers WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.Phone, &c.Address, &c.Notes, &c.CreditLimit, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("customer get: %w", err)
	}
	return c, nil
}

func (r *customerRepo) Create(ctx context.Context, c *model.Customer) (*model.Customer, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO customers (name, phone, address, notes, credit_limit) VALUES (?, ?, ?, ?, ?)`,
		c.Name, c.Phone, c.Address, c.Notes, c.CreditLimit)
	if err != nil {
		return nil, fmt.Errorf("customer create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("customer last id: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *customerRepo) Update(ctx context.Context, c *model.Customer) (*model.Customer, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE customers SET name = ?, phone = ?, address = ?, notes = ?, credit_limit = ? WHERE id = ?`,
		c.Name, c.Phone, c.Address, c.Notes, c.CreditLimit, c.ID)
	if err != nil {
		return nil, fmt.Errorf("customer update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, c.ID)
}

// Delete deactivates an active customer (active = 0). The row is kept because
// sales and payments reference it; an unknown or already inactive id is
// ErrNotFound.
func (r *customerRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE customers SET active = 0 WHERE id = ? AND active = 1`, id)
	if err != nil {
		return fmt.Errorf("customer delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Balance calculates the on-the-fly balance:
// opening balance (US-092) + SUM(credit sales total) - SUM(customer_payments
// amount). Same formula as customerBalanceExpr.
func (r *customerRepo) Balance(ctx context.Context, id int64) (float64, error) {
	var balance float64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE((SELECT amount FROM customer_opening_balances WHERE customer_id = ?), 0)
			+ COALESCE((
				SELECT SUM(sp.amount) FROM sale_payments sp
				JOIN sales s ON s.id = sp.sale_id
				WHERE s.customer_id = ? AND sp.method = 'credit' AND s.status = 'completed'
			), 0)
			- COALESCE((SELECT SUM(amount) FROM customer_payments WHERE customer_id = ?), 0)
	`, id, id, id).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("customer balance: %w", err)
	}
	return balance, nil
}

// TotalPortfolioBalance returns the sum of the outstanding balances of
// customers whose balance is > 0 (spec/domain/glosario.md: cartera). It
// aggregates the same per-customer formula as List/Balance/Debtors/
// AgingReceivables (customerBalanceExpr/customerBalanceFrom in metrics.go),
// so a customer who overpaid (negative balance) never reduces what the
// others owe.
func (r *customerRepo) TotalPortfolioBalance(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(balance), 0) FROM (
			SELECT `+customerBalanceExpr+` AS balance`+customerBalanceFrom+`
		) balances
		WHERE balance > 0
	`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("portfolio balance: %w", err)
	}
	return total, nil
}

// ListPayments returns all payments for a given customer, ordered by most recent first.
func (r *customerRepo) ListPayments(ctx context.Context, customerID int64) ([]*model.CustomerPayment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, customer_id, amount, method, register_id, COALESCE(note,''), created_at
		 FROM customer_payments WHERE customer_id = ? ORDER BY created_at DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var payments []*model.CustomerPayment
	for rows.Next() {
		p := &model.CustomerPayment{}
		if err := rows.Scan(&p.ID, &p.CustomerID, &p.Amount, &p.Method, &p.RegisterID, &p.Note, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

// CreatePayment inserts a new customer payment (abono).
func (r *customerRepo) CreatePayment(ctx context.Context, p *model.CustomerPayment) (*model.CustomerPayment, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO customer_payments (customer_id, amount, method, register_id, note) VALUES (?, ?, ?, ?, ?)`,
		p.CustomerID, p.Amount, p.Method, p.RegisterID, p.Note)
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("payment last id: %w", err)
	}
	created := &model.CustomerPayment{}
	err = r.db.QueryRowContext(ctx,
		`SELECT id, customer_id, amount, method, register_id, COALESCE(note,''), created_at FROM customer_payments WHERE id = ?`, id).
		Scan(&created.ID, &created.CustomerID, &created.Amount, &created.Method, &created.RegisterID, &created.Note, &created.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get created payment: %w", err)
	}
	return created, nil
}

// ListCreditSales returns all credit sales for a customer (completed only), most recent first.
func (r *customerRepo) ListCreditSales(ctx context.Context, customerID int64) ([]*model.Sale, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, customer_id, COALESCE(register_id, 0), subtotal, discount, tax, total,
		        payment_method, status, created_at
		 FROM sales
		 WHERE customer_id = ? AND status = 'completed'
		   AND id IN (SELECT sale_id FROM sale_payments WHERE method = 'credit')
		 ORDER BY created_at DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("list credit sales: %w", err)
	}
	defer rows.Close()

	var sales []*model.Sale
	for rows.Next() {
		s := &model.Sale{}
		var regID int64
		if err := rows.Scan(&s.ID, &s.UserID, &s.CustomerID, &regID,
			&s.Subtotal, &s.Discount, &s.Tax, &s.Total,
			&s.PaymentMethod, &s.Status, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan credit sale: %w", err)
		}
		if regID != 0 {
			s.RegisterID = &regID
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

// GetPOSSummary returns the customer context for the POS checkout.
func (r *customerRepo) GetPOSSummary(ctx context.Context, id int64) (*model.CustomerPosSummary, error) {
	summary := &model.CustomerPosSummary{}

	c, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	summary.CreditLimit = c.CreditLimit

	var balance float64
	var oldestDebt sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT
			`+customerBalanceExpr+` AS balance,
			`+customerOldestDebtExpr+` AS oldest_debt_at
		`+customerBalanceFrom+`
		WHERE c.id = ?`, id).Scan(&balance, &oldestDebt)
	if err != nil {
		return nil, fmt.Errorf("customer summary balance: %w", err)
	}
	summary.Balance = balance
	oldest := parseOldestDebt(oldestDebt)
	if balance > 0 && oldest != nil {
		days := int(time.Since(*oldest).Hours() / 24)
		summary.DaysOverdue = &days
	}

	var lastVisit sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT MAX(created_at) FROM sales WHERE customer_id = ? AND status = 'completed'`, id).Scan(&lastVisit)
	if err != nil {
		return nil, fmt.Errorf("customer summary last visit: %w", err)
	}
	if lastVisit.Valid {
		summary.LastVisitAt = &lastVisit.String
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			p.id, p.name, COUNT(DISTINCT si.sale_id) AS times_bought
		FROM sale_items si
		JOIN sales s ON s.id = si.sale_id
		JOIN products p ON p.id = si.product_id
		WHERE s.customer_id = ? AND s.status = 'completed' AND p.active = 1
		GROUP BY p.id, p.name
		ORDER BY times_bought DESC, p.name ASC
		LIMIT 3`, id)
	if err != nil {
		return nil, fmt.Errorf("customer summary top products: %w", err)
	}
	defer rows.Close()

	summary.TopProducts = []model.CustomerTopProduct{}
	for rows.Next() {
		var tp model.CustomerTopProduct
		if err := rows.Scan(&tp.ProductID, &tp.Name, &tp.TimesBought); err != nil {
			return nil, fmt.Errorf("scan top product: %w", err)
		}
		summary.TopProducts = append(summary.TopProducts, tp)
	}

	return summary, nil
}

// IsActive reports whether the customer exists and was not deleted
// (active = 1).
func (r *customerRepo) IsActive(ctx context.Context, id int64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM customers WHERE id = ? AND active = 1`, id).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("customer active: %w", err)
	}
	return n > 0, nil
}

// openingDateLayout is the stored layout of customer_opening_balances.debt_date,
// the same as MIN(sales.created_at) so both compare as text.
const openingDateLayout = "2006-01-02 15:04:05"

// openingDateToStored converts a server-local calendar date YYYY-MM-DD to the
// stored UTC text of 00:00 local time of that date.
func openingDateToStored(date string) (string, error) {
	d, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return "", fmt.Errorf("opening debt date: %w", err)
	}
	return d.UTC().Format(openingDateLayout), nil
}

// openingDateFromStored converts the stored UTC text back to the server-local
// calendar date YYYY-MM-DD.
func openingDateFromStored(stored string) (string, error) {
	t, err := time.Parse(openingDateLayout, stored)
	if err != nil {
		return "", fmt.Errorf("opening debt date %q: %w", stored, err)
	}
	return t.Local().Format("2006-01-02"), nil
}

// GetOpeningBalance returns the customer's opening balance, or ErrNotFound when
// there is none.
func (r *customerRepo) GetOpeningBalance(ctx context.Context, customerID int64) (*model.CustomerOpeningBalance, error) {
	ob := &model.CustomerOpeningBalance{}
	var stored string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, customer_id, amount, debt_date, note, created_by, created_at, updated_at
		FROM customer_opening_balances WHERE customer_id = ?`, customerID).
		Scan(&ob.ID, &ob.CustomerID, &ob.Amount, &stored, &ob.Note, &ob.CreatedBy, &ob.CreatedAt, &ob.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("opening balance get: %w", err)
	}
	if ob.DebtDate, err = openingDateFromStored(stored); err != nil {
		return nil, err
	}
	return ob, nil
}

// CreateOpeningBalance inserts the opening balance and its audit_log row
// (opening_balance_create, by ob.CreatedBy) in one transaction. The
// UNIQUE(customer_id) constraint is the guard against a second one, also under
// concurrent requests: it maps to ErrOpeningBalanceExists.
func (r *customerRepo) CreateOpeningBalance(ctx context.Context, ob *model.CustomerOpeningBalance) (*model.CustomerOpeningBalance, error) {
	stored, err := openingDateToStored(ob.DebtDate)
	if err != nil {
		return nil, err
	}
	details, err := json.Marshal(map[string]any{"amount": ob.Amount, "debt_date": ob.DebtDate})
	if err != nil {
		return nil, fmt.Errorf("opening balance details: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("opening balance begin: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO customer_opening_balances (customer_id, amount, debt_date, note, created_by) VALUES (?, ?, ?, ?, ?)`,
		ob.CustomerID, ob.Amount, stored, ob.Note, ob.CreatedBy)
	if isUniqueViolation(err) {
		return nil, ErrOpeningBalanceExists
	}
	if err != nil {
		return nil, fmt.Errorf("opening balance insert: %w", err)
	}
	if err := InsertAuditLog(ctx, tx, ob.CreatedBy, "opening_balance_create", "customer", ob.CustomerID, string(details)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("opening balance commit: %w", err)
	}
	return r.GetOpeningBalance(ctx, ob.CustomerID)
}

// UpdateOpeningBalance replaces amount, debt date and note of the customer's
// opening balance and writes its audit_log row (opening_balance_update, by
// userID, with the old and new amount and date) in one transaction. A
// customer with none is ErrNotFound.
func (r *customerRepo) UpdateOpeningBalance(ctx context.Context, ob *model.CustomerOpeningBalance, userID int64) (*model.CustomerOpeningBalance, error) {
	stored, err := openingDateToStored(ob.DebtDate)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("opening balance begin: %w", err)
	}
	defer tx.Rollback()

	var fromAmount float64
	var fromStored string
	err = tx.QueryRowContext(ctx,
		`SELECT amount, debt_date FROM customer_opening_balances WHERE customer_id = ?`, ob.CustomerID).
		Scan(&fromAmount, &fromStored)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("opening balance read: %w", err)
	}
	fromDate, err := openingDateFromStored(fromStored)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE customer_opening_balances SET amount = ?, debt_date = ?, note = ?, updated_at = CURRENT_TIMESTAMP WHERE customer_id = ?`,
		ob.Amount, stored, ob.Note, ob.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("opening balance update: %w", err)
	}
	details, err := json.Marshal(map[string]any{
		"from_amount": fromAmount,
		"to_amount":   ob.Amount,
		"from_date":   fromDate,
		"to_date":     ob.DebtDate,
	})
	if err != nil {
		return nil, fmt.Errorf("opening balance details: %w", err)
	}
	if err := InsertAuditLog(ctx, tx, userID, "opening_balance_update", "customer", ob.CustomerID, string(details)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("opening balance commit: %w", err)
	}
	return r.GetOpeningBalance(ctx, ob.CustomerID)
}
