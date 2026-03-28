package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// --- mocks ---

type mockSaleRepo struct {
	created   *model.Sale
	createErr error
}

func (m *mockSaleRepo) Create(_ context.Context, sale *model.Sale) (*model.Sale, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
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
func (m *mockSaleRepo) Void(_ context.Context, _ int64, _ int64) (*model.Sale, error) {
	return m.created, nil
}

type mockProductRepo struct {
	taxRate float64 // TaxRate returned by GetByID; defaults to 0
}

func (m *mockProductRepo) Create(_ context.Context, p *model.Product) (*model.Product, error) {
	return p, nil
}
func (m *mockProductRepo) List(_ context.Context, _ repository.ProductFilter) ([]*model.Product, error) {
	return nil, nil
}
func (m *mockProductRepo) GetByID(_ context.Context, id int64) (*model.Product, error) {
	return &model.Product{ID: id, TaxRate: m.taxRate, Type: "product", Stock: 9999}, nil
}
func (m *mockProductRepo) Update(_ context.Context, p *model.Product) (*model.Product, error) {
	return p, nil
}
func (m *mockProductRepo) Delete(_ context.Context, _ int64) error { return nil }
func (m *mockProductRepo) LowStock(_ context.Context) ([]*model.Product, error) {
	return nil, nil
}

type mockRegisterRepo struct {
	openRegister *model.Register
	openErr      error
}

func (m *mockRegisterRepo) Create(_ context.Context, r *model.Register) (*model.Register, error) {
	return r, nil
}
func (m *mockRegisterRepo) List(_ context.Context) ([]*model.Register, error) { return nil, nil }
func (m *mockRegisterRepo) GetByID(_ context.Context, _ int64) (*model.Register, error) {
	return nil, nil
}
func (m *mockRegisterRepo) GetOpen(_ context.Context) (*model.Register, error) {
	if m.openErr != nil {
		return nil, m.openErr
	}
	return m.openRegister, nil
}
func (m *mockRegisterRepo) Close(_ context.Context, _ int64, _, _, _ float64) (*model.Register, error) {
	return nil, nil
}
func (m *mockRegisterRepo) AddMovement(_ context.Context, mv *model.RegisterMovement) (*model.RegisterMovement, error) {
	return mv, nil
}
func (m *mockRegisterRepo) ListMovements(_ context.Context, _ int64) ([]*model.RegisterMovement, error) {
	return nil, nil
}
func (m *mockRegisterRepo) SumCashSales(_ context.Context, _ int64) (float64, error) { return 0, nil }
func (m *mockRegisterRepo) SumCashOut(_ context.Context, _ int64) (float64, error)   { return 0, nil }
func (m *mockRegisterRepo) SumCashIn(_ context.Context, _ int64) (float64, error)    { return 0, nil }

// --- helpers ---

func openRegister() *model.Register {
	id := int64(10)
	_ = id
	return &model.Register{ID: 10, Status: "open"}
}

func validSale() *model.Sale {
	return &model.Sale{
		PaymentMethod: "cash",
		Total:         20,
		Items: []model.SaleItem{
			{ProductID: 1, Quantity: 2, UnitPrice: 10},
		},
	}
}

func newSvc() *SaleService {
	return NewSaleService(
		&mockSaleRepo{},
		&mockProductRepo{},
		&mockRegisterRepo{openRegister: openRegister()},
	)
}

func newSvcNoRegister() *SaleService {
	return NewSaleService(
		&mockSaleRepo{},
		&mockProductRepo{},
		&mockRegisterRepo{openErr: repository.ErrNotFound},
	)
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

// --- New required tests ---

func TestSaleService_Create_AssignsOpenRegisterID(t *testing.T) {
	svc := newSvc()
	sale, err := svc.Create(context.Background(), validSale())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sale.RegisterID == nil || *sale.RegisterID != 10 {
		t.Errorf("expected register_id 10, got %v", sale.RegisterID)
	}
}

func TestSaleService_Create_NoOpenRegister_ReturnsError(t *testing.T) {
	svc := newSvcNoRegister()
	_, err := svc.Create(context.Background(), validSale())
	if err == nil {
		t.Fatal("expected error: no open register")
	}
	if err.Error() != "no hay caja abierta" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSaleService_Create_InconsistentTotal_ReturnsError(t *testing.T) {
	svc := newSvc()
	s := validSale()
	// items sum to 20, but frontend sends 50
	s.Total = 50
	_, err := svc.Create(context.Background(), s)
	if err == nil {
		t.Fatal("expected error: inconsistent total")
	}
}

func TestSaleService_Create_RecalculatesTotalCorrectly(t *testing.T) {
	svc := newSvc()
	s := validSale()
	// 2 items * $10 = $20 subtotal, $0 discount, $0 tax → total = 20
	s.Total = 20
	sale, err := svc.Create(context.Background(), s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sale.Total != 20 {
		t.Errorf("expected total 20, got %v", sale.Total)
	}
	if sale.Subtotal != 20 {
		t.Errorf("expected subtotal 20, got %v", sale.Subtotal)
	}
}

func TestSaleService_Create_CreditWithCustomer_Succeeds(t *testing.T) {
	svc := newSvc()
	s := validSale()
	s.PaymentMethod = "credit"
	id := int64(5)
	s.CustomerID = &id
	sale, err := svc.Create(context.Background(), s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sale.PaymentMethod != "credit" {
		t.Errorf("expected payment method 'credit', got %q", sale.PaymentMethod)
	}
}

func TestSaleService_Create_CreditWithoutCustomer_ReturnsError(t *testing.T) {
	svc := newSvc()
	s := validSale()
	s.PaymentMethod = "credit"
	s.CustomerID = nil
	_, err := svc.Create(context.Background(), s)
	if err == nil {
		t.Fatal("expected error: credit without customer")
	}
}

func TestSaleService_Create_RepoError_Propagates(t *testing.T) {
	svc := NewSaleService(
		&mockSaleRepo{createErr: errors.New("db failure")},
		&mockProductRepo{},
		&mockRegisterRepo{openRegister: openRegister()},
	)
	_, err := svc.Create(context.Background(), validSale())
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

// --- B3: IVA / tax_rate tests ---

func TestSaleService_Create_TaxCalculatedServerSide(t *testing.T) {
	// Product has 19% IVA. 2 units × $100 = $200 subtotal → tax = $38, total = $238.
	svc := NewSaleService(
		&mockSaleRepo{},
		&mockProductRepo{taxRate: 0.19},
		&mockRegisterRepo{openRegister: openRegister()},
	)
	sale := &model.Sale{
		PaymentMethod: "cash",
		Total:         238, // frontend sends correct total
		Items: []model.SaleItem{
			{ProductID: 1, Quantity: 2, UnitPrice: 100},
		},
	}
	result, err := svc.Create(context.Background(), sale)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Tax != 38 {
		t.Errorf("expected tax 38, got %.2f", result.Tax)
	}
	if result.Total != 238 {
		t.Errorf("expected total 238, got %.2f", result.Total)
	}
	if result.Items[0].TaxRate != 0.19 {
		t.Errorf("expected item tax_rate 0.19, got %.2f", result.Items[0].TaxRate)
	}
}

func TestSaleService_Create_FrontendTaxZeroOverriddenByBackend(t *testing.T) {
	// Frontend sends tax=0 but product has 19% IVA — backend must override.
	svc := NewSaleService(
		&mockSaleRepo{},
		&mockProductRepo{taxRate: 0.19},
		&mockRegisterRepo{openRegister: openRegister()},
	)
	sale := &model.Sale{
		PaymentMethod: "cash",
		Tax:           0,    // frontend says no tax
		Total:         238,  // but sends correct total anyway
		Items: []model.SaleItem{
			{ProductID: 1, Quantity: 2, UnitPrice: 100},
		},
	}
	result, err := svc.Create(context.Background(), sale)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Tax != 38 {
		t.Errorf("expected tax overridden to 38, got %.2f", result.Tax)
	}
}

func TestSaleService_Create_ServiceTaxZeroProductNoIVA(t *testing.T) {
	// Product with tax_rate=0 → tax is 0, total equals subtotal.
	svc := NewSaleService(
		&mockSaleRepo{},
		&mockProductRepo{taxRate: 0},
		&mockRegisterRepo{openRegister: openRegister()},
	)
	sale := &model.Sale{
		PaymentMethod: "cash",
		Total:         20,
		Items: []model.SaleItem{
			{ProductID: 1, Quantity: 2, UnitPrice: 10},
		},
	}
	result, err := svc.Create(context.Background(), sale)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Tax != 0 {
		t.Errorf("expected tax 0, got %.2f", result.Tax)
	}
	if result.Items[0].TaxRate != 0 {
		t.Errorf("expected item tax_rate 0, got %.2f", result.Items[0].TaxRate)
	}
}
