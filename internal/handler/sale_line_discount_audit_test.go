package handler

// US-106 contract section 4, T-1..T-7: every per-line discount writes one
// audit_log row (action discount, entity sale_item, entity_id = sale id,
// user_id = seller) in the sale transaction. Fixture: newSaleMoneyEnv (P1 Item
// $10.001 IVA 19%, P4 Queso $9.999/kg IVA 0, P5 Arroz $50.000 IVA 0 stock 20;
// admin 1, cashier 2 "cajero1"; register open with $100.000) and, for the
// table checkout, newOrderFixture.

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/model"
)

type lineAuditRow struct {
	Action   string
	Entity   string
	EntityID int64
	UserID   int64
	Details  string
}

func (e *moneyEnv) auditRows(t *testing.T) []lineAuditRow {
	t.Helper()
	rows, err := e.db.DB.Query(`SELECT action, entity, entity_id, user_id, COALESCE(details, '') FROM audit_log ORDER BY id`)
	if err != nil {
		t.Fatalf("query audit_log: %v", err)
	}
	defer rows.Close()
	var out []lineAuditRow
	for rows.Next() {
		var r lineAuditRow
		if err := rows.Scan(&r.Action, &r.Entity, &r.EntityID, &r.UserID, &r.Details); err != nil {
			t.Fatalf("scan audit_log: %v", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate audit_log: %v", err)
	}
	return out
}

// lineDiscountDetails is the contract row shape (section 3).
type lineDiscountDetails struct {
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
}

// checkLineDetails asserts the exact key set, the values and whole-peso JSON
// numbers (no "2000.00").
func checkLineDetails(t *testing.T, step, raw string, want lineDiscountDetails, wantAmountLiteral string) {
	t.Helper()
	var keys map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		t.Fatalf("%s: details %q are not JSON: %v", step, raw, err)
	}
	got := make([]string, 0, len(keys))
	for k := range keys {
		got = append(got, k)
	}
	sort.Strings(got)
	wantKeys := []string{"amount", "product_id", "product_name", "quantity", "unit_price"}
	if strings.Join(got, ",") != strings.Join(wantKeys, ",") {
		t.Errorf("%s: details keys = %v, want exactly %v", step, got, wantKeys)
	}
	var d lineDiscountDetails
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("%s: decode details %q: %v", step, raw, err)
	}
	if d != want {
		t.Errorf("%s: details = %+v, want %+v (raw %s)", step, d, want, raw)
	}
	if !strings.Contains(raw, wantAmountLiteral) {
		t.Errorf("%s: raw details %s do not contain %s", step, raw, wantAmountLiteral)
	}
	if strings.Contains(raw, ".00") {
		t.Errorf("%s: raw details %s contain a decimal money literal", step, raw)
	}
}

// T-1 (AC-1): one row per discounted line, in cart order; none for discount 0.
func TestSaleLineDiscountAudit_T1_OneRowPerDiscountedLine(t *testing.T) {
	e := newSaleMoneyEnv(t)
	// Arroz 2 x 50.000 - 2.000 = 98.000; Item 10.001 + IVA 1.900; Queso
	// 0,5 kg x 9.999 = 5.000 - 1.000 = 4.000. Total 113.901.
	s := e.expectSaleCreated(t, "three lines, two discounted", `{"items":[`+
		`{"product_id":5,"quantity":2,"unit_price":50000,"discount":2000},`+
		`{"product_id":1,"quantity":1,"unit_price":10001,"discount":0},`+
		`{"product_id":4,"quantity":0.5,"unit_price":9999,"discount":1000}`+
		`],"total":113901,"payments":[{"method":"cash","amount":113901}]}`)
	if s.Total != 113901 {
		t.Fatalf("created total = %v, want 113901", s.Total)
	}

	rows := e.auditRows(t)
	if len(rows) != 2 {
		t.Fatalf("audit_log rows = %d (%+v), want 2", len(rows), rows)
	}
	for i, r := range rows {
		if r.Action != "discount" || r.Entity != "sale_item" || r.EntityID != 1 || r.UserID != 2 {
			t.Errorf("row %d = (%s, %s, %d, %d), want (discount, sale_item, 1, 2)", i, r.Action, r.Entity, r.EntityID, r.UserID)
		}
	}
	checkLineDetails(t, "row 0 Arroz", rows[0].Details,
		lineDiscountDetails{ProductID: 5, ProductName: "Arroz", Quantity: 2, UnitPrice: 50000, Amount: 2000}, `"amount":2000`)
	checkLineDetails(t, "row 1 Queso", rows[1].Details,
		lineDiscountDetails{ProductID: 4, ProductName: "Queso", Quantity: 0.5, UnitPrice: 9999, Amount: 1000}, `"amount":1000`)

	if n := e.count(t, `SELECT COUNT(*) FROM audit_log WHERE entity = 'sale'`); n != 0 {
		t.Errorf("sale-level audit rows = %d, want 0 (no sale discount)", n)
	}
}

// T-2 (AC-2): a 100% line discount on a zero-total sale still writes its row.
func TestSaleLineDiscountAudit_T2_FullDiscountZeroTotal(t *testing.T) {
	e := newSaleMoneyEnv(t)
	s := e.expectSaleCreated(t, "Arroz 100% off",
		`{"items":[{"product_id":5,"quantity":1,"unit_price":50000,"discount":50000}],"total":0,"payments":[]}`)
	if s.Total != 0 {
		t.Fatalf("created total = %v, want 0", s.Total)
	}

	rows := e.auditRows(t)
	if len(rows) != 1 {
		t.Fatalf("audit_log rows = %d (%+v), want 1", len(rows), rows)
	}
	r := rows[0]
	if r.Action != "discount" || r.Entity != "sale_item" || r.EntityID != 1 || r.UserID != 2 {
		t.Errorf("row = (%s, %s, %d, %d), want (discount, sale_item, 1, 2)", r.Action, r.Entity, r.EntityID, r.UserID)
	}
	checkLineDetails(t, "Arroz 100%", r.Details,
		lineDiscountDetails{ProductID: 5, ProductName: "Arroz", Quantity: 1, UnitPrice: 50000, Amount: 50000}, `"amount":50000`)
	if n := e.count(t, `SELECT COUNT(*) FROM sale_payments`); n != 0 {
		t.Errorf("sale_payments = %d, want 0", n)
	}
}

// T-3 (AC-3): line and sale-level discount rows coexist; the sale row is unchanged.
func TestSaleLineDiscountAudit_T3_LineAndSaleDiscount(t *testing.T) {
	e := newSaleMoneyEnv(t)
	e.expectSaleCreated(t, "line 2.000 + sale 1.000",
		`{"items":[{"product_id":5,"quantity":2,"unit_price":50000,"discount":2000}],"discount":1000,"total":97000,"payments":[{"method":"cash","amount":97000}]}`)

	if n := e.count(t, `SELECT COUNT(*) FROM audit_log WHERE action = 'discount' AND entity_id = 1`); n != 2 {
		t.Fatalf("discount rows for sale 1 = %d, want 2", n)
	}
	var saleDetails string
	if err := e.db.DB.QueryRow(`SELECT details FROM audit_log WHERE action = 'discount' AND entity = 'sale' AND entity_id = 1`).Scan(&saleDetails); err != nil {
		t.Fatalf("sale-level discount row: %v", err)
	}
	if saleDetails != `{"amount":1000.00}` {
		t.Errorf("sale-level details = %s, want {\"amount\":1000.00}", saleDetails)
	}
	var lineDetails string
	if err := e.db.DB.QueryRow(`SELECT details FROM audit_log WHERE action = 'discount' AND entity = 'sale_item' AND entity_id = 1 AND user_id = 2`).Scan(&lineDetails); err != nil {
		t.Fatalf("line discount row: %v", err)
	}
	checkLineDetails(t, "Arroz line", lineDetails,
		lineDiscountDetails{ProductID: 5, ProductName: "Arroz", Quantity: 2, UnitPrice: 50000, Amount: 2000}, `"amount":2000`)

	var entries []model.AuditLogEntry
	e.get(t, "/audit?action=discount", &entries)
	if len(entries) != 2 {
		t.Fatalf("GET /audit?action=discount = %d entries, want 2: %+v", len(entries), entries)
	}
	entities := []string{entries[0].Entity, entries[1].Entity}
	sort.Strings(entities)
	if entities[0] != "sale" || entities[1] != "sale_item" {
		t.Errorf("entities = %v, want [sale sale_item]", entities)
	}
	for i, en := range entries {
		if en.Username != "cajero1" {
			t.Errorf("entry %d username = %q, want cajero1", i, en.Username)
		}
		if en.EntityID != 1 {
			t.Errorf("entry %d entity_id = %d, want 1", i, en.EntityID)
		}
	}
}

// T-4 (AC-4): admin price override plus line discount: both rows, discount
// carries the overridden unit price.
func TestSaleLineDiscountAudit_T4_PriceOverrideAndDiscount(t *testing.T) {
	e := newSaleMoneyEnv(t)
	code, resp := e.send(t, http.MethodPost, "/sales", e.admin,
		`{"items":[{"product_id":5,"quantity":1,"unit_price":45000,"discount":5000}],"total":40000,"payments":[{"method":"cash","amount":40000}]}`)
	if code != http.StatusCreated {
		t.Fatalf("admin override + discount: status %d, want 201; body %s", code, resp)
	}

	var overrideDetails string
	if err := e.db.DB.QueryRow(`SELECT details FROM audit_log WHERE action = 'price_override' AND entity = 'sale_item' AND entity_id = 1 AND user_id = 1`).Scan(&overrideDetails); err != nil {
		t.Fatalf("price_override row: %v", err)
	}
	if want := `{"product_id":5,"original_price":50000.00,"override_price":45000.00}`; overrideDetails != want {
		t.Errorf("price_override details = %s, want %s", overrideDetails, want)
	}

	if n := e.count(t, `SELECT COUNT(*) FROM audit_log WHERE action = 'discount' AND entity = 'sale_item' AND entity_id = 1 AND user_id = 1`); n != 1 {
		t.Fatalf("line discount rows by admin for sale 1 = %d, want 1", n)
	}
	var lineDetails string
	if err := e.db.DB.QueryRow(`SELECT details FROM audit_log WHERE action = 'discount' AND entity = 'sale_item' AND entity_id = 1`).Scan(&lineDetails); err != nil {
		t.Fatalf("line discount row: %v", err)
	}
	checkLineDetails(t, "Arroz override", lineDetails,
		lineDiscountDetails{ProductID: 5, ProductName: "Arroz", Quantity: 1, UnitPrice: 45000, Amount: 5000}, `"amount":5000`)
	if n := e.count(t, `SELECT COUNT(*) FROM audit_log`); n != 2 {
		t.Errorf("audit_log rows = %d, want 2 (price_override + line discount)", n)
	}
}

// T-5 (AC-5): a rejected sale with a discounted line writes no audit row.
func TestSaleLineDiscountAudit_T5_RejectedSaleWritesNothing(t *testing.T) {
	e := newSaleMoneyEnv(t)
	e.expectSaleRejected(t, "Item discount 10.002 over base 10.001",
		`{"items":[{"product_id":1,"quantity":1,"unit_price":10001,"discount":10002}],"total":0,"payments":[]}`, "descuento")
	e.expectSaleRejected(t, "Arroz 2 disc 2.000 with total 1",
		`{"items":[{"product_id":5,"quantity":2,"unit_price":50000,"discount":2000}],"total":1,"payments":[{"method":"cash","amount":1}]}`,
		"total inconsistente: esperado 98000, recibido 1")
	if n := e.count(t, `SELECT COUNT(*) FROM audit_log`); n != 0 {
		t.Errorf("audit_log rows after two rejections = %d, want 0", n)
	}
}

// T-6 (AC-6): a failing line-discount audit insert rolls the whole sale back.
func TestSaleLineDiscountAudit_T6_AuditFailureRollsBack(t *testing.T) {
	e := newSaleMoneyEnv(t)
	e.exec(t, `ALTER TABLE audit_log RENAME TO audit_log_x`)

	code, resp := e.postSale(t,
		`{"items":[{"product_id":5,"quantity":2,"unit_price":50000,"discount":2000}],"total":98000,"payments":[{"method":"cash","amount":98000}]}`)
	if code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400; body %s", code, resp)
	}
	if msg := moneyErrMsg(resp); !strings.Contains(msg, "audit_log insert") {
		t.Errorf("error %q does not contain %q", msg, "audit_log insert")
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 0 {
		t.Errorf("sales = %d, want 0", n)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sale_items`); n != 0 {
		t.Errorf("sale_items = %d, want 0", n)
	}
	if got := e.float(t, `SELECT stock FROM products WHERE id = 5`); got != 20 {
		t.Errorf("Arroz stock = %v, want 20", got)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM register_movements`); n != 1 {
		t.Errorf("register_movements = %d, want 1 (opening only)", n)
	}
}

// T-7 (AC-7): table checkout drops client line discounts and writes no
// discount row (regression guard for internal/handler/order.go:200-215).
func TestSaleLineDiscountAudit_T7_TableCheckoutIgnoresLineDiscount(t *testing.T) {
	f := newOrderFixture(t)
	f.exec(`INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 0, 'open')`)
	f.exec(`UPDATE products SET type = 'product', stock = 100`)

	tableID := f.mkTable("Mesa 1")
	orderID := f.openOrder(tableID)
	f.addItem(orderID, cafeID, 2)

	w := f.do(http.MethodPost, "/orders/"+jsonStr(orderID)+"/checkout",
		`{"items":[{"product_id":1,"quantity":2,"unit_price":4500,"discount":9000}],"payments":[{"method":"cash","amount":9000}]}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("checkout: status %d, want 201; body %s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			SaleID int64   `json:"sale_id"`
			Total  float64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode checkout: %v; body %s", err, w.Body.String())
	}
	if env.Data.Total != 9000 {
		t.Errorf("checkout total = %v, want 9000", env.Data.Total)
	}
	if n := f.count(`SELECT COALESCE(SUM(discount), 0) FROM sale_items WHERE sale_id = ?`, env.Data.SaleID); n != 0 {
		t.Errorf("SUM(sale_items.discount) = %d, want 0", n)
	}
	if n := f.count(`SELECT COUNT(*) FROM audit_log WHERE action = 'discount'`); n != 0 {
		t.Errorf("discount audit rows = %d, want 0", n)
	}
}
