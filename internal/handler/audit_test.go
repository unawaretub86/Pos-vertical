package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreyo/pos/internal/database"
	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
	"github.com/go-chi/chi/v5"
)

func withURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func withClaims(r *http.Request, claims *model.Claims) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), appmw.ClaimsKey, claims))
}

func execSeed(t *testing.T, db *database.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.DB.ExecContext(context.Background(), q, utcArgs(args)...); err != nil {
		t.Fatalf("seed exec: %v\nSQL: %s", err, q)
	}
}

func countAuditRows(t *testing.T, db *database.DB, action, entity string, entityID int64) int {
	t.Helper()
	var count int
	err := db.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM audit_log WHERE action = ? AND entity = ? AND entity_id = ?`,
		action, entity, entityID).Scan(&count)
	if err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	return count
}

func TestVoidSale_WritesAuditLog(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	execSeed(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (1, 'Prod A', 'A', 10000, 5000, 'unit', 1, 10)`)
	execSeed(t, db, `INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 0, 'open')`)
	execSeed(t, db, `INSERT INTO sales (id, user_id, register_id, subtotal, total, status, payment_method, reference) VALUES (453, 1, 1, 10000, 10000, 'completed', 'cash', '')`)
	execSeed(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (453, 1, 1, 10000, 10000)`)

	req := httptest.NewRequest(http.MethodDelete, "/api/sales/453/void", nil)
	req = withURLParam(req, "id", "453")
	req = withClaims(req, &model.Claims{UserID: 1, Username: "admin", Role: "admin"})
	w := httptest.NewRecorder()

	h.voidSale(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body: %s", w.Code, w.Body.String())
	}

	if got := countAuditRows(t, db, "void", "sale", 453); got != 1 {
		t.Errorf("audit_log rows for void sale 453 = %d, want 1", got)
	}
}

func TestUpdateProduct_PriceChange_WritesAuditLog(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	execSeed(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (1, 'Prod A', 'A', 10000, 5000, 'unit', 1, 10)`)

	body, _ := json.Marshal(map[string]any{
		"name": "Prod A", "code": "A", "type": "product", "sale_price": 12000, "cost": 5000, "unit": "unit", "active": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/products/1", bytes.NewReader(body))
	req = withURLParam(req, "id", "1")
	req = withClaims(req, &model.Claims{UserID: 1, Username: "admin", Role: "admin"})
	w := httptest.NewRecorder()

	h.updateProduct(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body: %s", w.Code, w.Body.String())
	}

	if got := countAuditRows(t, db, "product_price_change", "product", 1); got != 1 {
		t.Errorf("audit_log rows for product 1 price change = %d, want 1", got)
	}
}

func TestUpdateProduct_NameOnly_NoAuditLog(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	execSeed(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (1, 'Prod A', 'A', 10000, 5000, 'unit', 1, 10)`)

	body, _ := json.Marshal(map[string]any{
		"name": "Prod A Renamed", "code": "A", "type": "product", "sale_price": 10000, "cost": 5000, "unit": "unit", "active": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/products/1", bytes.NewReader(body))
	req = withURLParam(req, "id", "1")
	req = withClaims(req, &model.Claims{UserID: 1, Username: "admin", Role: "admin"})
	w := httptest.NewRecorder()

	h.updateProduct(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body: %s", w.Code, w.Body.String())
	}

	if got := countAuditRows(t, db, "product_price_change", "product", 1); got != 0 {
		t.Errorf("audit_log rows for name-only update = %d, want 0", got)
	}
}

func TestListAudit_FilterByAction(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	execSeed(t, db, `INSERT INTO audit_log (user_id, action, entity, entity_id, details) VALUES (1, 'cash_out', 'register_movement', 1, '{}')`)
	execSeed(t, db, `INSERT INTO audit_log (user_id, action, entity, entity_id, details) VALUES (1, 'void', 'sale', 1, '{}')`)
	execSeed(t, db, `INSERT INTO audit_log (user_id, action, entity, entity_id, details) VALUES (1, 'cash_out', 'register_movement', 2, '{}')`)

	req := httptest.NewRequest(http.MethodGet, "/api/audit?action=cash_out", nil)
	w := httptest.NewRecorder()

	h.listAudit(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body: %s", w.Code, w.Body.String())
	}

	var envelope struct {
		Data []model.AuditLogEntry `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(envelope.Data) != 2 {
		t.Fatalf("entries = %d, want 2", len(envelope.Data))
	}
	for _, e := range envelope.Data {
		if e.Action != "cash_out" {
			t.Errorf("entry action = %q, want cash_out", e.Action)
		}
	}
}
