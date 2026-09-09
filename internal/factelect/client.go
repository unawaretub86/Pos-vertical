package factelect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const factusBaseURL = "https://api.factus.com.co"
const factusSandboxURL = "https://api-sandbox.factus.com.co"

// Client is the Factus HTTP client with cached OAuth2 token.
type Client struct {
	mu           sync.Mutex
	httpClient   *http.Client
	baseURL      string
	clientID     string
	clientSecret string
	token        string
	tokenExpiry  time.Time
}

// NewClient creates a Factus client.
// ambiente: "test" → sandbox, "production" → production.
func NewClient(clientID, clientSecret, ambiente string) *Client {
	base := factusSandboxURL
	if ambiente == "production" {
		base = factusBaseURL
	}
	return &Client{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		baseURL:      base,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (c *Client) ensureToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Now().Before(c.tokenExpiry.Add(-30 * time.Second)) {
		return nil
	}
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/oauth/token",
		bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("factus oauth: %w", err)
	}
	defer resp.Body.Close()
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return fmt.Errorf("factus oauth decode: %w", err)
	}
	c.token = tok.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return nil
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	if err := c.ensureToken(ctx); err != nil {
		return err
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("factus POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		rawErr, _ := io.ReadAll(resp.Body)
		slog.Error("factus error response", "path", path, "status", resp.StatusCode, "body", string(rawErr))
		return fmt.Errorf("factus %s status %d: %s", path, resp.StatusCode, rawErr)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	if err := c.ensureToken(ctx); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("factus GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return fmt.Errorf("factus GET %s: not found", path)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("factus GET %s: status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// CreateBill emits an equivalent document in Factus.
func (c *Client) CreateBill(ctx context.Context, req *BillRequest) (*BillResponse, error) {
	payload := billRequestToFactus(req)
	var raw map[string]any
	if err := c.post(ctx, "/api/v1/bills", payload, &raw); err != nil {
		return nil, err
	}
	return parseFactusBillResponse(raw), nil
}

// GetBillStatus queries the status of a document by its Factus ID.
func (c *Client) GetBillStatus(ctx context.Context, providerID string) (*StatusResponse, error) {
	var raw map[string]any
	if err := c.get(ctx, "/api/v1/bills/"+providerID, &raw); err != nil {
		return nil, err
	}
	return parseFactusStatusResponse(raw), nil
}

// CreateCreditNote emits a credit note in Factus.
func (c *Client) CreateCreditNote(ctx context.Context, req *CreditNoteRequest) (*BillResponse, error) {
	payload := creditNoteToFactus(req)
	var raw map[string]any
	if err := c.post(ctx, "/api/v1/credit-notes", payload, &raw); err != nil {
		return nil, err
	}
	return parseFactusBillResponse(raw), nil
}

// ── Payload mapping ───────────────────────────────────────────────────────────
// Verified fields (developers.factus.com.co):
//   numbering_range_id  int       — from fiscal_config; must be non-zero
//   tribute_id          int       — 1=IVA19%, 4=IVA5%, 22=Excluido/Exento
//   unit_measure_id     int       — 70=Unidad (UND)
//   identification_document_id int — 3=CC, 5=CE, 6=NIT, 7=TI, 10=PASAPORTE
//   payment_method_code string    — "10"=Efectivo, "48"=Transferencia, "20"=Cheque
//   payment_form        int       — 1=Contado, 2=Crédito

func billRequestToFactus(req *BillRequest) map[string]any {
	items := make([]map[string]any, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, map[string]any{
			"code_reference":  it.ProductCode,
			"name":            it.Description,
			"quantity":        it.Quantity,
			"discount_rate":   discountRate(it),
			"price":           it.UnitPrice,
			"tax_rate":        it.TaxRate,
			"unit_measure_id": it.UnitCodeDIAN,
			"standard_code":   it.ProductCode,
			"is_excluded":     it.TaxRate == 0,
			"tribute_id":      tributeID(it.TaxRate),
		})
	}
	payments := make([]map[string]any, 0, len(req.PaymentMethods))
	for _, pm := range req.PaymentMethods {
		payments = append(payments, map[string]any{
			"payment_method_code": pm.Method,
			"amount":              pm.Amount,
		})
	}
	buyer := map[string]any{
		"identification":              req.Buyer.DocumentNumber,
		"dv":                          "",
		"company":                     req.Buyer.Name,
		"trade_name":                  req.Buyer.Name,
		"names":                       req.Buyer.Name,
		"address":                     "",
		"email":                       req.Buyer.Email,
		"phone":                       "",
		"identification_document_id":  documentTypeID(req.Buyer.DocumentType),
		"municipality_id":             req.Issuer.MunicipalityID,
	}
	if req.Buyer.IsAnonymous {
		buyer["identification"] = "222222222222"
		buyer["company"] = "Consumidor Final"
		buyer["names"] = "Consumidor Final"
		buyer["identification_document_id"] = 3
	}
	return map[string]any{
		"numbering_range_id": req.Issuer.NumberingRangeID,
		"reference_code":     "",
		"observation":        req.Notes,
		"payment_form":       1,
		"payment_due_date":   req.IssuedAt.Format("2006-01-02"),
		"buyer":              buyer,
		"items":              items,
		"payment_methods":    payments,
	}
}

func creditNoteToFactus(req *CreditNoteRequest) map[string]any {
	base := billRequestToFactus(&BillRequest{
		Issuer:   req.Issuer,
		Buyer:    req.Buyer,
		Items:    req.Items,
		Totals:   req.Totals,
		IssuedAt: req.IssuedAt,
	})
	base["reference_id"] = req.OriginalID
	base["concept_id"] = conceptID(req.Motive)
	return base
}

func parseFactusBillResponse(raw map[string]any) *BillResponse {
	resp := &BillResponse{}
	if data, ok := raw["data"].(map[string]any); ok {
		resp.ProviderID = strVal(data["id"])
		resp.Number = strVal(data["number"])
		resp.CUDE = strVal(data["cude"])
		resp.Status = strVal(data["status"])
		if links, ok := data["links"].(map[string]any); ok {
			resp.PDFURL = strVal(links["pdf"])
			resp.XMLURL = strVal(links["xml"])
		}
	}
	return resp
}

func parseFactusStatusResponse(raw map[string]any) *StatusResponse {
	resp := &StatusResponse{}
	if data, ok := raw["data"].(map[string]any); ok {
		resp.ProviderID = strVal(data["id"])
		resp.Status = strVal(data["status"])
		resp.CUDE = strVal(data["cude"])
	}
	return resp
}

func tributeID(taxRate float64) int {
	switch taxRate {
	case 19:
		return 1
	case 5:
		return 4
	default:
		return 22
	}
}

func documentTypeID(docType string) int {
	m := map[string]int{"CC": 3, "NIT": 6, "CE": 5, "TI": 7, "PASAPORTE": 10}
	if id, ok := m[docType]; ok {
		return id
	}
	return 3
}

func conceptID(motive string) int {
	m := map[string]int{
		"DEVOLUCIÓN_PARCIAL": 1,
		"DEVOLUCIÓN_DINERO":  2,
		"REBAJA_DESCUENTO":   3,
		"AJUSTE_PRECIO":      4,
	}
	if id, ok := m[motive]; ok {
		return id
	}
	return 2
}

func discountRate(it BillItem) float64 {
	if it.UnitPrice == 0 {
		return 0
	}
	return it.Discount / (it.UnitPrice * it.Quantity) * 100
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case fmt.Stringer:
		return s.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
