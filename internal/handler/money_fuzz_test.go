package handler

// US-096 contract §4, T-4 and T-5 (AC-10): fuzz targets on the amounts of
// POST /sales and POST /customers/{id}/payments. CI runs only the seed corpus;
// active fuzzing is by hand, e.g.
//
//	go test -run '^$' -fuzz FuzzSaleAmounts -fuzztime 60s ./internal/handler/
//
// Every input runs on a fresh file-backed DB (moneyEnv) with an open register.
// For every input: no panic (h.Routes() has no Recoverer) and no 5xx; on 201
// the stored rows satisfy AC-2 and the positivity of AC-7; on 4xx nothing
// changed (AC-6). Inputs JSON cannot express (NaN, ±Inf) are dropped.

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"testing"
)

var moneyFuzzMethods = []string{"cash", "transfer", "card", "credit"}

// T-4. The sale is posted by the admin, so a fuzzed unit_price that differs
// from the product price is a legal override rather than a 400.
func FuzzSaleAmounts(f *testing.F) {
	// productIdx, unitPrice, quantity, itemDiscount, saleDiscount, tax, nPays, a1, m1, a2, m2, a3, m3
	// Sale A: 1 x Gaseosa $1.001 at 19%, client tax 190, cash $1.191.
	f.Add(uint8(1), 1001.0, 1.0, 0.0, 0.0, 190.0, uint8(1), 1191.0, "cash", 0.0, "", 0.0, "")
	// Sale B: client tax 191, cash $1.192,4 (recalculated total $1.191,19).
	f.Add(uint8(1), 1001.0, 1.0, 0.0, 0.0, 191.0, uint8(1), 1192.4, "cash", 0.0, "", 0.0, "")
	// 0.125 kg x Queso $9.999 -> subtotal 1249.875.
	f.Add(uint8(3), 9999.0, 0.125, 0.0, 0.0, 0.0, uint8(1), 1249.875, "cash", 0.0, "", 0.0, "")
	// 0.1 kg x Queso $9.999 -> subtotal 999.9000000000001.
	f.Add(uint8(3), 9999.0, 0.1, 0.0, 0.0, 0.0, uint8(1), 999.9, "cash", 0.0, "", 0.0, "")
	// Sale-level discount -10000 on a $50.000 subtotal, cash $60.000.
	f.Add(uint8(0), 50000.0, 1.0, 0.0, -10000.0, 0.0, uint8(1), 60000.0, "cash", 0.0, "", 0.0, "")
	// 2 x $1e308 with two credit lines of 1.7e308 (overflow -> NaN tax).
	f.Add(uint8(0), 1e308, 2.0, 0.0, 0.0, 0.0, uint8(2), 1.7e308, "credit", 1.7e308, "credit", 0.0, "")

	template := moneyTemplate(f, f.TempDir())
	f.Fuzz(func(t *testing.T, productIdx uint8, unitPrice, quantity, itemDiscount, saleDiscount, tax float64,
		nPays uint8, a1 float64, m1 string, a2 float64, m2 string, a3 float64, m3 string) {
		catalog := moneyFullCatalog()
		productID := catalog[int(productIdx)%len(catalog)].id
		customerID := int64(1)
		req := moneySaleReq{
			CustomerID: &customerID,
			Items:      []moneyItemReq{{ProductID: productID, Quantity: quantity, UnitPrice: unitPrice, Discount: itemDiscount}},
			Discount:   saleDiscount,
			Tax:        tax,
		}
		amounts := []float64{a1, a2, a3}
		methods := []string{m1, m2, m3}
		count := int(nPays) // 1..3 as given; any other value wraps into 1..3
		if count < 1 || count > 3 {
			count = 1 + count%3
		}
		for i := 0; i < count; i++ {
			req.Payments = append(req.Payments, moneyPayReqJSON{Method: methods[i], Amount: amounts[i]})
		}
		raw, err := json.Marshal(req)
		if err != nil {
			return // NaN or ±Inf: not expressible in JSON
		}

		e := newMoneyEnv(t, t.TempDir(), template, catalog)
		assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)
		before := e.snapshot(t)
		code, body := e.send(t, http.MethodPost, "/sales", e.admin, string(raw))
		switch {
		case code >= 500:
			t.Fatalf("POST /sales %s -> %d (5xx); body %s", raw, code, body)
		case code == http.StatusCreated:
			e.checkStoredRows(t)
		case code >= 400:
			e.assertUnchanged(t, fmt.Sprintf("POST /sales %s -> %d", raw, code), before)
		default:
			t.Fatalf("POST /sales %s -> unexpected %d; body %s", raw, code, body)
		}
	})
}

// moneyAbonoSeedWant is the status each T-5 seed must get (method cash).
var moneyAbonoSeedWant = map[float64]int{
	0.4:    http.StatusCreated,
	1e-9:   http.StatusCreated,
	1e308:  http.StatusBadRequest,
	0:      http.StatusBadRequest,
	-1:     http.StatusBadRequest,
	2999.5: http.StatusCreated,
}

// T-5. Each input runs on a fresh DB where Dora owes $2.999,50 and a register
// is open; the cashier posts the abono.
func FuzzAbonoAmount(f *testing.F) {
	for _, a := range []float64{0.4, 1e-9, 1e308, 0, -1, 2999.5} {
		f.Add(a, "cash")
	}

	template := moneyTemplate(f, f.TempDir())
	f.Fuzz(func(t *testing.T, amount float64, method string) {
		if math.IsNaN(amount) || math.IsInf(amount, 0) {
			return // not expressible in JSON
		}
		raw, err := json.Marshal(map[string]any{"amount": amount, "method": method})
		if err != nil {
			return
		}

		e := newMoneyEnv(t, t.TempDir(), template, moneyFullCatalog())
		assertCode(t, "open register", e.doOpen(t, 100000), http.StatusCreated)
		e.exec(t, `INSERT INTO sales (id, user_id, customer_id, register_id, subtotal, total, status, payment_method) VALUES (1, 1, 1, 1, 2999.5, 2999.5, 'completed', 'credit')`)
		e.exec(t, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 2999.5)`)
		if got := e.customerBalance(t, 1); got != 2999.5 {
			t.Fatalf("seeded Dora balance = %v, want 2999.5", got)
		}

		before := e.snapshot(t)
		code, body := e.send(t, http.MethodPost, "/customers/1/payments", e.cashier, string(raw))
		if want, ok := moneyAbonoSeedWant[amount]; ok && method == "cash" && code != want {
			t.Errorf("abono %v cash -> %d, want %d; body %s", amount, code, want, body)
		}
		switch {
		case code >= 500:
			t.Fatalf("abono %s -> %d (5xx); body %s", raw, code, body)
		case code == http.StatusCreated:
			e.checkStoredRows(t)
			if got := e.float(t, `SELECT amount FROM customer_payments ORDER BY id DESC LIMIT 1`); got != amount {
				t.Errorf("stored abono = %v, sent %v", got, amount)
			}
		case code >= 400:
			e.assertUnchanged(t, fmt.Sprintf("abono %s -> %d", raw, code), before)
		default:
			t.Fatalf("abono %s -> unexpected %d; body %s", raw, code, body)
		}
	})
}
