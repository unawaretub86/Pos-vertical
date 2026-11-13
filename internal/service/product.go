package service

import (
	"context"
	"errors"
	"strings"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// ProductService handles product business logic.
type ProductService struct {
	repo repository.ProductRepository
}

// NewProductService creates a ProductService backed by the given repository.
func NewProductService(repo repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) Create(ctx context.Context, p *model.Product) (*model.Product, error) {
	if err := validateProduct(p); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, p)
}

func (s *ProductService) List(ctx context.Context, filter repository.ProductFilter) ([]*model.Product, error) {
	return s.repo.List(ctx, filter)
}

func (s *ProductService) GetByID(ctx context.Context, id int64) (*model.Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProductService) Update(ctx context.Context, p *model.Product) (*model.Product, error) {
	if err := validateProduct(p); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, p)
}

func (s *ProductService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *ProductService) GetByBarcode(ctx context.Context, barcode string) (*model.Product, error) {
	return s.repo.GetByBarcode(ctx, barcode)
}

func (s *ProductService) LowStock(ctx context.Context) ([]*model.Product, error) {
	return s.repo.LowStock(ctx)
}

func validateProduct(p *model.Product) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return errors.New("el nombre del producto es requerido")
	}
	if p.SalePrice < 0 {
		return errors.New("el precio de venta no puede ser negativo")
	}
	if !IsWholePeso(p.SalePrice) {
		return errors.New("el precio de venta debe ser un número entero de pesos")
	}
	if !IsWholePeso(p.Cost) {
		return errors.New("el costo debe ser un número entero de pesos")
	}
	if p.WholesalePrice != nil && !IsWholePeso(*p.WholesalePrice) {
		return errors.New("el precio mayorista debe ser un número entero de pesos")
	}
	if p.Type != "product" && p.Type != "service" {
		return errors.New("tipo inválido: debe ser 'product' o 'service'")
	}
	validUnits := map[string]bool{"unit": true, "kg": true, "lb": true, "liter": true}
	if !validUnits[p.Unit] {
		return errors.New("unidad inválida: debe ser unit, kg, lb o liter")
	}
	return nil
}
