package handler

// US-096 contract §4, T-7 .. T-12: the server-side validations approved at
// GATE 1 (Q-4, G-8, Q-8). Each goes through h.Routes() on a file-backed DB
// (moneyEnv) and checks the exact 400 message and that nothing was stored.
// US-102 (R-133) changes T-8 (whole pesos), T-10 (exact total), T-11 (whole
// payment line) and T-12 (a zero total is a valid sale), and adds T-11 of its
// own contract (non-whole inputs rejected).

import (
	"net/http"
	"testing"
)

func assertError(t *testing.T, step string, code int, body []byte, wantCode int, wantMsg string) {
	t.Helper()
	if code != wantCode {
		t.Errorf("%s: status %d, want %d; body %s", step, code, wantCode, body)
	}
	if got := moneyErrMsg(body); got != wantMsg {
		t.Errorf("%s: error %q, want %q", step, got, wantMsg)
	}
}

// T-7 (AC-12).
func TestMoneyValidation_T7_NegativeOpeningAmount(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())

	code, body := e.send(t, http.MethodPost, "/registers", e.admin, `{"opening_amount":-5000}`)
	assertError(t, "opening -5000", code, body, http.StatusBadRequest, "el monto de apertura no puede ser negativo")
	if n := e.count(t, `SELECT COUNT(*) FROM registers`); n != 0 {
		t.Errorf("registers after opening -5000 = %d, want 0", n)
	}

	code, body = e.send(t, http.MethodPost, "/registers", e.admin, `{"opening_amount":0}`)
	if code != http.StatusCreated {
		t.Fatalf("opening 0: status %d, want 201; body %s", code, body)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM registers`); n != 1 {
		t.Errorf("registers after opening 0 = %d, want 1", n)
	}
}

// T-8 (AC-13).
func TestMoneyValidation_T8_MovementAmountMustBePositive(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())
	assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)

	code, body := e.send(t, http.MethodPost, "/registers/1/movements", e.cashier, `{"type":"cash_in","amount":-20000,"description":"x"}`)
	assertError(t, "cash_in -20000", code, body, http.StatusBadRequest, "el monto debe ser mayor a 0")
	code, body = e.send(t, http.MethodPost, "/registers/1/movements", e.cashier, `{"type":"cash_out","amount":0,"description":"x"}`)
	assertError(t, "cash_out 0", code, body, http.StatusBadRequest, "el monto debe ser mayor a 0")

	if n := e.count(t, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1`); n != 1 {
		t.Errorf("movements after the two 400s = %d, want 1 (opening)", n)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM audit_log WHERE action = 'cash_out'`); n != 0 {
		t.Errorf("cash_out audit rows after the two 400s = %d, want 0", n)
	}
	var s moneyRegisterSummary
	e.get(t, "/registers/1", &s)
	assertMoney(t, "expected after the two 400s", s.Expected, 100000)

	// US-102 (AC-7): money is whole pesos, so $0,40 is rejected and $1 is not.
	code, body = e.send(t, http.MethodPost, "/registers/1/movements", e.cashier, `{"type":"cash_out","amount":0.4,"description":"x"}`)
	if code != http.StatusBadRequest {
		t.Fatalf("cash_out 0.4: status %d, want 400; body %s", code, body)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1`); n != 1 {
		t.Errorf("movements after cash_out 0.4 = %d, want 1", n)
	}
	code, body = e.send(t, http.MethodPost, "/registers/1/movements", e.cashier, `{"type":"cash_out","amount":1,"description":"x"}`)
	if code != http.StatusCreated {
		t.Fatalf("cash_out 1: status %d, want 201; body %s", code, body)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1`); n != 2 {
		t.Errorf("movements after cash_out 1 = %d, want 2", n)
	}
}

// T-9 (AC-14). Arroz is priced $50.000 here, as in the contract case.
func TestMoneyValidation_T9_NegativeSaleDiscount(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", []moneyProduct{{1, "Arroz", "product", "unit", 50000, 0, 500}})
	assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)

	code, body := e.send(t, http.MethodPost, "/sales", e.cashier,
		`{"items":[{"product_id":1,"quantity":1,"unit_price":50000,"discount":0}],"discount":-10000,"tax":0,"total":60000,"payments":[{"method":"cash","amount":60000}]}`)
	assertError(t, "sale discount -10000", code, body, http.StatusBadRequest, "el descuento no puede ser negativo")
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 0 {
		t.Errorf("sales = %d, want 0", n)
	}
	if got := e.float(t, `SELECT stock FROM products WHERE id = 1`); got != 500 {
		t.Errorf("Arroz stock = %v, want 500", got)
	}
}

// T-10 (AC-15, US-102 AC-2 and AC-4): the sale is priced by the server and the
// client total must match it exactly (tolerance 0).
func TestMoneyValidation_T10_PaymentsVsRecalculatedTotal(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())
	assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)

	// Sale B: client tax 191, total and cash $1.192; server total is $1.191 (IVA 190).
	code, body := e.send(t, http.MethodPost, "/sales", e.cashier,
		`{"items":[{"product_id":2,"quantity":1,"unit_price":1001,"discount":0}],"discount":0,"tax":191,"total":1192,"payments":[{"method":"cash","amount":1192}]}`)
	assertError(t, "sale B", code, body, http.StatusBadRequest, "total inconsistente: esperado 1191, recibido 1192")
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 0 {
		t.Errorf("sales after sale B = %d, want 0", n)
	}

	// Sale A: client tax 190, cash $1.191 on the server total $1.191.
	code, body = e.send(t, http.MethodPost, "/sales", e.cashier,
		`{"items":[{"product_id":2,"quantity":1,"unit_price":1001,"discount":0}],"discount":0,"tax":190,"total":1191,"payments":[{"method":"cash","amount":1191}]}`)
	if code != http.StatusCreated {
		t.Fatalf("sale A: status %d, want 201; body %s", code, body)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 1 {
		t.Errorf("sales after sale A = %d, want 1", n)
	}
	if got := e.float(t, `SELECT total FROM sales`); got != 1191 {
		t.Errorf("sale A stored total = %v, want 1191", got)
	}
}

// T-11 (AC-16): 4 x Arroz $2.500 (subtotal $10.000) with discount $12.000.
// The payment line is a whole $1 so that the discount bound is what fails.
func TestMoneyValidation_T11_DiscountAboveSubtotal(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())
	assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)

	code, body := e.send(t, http.MethodPost, "/sales", e.cashier,
		`{"items":[{"product_id":1,"quantity":4,"unit_price":2500,"discount":0}],"discount":12000,"tax":0,"total":-2000,"payments":[{"method":"cash","amount":1}]}`)
	assertError(t, "discount 12000 on 10000", code, body, http.StatusBadRequest, "el descuento no puede superar el subtotal de la venta")
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 0 {
		t.Errorf("sales = %d, want 0", n)
	}
	if got := e.float(t, `SELECT stock FROM products WHERE id = 1`); got != 500 {
		t.Errorf("Arroz stock = %v, want 500", got)
	}
}

// T-12 (AC-17, rewritten by US-102 AC-4a): 4 x Arroz $2.500 with discount
// $10.000 -> total 0 is a valid sale with an empty payments list.
func TestMoneyValidation_T12_TotalMustBePositive(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())
	assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)
	items := `[{"product_id":1,"quantity":4,"unit_price":2500,"discount":0}]`

	// A payment line on a zero total is rejected and nothing is stored.
	before := e.snapshot(t)
	code, body := e.send(t, http.MethodPost, "/sales", e.cashier,
		`{"items":`+items+`,"discount":10000,"tax":0,"total":0,"payments":[{"method":"cash","amount":1}]}`)
	if code != http.StatusBadRequest {
		t.Errorf("zero total with cash 1: status %d, want 400; body %s", code, body)
	}
	e.assertUnchanged(t, "zero total with cash 1", before)

	// No discount and no payments is still rejected.
	code, body = e.send(t, http.MethodPost, "/sales", e.cashier,
		`{"items":`+items+`,"discount":0,"tax":0,"total":10000,"payments":[]}`)
	assertError(t, "no payments on 10000", code, body, http.StatusBadRequest, "la venta debe tener al menos un método de pago")
	e.assertUnchanged(t, "no payments on 10000", before)

	code, body = e.send(t, http.MethodPost, "/sales", e.cashier,
		`{"items":`+items+`,"discount":10000,"tax":0,"total":0,"payments":[]}`)
	if code != http.StatusCreated {
		t.Fatalf("zero total, empty payments: status %d, want 201; body %s", code, body)
	}
	checks := []struct {
		what string
		q    string
		want int
	}{
		{"sales", `SELECT COUNT(*) FROM sales`, 1},
		{"sale_payments", `SELECT COUNT(*) FROM sale_payments`, 0},
	}
	for _, c := range checks {
		if got := e.count(t, c.q); got != c.want {
			t.Errorf("%s = %d, want %d", c.what, got, c.want)
		}
	}
	if got := e.float(t, `SELECT total FROM sales WHERE id = 1`); got != 0 {
		t.Errorf("stored total = %v, want 0", got)
	}
	var sum moneyRegisterSummary
	e.get(t, "/registers/1", &sum)
	assertMoney(t, "register expected", sum.Expected, 100000)
	if got := e.float(t, `SELECT stock FROM products WHERE id = 1`); got != 496 {
		t.Errorf("Arroz stock = %v, want 496", got)
	}
}

// US-102 T-11 (AC-7): every non-whole money input is a 400 and stores nothing.
func TestMoneyValidation_US102_T11_NonWholeInputsRejected(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())

	reject := func(step, method, path, token, body string) {
		t.Helper()
		before := e.snapshot(t)
		code, resp := e.send(t, method, path, token, body)
		if code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400; body %s", step, code, resp)
		}
		if moneyErrMsg(resp) == "" {
			t.Errorf("%s: empty error message", step)
		}
		e.assertUnchanged(t, step, before)
	}

	reject("opening_amount 100000.5", http.MethodPost, "/registers", e.admin, `{"opening_amount":100000.5}`)
	if n := e.count(t, `SELECT COUNT(*) FROM registers`); n != 0 {
		t.Errorf("registers after opening 100000.5 = %d, want 0", n)
	}
	assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)

	// Dora owes $5.000 (seeded directly, whole pesos).
	e.exec(t, `INSERT INTO sales (id, user_id, customer_id, register_id, subtotal, total, status, payment_method) VALUES (1, 1, 1, 1, 5000, 5000, 'completed', 'credit')`)
	e.exec(t, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 5000)`)
	e.exec(t, `INSERT INTO suppliers (id, name) VALUES (1, 'Proveedor')`)

	reject("abono 100.5", http.MethodPost, "/customers/1/payments", e.cashier, `{"amount":100.5,"method":"cash"}`)
	reject("movement 1000.5", http.MethodPost, "/registers/1/movements", e.cashier, `{"type":"cash_in","amount":1000.5,"description":"x"}`)
	reject("expense 2500.5", http.MethodPost, "/expenses", e.cashier, `{"category":"servicios","amount":2500.5,"note":""}`)
	reject("product sale_price 2500.5", http.MethodPost, "/products", e.admin,
		`{"name":"Nuevo","code":"N1","type":"product","unit":"unit","sale_price":2500.5,"cost":1000}`)
	reject("product cost 1200.5", http.MethodPost, "/products", e.admin,
		`{"name":"Nuevo","code":"N2","type":"product","unit":"unit","sale_price":2500,"cost":1200.5}`)
	reject("product wholesale_price 900.5", http.MethodPost, "/products", e.admin,
		`{"name":"Nuevo","code":"N3","type":"product","unit":"unit","sale_price":2500,"cost":1000,"wholesale_price":900.5}`)
	reject("purchase unit_cost 3000.5", http.MethodPost, "/purchases", e.admin,
		`{"supplier_id":1,"items":[{"product_id":1,"quantity":2,"unit_cost":3000.5}]}`)

	for _, c := range []struct {
		what string
		q    string
		want int
	}{
		{"customer_payments", `SELECT COUNT(*) FROM customer_payments`, 0},
		{"expenses", `SELECT COUNT(*) FROM expenses`, 0},
		{"products", `SELECT COUNT(*) FROM products`, 5},
		{"purchases", `SELECT COUNT(*) FROM purchases`, 0},
		{"register_movements", `SELECT COUNT(*) FROM register_movements WHERE register_id = 1`, 1},
	} {
		if got := e.count(t, c.q); got != c.want {
			t.Errorf("%s = %d, want %d", c.what, got, c.want)
		}
	}
}
