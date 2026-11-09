package handler

// US-096 contract §4, T-7 .. T-12: the server-side validations approved at
// GATE 1 (Q-4, G-8, Q-8). Each goes through h.Routes() on a file-backed DB
// (moneyEnv) and checks the exact 400 message and that nothing was stored.

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

	code, body = e.send(t, http.MethodPost, "/registers/1/movements", e.cashier, `{"type":"cash_out","amount":0.4,"description":"x"}`)
	if code != http.StatusCreated {
		t.Fatalf("cash_out 0.4: status %d, want 201; body %s", code, body)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1`); n != 2 {
		t.Errorf("movements after cash_out 0.4 = %d, want 2", n)
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

// T-10 (AC-15): payments are checked against the server-recalculated total.
func TestMoneyValidation_T10_PaymentsVsRecalculatedTotal(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())
	assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)

	// Sale B: client tax 191, cash $1.192,4; recalculated total $1.191,19 (gap 1.21).
	code, body := e.send(t, http.MethodPost, "/sales", e.cashier,
		`{"items":[{"product_id":2,"quantity":1,"unit_price":1001,"discount":0}],"discount":0,"tax":191,"total":1192,"payments":[{"method":"cash","amount":1192.4}]}`)
	assertError(t, "sale B", code, body, http.StatusBadRequest, "la suma de los pagos ($1192) no coincide con el total de la venta ($1191)")
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 0 {
		t.Errorf("sales after sale B = %d, want 0", n)
	}

	// Sale A: client tax 190, cash $1.191 on $1.191,19 (gap 0.19) is still accepted.
	code, body = e.send(t, http.MethodPost, "/sales", e.cashier,
		`{"items":[{"product_id":2,"quantity":1,"unit_price":1001,"discount":0}],"discount":0,"tax":190,"total":1191,"payments":[{"method":"cash","amount":1191}]}`)
	if code != http.StatusCreated {
		t.Fatalf("sale A: status %d, want 201; body %s", code, body)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 1 {
		t.Errorf("sales after sale A = %d, want 1", n)
	}
	if got := e.float(t, `SELECT total FROM sales`); !moneyClose(got, 1191.19) {
		t.Errorf("sale A stored total = %v, want 1191.19", got)
	}
}

// T-11 (AC-16): 4 x Arroz $2.500 (subtotal $10.000) with discount $12.000.
func TestMoneyValidation_T11_DiscountAboveSubtotal(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())
	assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)

	code, body := e.send(t, http.MethodPost, "/sales", e.cashier,
		`{"items":[{"product_id":1,"quantity":4,"unit_price":2500,"discount":0}],"discount":12000,"tax":0,"total":-2000,"payments":[{"method":"cash","amount":0.3}]}`)
	assertError(t, "discount 12000 on 10000", code, body, http.StatusBadRequest, "el descuento no puede superar el subtotal de la venta")
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 0 {
		t.Errorf("sales = %d, want 0", n)
	}
	if got := e.float(t, `SELECT stock FROM products WHERE id = 1`); got != 500 {
		t.Errorf("Arroz stock = %v, want 500", got)
	}
}

// T-12 (AC-17): 4 x Arroz $2.500 with discount $10.000 -> total 0, paid $0,30.
func TestMoneyValidation_T12_TotalMustBePositive(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())
	assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)

	code, body := e.send(t, http.MethodPost, "/sales", e.cashier,
		`{"items":[{"product_id":1,"quantity":4,"unit_price":2500,"discount":0}],"discount":10000,"tax":0,"total":0,"payments":[{"method":"cash","amount":0.3}]}`)
	assertError(t, "discount 10000 on 10000", code, body, http.StatusBadRequest, "el total de la venta debe ser mayor a cero")
	checks := []struct {
		what string
		q    string
		want int
	}{
		{"sales", `SELECT COUNT(*) FROM sales`, 0},
		{"sale_payments", `SELECT COUNT(*) FROM sale_payments`, 0},
		{"register movements (opening only)", `SELECT COUNT(*) FROM register_movements WHERE register_id = 1`, 1},
		{"discount audit rows", `SELECT COUNT(*) FROM audit_log WHERE action = 'discount'`, 0},
	}
	for _, c := range checks {
		if got := e.count(t, c.q); got != c.want {
			t.Errorf("%s = %d, want %d", c.what, got, c.want)
		}
	}
	if got := e.float(t, `SELECT stock FROM products WHERE id = 1`); got != 500 {
		t.Errorf("Arroz stock = %v, want 500", got)
	}
}
