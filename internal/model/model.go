package model

import "time"

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
	CreatedAt     time.Time  `json:"created_at"`
	Items         []SaleItem `json:"items,omitempty"`
}

type SaleItem struct {
	ID        int64   `json:"id"`
	SaleID    int64   `json:"sale_id"`
	ProductID int64   `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Discount  float64 `json:"discount"`
	Subtotal  float64 `json:"subtotal"`
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

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
