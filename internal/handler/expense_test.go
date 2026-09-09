package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreyo/pos/internal/model"
)

func TestCreateExpense_InvalidAmount_Rejected(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	body, _ := json.Marshal(map[string]any{"category": "servicios", "amount": 0, "note": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/expenses", bytes.NewReader(body))
	req = withClaims(req, &model.Claims{UserID: 1, Username: "admin", Role: "admin"})
	w := httptest.NewRecorder()

	h.createExpense(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400, body: %s", w.Code, w.Body.String())
	}
}

func TestCreateExpense_WithOpenRegister_CreatesCashOutMovement(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	execSeed(t, db, `INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 0, 'open')`)

	body, _ := json.Marshal(map[string]any{"category": "servicios", "amount": 80000, "note": "Luz"})
	req := httptest.NewRequest(http.MethodPost, "/api/expenses", bytes.NewReader(body))
	req = withClaims(req, &model.Claims{UserID: 1, Username: "admin", Role: "admin"})
	w := httptest.NewRecorder()

	h.createExpense(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201, body: %s", w.Code, w.Body.String())
	}

	var count int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM register_movements WHERE type = 'cash_out' AND register_id = 1`).Scan(&count); err != nil {
		t.Fatalf("count movements: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 cash_out movement, got %d", count)
	}
}

func TestListExpenses_FilterByCategory(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	execSeed(t, db, `INSERT INTO expenses (user_id, category, amount, note) VALUES (1, 'servicios', 80000, '')`)
	execSeed(t, db, `INSERT INTO expenses (user_id, category, amount, note) VALUES (1, 'arriendo', 500000, '')`)

	req := httptest.NewRequest(http.MethodGet, "/api/expenses?category=servicios", nil)
	w := httptest.NewRecorder()

	h.listExpenses(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body: %s", w.Code, w.Body.String())
	}

	var envelope struct {
		Data struct {
			Expenses []model.Expense `json:"expenses"`
			Total    float64         `json:"total"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(envelope.Data.Expenses) != 1 {
		t.Fatalf("expenses = %d, want 1", len(envelope.Data.Expenses))
	}
	if envelope.Data.Total != 80000 {
		t.Errorf("total = %.0f, want 80000", envelope.Data.Total)
	}
}
