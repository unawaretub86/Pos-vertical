package service

import (
	"context"
	"errors"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

var (
	ErrPurchaseNoItems      = errors.New("la compra debe tener al menos un producto")
	ErrPurchaseInvalidQty   = errors.New("la cantidad debe ser mayor a 0")
	ErrPurchaseInvalidCost  = errors.New("el costo unitario no puede ser negativo")
	ErrPurchaseSupplierGone = errors.New("proveedor no encontrado")
)

// PurchaseService validates and records supplier purchases (US-052).
type PurchaseService struct {
	repo      repository.PurchaseRepository
	suppliers repository.SupplierRepository
}

func NewPurchaseService(repo repository.PurchaseRepository, suppliers repository.SupplierRepository) *PurchaseService {
	return &PurchaseService{repo: repo, suppliers: suppliers}
}

// Create validates the purchase and its items, recalculates subtotal/total
// server-side (never trust client-sent money, same rule as sales — see
// SPRINT-11 US-050), and delegates the stock/cost update to the repository.
func (s *PurchaseService) Create(ctx context.Context, userID int64, p *model.Purchase) (*model.Purchase, error) {
	if len(p.Items) == 0 {
		return nil, ErrPurchaseNoItems
	}
	if _, err := s.suppliers.GetByID(ctx, p.SupplierID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrPurchaseSupplierGone
		}
		return nil, err
	}

	var total float64
	for i := range p.Items {
		if p.Items[i].Quantity <= 0 {
			return nil, ErrPurchaseInvalidQty
		}
		if p.Items[i].UnitCost < 0 {
			return nil, ErrPurchaseInvalidCost
		}
		if !IsWholePeso(p.Items[i].UnitCost) {
			return nil, ErrNotWholePeso
		}
		p.Items[i].Subtotal = LineBase(p.Items[i].Quantity, p.Items[i].UnitCost)
		total += p.Items[i].Subtotal
	}
	p.Total = total
	p.UserID = userID

	return s.repo.Create(ctx, p)
}

func (s *PurchaseService) List(ctx context.Context, filter repository.PurchaseFilter) ([]*model.Purchase, error) {
	return s.repo.List(ctx, filter)
}
