package factelect

import "context"

// SaleAdapter converts a vertical's sale to a BillRequest.
// Each vertical (POS, Taller) implements this interface.
type SaleAdapter interface {
	ToBillRequest(ctx context.Context, saleID int64, buyerEmail string) (*BillRequest, error)
}
