package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

var validPaymentMethods = map[string]bool{
	"cash":     true,
	"transfer": true,
	"card":     true,
	"credit":   true,
}

// SaleService handles sale business logic.
type SaleService struct {
	repo         repository.SaleRepository
	productRepo  repository.ProductRepository
	registerRepo repository.RegisterRepository
}

// NewSaleService creates a SaleService backed by the given repositories.
func NewSaleService(repo repository.SaleRepository, productRepo repository.ProductRepository, registerRepo repository.RegisterRepository) *SaleService {
	return &SaleService{repo: repo, productRepo: productRepo, registerRepo: registerRepo}
}

func (s *SaleService) Create(ctx context.Context, sale *model.Sale) (*model.Sale, error) {
	if err := validateSale(sale); err != nil {
		return nil, err
	}

	// Verify open register
	reg, err := s.registerRepo.GetOpen(ctx)
	if err != nil {
		return nil, errors.New("no hay caja abierta")
	}
	sale.RegisterID = &reg.ID

	// Recalculate and validate totals server-side
	var subtotal float64
	for i := range sale.Items {
		itemSub := sale.Items[i].Quantity*sale.Items[i].UnitPrice - sale.Items[i].Discount
		sale.Items[i].Subtotal = itemSub
		subtotal += itemSub
	}
	sale.Subtotal = subtotal
	recalcTotal := subtotal - sale.Discount + sale.Tax
	if math.Abs(recalcTotal-sale.Total) > 1 {
		return nil, fmt.Errorf("total inconsistente: esperado %.0f, recibido %.0f", recalcTotal, sale.Total)
	}
	sale.Total = recalcTotal

	if sale.Status == "" {
		sale.Status = "completed"
	}
	return s.repo.Create(ctx, sale)
}

func (s *SaleService) List(ctx context.Context, filter repository.SaleFilter) ([]*model.Sale, error) {
	return s.repo.List(ctx, filter)
}

func (s *SaleService) GetByID(ctx context.Context, id int64) (*model.Sale, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SaleService) Void(ctx context.Context, saleID int64, userID int64) (*model.Sale, error) {
	return s.repo.Void(ctx, saleID, userID)
}

func validateSale(sale *model.Sale) error {
	if !validPaymentMethods[sale.PaymentMethod] {
		return errors.New("método de pago inválido: debe ser cash, transfer, card o credit")
	}
	if sale.PaymentMethod == "credit" && sale.CustomerID == nil {
		return errors.New("el método de pago 'credito' requiere un cliente")
	}
	if len(sale.Items) == 0 {
		return errors.New("la venta debe tener al menos un artículo")
	}
	for _, item := range sale.Items {
		if item.Quantity <= 0 {
			return errors.New("la cantidad de cada artículo debe ser mayor a cero")
		}
		if item.UnitPrice < 0 {
			return errors.New("el precio unitario no puede ser negativo")
		}
	}
	return nil
}
