package model

import "time"

// FiscalConfig stores electronic billing configuration. Single-row table (id always = 1).
type FiscalConfig struct {
	ID               int64      `json:"id"`
	Enabled          bool       `json:"enabled"`
	NIT              string     `json:"nit"`
	RazonSocial      string     `json:"razon_social"`
	Direccion        string     `json:"direccion"`
	CIIU             string     `json:"ciiu"`
	ResolucionDIAN   string     `json:"resolucion_dian"`
	Prefijo          string     `json:"prefijo"`
	RangoDesde       int64      `json:"rango_desde"`
	RangoHasta       int64      `json:"rango_hasta"`
	FechaResolucion  *time.Time `json:"fecha_resolucion"`
	FechaVencimiento *time.Time `json:"fecha_vencimiento"`
	ClaveTecnica     string     `json:"clave_tecnica"`
	Ambiente         string     `json:"ambiente"` // "test" | "production"
	SoftwareID       string     `json:"software_id"`
	SoftwarePIN      string     `json:"software_pin"`
	Provider         string     `json:"provider"` // "factus"
	ProviderClientID string     `json:"provider_client_id"`
	ProviderSecret   string     `json:"-"`
	WebhookURL       string     `json:"webhook_url"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ElectronicDocument tracks the DIAN status of an electronic document.
type ElectronicDocument struct {
	ID              string     `json:"id"` // UUID v4
	SaleID          int64      `json:"sale_id"`
	Type            string     `json:"type"`   // "doc_equivalente" | "nota_credito"
	Status          string     `json:"status"` // "draft"|"pending"|"accepted"|"rejected"|"cancelled"
	Provider        string     `json:"provider"`
	ProviderID      string     `json:"provider_id"`
	CUDE            string     `json:"cude"`
	Number          string     `json:"number"` // "POS-000001"
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

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
}

type Category struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Product struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	Code           string   `json:"code"`
	Barcode        string   `json:"barcode"`
	CategoryID     *int64   `json:"category_id"`
	Type           string   `json:"type"`
	SalePrice      float64  `json:"sale_price"`
	WholesalePrice *float64 `json:"wholesale_price"`
	Cost           float64  `json:"cost"`
	Unit           string   `json:"unit"`
	Stock          float64  `json:"stock"`
	MinStock       float64  `json:"min_stock"`
	TaxRate        float64  `json:"tax_rate"`
	Active         bool     `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
}

type Customer struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

// CustomerWithBalance extends Customer with on-the-fly credit balance data.
type CustomerWithBalance struct {
	Customer
	Balance      float64    `json:"balance"`
	OldestDebtAt *time.Time `json:"oldest_debt_at"`
}

type CustomerPayment struct {
	ID         int64     `json:"id"`
	CustomerID int64     `json:"customer_id"`
	Amount     float64   `json:"amount"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

type Sale struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	CustomerID    *int64     `json:"customer_id"`
	RegisterID    *int64     `json:"register_id"`
	Subtotal      float64    `json:"subtotal"`
	Discount      float64    `json:"discount"`
	Tax           float64    `json:"tax"`
	Total         float64    `json:"total"`
	PaymentMethod string     `json:"payment_method"`
	CashTendered  *float64   `json:"cash_tendered"`
	ChangeAmount  *float64   `json:"change_amount"`
	Reference     string     `json:"reference"`
	Status        string     `json:"status"`
	VoidedBy      *int64     `json:"voided_by,omitempty"`
	VoidedAt      *time.Time `json:"voided_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	Items         []SaleItem `json:"items,omitempty"`
}

type SaleItem struct {
	ID            int64   `json:"id"`
	SaleID        int64   `json:"sale_id"`
	ProductID     int64   `json:"product_id"`
	Quantity      float64 `json:"quantity"`
	UnitPrice     float64 `json:"unit_price"`
	OriginalPrice float64 `json:"original_price,omitempty"`
	Discount      float64 `json:"discount"`
	Subtotal      float64 `json:"subtotal"`
	TaxRate       float64 `json:"tax_rate"`
}

type AuditLog struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  int64     `json:"entity_id"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

type Register struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	OpeningAmount float64    `json:"opening_amount"`
	ExpectedClose *float64   `json:"expected_close"`
	ActualClose   *float64   `json:"actual_close"`
	Difference    *float64   `json:"difference"`
	Status        string     `json:"status"`
	OpenedAt      time.Time  `json:"opened_at"`
	ClosedAt      *time.Time `json:"closed_at"`
}

type RegisterMovement struct {
	ID          int64     `json:"id"`
	RegisterID  int64     `json:"register_id"`
	Type        string    `json:"type"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type StockEntry struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	Quantity  float64   `json:"quantity"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type TicketBusiness struct {
	Name         string `json:"name"`
	Address      string `json:"address"`
	NIT          string `json:"nit"`
	TicketHeader string `json:"ticket_header"`
	TicketFooter string `json:"ticket_footer"`
}

type TicketItem struct {
	ProductName string  `json:"product_name"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Discount    float64 `json:"discount"`
	Subtotal    float64 `json:"subtotal"`
	Unit        string  `json:"unit"`
}

type TicketData struct {
	SaleID        int64          `json:"sale_id"`
	CreatedAt     time.Time      `json:"created_at"`
	PaymentMethod string         `json:"payment_method"`
	CashTendered  *float64       `json:"cash_tendered,omitempty"`
	ChangeAmount  *float64       `json:"change_amount,omitempty"`
	Reference     string         `json:"reference,omitempty"`
	Status        string         `json:"status"`
	Subtotal      float64        `json:"subtotal"`
	Discount      float64        `json:"discount"`
	Tax           float64        `json:"tax"`
	Total         float64        `json:"total"`
	Items         []TicketItem   `json:"items"`
	Customer      *Customer      `json:"customer,omitempty"`
	Business      TicketBusiness `json:"business"`
}
