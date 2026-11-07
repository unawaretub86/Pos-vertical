package handler

// US-088 contract §4, T-2 .. T-6 and T-8. Voiding a sale must not insert a
// register movement, so the register's expected cash returns to its value
// before the sale, by the sale's cash portion only (R-110).
//
// These go through h.Routes() with a real admin JWT (tokenFor) and the
// abonoRequest/abonoData helpers of customer_payment_test.go. Registers are
// never closed through POST /registers/{id}/close here: in a :memory: database
// that path is flaky (contract §2, last row). T-5 closes through the service
// and T-6 seeds status = 'closed'.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

type voidRegisterJSON struct {
	Status        string   `json:"status"`
	ExpectedClose *float64 `json:"expected_close"`
}

type voidMovementJSON struct {
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

type voidSummaryJSON struct {
	Register  voidRegisterJSON   `json:"register"`
	CashSales float64            `json:"cash_sales"`
	CashOut   float64            `json:"cash_out"`
	Expected  float64            `json:"expected"`
	Movements []voidMovementJSON `json:"movements"`
}

// voidSeedCatalog adds product 1 ($50,000, stock 20) and customer 1 (Dora),
// enough for every sale these tests post.
func voidSeedCatalog(t *testing.T, db *database.DB) {
	t.Helper()
	abonoExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (1, 'Arroz', 'A', 50000, 30000, 'unit', 1, 20)`)
	abonoExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
}

func voidOpenRegister(t *testing.T, h *Handler, token string, opening float64) {
	t.Helper()
	w := abonoRequest(t, h, http.MethodPost, "/registers", token, fmt.Sprintf(`{"opening_amount":%.0f}`, opening))
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /registers = %d, want 201; body %s", w.Code, w.Body.String())
	}
}

// voidPostSale posts a $50,000 sale of 1 unit of product 1 with the given
// payments JSON array. customerID 0 omits customer_id.
func voidPostSale(t *testing.T, h *Handler, token, payments string, customerID int64) {
	t.Helper()
	customer := ""
	if customerID != 0 {
		customer = fmt.Sprintf(`"customer_id":%d,`, customerID)
	}
	body := fmt.Sprintf(`{%s"items":[{"product_id":1,"quantity":1,"unit_price":50000}],"total":50000,"payments":%s}`, customer, payments)
	w := abonoRequest(t, h, http.MethodPost, "/sales", token, body)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sales %s = %d, want 201; body %s", payments, w.Code, w.Body.String())
	}
}

func voidSummary(t *testing.T, h *Handler, token string, registerID int64) voidSummaryJSON {
	t.Helper()
	w := abonoRequest(t, h, http.MethodGet, fmt.Sprintf("/registers/%d", registerID), token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /registers/%d = %d, want 200; body %s", registerID, w.Code, w.Body.String())
	}
	var s voidSummaryJSON
	abonoData(t, w, &s)
	return s
}

func voidSale(t *testing.T, h *Handler, token string, saleID int64) {
	t.Helper()
	w := abonoRequest(t, h, http.MethodDelete, fmt.Sprintf("/sales/%d/void", saleID), token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /sales/%d/void = %d, want 200; body %s", saleID, w.Code, w.Body.String())
	}
}

func voidCountMovements(t *testing.T, db *database.DB, registerID int64) int {
	t.Helper()
	var n int
	if err := db.DB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM register_movements WHERE register_id = ?`, registerID).Scan(&n); err != nil {
		t.Fatalf("count movements of register %d: %v", registerID, err)
	}
	return n
}

func voidAssertNoAnulacion(t *testing.T, where string, s voidSummaryJSON) {
	t.Helper()
	for _, m := range s.Movements {
		if strings.HasPrefix(m.Description, "Anulación") {
			t.Errorf("%s: unexpected movement %s %.0f %q", where, m.Type, m.Amount, m.Description)
		}
	}
}

// voidRunT2 runs the R-110 scenario: open $100,000, cash sale $50,000, void.
func voidRunT2(t *testing.T) (*database.DB, *Handler, string) {
	t.Helper()
	db := setupTestDB(t)
	voidSeedCatalog(t, db)
	h := New(db)
	admin := tokenFor(t, 1, "admin")

	voidOpenRegister(t, h, admin, 100000)
	voidPostSale(t, h, admin, `[{"method":"cash","amount":50000}]`, 0)
	if got := voidSummary(t, h, admin, 1).Expected; got != 150000 {
		t.Fatalf("expected before void = %.0f, want 150000", got)
	}
	voidSale(t, h, admin, 1)
	return db, h, admin
}

func TestRegisterVoid_T2_CashSaleVoidRestoresExpected(t *testing.T) {
	_, h, admin := voidRunT2(t)

	s := voidSummary(t, h, admin, 1)
	if s.Expected != 100000 {
		t.Errorf("expected after void = %.0f, want 100000", s.Expected)
	}
	if s.CashSales != 0 {
		t.Errorf("cash_sales after void = %.0f, want 0", s.CashSales)
	}
	if s.CashOut != 0 {
		t.Errorf("cash_out after void = %.0f, want 0", s.CashOut)
	}
	voidAssertNoAnulacion(t, "after void", s)
}

func TestRegisterVoid_T3_NonCashSaleVoidKeepsExpected(t *testing.T) {
	for _, tc := range []struct {
		method     string
		customerID int64
	}{
		{"credit", 1},
		{"transfer", 0},
		{"card", 0},
	} {
		t.Run(tc.method, func(t *testing.T) {
			db := setupTestDB(t)
			voidSeedCatalog(t, db)
			h := New(db)
			admin := tokenFor(t, 1, "admin")

			voidOpenRegister(t, h, admin, 100000)
			voidPostSale(t, h, admin, fmt.Sprintf(`[{"method":%q,"amount":50000}]`, tc.method), tc.customerID)
			if got := voidSummary(t, h, admin, 1).Expected; got != 100000 {
				t.Fatalf("expected before void = %.0f, want 100000", got)
			}
			before := voidCountMovements(t, db, 1)

			voidSale(t, h, admin, 1)

			s := voidSummary(t, h, admin, 1)
			if s.Expected != 100000 {
				t.Errorf("expected after void = %.0f, want 100000", s.Expected)
			}
			if s.CashOut != 0 {
				t.Errorf("cash_out after void = %.0f, want 0", s.CashOut)
			}
			if after := voidCountMovements(t, db, 1); after != before {
				t.Errorf("register 1 movements = %d after void, want %d (unchanged)", after, before)
			}
		})
	}
}

func TestRegisterVoid_T4_MixedSaleVoidRemovesCashPortionOnly(t *testing.T) {
	for _, tc := range []struct {
		name       string
		payments   string
		customerID int64
	}{
		{"cash+card", `[{"method":"cash","amount":30000},{"method":"card","amount":20000}]`, 0},
		{"cash+credit", `[{"method":"cash","amount":30000},{"method":"credit","amount":20000}]`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupTestDB(t)
			voidSeedCatalog(t, db)
			h := New(db)
			admin := tokenFor(t, 1, "admin")

			voidOpenRegister(t, h, admin, 100000)
			voidPostSale(t, h, admin, tc.payments, tc.customerID)
			if got := voidSummary(t, h, admin, 1).Expected; got != 130000 {
				t.Fatalf("expected before void = %.0f, want 130000", got)
			}

			voidSale(t, h, admin, 1)

			if got := voidSummary(t, h, admin, 1).Expected; got != 100000 {
				t.Errorf("expected after void = %.0f, want 100000", got)
			}
		})
	}
}

func TestRegisterVoid_T5_CloseAfterVoidHasNoShortage(t *testing.T) {
	db, h, _ := voidRunT2(t)

	// Close through the service, not POST /registers/1/close (see header).
	summary, err := h.registers.Close(context.Background(), 1, 100000)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if summary.Expected != 100000 {
		t.Errorf("Close returned Expected = %.0f, want 100000", summary.Expected)
	}

	var expectedClose, actualClose, difference float64
	if err := db.DB.QueryRowContext(context.Background(),
		`SELECT expected_close, actual_close, difference FROM registers WHERE id = 1`).
		Scan(&expectedClose, &actualClose, &difference); err != nil {
		t.Fatalf("read register 1: %v", err)
	}
	if expectedClose != 100000 || actualClose != 100000 || difference != 0 {
		t.Errorf("stored expected_close/actual_close/difference = %.0f/%.0f/%.0f, want 100000/100000/0",
			expectedClose, actualClose, difference)
	}
}

func TestRegisterVoid_T6_VoidAfterCloseWritesNothing(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	admin := tokenFor(t, 1, "admin")

	// Register 1 closed with its cash sale #1 of $50,000; register 2 open.
	abonoExec(t, db, `INSERT INTO registers (id, user_id, opening_amount, expected_close, actual_close, difference, status) VALUES (1, 1, 100000, 150000, 150000, 0, 'closed')`)
	abonoExec(t, db, `INSERT INTO sales (id, user_id, register_id, subtotal, total, status, payment_method, reference) VALUES (1, 1, 1, 50000, 50000, 'completed', 'cash', '')`)
	abonoExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'cash', 50000)`)
	abonoExec(t, db, `INSERT INTO register_movements (register_id, type, amount, description) VALUES (1, 'sale', 50000, 'Venta #1')`)
	abonoExec(t, db, `INSERT INTO registers (id, user_id, opening_amount, status) VALUES (2, 1, 80000, 'open')`)

	before1 := voidCountMovements(t, db, 1)
	before2 := voidCountMovements(t, db, 2)

	voidSale(t, h, admin, 1)

	if got := voidCountMovements(t, db, 1); got != before1 {
		t.Errorf("register 1 movements = %d after void, want %d (unchanged)", got, before1)
	}
	if got := voidCountMovements(t, db, 2); got != before2 {
		t.Errorf("register 2 movements = %d after void, want %d (unchanged)", got, before2)
	}

	var expectedClose, actualClose, difference float64
	if err := db.DB.QueryRowContext(context.Background(),
		`SELECT expected_close, actual_close, difference FROM registers WHERE id = 1`).
		Scan(&expectedClose, &actualClose, &difference); err != nil {
		t.Fatalf("read register 1: %v", err)
	}
	if expectedClose != 150000 || actualClose != 150000 || difference != 0 {
		t.Errorf("stored expected_close/actual_close/difference = %.0f/%.0f/%.0f, want 150000/150000/0",
			expectedClose, actualClose, difference)
	}

	s := voidSummary(t, h, admin, 1)
	if s.Expected != 100000 {
		t.Errorf("closed register expected = %.0f, want 100000", s.Expected)
	}
	if s.CashSales != 0 {
		t.Errorf("closed register cash_sales = %.0f, want 0", s.CashSales)
	}
	if s.CashOut != 0 {
		t.Errorf("closed register cash_out = %.0f, want 0", s.CashOut)
	}
	if s.Register.Status != "closed" {
		t.Errorf("register.status = %q, want closed", s.Register.Status)
	}
	if s.Register.ExpectedClose == nil || *s.Register.ExpectedClose != 150000 {
		t.Errorf("register.expected_close = %v, want 150000", s.Register.ExpectedClose)
	}

	if got := voidSummary(t, h, admin, 2).Expected; got != 80000 {
		t.Errorf("open register 2 expected = %.0f, want 80000", got)
	}
}

func TestRegisterVoid_T8_SecondVoidAndUnknownID(t *testing.T) {
	db, h, admin := voidRunT2(t)

	movesBefore := voidCountMovements(t, db, 1)
	if got := countAuditRows(t, db, "void", "sale", 1); got != 1 {
		t.Fatalf("audit void rows after first void = %d, want 1", got)
	}

	w := abonoRequest(t, h, http.MethodDelete, "/sales/1/void", admin, "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("second void = %d, want 400; body %s", w.Code, w.Body.String())
	} else if msg := abonoError(t, w); msg != "la venta ya fue anulada" {
		t.Errorf("second void error = %q, want %q", msg, "la venta ya fue anulada")
	}
	if got := voidCountMovements(t, db, 1); got != movesBefore {
		t.Errorf("register 1 movements = %d after second void, want %d (unchanged)", got, movesBefore)
	}
	if got := countAuditRows(t, db, "void", "sale", 1); got != 1 {
		t.Errorf("audit void rows after second void = %d, want 1", got)
	}

	w = abonoRequest(t, h, http.MethodDelete, "/sales/999/void", admin, "")
	if w.Code != http.StatusNotFound {
		t.Errorf("void of unknown sale = %d, want 404; body %s", w.Code, w.Body.String())
	} else if msg := abonoError(t, w); msg != "venta no encontrada" {
		t.Errorf("unknown sale error = %q, want %q", msg, "venta no encontrada")
	}
}
