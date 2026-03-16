package service

import (
	"context"
	"testing"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// --- mocks ---

type mockSaleRepo struct {
	created *model.Sale
}

func (m *mockSaleRepo) Create(_ context.Context, sale *model.Sale) (*model.Sale, error) {
	sale.ID = 1
	m.created = sale
	return sale, nil
}
func (m *mockSaleRepo) List(_ context.Context, _ repository.SaleFilter) ([]*model.Sale, error) {
	return nil, nil
}
func (m *mockSaleRepo) GetByID(_ context.Context, _ int64) (*model.Sale, error) {
	return m.created, nil
}

type mockProductRepo struct{}

func (m *mockProductRepo) Create(_ context.Context, p *model.Product) (*model.Product, error) {
	return p, nil
}
func (m *mockProductRepo) List(_ context.Context, _ repository.ProductFilter) ([]*model.Product, error) {
	return nil, nil
}
func (m *mockProductRepo) GetByID(_ context.Context, _ int64) (*model.Product, error) {
	return nil, nil
}
func (m *mockProductRepo) Update(_ context.Context, p *model.Product) (*model.Product, error) {
	return p, nil
}
func (m *mockProductRepo) Delete(_ context.Context, _ int64) error { return nil }

// --- helpers ---

func validSale() *model.Sale {
	return &model.Sale{
		PaymentMethod: "cash",
		Items: []model.SaleItem{
			{ProductID: 1, Quantity: 2, UnitPrice: 10},
		},
	}
}

func newSvc() *SaleService {
	return NewSaleService(&mockSaleRepo{}, &mockProductRepo{})
}

// --- validateSale tests ---

func TestValidateSale_Valid(t *testing.T) {
	if err := validateSale(validSale()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateSale_InvalidPaymentMethod(t *testing.T) {
	s := validSale()
	s.PaymentMethod = "bitcoin"
	if err := validateSale(s); err == nil {
		t.Fatal("expected error for invalid payment method")
	}
}

func TestValidateSale_CreditRequiresCustomer(t *testing.T) {
	s := validSale()
	s.PaymentMethod = "credit"
	s.CustomerID = nil
	if err := validateSale(s); err == nil {
		t.Fatal("expected error: credit without customer")
	}
}

func TestValidateSale_CreditWithCustomerOK(t *testing.T) {
	s := validSale()
	s.PaymentMethod = "credit"
	id := int64(5)
	s.CustomerID = &id
	if err := validateSale(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSale_NoItems(t *testing.T) {
	s := validSale()
	s.Items = nil
	if err := validateSale(s); err == nil {
		t.Fatal("expected error for empty items")
	}
}

func TestValidateSale_ZeroQuantity(t *testing.T) {
	s := validSale()
	s.Items[0].Quantity = 0
	if err := validateSale(s); err == nil {
		t.Fatal("expected error for zero quantity")
	}
}

func TestValidateSale_NegativeUnitPrice(t *testing.T) {
	s := validSale()
	s.Items[0].UnitPrice = -1
	if err := validateSale(s); err == nil {
		t.Fatal("expected error for negative unit price")
	}
}

// --- SaleService.Create tests ---

func TestSaleService_Create_SetsDefaultStatus(t *testing.T) {
	svc := newSvc()
	sale, err := svc.Create(context.Background(), validSale())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sale.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", sale.Status)
	}
}

func TestSaleService_Create_PreservesExistingStatus(t *testing.T) {
	svc := newSvc()
	s := validSale()
	s.Status = "pending"
	sale, err := svc.Create(context.Background(), s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sale.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", sale.Status)
	}
}

func TestSaleService_Create_InvalidSaleReturnsError(t *testing.T) {
	svc := newSvc()
	s := validSale()
	s.PaymentMethod = ""
	if _, err := svc.Create(context.Background(), s); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSaleService_Create_AllPaymentMethods(t *testing.T) {
	methods := []string{"cash", "transfer", "card", "credit"}
	for _, m := range methods {
		s := validSale()
		s.PaymentMethod = m
		if m == "credit" {
			id := int64(1)
			s.CustomerID = &id
		}
		svc := newSvc()
		if _, err := svc.Create(context.Background(), s); err != nil {
			t.Errorf("payment method %q should be valid, got: %v", m, err)
		}
	}
}
