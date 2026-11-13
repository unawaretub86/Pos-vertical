package handler

// US-104 contract §4, T-5 and T-6: the server already accepts a close with a
// faltante (shortage) and stores the negative difference; these tests lock
// that behavior (AC-4) and the surplus/exact paths (AC-8).
//
// Fixture: newShiftEnv (file-backed DB under t.TempDir()), never ":memory:",
// because POST /registers/{id}/close starts the email goroutine (R-115).

import (
	"encoding/json"
	"net/http"
	"testing"
)

const registerAlreadyClosedMsg = "la caja ya está cerrada"

// openWithCashSale opens the register with $100,000 and sells Arroz for
// $30,000 in cash, as the cashier: expected cash at close is $130,000.
func (e *shiftEnv) openWithCashSale(t *testing.T) {
	t.Helper()
	e.openRegister(t, 100000)
	e.postSale(t, 1, 30000, `[{"method":"cash","amount":30000}]`, 0)
}

// closeAs posts the close of register 1 with actualClose as the cashier,
// expects 200 and returns the decoded summary.
func (e *shiftEnv) closeAs(t *testing.T, body string) shiftSummaryJSON {
	t.Helper()
	raw := e.expectStatus(t, http.MethodPost, "/registers/1/close", e.cashier, body, http.StatusOK)
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("decode close envelope: %v; body %s", err, raw)
	}
	var s shiftSummaryJSON
	if err := json.Unmarshal(env.Data, &s); err != nil {
		t.Fatalf("decode close data: %v; body %s", err, raw)
	}
	return s
}

// T-5 (AC-4): cashier closes $125,000 against $130,000 expected → 200,
// difference −5,000 stored; second close → 409; no audit row.
func TestRegisterShortage_T5_CashierClosesWithFaltante(t *testing.T) {
	e := newShiftEnv(t)
	e.openWithCashSale(t)

	closed := e.closeAs(t, `{"actual_close":125000}`)
	if closed.Expected != 130000 {
		t.Errorf("close response expected = %.0f, want 130000", closed.Expected)
	}
	if closed.Register.Difference == nil || *closed.Register.Difference != -5000 {
		t.Errorf("close response register.difference = %v, want -5000", closed.Register.Difference)
	}

	r := e.storedRegister(t, 1)
	if r.status != "closed" {
		t.Errorf("register 1 status = %q, want closed", r.status)
	}
	assertNullFloat(t, "register 1 expected_close", r.expectedClose, 130000)
	assertNullFloat(t, "register 1 actual_close", r.actualClose, 125000)
	assertNullFloat(t, "register 1 difference", r.difference, -5000)

	w := abonoRequest(t, e.h, http.MethodPost, "/registers/1/close", e.cashier, `{"actual_close":125000}`)
	if w.Code != http.StatusConflict {
		t.Errorf("second close = %d, want 409; body %s", w.Code, w.Body.String())
	} else if msg := abonoError(t, w); msg != registerAlreadyClosedMsg {
		t.Errorf("second close error = %q, want %q", msg, registerAlreadyClosedMsg)
	}
	r = e.storedRegister(t, 1)
	assertNullFloat(t, "register 1 difference after second close", r.difference, -5000)

	if n := e.num(t, `SELECT COUNT(*) FROM audit_log`); n != 0 {
		t.Errorf("audit_log rows = %.0f, want 0", n)
	}
}

// T-6 (AC-8): fresh environments; $131,000 counted → difference 1,000;
// $130,000 counted → difference 0.
func TestRegisterShortage_T6_SurplusAndExactClose(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		actual   float64
		wantDiff float64
	}{
		{"surplus", `{"actual_close":131000}`, 131000, 1000},
		{"exact", `{"actual_close":130000}`, 130000, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newShiftEnv(t)
			e.openWithCashSale(t)

			closed := e.closeAs(t, tc.body)
			if closed.Expected != 130000 {
				t.Errorf("close response expected = %.0f, want 130000", closed.Expected)
			}
			r := e.storedRegister(t, 1)
			if r.status != "closed" {
				t.Errorf("register 1 status = %q, want closed", r.status)
			}
			assertNullFloat(t, "register 1 actual_close", r.actualClose, tc.actual)
			assertNullFloat(t, "register 1 difference", r.difference, tc.wantDiff)
		})
	}
}
