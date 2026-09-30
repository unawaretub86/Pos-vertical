package model

import "time"

// Order is a table order (type 'table') or, in the future, a takeaway order.
type Order struct {
	ID        int64       `json:"id"`
	Type      string      `json:"type"`
	TableID   *int64      `json:"table_id"`
	Status    string      `json:"status"`
	UserID    int64       `json:"user_id"`
	Total     float64     `json:"total"`
	CreatedAt time.Time   `json:"created_at"`
	Items     []OrderItem `json:"items"`
}

// OrderItem is one line of an order. UnitPrice is a snapshot of the product
// price at the moment the item was added.
type OrderItem struct {
	ID          int64   `json:"id"`
	OrderID     int64   `json:"order_id"`
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Subtotal    float64 `json:"subtotal"`
	Note        string  `json:"note"`
	Status      string  `json:"status"`
}

// OrderSummary is a row of the open-orders list.
type OrderSummary struct {
	ID            int64   `json:"id"`
	TableID       *int64  `json:"table_id"`
	TableName     string  `json:"table_name"`
	Total         float64 `json:"total"`
	ItemsPending  int     `json:"items_pending"`
	OpenedMinutes int     `json:"opened_minutes"`
}
