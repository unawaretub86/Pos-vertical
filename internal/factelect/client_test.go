package factelect

import (
	"testing"
	"time"
)

func TestBillRequestToFactus_NumberingRangeAndMunicipality(t *testing.T) {
	req := &BillRequest{
		Issuer: IssuerInfo{
			NIT:              "900123456",
			RazonSocial:      "Tienda Test",
			NumberingRangeID: 5,
			MunicipalityID:   250,
		},
		Buyer: BuyerInfo{
			IsAnonymous: true,
		},
		Items: []BillItem{
			{
				Description:  "Producto",
				UnitCodeDIAN: "94",
				Quantity:     1,
				UnitPrice:    10000,
				TaxRate:      0,
				Subtotal:     10000,
			},
		},
		Totals: BillTotals{
			Subtotal: 10000,
			Total:    10000,
		},
		PaymentMethods: []PaymentMethod{{Method: "10", Amount: 10000}},
		IssuedAt:       time.Now(),
	}

	payload := billRequestToFactus(req)

	if got, ok := payload["numbering_range_id"].(int); !ok || got != 5 {
		t.Errorf("numbering_range_id = %v, want 5", payload["numbering_range_id"])
	}

	buyer, ok := payload["buyer"].(map[string]any)
	if !ok {
		t.Fatal("buyer not a map")
	}
	if got, ok := buyer["municipality_id"].(int); !ok || got != 250 {
		t.Errorf("municipality_id = %v, want 250", buyer["municipality_id"])
	}
}

func TestBillRequestToFactus_PaymentFormIsInt(t *testing.T) {
	req := &BillRequest{
		Issuer:         IssuerInfo{NumberingRangeID: 1, MunicipalityID: 149},
		Buyer:          BuyerInfo{IsAnonymous: true},
		Items:          []BillItem{{Description: "x", UnitCodeDIAN: "94", Quantity: 1, UnitPrice: 1000, Subtotal: 1000}},
		Totals:         BillTotals{Subtotal: 1000, Total: 1000},
		PaymentMethods: []PaymentMethod{{Method: "10", Amount: 1000}},
		IssuedAt:       time.Now(),
	}
	payload := billRequestToFactus(req)
	pf, ok := payload["payment_form"].(int)
	if !ok || pf != 1 {
		t.Errorf("payment_form must be int 1, got %T(%v)", payload["payment_form"], payload["payment_form"])
	}
}

func TestBillRequestToFactus_NoHardcodedZeroOrBogota(t *testing.T) {
	req := &BillRequest{
		Issuer: IssuerInfo{
			NumberingRangeID: 7,
			MunicipalityID:   300,
		},
		Buyer:          BuyerInfo{IsAnonymous: true},
		Items:          []BillItem{{Description: "x", UnitCodeDIAN: "94", Quantity: 1, UnitPrice: 1000, Subtotal: 1000}},
		Totals:         BillTotals{Subtotal: 1000, Total: 1000},
		PaymentMethods: []PaymentMethod{{Method: "10", Amount: 1000}},
		IssuedAt:       time.Now(),
	}

	payload := billRequestToFactus(req)

	if got := payload["numbering_range_id"]; got == 0 {
		t.Error("numbering_range_id must not be hardcoded 0")
	}
	buyer := payload["buyer"].(map[string]any)
	if got := buyer["municipality_id"]; got == 149 {
		t.Error("municipality_id must not be hardcoded 149 when caller passes 300")
	}
}
