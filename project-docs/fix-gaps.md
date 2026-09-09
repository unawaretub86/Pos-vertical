# Fix Gaps — Facturación Electrónica: Decisiones y Plan de Implementación

> Documento de referencia para implementar `internal/factelect` en POS Vertical.
> Todas las decisiones de arquitectura están aquí. Leer antes de escribir código.
> Fecha: Mayo 2026

---

## Decisiones finales por gap

| Gap | Decisión |
|-----|----------|
| Gap 1 — CIIU faltante en `business_config` | Migration v8: `ALTER TABLE business_config ADD COLUMN ciiu TEXT` |
| Gap 2 — Pagos mixtos | **Bloqueado en MVP**. Un método por venta. `BillRequest.PaymentMethods` = slice de 1 item mapeado del `payment_method` actual. Sin cambio al modelo `Sale`. |
| Gap 3 — Webhook offline | **Polling**. Job cada 30s hace `GET /bills/{provider_id}` a Factus hasta estado final. Sin URL pública. Sin webhook. |
| Gap 4 — Void + nota crédito | `SaleService.Void` verifica `electronic_documents`. Si `accepted/pending` → retorna `*repository.ErrRequiresCreditNote`. Handler responde 409. Nuevo endpoint `POST /sales/{id}/void-with-credit-note`. |
| Gap 5 — Numeración | Factus maneja contador internamente. Solo validar `fiscal_config.fecha_vencimiento` antes de emitir. Si venció → error claro, no llamar a Factus. |
| Gap 6 — `clave_tecnica` | Guardar en `fiscal_config` para auditoría. **NO** enviar a Factus (ellos la tienen del registro de la resolución en su sistema). |
| Gap 7 — Email comprador anónimo | Campo `buyer_email string` (opcional) en el body del `POST /sales`. No persiste en `sales`. Se pasa como argumento a `ToBillRequest`. |
| Gap 8 — Unidad de medida DIAN | Mapa hardcoded en `mapper.go`. Default `"94"` (unidad) para valores desconocidos. |

---

## Árbol de archivos

```
internal/
├── model/model.go              ← MODIFICAR: agregar FiscalConfig, ElectronicDocument
├── repository/
│   ├── repository.go           ← MODIFICAR: agregar ElectronicDocRepository interface
│   ├── errors.go               ← MODIFICAR: agregar ErrRequiresCreditNote
│   └── eledoc.go               ← CREAR: SQLite impl de ElectronicDocRepository
├── database/
│   └── migrate.go              ← MODIFICAR: agregar migration v8
├── factelect/
│   ├── types.go                ← CREAR: BillRequest y todos los DTOs
│   ├── adapter.go              ← CREAR: interface SaleAdapter
│   ├── pos_adapter.go          ← CREAR: PosSaleAdapter (impl para este repo)
│   ├── mapper.go               ← CREAR: unidades DIAN, payment method codes
│   ├── config.go               ← CREAR: FiscalConfigRepository interface + SQLite impl
│   ├── client.go               ← CREAR: Factus HTTP client con OAuth2 token cache
│   ├── service.go              ← CREAR: FacElectService (EmitFromSale, CreateCreditNote)
│   └── poller.go               ← CREAR: PollPending job
├── service/
│   └── sale.go                 ← MODIFICAR: Void() verifica eledoc antes de anular
├── handler/
│   ├── sale.go                 ← MODIFICAR: buyer_email en createSale, 409 en voidSale
│   ├── fiscal.go               ← CREAR: handlers fiscal config + documentos
│   └── handler.go              ← MODIFICAR: wiring FacElectService + nuevas rutas
```

---

## Migration v8

Agregar al final de `internal/database/migrate.go`:

```go
// En dbMigrations slice:
{version: 8, up: migration8},

// Constante SQL:
const migration8 = `
ALTER TABLE business_config ADD COLUMN ciiu TEXT;

ALTER TABLE customers ADD COLUMN email TEXT;
ALTER TABLE customers ADD COLUMN document_type TEXT DEFAULT 'CC';
ALTER TABLE customers ADD COLUMN document_number TEXT;
ALTER TABLE customers ADD COLUMN tax_regime TEXT DEFAULT 'simplified';

CREATE TABLE IF NOT EXISTS fiscal_config (
    id                  INTEGER PRIMARY KEY CHECK (id = 1),
    enabled             INTEGER NOT NULL DEFAULT 0,
    nit                 TEXT NOT NULL DEFAULT '',
    razon_social        TEXT NOT NULL DEFAULT '',
    direccion           TEXT NOT NULL DEFAULT '',
    ciiu                TEXT NOT NULL DEFAULT '',
    resolucion_dian     TEXT NOT NULL DEFAULT '',
    prefijo             TEXT DEFAULT '',
    rango_desde         INTEGER NOT NULL DEFAULT 0,
    rango_hasta         INTEGER NOT NULL DEFAULT 0,
    fecha_resolucion    DATE,
    fecha_vencimiento   DATE,
    clave_tecnica       TEXT NOT NULL DEFAULT '',
    ambiente            TEXT NOT NULL DEFAULT 'test',
    software_id         TEXT NOT NULL DEFAULT '',
    software_pin        TEXT NOT NULL DEFAULT '',
    provider            TEXT NOT NULL DEFAULT 'factus',
    provider_client_id  TEXT NOT NULL DEFAULT '',
    provider_secret     TEXT NOT NULL DEFAULT '',
    webhook_url         TEXT,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT OR IGNORE INTO fiscal_config (id) VALUES (1);

CREATE TABLE IF NOT EXISTS electronic_documents (
    id               TEXT PRIMARY KEY,
    sale_id          INTEGER NOT NULL REFERENCES sales(id),
    type             TEXT NOT NULL DEFAULT 'doc_equivalente',
    status           TEXT NOT NULL DEFAULT 'draft',
    provider         TEXT NOT NULL DEFAULT 'factus',
    provider_id      TEXT,
    cude             TEXT,
    number           TEXT,
    xml_url          TEXT,
    pdf_url          TEXT,
    dian_response    TEXT,
    rejection_reason TEXT,
    retry_count      INTEGER NOT NULL DEFAULT 0,
    issued_at        DATETIME,
    accepted_at      DATETIME,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_eledocs_sale_id ON electronic_documents(sale_id);
CREATE INDEX IF NOT EXISTS idx_eledocs_status ON electronic_documents(status);
CREATE INDEX IF NOT EXISTS idx_eledocs_provider_id ON electronic_documents(provider_id);
`
```

---

## Modelos nuevos en `internal/model/model.go`

```go
// FiscalConfig almacena la configuración de facturación electrónica del negocio.
// Es una tabla de una sola fila (id siempre = 1).
type FiscalConfig struct {
    ID                int64      `json:"id"`
    Enabled           bool       `json:"enabled"`
    NIT               string     `json:"nit"`
    RazonSocial       string     `json:"razon_social"`
    Direccion         string     `json:"direccion"`
    CIIU              string     `json:"ciiu"`
    ResolucionDIAN    string     `json:"resolucion_dian"`
    Prefijo           string     `json:"prefijo"`
    RangoDesde        int64      `json:"rango_desde"`
    RangoHasta        int64      `json:"rango_hasta"`
    FechaResolucion   *time.Time `json:"fecha_resolucion"`
    FechaVencimiento  *time.Time `json:"fecha_vencimiento"`
    ClaveTecnica      string     `json:"clave_tecnica"`
    Ambiente          string     `json:"ambiente"` // "test" | "production"
    SoftwareID        string     `json:"software_id"`
    SoftwarePIN       string     `json:"software_pin"`
    Provider          string     `json:"provider"` // "factus"
    ProviderClientID  string     `json:"provider_client_id"`
    ProviderSecret    string     `json:"-"` // omitir de JSON por seguridad
    WebhookURL        string     `json:"webhook_url"`
    CreatedAt         time.Time  `json:"created_at"`
    UpdatedAt         time.Time  `json:"updated_at"`
}

// ElectronicDocument rastrea el estado de un documento electrónico ante la DIAN.
type ElectronicDocument struct {
    ID              string     `json:"id"`       // UUID v4
    SaleID          int64      `json:"sale_id"`
    Type            string     `json:"type"`     // "doc_equivalente" | "nota_credito"
    Status          string     `json:"status"`   // "draft"|"pending"|"accepted"|"rejected"|"cancelled"
    Provider        string     `json:"provider"`
    ProviderID      string     `json:"provider_id"`
    CUDE            string     `json:"cude"`
    Number          string     `json:"number"`   // "POS-000001"
    XMLURL          string     `json:"xml_url"`
    PDFURL          string     `json:"pdf_url"`
    DIANResponse    string     `json:"dian_response"`
    RejectionReason string     `json:"rejection_reason"`
    RetryCount      int        `json:"retry_count"`
    IssuedAt        *time.Time `json:"issued_at"`
    AcceptedAt      *time.Time `json:"accepted_at"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}
```

---

## `internal/repository/errors.go` — agregar

```go
// ErrRequiresCreditNote se retorna cuando se intenta anular una venta
// que ya tiene un documento electrónico aceptado por la DIAN.
type ErrRequiresCreditNote struct {
    DocumentID string
    CUDE       string
}

func (e *ErrRequiresCreditNote) Error() string {
    return "la venta tiene un documento electrónico aceptado; se requiere nota crédito para anular"
}
```

---

## `internal/repository/repository.go` — agregar interface

```go
// ElectronicDocRepository maneja persistencia de documentos electrónicos.
type ElectronicDocRepository interface {
    Create(ctx context.Context, doc *model.ElectronicDocument) error
    GetBySaleID(ctx context.Context, saleID int64) (*model.ElectronicDocument, error)
    GetByProviderID(ctx context.Context, providerID string) (*model.ElectronicDocument, error)
    Update(ctx context.Context, doc *model.ElectronicDocument) error
    ListByStatus(ctx context.Context, status string, limit int) ([]*model.ElectronicDocument, error)
}
```

---

## `internal/repository/eledoc.go` — nuevo archivo

```go
package repository

import (
    "context"
    "database/sql"
    "time"

    "github.com/andreyo/pos/internal/model"
    "github.com/google/uuid"
)

type eledocRepository struct {
    db *sql.DB
}

func NewElectronicDocRepository(db *sql.DB) ElectronicDocRepository {
    return &eledocRepository{db: db}
}

func (r *eledocRepository) Create(ctx context.Context, doc *model.ElectronicDocument) error {
    if doc.ID == "" {
        doc.ID = uuid.New().String()
    }
    now := time.Now()
    doc.CreatedAt = now
    doc.UpdatedAt = now
    _, err := r.db.ExecContext(ctx, `
        INSERT INTO electronic_documents
            (id, sale_id, type, status, provider, provider_id, cude, number,
             xml_url, pdf_url, dian_response, rejection_reason, retry_count,
             issued_at, accepted_at, created_at, updated_at)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
        doc.ID, doc.SaleID, doc.Type, doc.Status, doc.Provider,
        nullStr(doc.ProviderID), nullStr(doc.CUDE), nullStr(doc.Number),
        nullStr(doc.XMLURL), nullStr(doc.PDFURL), nullStr(doc.DIANResponse),
        nullStr(doc.RejectionReason), doc.RetryCount,
        doc.IssuedAt, doc.AcceptedAt, doc.CreatedAt, doc.UpdatedAt,
    )
    return err
}

func (r *eledocRepository) GetBySaleID(ctx context.Context, saleID int64) (*model.ElectronicDocument, error) {
    row := r.db.QueryRowContext(ctx,
        `SELECT id, sale_id, type, status, provider, COALESCE(provider_id,''),
                COALESCE(cude,''), COALESCE(number,''), COALESCE(xml_url,''),
                COALESCE(pdf_url,''), COALESCE(dian_response,''),
                COALESCE(rejection_reason,''), retry_count,
                issued_at, accepted_at, created_at, updated_at
         FROM electronic_documents WHERE sale_id = ?`, saleID)
    return scanDoc(row)
}

func (r *eledocRepository) GetByProviderID(ctx context.Context, providerID string) (*model.ElectronicDocument, error) {
    row := r.db.QueryRowContext(ctx,
        `SELECT id, sale_id, type, status, provider, COALESCE(provider_id,''),
                COALESCE(cude,''), COALESCE(number,''), COALESCE(xml_url,''),
                COALESCE(pdf_url,''), COALESCE(dian_response,''),
                COALESCE(rejection_reason,''), retry_count,
                issued_at, accepted_at, created_at, updated_at
         FROM electronic_documents WHERE provider_id = ?`, providerID)
    return scanDoc(row)
}

func (r *eledocRepository) Update(ctx context.Context, doc *model.ElectronicDocument) error {
    doc.UpdatedAt = time.Now()
    _, err := r.db.ExecContext(ctx, `
        UPDATE electronic_documents SET
            status=?, provider_id=?, cude=?, number=?, xml_url=?, pdf_url=?,
            dian_response=?, rejection_reason=?, retry_count=?,
            issued_at=?, accepted_at=?, updated_at=?
        WHERE id=?`,
        doc.Status, nullStr(doc.ProviderID), nullStr(doc.CUDE), nullStr(doc.Number),
        nullStr(doc.XMLURL), nullStr(doc.PDFURL), nullStr(doc.DIANResponse),
        nullStr(doc.RejectionReason), doc.RetryCount,
        doc.IssuedAt, doc.AcceptedAt, doc.UpdatedAt, doc.ID,
    )
    return err
}

func (r *eledocRepository) ListByStatus(ctx context.Context, status string, limit int) ([]*model.ElectronicDocument, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT id, sale_id, type, status, provider, COALESCE(provider_id,''),
               COALESCE(cude,''), COALESCE(number,''), COALESCE(xml_url,''),
               COALESCE(pdf_url,''), COALESCE(dian_response,''),
               COALESCE(rejection_reason,''), retry_count,
               issued_at, accepted_at, created_at, updated_at
        FROM electronic_documents WHERE status=? LIMIT ?`, status, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var docs []*model.ElectronicDocument
    for rows.Next() {
        doc, err := scanDoc(rows)
        if err != nil {
            return nil, err
        }
        docs = append(docs, doc)
    }
    return docs, rows.Err()
}

// scanner matches both *sql.Row and *sql.Rows.
type scanner interface {
    Scan(dest ...any) error
}

func scanDoc(s scanner) (*model.ElectronicDocument, error) {
    var doc model.ElectronicDocument
    err := s.Scan(
        &doc.ID, &doc.SaleID, &doc.Type, &doc.Status, &doc.Provider,
        &doc.ProviderID, &doc.CUDE, &doc.Number, &doc.XMLURL, &doc.PDFURL,
        &doc.DIANResponse, &doc.RejectionReason, &doc.RetryCount,
        &doc.IssuedAt, &doc.AcceptedAt, &doc.CreatedAt, &doc.UpdatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    return &doc, err
}

func nullStr(s string) any {
    if s == "" {
        return nil
    }
    return s
}
```

---

## `internal/factelect/types.go`

```go
// Package factelect maneja la emisión de documentos electrónicos DIAN
// vía proveedor tecnológico (Factus por defecto).
// Es agnóstico al vertical: POS y Taller usan el mismo paquete via SaleAdapter.
package factelect

import "time"

// BillRequest es el contrato entre cualquier vertical y Factus.
// Se construye vía SaleAdapter.ToBillRequest().
type BillRequest struct {
    Issuer         IssuerInfo
    Buyer          BuyerInfo
    Items          []BillItem
    Totals         BillTotals
    PaymentMethods []PaymentMethod // MVP: siempre 1 elemento
    IssuedAt       time.Time       // hora en UTC-5 (América/Bogotá)
    Notes          string
    Environment    string            // "test" | "production"
    Extra          map[string]string // datos libres por vertical
}

// IssuerInfo datos del negocio emisor (de fiscal_config + business_config).
type IssuerInfo struct {
    NIT         string
    RazonSocial string
    Direccion   string
    CIIU        string
    SoftwareID  string
    SoftwarePIN string
    Resolucion  string
    Prefijo     string
}

// BuyerInfo datos del comprador. IsAnonymous=true → "consumidor final".
type BuyerInfo struct {
    IsAnonymous    bool
    DocumentType   string // CC | NIT | CE | TI | PASAPORTE
    DocumentNumber string
    Name           string
    Email          string
    TaxRegime      string // "simplified" | "regular"
}

// BillItem una línea del documento equivalente.
type BillItem struct {
    Description  string
    ProductCode  string
    UnitCodeDIAN string  // código DIAN: "94"=unidad, "KGM"=kg, "MTR"=metro, etc.
    Quantity     float64
    UnitPrice    float64
    Discount     float64
    TaxRate      float64 // 0, 5 o 19
    TaxAmount    float64
    Subtotal     float64
    ProductType  string // "01"=mercancía, "04"=servicio
}

// BillTotals totales consolidados del documento.
type BillTotals struct {
    Subtotal float64
    Discount float64
    TaxBase  float64
    TotalIVA float64
    Total    float64
}

// PaymentMethod forma de pago con código DIAN.
// Códigos: "10"=efectivo, "42"=tarjeta débito, "48"=tarjeta crédito, "20"=transferencia.
type PaymentMethod struct {
    Method string
    Amount float64
}

// BillResponse respuesta del proveedor al crear un documento.
type BillResponse struct {
    ProviderID string
    Number     string
    CUDE       string
    Status     string
    PDFURL     string
    XMLURL     string
    Message    string
}

// StatusResponse respuesta al consultar estado de un documento.
type StatusResponse struct {
    ProviderID      string
    Status          string // "draft"|"pending"|"accepted"|"rejected"
    CUDE            string
    PDFURL          string
    XMLURL          string
    DIANResponse    string
    RejectionReason string
}

// CreditNoteRequest para emitir nota crédito referenciando un DEE aceptado.
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
```

---

## `internal/factelect/adapter.go`

```go
package factelect

import "context"

// SaleAdapter convierte una venta del vertical a BillRequest.
// Cada vertical (POS, Taller) implementa esta interface.
// El taller importa este paquete y provee su propia implementación.
type SaleAdapter interface {
    ToBillRequest(ctx context.Context, saleID int64, buyerEmail string) (*BillRequest, error)
}
```

---

## `internal/factelect/mapper.go`

```go
package factelect

// unitToDIAN traduce el campo unit del producto al código DIAN.
// Default "94" (unidad) para valores no mapeados.
var unitToDIAN = map[string]string{
    "unit":  "94",
    "und":   "94",
    "unidad":"94",
    "kg":    "KGM",
    "liter": "LTR",
    "lb":    "LBR",
    "m":     "MTR",
    "m2":    "MTK",
    "m3":    "MTQ",
    "caja":  "BX",
    "par":   "PR",
    "hora":  "HUR",
}

// UnitCode retorna el código DIAN para una unidad de medida.
func UnitCode(unit string) string {
    if code, ok := unitToDIAN[unit]; ok {
        return code
    }
    return "94" // unidad por defecto
}

// paymentMethodCode traduce el payment_method del POS al código DIAN.
// "10"=efectivo, "20"=transferencia, "42"=débito, "48"=crédito.
var paymentMethodCode = map[string]string{
    "cash":     "10",
    "transfer": "20",
    "card":     "42", // tarjeta débito por defecto; ajustar si se distingue débito/crédito
    "credit":   "48",
}

// PaymentCode retorna el código DIAN para un método de pago.
func PaymentCode(method string) string {
    if code, ok := paymentMethodCode[method]; ok {
        return code
    }
    return "10"
}

// productTypeCode retorna el tipo de ítem DIAN según el tipo de producto.
// "01"=mercancía, "04"=servicio.
func productTypeCode(productType string) string {
    if productType == "service" {
        return "04"
    }
    return "01"
}
```

---

## `internal/factelect/config.go`

```go
package factelect

import (
    "context"
    "database/sql"
    "time"

    "github.com/andreyo/pos/internal/model"
)

// FiscalConfigRepository persiste la configuración fiscal del negocio.
type FiscalConfigRepository interface {
    Get(ctx context.Context) (*model.FiscalConfig, error)
    Update(ctx context.Context, cfg *model.FiscalConfig) error
}

type fiscalConfigSQLite struct {
    db *sql.DB
}

// NewFiscalConfigRepository crea una implementación SQLite.
func NewFiscalConfigRepository(db *sql.DB) FiscalConfigRepository {
    return &fiscalConfigSQLite{db: db}
}

func (r *fiscalConfigSQLite) Get(ctx context.Context) (*model.FiscalConfig, error) {
    var cfg model.FiscalConfig
    err := r.db.QueryRowContext(ctx, `
        SELECT id, enabled, nit, razon_social, direccion, ciiu,
               resolucion_dian, COALESCE(prefijo,''), rango_desde, rango_hasta,
               fecha_resolucion, fecha_vencimiento,
               clave_tecnica, ambiente, software_id, software_pin,
               provider, provider_client_id, provider_secret,
               COALESCE(webhook_url,''), created_at, updated_at
        FROM fiscal_config WHERE id=1`,
    ).Scan(
        &cfg.ID, &cfg.Enabled, &cfg.NIT, &cfg.RazonSocial, &cfg.Direccion, &cfg.CIIU,
        &cfg.ResolucionDIAN, &cfg.Prefijo, &cfg.RangoDesde, &cfg.RangoHasta,
        &cfg.FechaResolucion, &cfg.FechaVencimiento,
        &cfg.ClaveTecnica, &cfg.Ambiente, &cfg.SoftwareID, &cfg.SoftwarePIN,
        &cfg.Provider, &cfg.ProviderClientID, &cfg.ProviderSecret,
        &cfg.WebhookURL, &cfg.CreatedAt, &cfg.UpdatedAt,
    )
    return &cfg, err
}

func (r *fiscalConfigSQLite) Update(ctx context.Context, cfg *model.FiscalConfig) error {
    cfg.UpdatedAt = time.Now()
    _, err := r.db.ExecContext(ctx, `
        UPDATE fiscal_config SET
            enabled=?, nit=?, razon_social=?, direccion=?, ciiu=?,
            resolucion_dian=?, prefijo=?, rango_desde=?, rango_hasta=?,
            fecha_resolucion=?, fecha_vencimiento=?,
            clave_tecnica=?, ambiente=?, software_id=?, software_pin=?,
            provider=?, provider_client_id=?, provider_secret=?,
            webhook_url=?, updated_at=?
        WHERE id=1`,
        cfg.Enabled, cfg.NIT, cfg.RazonSocial, cfg.Direccion, cfg.CIIU,
        cfg.ResolucionDIAN, cfg.Prefijo, cfg.RangoDesde, cfg.RangoHasta,
        cfg.FechaResolucion, cfg.FechaVencimiento,
        cfg.ClaveTecnica, cfg.Ambiente, cfg.SoftwareID, cfg.SoftwarePIN,
        cfg.Provider, cfg.ProviderClientID, cfg.ProviderSecret,
        cfg.WebhookURL, cfg.UpdatedAt,
    )
    return err
}
```

---

## `internal/factelect/client.go`

```go
package factelect

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "sync"
    "time"
)

const factusBaseURL = "https://api.factus.com.co"
const factusSandboxURL = "https://api-sandbox.factus.com.co"

// Client es el cliente HTTP contra Factus con token OAuth2 cacheado.
type Client struct {
    mu          sync.Mutex
    httpClient  *http.Client
    baseURL     string
    clientID    string
    clientSecret string
    token        string
    tokenExpiry  time.Time
}

// NewClient crea un cliente Factus.
// ambiente: "test" → sandbox, "production" → producción.
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

// ensureToken obtiene/renueva el token OAuth2 si está vencido.
func (c *Client) ensureToken(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    if time.Now().Before(c.tokenExpiry.Add(-30 * time.Second)) {
        return nil // todavía válido
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

// post hace POST JSON a Factus y decodifica la respuesta en out.
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
        var errBody map[string]any
        _ = json.NewDecoder(resp.Body).Decode(&errBody)
        return fmt.Errorf("factus %s status %d: %v", path, resp.StatusCode, errBody)
    }
    if out != nil {
        return json.NewDecoder(resp.Body).Decode(out)
    }
    return nil
}

// get hace GET a Factus y decodifica la respuesta en out.
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

// CreateBill emite un documento equivalente en Factus.
// El mapeo de BillRequest → payload Factus va aquí.
// NOTA: El payload exacto depende de la documentación de Factus.
// Esta función es el único lugar donde vive el mapeo específico al proveedor.
func (c *Client) CreateBill(ctx context.Context, req *BillRequest) (*BillResponse, error) {
    payload := billRequestToFactus(req)
    var raw map[string]any
    if err := c.post(ctx, "/api/v1/bills", payload, &raw); err != nil {
        return nil, err
    }
    return parseFactusBillResponse(raw), nil
}

// GetBillStatus consulta el estado de un documento por su ID en Factus.
func (c *Client) GetBillStatus(ctx context.Context, providerID string) (*StatusResponse, error) {
    var raw map[string]any
    if err := c.get(ctx, "/api/v1/bills/"+providerID, &raw); err != nil {
        return nil, err
    }
    return parseFactusStatusResponse(raw), nil
}

// CreateCreditNote emite una nota crédito en Factus.
func (c *Client) CreateCreditNote(ctx context.Context, req *CreditNoteRequest) (*BillResponse, error) {
    payload := creditNoteToFactus(req)
    var raw map[string]any
    if err := c.post(ctx, "/api/v1/credit-notes", payload, &raw); err != nil {
        return nil, err
    }
    return parseFactusBillResponse(raw), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Mapeo BillRequest → payload Factus
// Adaptar campos según documentación real de Factus (developers.factus.com.co)
// ──────────────────────────────────────────────────────────────────────────────

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
        "identification":      req.Buyer.DocumentNumber,
        "dv":                  "",
        "company":             req.Buyer.Name,
        "trade_name":          req.Buyer.Name,
        "names":               req.Buyer.Name,
        "address":             "",
        "email":               req.Buyer.Email,
        "phone":               "",
        "identification_document_id": documentTypeID(req.Buyer.DocumentType),
        "municipality_id":     149, // Bogotá por defecto; ajustar si se conoce municipio
    }
    if req.Buyer.IsAnonymous {
        buyer["identification"] = "222222222222"
        buyer["company"] = "Consumidor Final"
        buyer["names"] = "Consumidor Final"
        buyer["identification_document_id"] = 3 // CC genérico
    }
    return map[string]any{
        "numbering_range_id": 0, // Factus lo asigna según la resolución configurada
        "reference_code":     "",
        "observation":        req.Notes,
        "payment_form":       "1", // 1=contado
        "payment_due_date":   req.IssuedAt.Format("2006-01-02"),
        "buyer":              buyer,
        "items":              items,
        "payment_methods":    payments,
    }
}

func creditNoteToFactus(req *CreditNoteRequest) map[string]any {
    // Mismo patrón que billRequestToFactus pero con referencia al CUDE original.
    base := billRequestToFactus(&BillRequest{
        Issuer:         req.Issuer,
        Buyer:          req.Buyer,
        Items:          req.Items,
        Totals:         req.Totals,
        IssuedAt:       req.IssuedAt,
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
        return 22 // excluido
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
```

> **IMPORTANTE:** Los nombres de campos del payload Factus (`numbering_range_id`, `tribute_id`, etc.)
> deben verificarse contra [developers.factus.com.co](https://developers.factus.com.co) antes de usar en producción.
> Esta implementación es la estructura base; los nombres exactos pueden diferir.

---

## `internal/factelect/pos_adapter.go`

```go
package factelect

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/andreyo/pos/internal/model"
    "github.com/andreyo/pos/internal/repository"
)

// PosSaleAdapter implementa SaleAdapter para el vertical POS.
// El taller crea su propio adapter en su repo, siguiendo el mismo patrón.
type PosSaleAdapter struct {
    salesRepo    repository.SaleRepository
    productsRepo repository.ProductRepository
    setupRepo    *repository.SetupRepository
    fiscalRepo   FiscalConfigRepository
}

// NewPosSaleAdapter construye el adapter inyectando dependencias.
func NewPosSaleAdapter(
    sales repository.SaleRepository,
    products repository.ProductRepository,
    setup *repository.SetupRepository,
    fiscal FiscalConfigRepository,
) *PosSaleAdapter {
    return &PosSaleAdapter{
        salesRepo:    sales,
        productsRepo: products,
        setupRepo:    setup,
        fiscalRepo:   fiscal,
    }
}

// ToBillRequest convierte una Sale del POS a un BillRequest listo para Factus.
func (a *PosSaleAdapter) ToBillRequest(ctx context.Context, saleID int64, buyerEmail string) (*BillRequest, error) {
    sale, err := a.salesRepo.GetByID(ctx, saleID)
    if err != nil {
        return nil, fmt.Errorf("pos_adapter: obtener venta %d: %w", saleID, err)
    }

    cfg, err := a.fiscalRepo.Get(ctx)
    if err != nil {
        return nil, fmt.Errorf("pos_adapter: obtener fiscal_config: %w", err)
    }

    setup, err := a.setupRepo.Get()
    if err != nil {
        return nil, fmt.Errorf("pos_adapter: obtener business_config: %w", err)
    }

    // Validar resolución no vencida
    if cfg.FechaVencimiento != nil && time.Now().After(*cfg.FechaVencimiento) {
        return nil, errors.New("la resolución de facturación está vencida; renueva ante la DIAN")
    }

    issuer := IssuerInfo{
        NIT:         setup.NIT,
        RazonSocial: setup.Name,
        Direccion:   setup.Address,
        CIIU:        setup.CIIU,
        SoftwareID:  cfg.SoftwareID,
        SoftwarePIN: cfg.SoftwarePIN,
        Resolucion:  cfg.ResolucionDIAN,
        Prefijo:     cfg.Prefijo,
    }

    buyer, err := a.buildBuyer(ctx, sale, buyerEmail)
    if err != nil {
        return nil, err
    }

    items, totals, err := a.buildItems(ctx, sale)
    if err != nil {
        return nil, err
    }

    payments := []PaymentMethod{{
        Method: PaymentCode(sale.PaymentMethod),
        Amount: sale.Total,
    }}

    return &BillRequest{
        Issuer:         issuer,
        Buyer:          buyer,
        Items:          items,
        Totals:         totals,
        PaymentMethods: payments,
        IssuedAt:       sale.CreatedAt,
        Environment:    cfg.Ambiente,
    }, nil
}

func (a *PosSaleAdapter) buildBuyer(ctx context.Context, sale *model.Sale, buyerEmail string) (BuyerInfo, error) {
    if sale.CustomerID == nil {
        return BuyerInfo{IsAnonymous: true, Email: buyerEmail}, nil
    }
    cust, err := a.salesRepo.(interface {
        // GetByID en CustomerRepository — accedemos via cast si SaleRepo no lo expone.
        // Nota: en la práctica pasaríamos CustomerRepository al adapter también.
        // Ver nota de implementación abajo.
    }); _ = cust; _ = err
    // NOTA: Si se quiere enriquecer con datos del customer (document_type, document_number),
    // agregar CustomerRepository como dependencia del adapter.
    // Para MVP, con customer_id identificado basta con consumidor final + email del customer si lo tiene.
    return BuyerInfo{IsAnonymous: true, Email: buyerEmail}, nil
}

func (a *PosSaleAdapter) buildItems(ctx context.Context, sale *model.Sale) ([]BillItem, BillTotals, error) {
    items := make([]BillItem, 0, len(sale.Items))
    var totalIVA float64
    for _, si := range sale.Items {
        prod, err := a.productsRepo.GetByID(ctx, si.ProductID)
        if err != nil {
            return nil, BillTotals{}, fmt.Errorf("pos_adapter: producto %d: %w", si.ProductID, err)
        }
        taxAmt := si.Subtotal * si.TaxRate
        totalIVA += taxAmt
        items = append(items, BillItem{
            Description:  prod.Name,
            ProductCode:  prod.Code,
            UnitCodeDIAN: UnitCode(prod.Unit),
            Quantity:     si.Quantity,
            UnitPrice:    si.UnitPrice,
            Discount:     si.Discount,
            TaxRate:      si.TaxRate * 100, // convertir 0.19 → 19
            TaxAmount:    taxAmt,
            Subtotal:     si.Subtotal,
            ProductType:  productTypeCode(prod.Type),
        })
    }
    totals := BillTotals{
        Subtotal: sale.Subtotal,
        Discount: sale.Discount,
        TaxBase:  sale.Subtotal - sale.Discount,
        TotalIVA: totalIVA,
        Total:    sale.Total,
    }
    return items, totals, nil
}
```

> **Nota de implementación:** `buildBuyer` está simplificado. Para leer datos fiscales del customer
> (document_type, document_number), agregar `repository.CustomerRepository` al `PosSaleAdapter`.

---

## `internal/factelect/service.go`

```go
package factelect

import (
    "context"
    "fmt"
    "log/slog"
    "time"

    "github.com/andreyo/pos/internal/model"
    "github.com/andreyo/pos/internal/repository"
    "github.com/google/uuid"
)

// FacElectService orquesta la emisión de documentos electrónicos.
type FacElectService struct {
    adapter    SaleAdapter
    client     *Client
    eledocRepo repository.ElectronicDocRepository
    fiscalRepo FiscalConfigRepository
}

// NewFacElectService construye el servicio con todas sus dependencias.
func NewFacElectService(
    adapter SaleAdapter,
    client *Client,
    eledocRepo repository.ElectronicDocRepository,
    fiscalRepo FiscalConfigRepository,
) *FacElectService {
    return &FacElectService{
        adapter:    adapter,
        client:     client,
        eledocRepo: eledocRepo,
        fiscalRepo: fiscalRepo,
    }
}

// EmitFromSale emite un DEE para la venta dada. Se llama en goroutine (async).
// No bloquea la confirmación de la venta al cajero.
func (s *FacElectService) EmitFromSale(ctx context.Context, saleID int64, buyerEmail string) {
    cfg, err := s.fiscalRepo.Get(ctx)
    if err != nil || !cfg.Enabled {
        return // FE deshabilitada o config no disponible
    }

    doc := &model.ElectronicDocument{
        ID:       uuid.New().String(),
        SaleID:   saleID,
        Type:     "doc_equivalente",
        Status:   "draft",
        Provider: cfg.Provider,
    }
    if err := s.eledocRepo.Create(ctx, doc); err != nil {
        slog.Error("factelect: crear draft", "sale_id", saleID, "err", err)
        return
    }

    req, err := s.adapter.ToBillRequest(ctx, saleID, buyerEmail)
    if err != nil {
        slog.Error("factelect: construir BillRequest", "sale_id", saleID, "err", err)
        return
    }

    resp, err := s.client.CreateBill(ctx, req)
    if err != nil {
        slog.Error("factelect: CreateBill falló", "sale_id", saleID, "err", err)
        // doc queda en 'draft' → poller reintentará
        return
    }

    now := time.Now()
    doc.Status = "pending"
    doc.ProviderID = resp.ProviderID
    doc.Number = resp.Number
    doc.IssuedAt = &now
    if err := s.eledocRepo.Update(ctx, doc); err != nil {
        slog.Error("factelect: actualizar a pending", "sale_id", saleID, "err", err)
    }
}

// CreateCreditNote emite una nota crédito referenciando un DEE aceptado.
// Retorna el doc de nota crédito creado, o error si Factus rechaza.
func (s *FacElectService) CreateCreditNote(ctx context.Context, saleID int64, motive string) (*model.ElectronicDocument, error) {
    orig, err := s.eledocRepo.GetBySaleID(ctx, saleID)
    if err != nil {
        return nil, fmt.Errorf("factelect: doc original no encontrado: %w", err)
    }

    req, err := s.adapter.ToBillRequest(ctx, saleID, "")
    if err != nil {
        return nil, err
    }
    cnReq := &CreditNoteRequest{
        OriginalCUDE: orig.CUDE,
        OriginalID:   orig.ProviderID,
        Motive:       motive,
        Issuer:       req.Issuer,
        Buyer:        req.Buyer,
        Items:        req.Items,
        Totals:       req.Totals,
        IssuedAt:     time.Now(),
    }

    resp, err := s.client.CreateCreditNote(ctx, cnReq)
    if err != nil {
        return nil, fmt.Errorf("factus nota crédito rechazada: %w", err)
    }

    cfg, _ := s.fiscalRepo.Get(ctx)
    now := time.Now()
    noteDoc := &model.ElectronicDocument{
        ID:         uuid.New().String(),
        SaleID:     saleID,
        Type:       "nota_credito",
        Status:     "pending",
        Provider:   cfg.Provider,
        ProviderID: resp.ProviderID,
        Number:     resp.Number,
        IssuedAt:   &now,
    }
    if err := s.eledocRepo.Create(ctx, noteDoc); err != nil {
        return nil, err
    }
    return noteDoc, nil
}

// GetDocBySaleID retorna el documento electrónico de una venta, o nil si no existe.
func (s *FacElectService) GetDocBySaleID(ctx context.Context, saleID int64) (*model.ElectronicDocument, error) {
    doc, err := s.eledocRepo.GetBySaleID(ctx, saleID)
    if err == repository.ErrNotFound {
        return nil, nil
    }
    return doc, err
}
```

---

## `internal/factelect/poller.go`

```go
package factelect

import (
    "context"
    "log/slog"
    "time"

    "github.com/andreyo/pos/internal/repository"
)

const maxRetries = 10

// PollPending consulta el estado en Factus de todos los docs en status='pending'.
// Llamar en un ticker cada 30 segundos desde main.
func (s *FacElectService) PollPending(ctx context.Context) {
    docs, err := s.eledocRepo.ListByStatus(ctx, "pending", 50)
    if err != nil {
        slog.Error("factelect: listar pendientes", "err", err)
        return
    }
    for _, doc := range docs {
        if doc.RetryCount >= maxRetries {
            doc.Status = "rejected"
            doc.RejectionReason = "máximo de reintentos alcanzado"
            _ = s.eledocRepo.Update(ctx, doc)
            continue
        }
        status, err := s.client.GetBillStatus(ctx, doc.ProviderID)
        if err != nil {
            slog.Warn("factelect: consultar estado", "provider_id", doc.ProviderID, "err", err)
            doc.RetryCount++
            _ = s.eledocRepo.Update(ctx, doc)
            continue
        }
        switch status.Status {
        case "accepted":
            now := time.Now()
            doc.Status = "accepted"
            doc.CUDE = status.CUDE
            doc.PDFURL = status.PDFURL
            doc.XMLURL = status.XMLURL
            doc.AcceptedAt = &now
        case "rejected":
            doc.Status = "rejected"
            doc.RejectionReason = status.RejectionReason
            doc.DIANResponse = status.DIANResponse
        default:
            doc.RetryCount++
        }
        if err := s.eledocRepo.Update(ctx, doc); err != nil {
            slog.Error("factelect: actualizar doc", "id", doc.ID, "err", err)
        }
    }
}

// StartPoller lanza el job de polling en background.
// Cancelar ctx para detenerlo (en main: defer cancel()).
func (s *FacElectService) StartPoller(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                s.PollPending(ctx)
            }
        }
    }()
}
```

---

## Cambio en `internal/service/sale.go`

Agregar campo `eledocRepo` opcional y modificar `Void`:

```go
// Agregar campo al struct:
type SaleService struct {
    repo         repository.SaleRepository
    productRepo  repository.ProductRepository
    registerRepo repository.RegisterRepository
    eledocRepo   repository.ElectronicDocRepository // nil si FE deshabilitada
}

// Agregar método setter (evita cambiar el constructor existente):
func (s *SaleService) WithElectronicDocs(r repository.ElectronicDocRepository) {
    s.eledocRepo = r
}

// Reemplazar Void:
func (s *SaleService) Void(ctx context.Context, saleID int64, userID int64) (*model.Sale, error) {
    if s.eledocRepo != nil {
        doc, err := s.eledocRepo.GetBySaleID(ctx, saleID)
        if err == nil && doc != nil &&
            (doc.Status == "accepted" || doc.Status == "pending") {
            return nil, &repository.ErrRequiresCreditNote{
                DocumentID: doc.ID,
                CUDE:       doc.CUDE,
            }
        }
    }
    return s.repo.Void(ctx, saleID, userID)
}
```

---

## Cambio en `internal/handler/sale.go`

**createSale** — capturar `buyer_email`:

```go
func (h *Handler) createSale(w http.ResponseWriter, r *http.Request) {
    var req struct {
        model.Sale
        BuyerEmail string `json:"buyer_email"`
    }
    if err := decode(r, &req); err != nil {
        respondError(w, http.StatusBadRequest, "request inválido")
        return
    }
    // ... lógica existente usando req.Sale ...
    created, err := h.sales.Create(r.Context(), &req.Sale)
    if err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }
    // Emitir DEE async si FE está configurada
    if h.factelect != nil {
        go h.factelect.EmitFromSale(r.Context(), created.ID, req.BuyerEmail)
    }
    respond(w, http.StatusCreated, created)
}
```

**voidSale** — manejar ErrRequiresCreditNote:

```go
func (h *Handler) voidSale(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        respondError(w, http.StatusBadRequest, "id inválido")
        return
    }
    claims := appmw.GetClaims(r)
    var userID int64
    if claims != nil {
        userID = claims.UserID
    }
    sale, err := h.sales.Void(r.Context(), id, userID)
    if errors.Is(err, repository.ErrNotFound) {
        respondError(w, http.StatusNotFound, "venta no encontrada")
        return
    }
    var cnErr *repository.ErrRequiresCreditNote
    if errors.As(err, &cnErr) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusConflict) // 409
        json.NewEncoder(w).Encode(map[string]any{
            "error":               "sale_has_accepted_invoice",
            "message":             cnErr.Error(),
            "document_id":         cnErr.DocumentID,
            "cude":                cnErr.CUDE,
            "requires_credit_note": true,
        })
        return
    }
    if err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }
    respond(w, http.StatusOK, sale)
}
```

**Nuevo handler voidWithCreditNote:**

```go
func (h *Handler) voidSaleWithCreditNote(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        respondError(w, http.StatusBadRequest, "id inválido")
        return
    }
    var body struct {
        Motive string `json:"motive"` // "DEVOLUCIÓN_DINERO" etc.
    }
    if err := decode(r, &body); err != nil || body.Motive == "" {
        respondError(w, http.StatusBadRequest, "motive requerido")
        return
    }
    if h.factelect == nil {
        respondError(w, http.StatusBadRequest, "facturación electrónica no habilitada")
        return
    }
    _, err = h.factelect.CreateCreditNote(r.Context(), id, body.Motive)
    if err != nil {
        respondError(w, http.StatusBadGateway, err.Error())
        return
    }
    claims := appmw.GetClaims(r)
    var userID int64
    if claims != nil {
        userID = claims.UserID
    }
    sale, err := h.sales.Void(r.Context(), id, userID)
    if err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }
    respond(w, http.StatusOK, sale)
}
```

---

## Cambios en `internal/handler/handler.go`

Agregar campo `factelect *factelect.FacElectService` al struct `Handler`.

En `New()`:

```go
func New(db *database.DB) *Handler {
    // ... repos existentes ...
    eledocRepo := repository.NewElectronicDocRepository(db.DB)
    fiscalRepo := factelect.NewFiscalConfigRepository(db.DB)

    // Obtener config para construir el client (si FE habilitada)
    cfg, _ := fiscalRepo.Get(context.Background())
    var factelSvc *factelect.FacElectService
    if cfg != nil && cfg.Enabled {
        client := factelect.NewClient(cfg.ProviderClientID, cfg.ProviderSecret, cfg.Ambiente)
        adapter := factelect.NewPosSaleAdapter(saleRepo, prodRepo, setupRepo, fiscalRepo)
        factelSvc = factelect.NewFacElectService(adapter, client, eledocRepo, fiscalRepo)
        factelSvc.StartPoller(context.Background()) // goroutine background
    }

    salesSvc := service.NewSaleService(saleRepo, prodRepo, regRepo)
    salesSvc.WithElectronicDocs(eledocRepo)

    return &Handler{
        // ... campos existentes ...
        factelect: factelSvc,
    }
}
```

En `Routes()` — agregar dentro del grupo admin:

```go
// Fiscal config
r.Get("/fiscal/config", h.getFiscalConfig)
r.Put("/fiscal/config", h.updateFiscalConfig)

// Documentos electrónicos
r.Get("/sales/{id}/factura", h.getSaleFactura)
r.Get("/sales/{id}/factura/pdf", h.getSaleFacturaPDF)
r.Post("/sales/{id}/void-with-credit-note", h.voidSaleWithCreditNote)
```

---

## `internal/handler/fiscal.go` — nuevo archivo

```go
package handler

import (
    "net/http"
    "strconv"
    "github.com/go-chi/chi/v5"
)

func (h *Handler) getFiscalConfig(w http.ResponseWriter, r *http.Request) {
    cfg, err := h.fiscalRepo.Get(r.Context())
    if err != nil {
        respondError(w, http.StatusInternalServerError, "error obteniendo configuración fiscal")
        return
    }
    respond(w, http.StatusOK, cfg)
}

func (h *Handler) updateFiscalConfig(w http.ResponseWriter, r *http.Request) {
    cfg, err := h.fiscalRepo.Get(r.Context())
    if err != nil {
        respondError(w, http.StatusInternalServerError, "error obteniendo configuración fiscal")
        return
    }
    if err := decode(r, cfg); err != nil {
        respondError(w, http.StatusBadRequest, "request inválido")
        return
    }
    if err := h.fiscalRepo.Update(r.Context(), cfg); err != nil {
        respondError(w, http.StatusInternalServerError, "error actualizando configuración fiscal")
        return
    }
    respond(w, http.StatusOK, cfg)
}

func (h *Handler) getSaleFactura(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        respondError(w, http.StatusBadRequest, "id inválido")
        return
    }
    if h.factelect == nil {
        respondError(w, http.StatusBadRequest, "facturación electrónica no habilitada")
        return
    }
    doc, err := h.factelect.GetDocBySaleID(r.Context(), id)
    if err != nil || doc == nil {
        respondError(w, http.StatusNotFound, "documento no encontrado")
        return
    }
    respond(w, http.StatusOK, doc)
}

func (h *Handler) getSaleFacturaPDF(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        respondError(w, http.StatusBadRequest, "id inválido")
        return
    }
    if h.factelect == nil {
        respondError(w, http.StatusNotFound, "facturación electrónica no habilitada")
        return
    }
    doc, err := h.factelect.GetDocBySaleID(r.Context(), id)
    if err != nil || doc == nil || doc.PDFURL == "" {
        respondError(w, http.StatusNotFound, "PDF no disponible aún")
        return
    }
    http.Redirect(w, r, doc.PDFURL, http.StatusTemporaryRedirect)
}
```

---

## Grafo de imports (sin ciclos)

```
model
  ↑
repository (usa model)
  ↑
factelect (usa model, repository) ← NO importa service
  ↑
service (usa model, repository) ← NO importa factelect
  ↑
handler (usa model, repository, service, factelect)
```

El handler es el único que conoce a `factelect` directamente.
`service.SaleService` recibe `repository.ElectronicDocRepository` (interface) → no conoce `factelect`.

---

## Acoplamiento para taller-vertical

El taller solo necesita:

```go
// En el repo del taller:
import "github.com/andreyo/pos/internal/factelect"

type TallerAdapter struct {
    ordenRepo  *repository.OrdenRepository
    fiscalRepo factelect.FiscalConfigRepository
}

func (a *TallerAdapter) ToBillRequest(ctx context.Context, ordenID int64, buyerEmail string) (*factelect.BillRequest, error) {
    orden, _ := a.ordenRepo.GetByID(ctx, ordenID)
    // Items: repuestos → ProductType "01", mano de obra → "04"
    // Extra para datos del vehículo:
    return &factelect.BillRequest{
        Extra: map[string]string{
            "placa": orden.Placa,
            "km":    fmt.Sprintf("%d", orden.KM),
        },
        // ... resto de campos ...
    }, nil
}
```

Opción de import: `replace github.com/andreyo/pos => ../pos-vertical` en el `go.mod` del taller.

---

## Checklist de verificación antes de marcar como done

- [ ] `go build ./...` sin errores de compilación
- [ ] Migration v8 corre limpia sobre BD existente (`go run . --migrate-only` o similar)
- [ ] `POST /sales` con `buyer_email` → doc en `electronic_documents` con `status=draft`
- [ ] Poller detecta docs `pending` → consulta Factus → actualiza a `accepted/rejected`
- [ ] `DELETE /sales/{id}/void` con DEE `accepted` → responde 409 con `requires_credit_note:true`
- [ ] `POST /sales/{id}/void-with-credit-note` → nota crédito OK → venta anulada
- [ ] `GET /fiscal/config` y `PUT /fiscal/config` funcionan
- [ ] `fiscal_config.enabled = false` → no se emite ningún documento (EmitFromSale retorna silencioso)
- [ ] Tests unitarios `mapper_test.go`: UnitCode, PaymentCode, productTypeCode
