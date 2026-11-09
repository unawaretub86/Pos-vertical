package handler

// US-095 contract §4, T-6 .. T-8: lock the server behavior the dashboard void
// UI relies on (AC-4, AC-6, AC-11). Test-only: no production Go change.
// Fixture: newShiftEnv (file-backed DB, see shift_certification_test.go).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

type saleVoidUIDashboardJSON struct {
	Register  *shiftSummaryJSON `json:"register"`
	Portfolio struct {
		Total float64 `json:"total"`
		Count int     `json:"count"`
	} `json:"portfolio"`
	RecentSales []struct {
		ID           int64  `json:"id"`
		CustomerName string `json:"customer_name"`
		Status       string `json:"status"`
	} `json:"recent_sales"`
}

type saleVoidUISaleJSON struct {
	ID       int64  `json:"id"`
	Status   string `json:"status"`
	Payments []struct {
		Method string  `json:"method"`
		Amount float64 `json:"amount"`
	} `json:"payments"`
}

func (e *shiftEnv) saleVoidUIDashboard(t *testing.T) saleVoidUIDashboardJSON {
	t.Helper()
	w := abonoRequest(t, e.h, http.MethodGet, "/dashboard", e.admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /dashboard = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var d saleVoidUIDashboardJSON
	abonoData(t, w, &d)
	return d
}

func (e *shiftEnv) saleVoidUISale(t *testing.T, id int64) saleVoidUISaleJSON {
	t.Helper()
	w := abonoRequest(t, e.h, http.MethodGet, fmt.Sprintf("/sales/%d", id), e.admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /sales/%d = %d, want 200; body %s", id, w.Code, w.Body.String())
	}
	var s saleVoidUISaleJSON
	abonoData(t, w, &s)
	return s
}

func saleVoidUIRecentIDs(d saleVoidUIDashboardJSON) []int64 {
	ids := make([]int64, 0, len(d.RecentSales))
	for _, r := range d.RecentSales {
		ids = append(ids, r.ID)
	}
	return ids
}

func saleVoidUIEqualIDs(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ─── T-6: AC-11 ──────────────────────────────────────────────────────────────

func TestSaleVoidUI_T6_ErrorBodies(t *testing.T) {
	e := newShiftEnv(t)
	e.openRegister(t, 100000)
	e.postSale(t, 1, 30000, `[{"method":"cash","amount":30000}]`, 0)
	abonoExec(t, e.db, `INSERT INTO electronic_documents (id, sale_id, status, cude) VALUES ('doc-1', 1, 'accepted', 'CUDE123')`)

	assert409 := func(where string) {
		t.Helper()
		w := abonoRequest(t, e.h, http.MethodDelete, "/sales/1/void", e.admin, "")
		if w.Code != http.StatusConflict {
			t.Fatalf("%s: DELETE /sales/1/void = %d, want 409; body %s", where, w.Code, w.Body.String())
		}
		var body struct {
			Error              string `json:"error"`
			DocumentID         string `json:"document_id"`
			RequiresCreditNote bool   `json:"requires_credit_note"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: decode 409 body: %v; body %s", where, err, w.Body.String())
		}
		if body.Error != "sale_has_accepted_invoice" {
			t.Errorf("%s: 409 error = %q, want sale_has_accepted_invoice", where, body.Error)
		}
		if !body.RequiresCreditNote {
			t.Errorf("%s: 409 requires_credit_note = false, want true", where)
		}
		if body.DocumentID != "doc-1" {
			t.Errorf("%s: 409 document_id = %q, want doc-1", where, body.DocumentID)
		}
		if got := e.saleVoidUISale(t, 1).Status; got != "completed" {
			t.Errorf("%s: sale 1 status = %q, want completed", where, got)
		}
		if got := e.stock(t, 1); got != 9 {
			t.Errorf("%s: Arroz stock = %.0f, want 9", where, got)
		}
	}

	assert409("accepted document")

	abonoExec(t, e.db, `UPDATE electronic_documents SET status = 'pending' WHERE id = 'doc-1'`)
	assert409("pending document")

	abonoExec(t, e.db, `UPDATE electronic_documents SET status = 'rejected' WHERE id = 'doc-1'`)
	w := abonoRequest(t, e.h, http.MethodDelete, "/sales/1/void", e.admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("rejected document: DELETE /sales/1/void = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var voided saleVoidUISaleJSON
	abonoData(t, w, &voided)
	if voided.Status != "voided" {
		t.Errorf("void response status = %q, want voided", voided.Status)
	}
	if got := e.stock(t, 1); got != 10 {
		t.Errorf("Arroz stock after void = %.0f, want 10", got)
	}

	w = abonoRequest(t, e.h, http.MethodDelete, "/sales/1/void", e.admin, "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("second void = %d, want 400; body %s", w.Code, w.Body.String())
	} else if msg := abonoError(t, w); msg != "la venta ya fue anulada" {
		t.Errorf("second void error = %q, want %q", msg, "la venta ya fue anulada")
	}

	w = abonoRequest(t, e.h, http.MethodDelete, "/sales/99/void", e.admin, "")
	if w.Code != http.StatusNotFound {
		t.Errorf("DELETE /sales/99/void = %d, want 404; body %s", w.Code, w.Body.String())
	} else if msg := abonoError(t, w); msg != "venta no encontrada" {
		t.Errorf("DELETE /sales/99/void error = %q, want %q", msg, "venta no encontrada")
	}

	e.postSale(t, 2, 50000, `[{"method":"cash","amount":50000}]`, 0)
	w = abonoRequest(t, e.h, http.MethodDelete, "/sales/2/void", e.cashier, "")
	if w.Code != http.StatusForbidden {
		t.Errorf("cashier DELETE /sales/2/void = %d, want 403; body %s", w.Code, w.Body.String())
	} else if msg := abonoError(t, w); msg != "forbidden" {
		t.Errorf("cashier void error = %q, want forbidden; body %s", msg, w.Body.String())
	}
	if got := e.saleVoidUISale(t, 2).Status; got != "completed" {
		t.Errorf("sale 2 status after cashier attempt = %q, want completed", got)
	}
}

// ─── T-7: AC-6, AC-11 ────────────────────────────────────────────────────────

func TestSaleVoidUI_T7_DashboardReflectsVoid(t *testing.T) {
	e := newShiftEnv(t)
	e.openRegister(t, 100000)
	e.postSale(t, 1, 30000, `[{"method":"cash","amount":30000}]`, 0)
	e.postSale(t, 2, 50000, `[{"method":"credit","amount":50000}]`, 1)
	e.postSale(t, 3, 12000, `[{"method":"cash","amount":7000},{"method":"credit","amount":5000}]`, 1)
	// recent_sales orders by created_at DESC, which has 1-second resolution;
	// the three sales above share one second, so space them out to make the
	// expected order [3,2,1] deterministic.
	abonoExec(t, e.db, `UPDATE sales SET created_at = datetime(created_at, '-' || (4 - id) || ' seconds')`)

	d := e.saleVoidUIDashboard(t)
	if d.Register == nil {
		t.Fatalf("GET /dashboard register = null, want the open register")
	}
	if d.Register.Expected != 137000 {
		t.Errorf("register.expected before void = %.0f, want 137000", d.Register.Expected)
	}
	if d.Register.Register.ID != 1 {
		t.Errorf("register.register.id = %d, want 1", d.Register.Register.ID)
	}
	if d.Portfolio.Total != 55000 || d.Portfolio.Count != 1 {
		t.Errorf("portfolio before void = %.0f / %d, want 55000 / 1", d.Portfolio.Total, d.Portfolio.Count)
	}
	if ids := saleVoidUIRecentIDs(d); !saleVoidUIEqualIDs(ids, []int64{3, 2, 1}) {
		t.Errorf("recent_sales ids before void = %v, want [3 2 1]", ids)
	} else if d.RecentSales[0].CustomerName != "Dora" {
		t.Errorf("recent_sales #3 customer_name = %q, want Dora", d.RecentSales[0].CustomerName)
	}

	voidSale(t, e.h, e.admin, 3)

	d = e.saleVoidUIDashboard(t)
	if d.Register == nil {
		t.Fatalf("GET /dashboard register after void = null, want the open register")
	}
	if d.Register.Expected != 130000 {
		t.Errorf("register.expected after void = %.0f, want 130000", d.Register.Expected)
	}
	if d.Portfolio.Total != 50000 {
		t.Errorf("portfolio.total after void = %.0f, want 50000", d.Portfolio.Total)
	}
	if ids := saleVoidUIRecentIDs(d); !saleVoidUIEqualIDs(ids, []int64{2, 1}) {
		t.Errorf("recent_sales ids after void = %v, want [2 1]", ids)
	}

	w := abonoRequest(t, e.h, http.MethodGet, "/dashboard/today", e.admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /dashboard/today = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var today struct {
		AgingReceivables []struct {
			Name    string  `json:"name"`
			Balance float64 `json:"balance"`
		} `json:"aging_receivables"`
	}
	abonoData(t, w, &today)
	if len(today.AgingReceivables) != 1 || today.AgingReceivables[0].Name != "Dora" || today.AgingReceivables[0].Balance != 50000 {
		t.Errorf("aging_receivables after void = %+v, want [Dora 50000]", today.AgingReceivables)
	}

	s := e.saleVoidUISale(t, 3)
	if s.Status != "voided" {
		t.Errorf("GET /sales/3 status = %q, want voided", s.Status)
	}
	var cash, credit float64
	for _, p := range s.Payments {
		switch p.Method {
		case "cash":
			cash += p.Amount
		case "credit":
			credit += p.Amount
		default:
			t.Errorf("GET /sales/3 unexpected payment %s %.0f", p.Method, p.Amount)
		}
	}
	if len(s.Payments) != 2 || cash != 7000 || credit != 5000 {
		t.Errorf("GET /sales/3 payments = %+v, want cash 7000 + credit 5000", s.Payments)
	}
	if got := e.stock(t, 3); got != 10 {
		t.Errorf("Panela stock after void = %.0f, want 10", got)
	}
}

// ─── T-8: AC-4, AC-11 ────────────────────────────────────────────────────────

func TestSaleVoidUI_T8_MixedVoidOnClosedRegister(t *testing.T) {
	e := newShiftEnv(t)
	e.openRegister(t, 100000)
	e.postSale(t, 3, 12000, `[{"method":"cash","amount":7000},{"method":"credit","amount":5000}]`, 1)
	e.expectStatus(t, http.MethodPost, "/registers/1/close", e.cashier, `{"actual_close":107000}`, http.StatusOK)
	assertNullFloat(t, "register 1 expected_close at close", e.storedRegister(t, 1).expectedClose, 107000)
	if got := e.balance(t); got != 5000 {
		t.Fatalf("balance before void = %.0f, want 5000", got)
	}
	e.openRegister(t, 50000)

	voidSale(t, e.h, e.admin, 1)

	r1 := e.storedRegister(t, 1)
	assertNullFloat(t, "register 1 expected_close after void", r1.expectedClose, 107000)
	assertNullFloat(t, "register 1 actual_close after void", r1.actualClose, 107000)
	assertNullFloat(t, "register 1 difference after void", r1.difference, 0)
	if got := e.summary(t, 1).Expected; got != 100000 {
		t.Errorf("GET /registers/1 expected after void = %.0f, want 100000", got)
	}

	s2 := e.summary(t, 2)
	if s2.Expected != 50000 {
		t.Errorf("register 2 expected = %.0f, want 50000", s2.Expected)
	}
	if len(s2.Movements) != 1 {
		t.Errorf("register 2 movements = %d (%+v), want 1", len(s2.Movements), s2.Movements)
	}

	if got := e.balance(t); got != 0 {
		t.Errorf("balance after void = %.0f, want 0", got)
	}

	d := e.saleVoidUIDashboard(t)
	if d.Register == nil {
		t.Fatalf("GET /dashboard register = null, want register 2")
	}
	if d.Register.Register.ID != 2 {
		t.Errorf("GET /dashboard register.register.id = %d, want 2", d.Register.Register.ID)
	}
}
