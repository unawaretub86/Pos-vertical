package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreyo/pos/internal/model"
)

// ── mock customer repo ──────────────────────────────────────────────────────

type mockCustomerRepo struct {
	customer    *model.Customer
	getErr      error
	balance     float64
	balanceErr  error
	payments    []*model.CustomerPayment
	createdPay  *model.CustomerPayment
	createPayErr error
	creditSales []*model.Sale
}

func (m *mockCustomerRepo) List(_ context.Context, _ string) ([]*model.CustomerWithBalance, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByID(_ context.Context, _ int64) (*model.Customer, error) {
	return m.customer, m.getErr
}
func (m *mockCustomerRepo) Create(_ context.Context, c *model.Customer) (*model.Customer, error) {
	c.ID = 1
	return c, nil
}
func (m *mockCustomerRepo) Update(_ context.Context, c *model.Customer) (*model.Customer, error) {
	return c, nil
}
func (m *mockCustomerRepo) Delete(_ context.Context, _ int64) error { return nil }
func (m *mockCustomerRepo) Balance(_ context.Context, _ int64) (float64, error) {
	return m.balance, m.balanceErr
}
func (m *mockCustomerRepo) TotalPortfolioBalance(_ context.Context) (float64, error) {
	return 0, nil
}
func (m *mockCustomerRepo) ListPayments(_ context.Context, _ int64) ([]*model.CustomerPayment, error) {
	return m.payments, nil
}
func (m *mockCustomerRepo) CreatePayment(_ context.Context, p *model.CustomerPayment) (*model.CustomerPayment, error) {
	if m.createPayErr != nil {
		return nil, m.createPayErr
	}
	p.ID = 1
	m.createdPay = p
	return p, nil
}
func (m *mockCustomerRepo) ListCreditSales(_ context.Context, _ int64) ([]*model.Sale, error) {
	return m.creditSales, nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

func newCustomerSvc(repo *mockCustomerRepo) *CustomerService {
	return NewCustomerService(repo)
}

func existingCustomer() *model.Customer {
	return &model.Customer{ID: 1, Name: "Carlos"}
}

// ── CreatePayment tests ──────────────────────────────────────────────────────

func TestCreatePayment_OK(t *testing.T) {
	repo := &mockCustomerRepo{
		customer: existingCustomer(),
		balance:  50000,
	}
	svc := newCustomerSvc(repo)

	p, err := svc.CreatePayment(context.Background(), 1, 20000, "primer abono")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Amount != 20000 {
		t.Errorf("expected amount 20000, got %v", p.Amount)
	}
	if p.Note != "primer abono" {
		t.Errorf("expected note 'primer abono', got %q", p.Note)
	}
}

func TestCreatePayment_ZeroAmount(t *testing.T) {
	repo := &mockCustomerRepo{customer: existingCustomer(), balance: 50000}
	svc := newCustomerSvc(repo)

	_, err := svc.CreatePayment(context.Background(), 1, 0, "")
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestCreatePayment_NegativeAmount(t *testing.T) {
	repo := &mockCustomerRepo{customer: existingCustomer(), balance: 50000}
	svc := newCustomerSvc(repo)

	_, err := svc.CreatePayment(context.Background(), 1, -100, "")
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestCreatePayment_ExceedsBalance(t *testing.T) {
	repo := &mockCustomerRepo{
		customer: existingCustomer(),
		balance:  10000,
	}
	svc := newCustomerSvc(repo)

	_, err := svc.CreatePayment(context.Background(), 1, 50000, "")
	if err == nil {
		t.Fatal("expected error when amount exceeds balance")
	}
}

func TestCreatePayment_ExactBalance(t *testing.T) {
	repo := &mockCustomerRepo{
		customer: existingCustomer(),
		balance:  30000,
	}
	svc := newCustomerSvc(repo)

	p, err := svc.CreatePayment(context.Background(), 1, 30000, "pago total")
	if err != nil {
		t.Fatalf("unexpected error for exact balance payment: %v", err)
	}
	if p.Amount != 30000 {
		t.Errorf("expected 30000, got %v", p.Amount)
	}
}

func TestCreatePayment_CustomerNotFound(t *testing.T) {
	repo := &mockCustomerRepo{
		getErr: errors.New("not found"),
	}
	svc := newCustomerSvc(repo)

	_, err := svc.CreatePayment(context.Background(), 99, 1000, "")
	if err == nil {
		t.Fatal("expected error for non-existent customer")
	}
}

func TestCreatePayment_BalanceError(t *testing.T) {
	repo := &mockCustomerRepo{
		customer:   existingCustomer(),
		balanceErr: errors.New("db error"),
	}
	svc := newCustomerSvc(repo)

	_, err := svc.CreatePayment(context.Background(), 1, 1000, "")
	if err == nil {
		t.Fatal("expected error when balance query fails")
	}
}

func TestCreatePayment_RepoError(t *testing.T) {
	repo := &mockCustomerRepo{
		customer:    existingCustomer(),
		balance:     50000,
		createPayErr: errors.New("db error"),
	}
	svc := newCustomerSvc(repo)

	_, err := svc.CreatePayment(context.Background(), 1, 10000, "")
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
}

// ── ListPayments tests ───────────────────────────────────────────────────────

func TestListPayments_ReturnsAll(t *testing.T) {
	repo := &mockCustomerRepo{
		payments: []*model.CustomerPayment{
			{ID: 1, CustomerID: 1, Amount: 10000, Note: "abono 1"},
			{ID: 2, CustomerID: 1, Amount: 20000, Note: "abono 2"},
		},
	}
	svc := newCustomerSvc(repo)

	payments, err := svc.ListPayments(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payments) != 2 {
		t.Errorf("expected 2 payments, got %d", len(payments))
	}
}

func TestListPayments_Empty(t *testing.T) {
	repo := &mockCustomerRepo{payments: nil}
	svc := newCustomerSvc(repo)

	payments, err := svc.ListPayments(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payments) != 0 {
		t.Errorf("expected 0 payments, got %d", len(payments))
	}
}
