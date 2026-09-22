package mailer

import (
	"strings"
	"testing"
	"time"
)

func baseData(overrides func(*RegisterSummaryData)) RegisterSummaryData {
	d := RegisterSummaryData{
		BusinessName: "Tienda La Esquina",
		Cashier:      "Laura",
		ClosedAt:     time.Date(2026, 9, 9, 20, 0, 0, 0, time.UTC),
		ByPaymentMethod: []PaymentLine{
			{Method: "cash", Total: 30000},
			{Method: "transfer", Total: 20000},
			{Method: "credit", Total: 10000},
		},
		CashIn:      0,
		CashOut:     5000,
		Expected:    25000,
		ActualClose: 24500,
		Difference:  -500,
	}
	if overrides != nil {
		overrides(&d)
	}
	return d
}

func TestBuildRegisterSummaryEmail_Shortfall(t *testing.T) {
	_, body := BuildRegisterSummaryEmail(baseData(nil))

	for _, want := range []string{
		"Faltante: $500",
		"Efectivo: $30.000",
		"Transferencia: $20.000",
		"Crédito: $10.000",
		"Entradas: $0",
		"Salidas: $5.000",
		"Esperado: $25.000",
		"Real: $24.500",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q; got:\n%s", want, body)
		}
	}
	if strings.Contains(body, "Diferencia") {
		t.Errorf("shortfall body should not contain \"Diferencia\"; got:\n%s", body)
	}
}

func TestBuildRegisterSummaryEmail_ExactMatch(t *testing.T) {
	_, body := BuildRegisterSummaryEmail(baseData(func(d *RegisterSummaryData) {
		d.ActualClose = 25000
		d.Difference = 0
	}))

	if !strings.Contains(body, "Diferencia: $0") {
		t.Errorf("expected \"Diferencia: $0\"; got:\n%s", body)
	}
	if strings.Contains(body, "Faltante") {
		t.Errorf("exact-match body should not contain \"Faltante\"; got:\n%s", body)
	}
}
