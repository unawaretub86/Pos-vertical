package service

import (
	"context"
	"testing"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

type stubProductRepo struct{}

func (s *stubProductRepo) Create(_ context.Context, p *model.Product) (*model.Product, error) {
	p.ID = 1
	return p, nil
}
func (s *stubProductRepo) List(_ context.Context, _ repository.ProductFilter) ([]*model.Product, error) {
	return nil, nil
}
func (s *stubProductRepo) GetByID(_ context.Context, _ int64) (*model.Product, error) { return nil, nil }
func (s *stubProductRepo) Update(_ context.Context, p *model.Product) (*model.Product, error) {
	return p, nil
}
func (s *stubProductRepo) Delete(_ context.Context, _ int64) error { return nil }
func (s *stubProductRepo) LowStock(_ context.Context) ([]*model.Product, error) {
	return nil, nil
}

func validProduct() *model.Product {
	return &model.Product{
		Name:      "Cafe",
		Type:      "product",
		SalePrice: 10,
		Unit:      "unit",
	}
}

func newProductSvc() *ProductService {
	return NewProductService(&stubProductRepo{})
}

func TestValidateProduct_Valid(t *testing.T) {
	if err := validateProduct(validProduct()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProduct_EmptyName(t *testing.T) {
	p := validProduct()
	p.Name = "   "
	if err := validateProduct(p); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestValidateProduct_TrimsName(t *testing.T) {
	p := validProduct()
	p.Name = "  Cafe  "
	if err := validateProduct(p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "Cafe" {
		t.Errorf("expected trimmed name 'Cafe', got %q", p.Name)
	}
}

func TestValidateProduct_NegativeSalePrice(t *testing.T) {
	p := validProduct()
	p.SalePrice = -1
	if err := validateProduct(p); err == nil {
		t.Fatal("expected error for negative price")
	}
}

func TestValidateProduct_ZeroPriceOK(t *testing.T) {
	p := validProduct()
	p.SalePrice = 0
	if err := validateProduct(p); err != nil {
		t.Fatalf("zero price should be allowed: %v", err)
	}
}

func TestValidateProduct_InvalidType(t *testing.T) {
	p := validProduct()
	p.Type = "digital"
	if err := validateProduct(p); err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestValidateProduct_ValidTypes(t *testing.T) {
	for _, typ := range []string{"product", "service"} {
		p := validProduct()
		p.Type = typ
		if err := validateProduct(p); err != nil {
			t.Errorf("type %q should be valid: %v", typ, err)
		}
	}
}

func TestValidateProduct_InvalidUnit(t *testing.T) {
	p := validProduct()
	p.Unit = "dozen"
	if err := validateProduct(p); err == nil {
		t.Fatal("expected error for invalid unit")
	}
}

func TestValidateProduct_ValidUnits(t *testing.T) {
	for _, u := range []string{"unit", "kg", "lb", "liter"} {
		p := validProduct()
		p.Unit = u
		if err := validateProduct(p); err != nil {
			t.Errorf("unit %q should be valid: %v", u, err)
		}
	}
}

func TestProductService_Create_OK(t *testing.T) {
	svc := newProductSvc()
	p, err := svc.Create(context.Background(), validProduct())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != 1 {
		t.Errorf("expected ID 1, got %d", p.ID)
	}
}

func TestProductService_Create_ValidationFails(t *testing.T) {
	svc := newProductSvc()
	p := validProduct()
	p.Name = ""
	if _, err := svc.Create(context.Background(), p); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestProductService_Update_ValidationFails(t *testing.T) {
	svc := newProductSvc()
	p := validProduct()
	p.Type = "invalid"
	if _, err := svc.Update(context.Background(), p); err == nil {
		t.Fatal("expected validation error on update")
	}
}
