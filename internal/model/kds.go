package model

import "time"

type KDSResponse struct {
	ServerTime time.Time  `json:"server_time"`
	Orders     []KDSOrder `json:"orders"`
}

type KDSOrder struct {
	OrderID    int64      `json:"order_id"`
	Type       string     `json:"type"`
	TableName  string     `json:"table_name"`
	TurnNumber *int       `json:"turn_number,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	Items      []KDSItem  `json:"items"`
}

type KDSItem struct {
	ID          int64     `json:"id"`
	ProductName string    `json:"product_name"`
	Quantity    float64   `json:"quantity"`
	Note        string    `json:"note"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}
