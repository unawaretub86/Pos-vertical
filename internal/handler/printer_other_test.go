//go:build !windows

package handler

// US-080 contract section 4, T-11 (AC-8): the Windows queue method is unsupported off Windows.

import (
	"net/http"
	"testing"
)

func TestPrintTicket_T11_WindowsMethodOffWindowsReturns501(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	seedPrintData(t, db, "escpos_windows", "", 9100, "POS-58")

	w := postPrint(h, tokenFor(t, 2, "cashier"), "/printer/print-ticket/1")
	if w.Code != http.StatusNotImplemented {
		t.Errorf("print-ticket: status %d, want 501; body %s", w.Code, w.Body.String())
	}
	if errorMessage(t, w) == "" {
		t.Errorf("501 must carry a Spanish message")
	}

	w = postPrint(h, tokenFor(t, 1, "admin"), "/printer/test")
	if w.Code != http.StatusNotImplemented {
		t.Errorf("test: status %d, want 501; body %s", w.Code, w.Body.String())
	}
}
