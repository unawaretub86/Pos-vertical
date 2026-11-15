package handler

// US-105 contract §4, T-1 .. T-7: GET /customers is shaped by role (the
// cashier gets names and phones, never balances or total_portfolio), the
// per-customer endpoints the POS needs stay open to her, and the customer
// detail moves under RequireAdmin. Fixture: seedCustomersT5 (Dora $30.000,
// Eli $15.000, Fer $0, Gus -$20.000) and a cashier user id 2. Requests go
// through h.Routes() with real JWTs (tokenFor, abonoRequest).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

const roleCashierID = 2

// seedRoleFixture seeds seedCustomersT5 plus the cashier user id 2.
func seedRoleFixture(t *testing.T) (*database.DB, *Handler) {
	t.Helper()
	db := setupTestDB(t)
	h := seedCustomersT5(t, db)
	abonoExec(t, db, `INSERT INTO users (id, username, password_hash, role) VALUES (2, 'cajero1', 'h', 'cashier')`)
	return db, h
}

// listData decodes the data envelope of GET /customers into the raw keys of
// data and the raw rows of data.customers.
func listData(t *testing.T, w *httptest.ResponseRecorder) (map[string]json.RawMessage, []map[string]json.RawMessage) {
	t.Helper()
	var data map[string]json.RawMessage
	abonoData(t, w, &data)
	var rows []map[string]json.RawMessage
	if raw, ok := data["customers"]; ok {
		if err := json.Unmarshal(raw, &rows); err != nil {
			t.Fatalf("decode customers: %v; body %s", err, w.Body.String())
		}
	}
	return data, rows
}

func sortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func rowName(t *testing.T, row map[string]json.RawMessage) string {
	t.Helper()
	var name string
	if err := json.Unmarshal(row["name"], &name); err != nil {
		t.Fatalf("decode name: %v", err)
	}
	return name
}

func rowNames(t *testing.T, rows []map[string]json.RawMessage) []string {
	t.Helper()
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, rowName(t, r))
	}
	return names
}

// assertCashierShape checks AC-1's shape: data == {customers}, every row has
// exactly the model.Customer keys, and no balance data anywhere in the body.
func assertCashierShape(t *testing.T, w *httptest.ResponseRecorder, wantNames []string) {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
	}
	data, rows := listData(t, w)
	if got := sortedKeys(data); strings.Join(got, ",") != "customers" {
		t.Errorf("data keys = %v, want [customers]", got)
	}
	if got := rowNames(t, rows); strings.Join(got, ",") != strings.Join(wantNames, ",") {
		t.Errorf("rows = %v, want %v", got, wantNames)
	}
	wantKeys := "address,created_at,credit_limit,id,name,notes,phone"
	for i, r := range rows {
		if got := strings.Join(sortedKeys(r), ","); got != wantKeys {
			t.Errorf("row %d keys = %s, want %s", i, got, wantKeys)
		}
	}
	body := w.Body.String()
	for _, leak := range []string{`"balance"`, `"total_portfolio"`, `"oldest_debt_at"`} {
		if strings.Contains(body, leak) {
			t.Errorf("body contains %s; body %s", leak, body)
		}
	}
}

// ─── T-1: AC-1 ───────────────────────────────────────────────────────────────

func TestCustomerRole_T1_CashierListHasNoBalances(t *testing.T) {
	_, h := seedRoleFixture(t)
	cashier := tokenFor(t, roleCashierID, "cashier")

	w := abonoRequest(t, h, http.MethodGet, "/customers", cashier, "")
	assertCashierShape(t, w, []string{"Dora", "Eli", "Fer", "Gus"})
}

// ─── T-2: AC-2 ───────────────────────────────────────────────────────────────

func TestCustomerRole_T2_CashierIgnoresHasBalanceButKeepsSearch(t *testing.T) {
	_, h := seedRoleFixture(t)
	cashier := tokenFor(t, roleCashierID, "cashier")

	// has_balance=true would return Dora and Eli (2 rows); for a cashier it is
	// ignored and all 4 come back.
	w := abonoRequest(t, h, http.MethodGet, "/customers?has_balance=true", cashier, "")
	assertCashierShape(t, w, []string{"Dora", "Eli", "Fer", "Gus"})

	w = abonoRequest(t, h, http.MethodGet, "/customers?q=Do", cashier, "")
	assertCashierShape(t, w, []string{"Dora"})
}

// ─── T-3: AC-3 ───────────────────────────────────────────────────────────────

func TestCustomerRole_T3_AdminListUnchanged(t *testing.T) {
	_, h := seedRoleFixture(t)
	admin := tokenFor(t, 1, "admin")

	type row struct {
		ID           int64   `json:"id"`
		Name         string  `json:"name"`
		Balance      float64 `json:"balance"`
		OldestDebtAt *string `json:"oldest_debt_at"`
	}
	var list struct {
		Customers      []row    `json:"customers"`
		TotalPortfolio *float64 `json:"total_portfolio"`
	}

	w := abonoRequest(t, h, http.MethodGet, "/customers", admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("admin GET /customers = %d, want 200; body %s", w.Code, w.Body.String())
	}
	abonoData(t, w, &list)
	if len(list.Customers) != 4 {
		t.Fatalf("admin rows = %d, want 4; body %s", len(list.Customers), w.Body.String())
	}
	want := map[string]float64{"Dora": 30000, "Eli": 15000, "Fer": 0, "Gus": -20000}
	for _, c := range list.Customers {
		if c.Balance != want[c.Name] {
			t.Errorf("%s balance = %v, want %v", c.Name, c.Balance, want[c.Name])
		}
	}
	if list.TotalPortfolio == nil || *list.TotalPortfolio != 45000 {
		t.Errorf("total_portfolio = %v, want 45000", list.TotalPortfolio)
	}
	if !strings.Contains(w.Body.String(), `"oldest_debt_at"`) {
		t.Errorf("admin body has no oldest_debt_at; body %s", w.Body.String())
	}

	list.Customers = nil
	w = abonoRequest(t, h, http.MethodGet, "/customers?has_balance=true", admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("admin has_balance = %d, want 200; body %s", w.Code, w.Body.String())
	}
	abonoData(t, w, &list)
	if len(list.Customers) != 2 || list.Customers[0].ID != 1 || list.Customers[1].ID != 2 {
		t.Errorf("admin has_balance rows = %+v, want ids [1 2]", list.Customers)
	}
}

// ─── T-4: AC-4 ───────────────────────────────────────────────────────────────

func TestCustomerRole_T4_NoClaimsFailsClosed(t *testing.T) {
	_, h := seedRoleFixture(t)

	w := httptest.NewRecorder()
	h.listCustomers(w, httptest.NewRequest(http.MethodGet, "/api/customers", nil))
	assertCashierShape(t, w, []string{"Dora", "Eli", "Fer", "Gus"})
}

// ─── T-5: AC-5 ───────────────────────────────────────────────────────────────

func TestCustomerRole_T5_CashierKeepsPerCustomerBalance(t *testing.T) {
	_, h := seedRoleFixture(t)
	cashier := tokenFor(t, roleCashierID, "cashier")

	w := abonoRequest(t, h, http.MethodGet, "/customers/1/pos-summary", cashier, "")
	if w.Code != http.StatusOK {
		t.Fatalf("cashier pos-summary = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var summary struct {
		Balance *float64 `json:"balance"`
	}
	abonoData(t, w, &summary)
	if summary.Balance == nil || *summary.Balance != 30000 {
		t.Errorf("pos-summary balance = %v, want 30000; body %s", summary.Balance, w.Body.String())
	}

	w = abonoRequest(t, h, http.MethodGet, "/customers/1/balance", cashier, "")
	if w.Code != http.StatusOK {
		t.Fatalf("cashier balance = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var bal map[string]float64
	abonoData(t, w, &bal)
	if len(bal) != 1 || bal["balance"] != 30000 {
		t.Errorf(`balance data = %v, want {"balance":30000}`, bal)
	}
}

// ─── T-6: AC-7 ───────────────────────────────────────────────────────────────

func TestCustomerRole_T6_DetailIsAdminOnly(t *testing.T) {
	_, h := seedRoleFixture(t)
	cashier := tokenFor(t, roleCashierID, "cashier")
	admin := tokenFor(t, 1, "admin")

	w := abonoRequest(t, h, http.MethodGet, "/customers/1/detail", cashier, "")
	if w.Code != http.StatusForbidden {
		t.Errorf("cashier detail = %d, want 403; body %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"balance"`) {
		t.Errorf("cashier detail body leaks balance; body %s", w.Body.String())
	}

	w = abonoRequest(t, h, http.MethodGet, "/customers/1/detail", admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("admin detail = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var detail struct {
		Balance *float64 `json:"balance"`
	}
	abonoData(t, w, &detail)
	if detail.Balance == nil || *detail.Balance != 30000 {
		t.Errorf("admin detail balance = %v, want 30000", detail.Balance)
	}
}

// ─── T-7: AC-13 ──────────────────────────────────────────────────────────────

func TestCustomerRole_T7_CashierAbonoFromPOSLowersBalance(t *testing.T) {
	db, h := seedRoleFixture(t)
	abonoExec(t, db, `INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 2, 50000, 'open')`)
	cashier := tokenFor(t, roleCashierID, "cashier")

	w := abonoRequest(t, h, http.MethodPost, "/customers/1/payments", cashier, `{"amount":10000,"method":"cash"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("cashier abono = %d, want 201; body %s", w.Code, w.Body.String())
	}
	var created paymentJSON
	abonoData(t, w, &created)
	assertPayment(t, "cashier POS abono", created, 10000, "cash", "1")

	w = abonoRequest(t, h, http.MethodGet, "/customers/1/balance", cashier, "")
	var bal map[string]float64
	abonoData(t, w, &bal)
	if bal["balance"] != 20000 {
		t.Errorf("Dora balance after $10.000 abono = %v, want 20000 (30000 - 10000)", bal["balance"])
	}
}
