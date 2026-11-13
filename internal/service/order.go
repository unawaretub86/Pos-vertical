package service

import (
	"context"
	"strings"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// ValidationError is a user-facing input error (maps to HTTP 400).
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(msg string) error { return &ValidationError{Msg: msg} }

var (
	itemStatuses  = map[string]bool{"pending": true, "preparing": true, "ready": true, "delivered": true}
	orderStatuses = map[string]bool{"open": true, "closed": true, "cancelled": true}
)

type OrderService struct {
	repo repository.OrderRepository
}

func NewOrderService(repo repository.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) Open(ctx context.Context, tableID int64, orderType string, userID int64) (*model.Order, error) {
	if orderType != "" && orderType != "table" {
		return nil, invalid("solo se admiten pedidos de mesa")
	}
	if tableID <= 0 {
		return nil, invalid("la mesa es requerida")
	}
	return s.repo.Open(ctx, tableID, userID)
}

func (s *OrderService) AddItem(ctx context.Context, orderID, productID int64, quantity int, note string) (*model.OrderItem, error) {
	if productID <= 0 {
		return nil, invalid("el producto es requerido")
	}
	if quantity <= 0 {
		return nil, invalid("la cantidad debe ser un entero mayor a 0")
	}
	return s.repo.AddItem(ctx, orderID, productID, quantity, strings.TrimSpace(note))
}

func (s *OrderService) UpdateItem(ctx context.Context, orderID, itemID int64, p repository.ItemPatch) (*model.OrderItem, error) {
	if p.Quantity != nil && *p.Quantity <= 0 {
		return nil, invalid("la cantidad debe ser un entero mayor a 0")
	}
	if p.Status != nil && !itemStatuses[*p.Status] {
		return nil, invalid("estado inválido")
	}
	if p.Note != nil {
		n := strings.TrimSpace(*p.Note)
		p.Note = &n
	}
	return s.repo.UpdateItem(ctx, orderID, itemID, p)
}

func (s *OrderService) VoidItem(ctx context.Context, orderID, itemID, userID int64) (*model.OrderItem, error) {
	return s.repo.VoidItem(ctx, orderID, itemID, userID)
}

func (s *OrderService) Cancel(ctx context.Context, orderID, userID int64, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return invalid("el motivo es requerido")
	}
	return s.repo.Cancel(ctx, orderID, userID, reason)
}

func (s *OrderService) Get(ctx context.Context, id int64) (*model.Order, error) {
	o, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	// Payable total with IVA, priced at read time with the same per-line
	// rule as a counter sale (money.go); tax is not stored on the order.
	o.Total = 0
	for i := range o.Items {
		it := &o.Items[i]
		it.Subtotal = LineBase(it.Quantity, it.UnitPrice)
		if it.Status != "void" {
			o.Total += it.Subtotal + LineTax(it.Subtotal, it.TaxRate)
		}
	}
	return o, nil
}

func (s *OrderService) List(ctx context.Context, status string) ([]*model.OrderSummary, error) {
	if !orderStatuses[status] {
		return nil, invalid("estado inválido")
	}
	return s.repo.ListOpen(ctx, status)
}
