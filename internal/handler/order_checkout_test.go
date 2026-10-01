package handler

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestOrder_US057_Checkout(t *testing.T) {
	f := newOrderFixture(t)
	// Seed register for sales
	f.exec(`INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 0, 'open')`)
	// Seed stock
	f.exec(`UPDATE products SET type = 'product', stock = 100`)

	tableID := f.mkTable("Mesa 1")
	orderID := f.openOrder(tableID)

	// Scenario 3: Checkout empty order -> 400
	wEmpty := f.do(http.MethodPost, "/orders/"+jsonStr(orderID)+"/checkout", `{"payments":[{"method":"cash","amount":0}]}`)
	if wEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty order checkout, got %d", wEmpty.Code)
	}

	// Add 3 items: Cafe x2 ($9000), Torta x1 ($8000), Cafe x1 ($4500)
	f.addItem(orderID, cafeID, 2)
	f.addItem(orderID, tortaID, 1)
	it3 := f.addItem(orderID, cafeID, 1)

	// Void the third item
	f.voidItem(orderID, it3)

	salesBefore := f.count(`SELECT COUNT(*) FROM sales`)

	// Scenario 1 & 2: Checkout with $20000 cash. Total should be 17000 (excluding voided)
	wCheck := f.do(http.MethodPost, "/orders/"+jsonStr(orderID)+"/checkout", `{
		"payments": [{"method":"cash", "amount": 17000}],
		"cash_tendered": 20000,
		"change_amount": 3000
	}`)
	if wCheck.Code != http.StatusCreated {
		t.Fatalf("checkout failed: %s", wCheck.Body.String())
	}

	var env struct {
		Data struct {
			OrderID int64   `json:"order_id"`
			SaleID  int64   `json:"sale_id"`
			Total   float64 `json:"total"`
		} `json:"data"`
	}
	json.Unmarshal(wCheck.Body.Bytes(), &env)
	res := env.Data

	if res.SaleID == 0 {
		t.Fatalf("checkout response missing sale_id: %s", wCheck.Body.String())
	}
	if res.Total != 17000 {
		t.Errorf("expected total 17000, got %v", res.Total)
	}

	salesAfter := f.count(`SELECT COUNT(*) FROM sales`)
	if salesAfter != salesBefore+1 {
		t.Errorf("expected 1 new sale, got %d", salesAfter-salesBefore)
	}

	// Verify sale_items has 2 rows (voided item excluded)
	saleItemsCount := f.count(`SELECT COUNT(*) FROM sale_items WHERE sale_id = ?`, res.SaleID)
	if saleItemsCount != 2 {
		t.Errorf("expected 2 sale_items, got %d", saleItemsCount)
	}

	// Verify order status is closed and sale_id populated
	var oStatus string
	var oSaleID *int64
	f.db.QueryRow(`SELECT status, sale_id FROM orders WHERE id = ?`, orderID).Scan(&oStatus, &oSaleID)
	if oStatus != "closed" {
		t.Errorf("expected order status closed, got %s", oStatus)
	}
	if oSaleID == nil || *oSaleID != res.SaleID {
		t.Errorf("expected order sale_id %d, got %v", res.SaleID, oSaleID)
	}

	// Scenario 4: Mixed payment checkout
	orderID2 := f.openOrder(f.mkTable("Mesa 2"))
	f.addItem(orderID2, cafeID, 2)
	f.addItem(orderID2, tortaID, 1)
	
	wCheck2 := f.do(http.MethodPost, "/orders/"+jsonStr(orderID2)+"/checkout", `{
		"payments": [
			{"method":"cash", "amount": 10000},
			{"method":"transfer", "amount": 7000}
		]
	}`)
	if wCheck2.Code != http.StatusCreated {
		t.Fatalf("mixed checkout failed: %s", wCheck2.Body.String())
	}
	
	var env2 struct {
		Data struct {
			SaleID int64 `json:"sale_id"`
		} `json:"data"`
	}
	json.Unmarshal(wCheck2.Body.Bytes(), &env2)
	saleID2 := env2.Data.SaleID

	var pm string
	f.db.QueryRow(`SELECT payment_method FROM sales WHERE id = ?`, saleID2).Scan(&pm)
	if pm != "mixed" {
		t.Errorf("expected payment_method mixed, got %s", pm)
	}
	
	mixedLines := f.count(`SELECT COUNT(*) FROM sale_payments WHERE sale_id = ?`, saleID2)
	if mixedLines != 2 {
		t.Errorf("expected 2 payment lines, got %d", mixedLines)
	}
}

func jsonStr(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}
