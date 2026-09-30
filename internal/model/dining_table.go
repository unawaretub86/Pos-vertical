package model

// DiningTable is a physical table of the restaurant.
type DiningTable struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Zone      string `json:"zone"`
	Seats     int    `json:"seats"`
	SortOrder int    `json:"sort_order"`
	Active    bool   `json:"active"`
}

// TableWithState is a DiningTable plus its derived state (never stored):
// libre, ocupada or por_cobrar.
type TableWithState struct {
	DiningTable
	State         string  `json:"state"`
	OpenOrderID   *int64  `json:"open_order_id"`
	OpenTotal     float64 `json:"open_total"`
	OpenedMinutes int     `json:"opened_minutes"`
}
