package handler

// US-102 (R-133) contract §5, T-6, T-7, T-9, T-10, T-21, T-23: whole-peso
// sales through h.Routes() on the moneyEnv fixture (file-backed DB, literal
// INSERT seeding, real JWTs). Catalog of the contract: P1 Item $10.001 IVA 19%,
// P2 Cafe $1.001 IVA 19%, P3 Panela $50 IVA 5%, P4 Queso $9.999/kg IVA 0,
// P5 Arroz $50.000 IVA 0. Customers: Dora (1), Beto (2).

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/model"
)

func saleMoneyCatalog() []moneyProduct {
	return []moneyProduct{
		{1, "Item", "product", "unit", 10001, 0.19, 20},
		{2, "Cafe", "product", "unit", 1001, 0.19, 20},
		{3, "Panela", "product", "unit", 50, 0.05, 20},
		{4, "Queso", "product", "kg", 9999, 0, 200},
		{5, "Arroz", "product", "unit", 50000, 0, 20},
	}
}

func newSaleMoneyEnv(t *testing.T) *moneyEnv {
	t.Helper()
	e := newMoneyEnv(t, t.TempDir(), "", saleMoneyCatalog())
	assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)
	return e
}

// postSale posts a raw JSON sale body as the cashier.
func (e *moneyEnv) postSale(t *testing.T, body string) (int, []byte) {
	t.Helper()
	return e.send(t, http.MethodPost, "/sales", e.cashier, body)
}

// expectSaleRejected asserts 400, a non-empty Spanish message containing every
// fragment, and that no stored figure changed.
func (e *moneyEnv) expectSaleRejected(t *testing.T, step, body string, fragments ...string) {
	t.Helper()
	before := e.snapshot(t)
	code, resp := e.postSale(t, body)
	if code != http.StatusBadRequest {
		t.Fatalf("%s: status %d, want 400; body %s", step, code, resp)
	}
	msg := moneyErrMsg(resp)
	if msg == "" {
		t.Errorf("%s: empty error message", step)
	}
	for _, f := range fragments {
		if !strings.Contains(strings.ToLower(msg), f) {
			t.Errorf("%s: error %q does not name %q", step, msg, f)
		}
	}
	e.assertUnchanged(t, step, before)
}

func (e *moneyEnv) expectSaleCreated(t *testing.T, step, body string) model.Sale {
	t.Helper()
	code, resp := e.postSale(t, body)
	if code != http.StatusCreated {
		t.Fatalf("%s: status %d, want 201; body %s", step, code, resp)
	}
	var s model.Sale
	moneyData(t, resp, &s)
	return s
}

const p1Item = `{"product_id":1,"quantity":1,"unit_price":10001,"discount":0}`

// T-6 (AC-3, AC-4).
func TestSaleMoneyHTTP_T6_WholePesoInputsAndExactPayments(t *testing.T) {
	e := newSaleMoneyEnv(t)

	e.expectSaleRejected(t, "payment 11901.5",
		`{"items":[`+p1Item+`],"total":11901,"payments":[{"method":"cash","amount":11901.5}]}`, "pago")
	e.expectSaleRejected(t, "discount 100.5",
		`{"items":[`+p1Item+`],"discount":100.5,"total":11801,"payments":[{"method":"cash","amount":11801}]}`, "descuento")
	e.expectSaleRejected(t, "cash_tendered 12000.5",
		`{"items":[`+p1Item+`],"total":11901,"cash_tendered":12000.5,"change_amount":99,"payments":[{"method":"cash","amount":11901}]}`)

	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 0 {
		t.Errorf("sales after the three rejections = %d, want 0", n)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sale_payments`); n != 0 {
		t.Errorf("sale_payments after the three rejections = %d, want 0", n)
	}

	// cash 5000 + credit 6900 on total 11901: short by 1.
	before := e.snapshot(t)
	code, resp := e.postSale(t, `{"customer_id":1,"items":[`+p1Item+`],"total":11901,"payments":[{"method":"cash","amount":5000},{"method":"credit","amount":6900}]}`)
	assertError(t, "5000 + 6900", code, resp, http.StatusBadRequest, "la suma de los pagos ($11900) no coincide con el total de la venta ($11901)")
	e.assertUnchanged(t, "5000 + 6900", before)

	s := e.expectSaleCreated(t, "5000 + 6901",
		`{"customer_id":1,"items":[`+p1Item+`],"total":11901,"payments":[{"method":"cash","amount":5000},{"method":"credit","amount":6901}]}`)
	if s.Total != 11901 {
		t.Errorf("created total = %v, want 11901", s.Total)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sale_payments`); n != 2 {
		t.Errorf("sale_payments after 5000 + 6901 = %d, want 2", n)
	}
}

// T-7 (AC-2): the server prices the sale; client tax and subtotal are ignored.
func TestSaleMoneyHTTP_T7_ServerPricesTheSale(t *testing.T) {
	e := newSaleMoneyEnv(t)

	s := e.expectSaleCreated(t, "no total, no tax, no subtotal",
		`{"items":[`+p1Item+`],"payments":[{"method":"cash","amount":11901}]}`)
	if s.Total != 11901 || s.Tax != 1900 || s.Subtotal != 10001 {
		t.Errorf("table-checkout shape: subtotal/tax/total = %v/%v/%v, want 10001/1900/11901", s.Subtotal, s.Tax, s.Total)
	}
	if got := e.float(t, `SELECT total FROM sales WHERE id = 1`); got != 11901 {
		t.Errorf("stored total = %v, want 11901", got)
	}

	e.expectSaleCreated(t, "wrong client tax and subtotal",
		`{"items":[`+p1Item+`],"tax":1000,"subtotal":5,"total":11901,"payments":[{"method":"cash","amount":11901}]}`)
	if got := e.float(t, `SELECT tax FROM sales WHERE id = 2`); got != 1900 {
		t.Errorf("stored tax with client tax 1000 = %v, want 1900", got)
	}
	if got := e.float(t, `SELECT subtotal FROM sales WHERE id = 2`); got != 10001 {
		t.Errorf("stored subtotal with client subtotal 5 = %v, want 10001", got)
	}

	e.expectSaleRejected(t, "total 11901.19",
		`{"items":[`+p1Item+`],"total":11901.19,"payments":[{"method":"cash","amount":11901}]}`)

	before := e.snapshot(t)
	code, resp := e.postSale(t, `{"items":[`+p1Item+`],"total":11902,"payments":[{"method":"cash","amount":11902}]}`)
	assertError(t, "total 11902", code, resp, http.StatusBadRequest, "total inconsistente: esperado 11901, recibido 11902")
	e.assertUnchanged(t, "total 11902", before)
}

// T-9 (AC-6): a fiado sale paid in full by an abono of its displayed amount.
func TestSaleMoneyHTTP_T9_AbonoOfDisplayedCreditLeavesZero(t *testing.T) {
	e := newSaleMoneyEnv(t)
	e.expectSaleCreated(t, "Dora credit",
		`{"customer_id":1,"items":[`+p1Item+`],"total":11901,"payments":[{"method":"credit","amount":11901}]}`)
	if got := e.customerBalance(t, 1); got != 11901 {
		t.Fatalf("Dora balance after the sale = %v, want 11901", got)
	}

	code, resp := e.send(t, http.MethodPost, "/customers/1/payments", e.cashier, `{"amount":11901,"method":"cash"}`)
	if code != http.StatusCreated {
		t.Fatalf("abono 11901: status %d, want 201; body %s", code, resp)
	}
	if got := e.customerBalance(t, 1); got != 0 {
		t.Errorf("GET /customers/1/balance = %v, want 0 (was 0.19)", got)
	}
	var list struct {
		Customers []struct {
			ID      int64   `json:"id"`
			Balance float64 `json:"balance"`
		} `json:"customers"`
		TotalPortfolio float64 `json:"total_portfolio"`
	}
	e.get(t, "/customers", &list)
	found := false
	for _, c := range list.Customers {
		if c.ID == 1 {
			found = true
			if c.Balance != 0 {
				t.Errorf("GET /customers row balance = %v, want 0", c.Balance)
			}
		}
	}
	if !found {
		t.Errorf("Dora missing from GET /customers")
	}
	if list.TotalPortfolio != 0 {
		t.Errorf("total_portfolio = %v, want 0", list.TotalPortfolio)
	}
}

// T-10 (AC-6): mixed sale, abono above the credit line, abono of the line.
func TestSaleMoneyHTTP_T10_MixedSaleAbono(t *testing.T) {
	e := newSaleMoneyEnv(t)
	e.expectSaleCreated(t, "Dora mixed",
		`{"customer_id":1,"items":[`+p1Item+`],"total":11901,"payments":[{"method":"cash","amount":5000},{"method":"credit","amount":6901}]}`)
	if got := e.customerBalance(t, 1); got != 6901 {
		t.Fatalf("Dora balance = %v, want 6901", got)
	}

	code, resp := e.send(t, http.MethodPost, "/customers/1/payments", e.cashier, `{"amount":6902,"method":"cash"}`)
	assertError(t, "abono 6902", code, resp, http.StatusBadRequest, "el pago ($6902) supera la deuda actual ($6901)")
	if got := e.customerBalance(t, 1); got != 6901 {
		t.Errorf("balance after the rejected abono = %v, want 6901", got)
	}

	code, resp = e.send(t, http.MethodPost, "/customers/1/payments", e.cashier, `{"amount":6901,"method":"cash"}`)
	if code != http.StatusCreated {
		t.Fatalf("abono 6901: status %d, want 201; body %s", code, resp)
	}
	if got := e.customerBalance(t, 1); got != 0 {
		t.Errorf("balance after abono 6901 = %v, want 0", got)
	}
}

// T-21 (AC-4a): a zero-total sale takes an empty payments list.
func TestSaleMoneyHTTP_T21_ZeroTotalSale(t *testing.T) {
	e := newSaleMoneyEnv(t)
	arroz := `{"product_id":5,"quantity":1,"unit_price":50000,"discount":50000}`
	arrozFull := `{"product_id":5,"quantity":1,"unit_price":50000,"discount":0}`

	e.expectSaleRejected(t, "zero total with cash 1",
		`{"items":[`+arroz+`],"total":0,"payments":[{"method":"cash","amount":1}]}`)
	e.expectSaleRejected(t, "zero total with cash 0",
		`{"items":[`+arroz+`],"total":0,"payments":[{"method":"cash","amount":0}]}`)

	before := e.snapshot(t)
	code, resp := e.postSale(t, `{"items":[`+arrozFull+`],"total":50000,"payments":[]}`)
	assertError(t, "no discount, no payments", code, resp, http.StatusBadRequest, "la venta debe tener al menos un método de pago")
	e.assertUnchanged(t, "no discount, no payments", before)

	s := e.expectSaleCreated(t, "100% discount, empty payments", `{"items":[`+arroz+`],"total":0,"payments":[]}`)
	if s.Total != 0 {
		t.Errorf("created total = %v, want 0", s.Total)
	}
	if got := e.float(t, `SELECT total FROM sales WHERE id = 1`); got != 0 {
		t.Errorf("stored total = %v, want 0", got)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 1 {
		t.Errorf("sales = %d, want 1", n)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sale_payments`); n != 0 {
		t.Errorf("sale_payments = %d, want 0", n)
	}
	var sum moneyRegisterSummary
	e.get(t, "/registers/1", &sum)
	if sum.Expected != 100000 {
		t.Errorf("register expected = %v, want 100000 (unchanged)", sum.Expected)
	}
	if got := e.float(t, `SELECT stock FROM products WHERE id = 5`); got != 19 {
		t.Errorf("Arroz stock = %v, want 19", got)
	}
}

// T-23 (AC-3): discount bounds.
func TestSaleMoneyHTTP_T23_DiscountBounds(t *testing.T) {
	e := newSaleMoneyEnv(t)
	item := func(discount int) string {
		return fmt.Sprintf(`{"product_id":1,"quantity":1,"unit_price":10001,"discount":%d}`, discount)
	}

	e.expectSaleRejected(t, "item discount 10002", `{"items":[`+item(10002)+`],"total":0,"payments":[]}`, "descuento")
	e.expectSaleRejected(t, "item discount -1", `{"items":[`+item(-1)+`],"total":11902,"payments":[{"method":"cash","amount":11902}]}`, "descuento")
	e.expectSaleRejected(t, "global discount -1", `{"items":[`+item(0)+`],"discount":-1,"total":11902,"payments":[{"method":"cash","amount":11902}]}`, "descuento")
	e.expectSaleRejected(t, "global discount 10002", `{"items":[`+item(0)+`],"discount":10002,"total":1899,"payments":[{"method":"cash","amount":1899}]}`, "descuento")
	// 0.5 kg Queso: lineBase 5000 (4999.5 rounded), so 5001 is over it.
	e.expectSaleRejected(t, "kg item discount 5001",
		`{"items":[{"product_id":4,"quantity":0.5,"unit_price":9999,"discount":5001}],"total":0,"payments":[]}`, "descuento")

	// Exactly the base is a free line.
	s := e.expectSaleCreated(t, "item discount 10001", `{"items":[`+item(10001)+`],"total":0,"payments":[]}`)
	if s.Total != 0 {
		t.Errorf("free line total = %v, want 0", s.Total)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sale_payments`); n != 0 {
		t.Errorf("sale_payments after the free line = %d, want 0", n)
	}
	// The unrounded 4999.5 would have rejected 5000.
	e.expectSaleCreated(t, "kg item discount 5000",
		`{"items":[{"product_id":4,"quantity":0.5,"unit_price":9999,"discount":5000}],"total":0,"payments":[]}`)

	// Global discount equal to the subtotal: tax stays 1900.
	g := e.expectSaleCreated(t, "global discount 10001",
		`{"items":[`+item(0)+`],"discount":10001,"total":1900,"payments":[{"method":"cash","amount":1900}]}`)
	if g.Tax != 1900 || g.Total != 1900 {
		t.Errorf("global discount 10001: tax/total = %v/%v, want 1900/1900", g.Tax, g.Total)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 3 {
		t.Errorf("sales after the three accepted cases = %d, want 3", n)
	}
}
