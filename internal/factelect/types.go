// Package factelect handles DIAN electronic document emission via a technology provider (Factus by default).
// It is vertical-agnostic: POS and Taller use the same package via SaleAdapter.
package factelect

import "time"

// BillRequest is the contract between any vertical and Factus.
// Built via SaleAdapter.ToBillRequest().
type BillRequest struct {
	Issuer         IssuerInfo
	Buyer          BuyerInfo
	Items          []BillItem
	Totals         BillTotals
	PaymentMethods []PaymentMethod // MVP: always 1 element
	IssuedAt       time.Time       // UTC-5 (América/Bogotá)
	Notes          string
	Environment    string            // "test" | "production"
	Extra          map[string]string // vertical-specific free fields
}

// IssuerInfo holds the issuing business data (from fiscal_config + business_config).
type IssuerInfo struct {
	NIT              string
	RazonSocial      string
	Direccion        string
	CIIU             string
	SoftwareID       string
	SoftwarePIN      string
	Resolucion       string
	Prefijo          string
	NumberingRangeID int
	MunicipalityID   int
}

// BuyerInfo holds buyer data. IsAnonymous=true → "consumidor final".
type BuyerInfo struct {
	IsAnonymous    bool
	DocumentType   string // CC | NIT | CE | TI | PASAPORTE
	DocumentNumber string
	Name           string
	Email          string
	TaxRegime      string // "simplified" | "regular"
}

// BillItem is one line of the equivalent document.
type BillItem struct {
	Description  string
	ProductCode  string
	UnitCodeDIAN string  // DIAN code: "94"=unit, "KGM"=kg, "MTR"=meter, etc.
	Quantity     float64
	UnitPrice    float64
	Discount     float64
	TaxRate      float64 // 0, 5 or 19
	TaxAmount    float64
	Subtotal     float64
	ProductType  string // "01"=merchandise, "04"=service
}

// BillTotals holds consolidated document totals.
type BillTotals struct {
	Subtotal float64
	Discount float64
	TaxBase  float64
	TotalIVA float64
	Total    float64
}

// PaymentMethod is a payment form with DIAN code.
// Codes: "10"=cash, "42"=debit card, "48"=credit card, "20"=transfer.
type PaymentMethod struct {
	Method string
	Amount float64
}

// BillResponse is the provider response when creating a document.
type BillResponse struct {
	ProviderID string
	Number     string
	CUDE       string
	Status     string
	PDFURL     string
	XMLURL     string
	Message    string
}

// StatusResponse is the response when querying document status.
type StatusResponse struct {
	ProviderID      string
	Status          string // "draft"|"pending"|"accepted"|"rejected"
	CUDE            string
	PDFURL          string
	XMLURL          string
	DIANResponse    string
	RejectionReason string
}

// CreditNoteRequest to emit a credit note referencing an accepted DEE.
type CreditNoteRequest struct {
	OriginalCUDE string
	OriginalID   string
	Motive       string // "DEVOLUCIÓN_DINERO" | "DEVOLUCIÓN_PARCIAL" | "REBAJA_DESCUENTO" | "AJUSTE_PRECIO"
	Issuer       IssuerInfo
	Buyer        BuyerInfo
	Items        []BillItem
	Totals       BillTotals
	IssuedAt     time.Time
}
