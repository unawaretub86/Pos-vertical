package handler

// US-102 (R-133) contract §5, T-8 and T-25: table checkout prices IVA on the
// server. P1 Item $10.001 IVA 19% (id 11), P2 Cafe $1.001 IVA 19% (id 12).
// The order total is computed at read time from the current product tax_rate;
// nothing about tax is persisted on orders or order_items.

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	taxedItemID = 11
	taxedCafeID = 12
)

func newTaxedOrderFixture(t *testing.T) *orderFixture {
	t.Helper()
	f := newOrderFixture(t)
	f.exec(`INSERT INTO products (id, name, code, type, sale_price, cost, unit, stock, tax_rate, active) VALUES (11, 'Item', 'ITM', 'product', 10001, 0, 'unit', 20, 0.19, 1)`)
	f.exec(`INSERT INTO products (id, name, code, type, sale_price, cost, unit, stock, tax_rate, active) VALUES (12, 'Cafe IVA', 'CIV', 'product', 1001, 0, 'unit', 20, 0.19, 1)`)
	f.exec(`INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 0, 'open')`)
	return f
}

func (f *orderFixture) checkoutCash(orderID int64, amount int) *httptest.ResponseRecorder {
	return f.do(http.MethodPost, fmt.Sprintf("/orders/%d/checkout", orderID),
		fmt.Sprintf(`{"payments":[{"method":"cash","amount":%d}]}`, amount))
}

func (f *orderFixture) floatQ(q string, args ...any) float64 {
	f.t.Helper()
	var v float64
	if err := f.db.QueryRow(q, args...).Scan(&v); err != nil {
		f.t.Fatalf("query: %v\nSQL: %s", err, q)
	}
	return v
}

// T-8 (AC-5): the order total includes IVA, checkout prices it server-side.
func TestOrderMoney_T8_TableCheckoutWithIVA(t *testing.T) {
	f := newTaxedOrderFixture(t)
	orderID := f.openOrder(f.mkTable("Mesa 1"))
	f.addItem(orderID, taxedItemID, 1)

	if got := f.getOrder(orderID).Total; got != 11901 {
		t.Errorf("GET /orders/{id} total = %v, want 11901 (10001 + IVA 1900)", got)
	}

	w := f.checkoutCash(orderID, 10001)
	if w.Code != http.StatusBadRequest {
		t.Errorf("checkout cash 10001 = %d, want 400; body %s", w.Code, w.Body.String())
	}
	if n := f.count(`SELECT COUNT(*) FROM sales`); n != 0 {
		t.Errorf("sales after the rejected checkout = %d, want 0", n)
	}

	w = f.checkoutCash(orderID, 11901)
	if w.Code != http.StatusCreated {
		t.Fatalf("checkout cash 11901 = %d, want 201; body %s", w.Code, w.Body.String())
	}
	res := decodeInto[struct {
		OrderID int64   `json:"order_id"`
		SaleID  int64   `json:"sale_id"`
		Total   float64 `json:"total"`
	}](t, w)
	if res.Total != 11901 || res.SaleID == 0 {
		t.Errorf("checkout = %+v, want total 11901 and a sale_id", res)
	}
	if got := f.floatQ(`SELECT total FROM sales WHERE id = ?`, res.SaleID); got != 11901 {
		t.Errorf("stored sale total = %v, want 11901", got)
	}
	if got := f.floatQ(`SELECT tax FROM sales WHERE id = ?`, res.SaleID); got != 1900 {
		t.Errorf("stored sale tax = %v, want 1900", got)
	}
	if n := f.count(`SELECT COUNT(*) FROM electronic_documents`); n != 0 {
		t.Errorf("electronic_documents = %d, want 0", n)
	}
	if got := f.floatQ(`SELECT stock FROM products WHERE id = ?`, taxedItemID); got != 19 {
		t.Errorf("stock after checkout = %v, want 19", got)
	}

	w = f.do(http.MethodDelete, fmt.Sprintf("/sales/%d/void", res.SaleID), "")
	if w.Code != http.StatusOK {
		t.Fatalf("void = %d, want 200; body %s", w.Code, w.Body.String())
	}
	if got := f.floatQ(`SELECT stock FROM products WHERE id = ?`, taxedItemID); got != 20 {
		t.Errorf("stock after void = %v, want 20", got)
	}
}

func tableColumns(f *orderFixture, table string) []string {
	f.t.Helper()
	rows, err := f.db.Query(fmt.Sprintf(`SELECT name FROM pragma_table_info('%s')`, table))
	if err != nil {
		f.t.Fatalf("table_info %s: %v", table, err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			f.t.Fatalf("scan: %v", err)
		}
		cols = append(cols, c)
	}
	return cols
}

// T-25 (AC-5): read-time pricing, the rate at checkout, void items excluded.
func TestOrderMoney_T25_TaxIsNotPersistedOnOrders(t *testing.T) {
	f := newTaxedOrderFixture(t)

	// Tax data does not exist on orders or order_items.
	for _, table := range []string{"orders", "order_items"} {
		cols := tableColumns(f, table)
		if len(cols) == 0 {
			t.Fatalf("no columns read for %s", table)
		}
		for _, c := range cols {
			if strings.Contains(strings.ToLower(c), "tax") {
				t.Errorf("%s has a tax column %q, want none", table, c)
			}
		}
	}

	// A changed tax_rate shows at read time: 11901 -> 10001.
	orderID := f.openOrder(f.mkTable("Mesa 1"))
	f.addItem(orderID, taxedItemID, 1)
	if got := f.getOrder(orderID).Total; got != 11901 {
		t.Fatalf("total at 19%% = %v, want 11901", got)
	}
	f.exec(`UPDATE products SET tax_rate = 0 WHERE id = ?`, taxedItemID)
	if got := f.getOrder(orderID).Total; got != 10001 {
		t.Errorf("total after tax_rate = 0 = %v, want 10001", got)
	}

	// Checkout uses the rate current at checkout: 5% -> tax round(500.05) = 500.
	f.exec(`UPDATE products SET tax_rate = 0.05 WHERE id = ?`, taxedItemID)
	w := f.checkoutCash(orderID, 10501)
	if w.Code != http.StatusCreated {
		t.Fatalf("checkout cash 10501 at 5%% = %d, want 201; body %s", w.Code, w.Body.String())
	}
	res := decodeInto[struct {
		SaleID int64   `json:"sale_id"`
		Total  float64 `json:"total"`
	}](t, w)
	if res.Total != 10501 {
		t.Errorf("checkout total = %v, want 10501", res.Total)
	}
	if got := f.floatQ(`SELECT tax FROM sales WHERE id = ?`, res.SaleID); got != 500 {
		t.Errorf("stored tax at 5%% = %v, want 500", got)
	}

	// Voided items are excluded: P1 + voided P2 -> 11901.
	f.exec(`UPDATE products SET tax_rate = 0.19 WHERE id = ?`, taxedItemID)
	order2 := f.openOrder(f.mkTable("Mesa 2"))
	f.addItem(order2, taxedItemID, 1)
	cafe := f.addItem(order2, taxedCafeID, 1)
	if got := f.getOrder(order2).Total; got != 11901+1001+190 {
		t.Errorf("total with both items = %v, want 13092", got)
	}
	if w := f.voidItem(order2, cafe); w.Code != http.StatusOK {
		t.Fatalf("void item = %d, want 200; body %s", w.Code, w.Body.String())
	}
	if got := f.getOrder(order2).Total; got != 11901 {
		t.Errorf("total after voiding P2 = %v, want 11901", got)
	}
}
