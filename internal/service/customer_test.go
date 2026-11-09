package service

import (
	"context"
	"errors"
	"testing"

	appmw "github.com/andreyo/pos/internal/middleware"
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
func (m *mockCustomerRepo) GetPOSSummary(_ context.Context, _ int64) (*model.CustomerPosSummary, error) {
	return nil, nil
}
func (m *mockCustomerRepo) IsActive(_ context.Context, _ int64) (bool, error) {
	return m.customer != nil, m.getErr
}
func (m *mockCustomerRepo) GetOpeningBalance(_ context.Context, _ int64) (*model.CustomerOpeningBalance, error) {
	return nil, nil
}
func (m *mockCustomerRepo) CreateOpeningBalance(_ context.Context, ob *model.CustomerOpeningBalance) (*model.CustomerOpeningBalance, error) {
	return ob, nil
}
func (m *mockCustomerRepo) UpdateOpeningBalance(_ context.Context, ob *model.CustomerOpeningBalance, _ int64) (*model.CustomerOpeningBalance, error) {
	return ob, nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

// newCustomerSvc wires a register stub with no open register (US-082: the
// service now resolves the open register when creating an abono).
func newCustomerSvc(repo *mockCustomerRepo) *CustomerService {
	return NewCustomerService(repo, &stubRegisterRepo{})
}

func newCustomerSvcWithRegister(repo *mockCustomerRepo, reg *stubRegisterRepo) *CustomerService {
	return NewCustomerService(repo, reg)
}

func ctxWithRole(role string) context.Context {
	return context.WithValue(context.Background(), appmw.ClaimsKey, &model.Claims{UserID: 1, Username: role, Role: role})
}

func adminCtx() context.Context   { return ctxWithRole("admin") }
func cashierCtx() context.Context { return ctxWithRole("cashier") }

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

	p, err := svc.CreatePayment(adminCtx(), 1, 20000, "cash", "primer abono")
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

	_, err := svc.CreatePayment(adminCtx(), 1, 0, "cash", "")
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestCreatePayment_NegativeAmount(t *testing.T) {
	repo := &mockCustomerRepo{customer: existingCustomer(), balance: 50000}
	svc := newCustomerSvc(repo)

	_, err := svc.CreatePayment(adminCtx(), 1, -100, "cash", "")
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

	_, err := svc.CreatePayment(adminCtx(), 1, 50000, "cash", "")
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

	p, err := svc.CreatePayment(adminCtx(), 1, 30000, "cash", "pago total")
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

	_, err := svc.CreatePayment(adminCtx(), 99, 1000, "cash", "")
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

	_, err := svc.CreatePayment(adminCtx(), 1, 1000, "cash", "")
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

	_, err := svc.CreatePayment(adminCtx(), 1, 10000, "cash", "")
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
}

// ── Create/Update with notes and credit_limit ────────────────────────────────

func TestCreate_WithNotesAndCreditLimit(t *testing.T) {
	repo := &mockCustomerRepo{}
	svc := newCustomerSvc(repo)

	limit := int64(150000)
	c, err := svc.Create(context.Background(), "María", "3001111111", "Calle 1", "paga los viernes", &limit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Notes != "paga los viernes" {
		t.Errorf("expected notes 'paga los viernes', got %q", c.Notes)
	}
	if c.CreditLimit == nil || *c.CreditLimit != 150000 {
		t.Errorf("expected credit_limit 150000, got %v", c.CreditLimit)
	}
}

func TestCreate_WithoutCreditLimit(t *testing.T) {
	repo := &mockCustomerRepo{}
	svc := newCustomerSvc(repo)

	c, err := svc.Create(context.Background(), "Pedro", "", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.CreditLimit != nil {
		t.Errorf("expected nil credit_limit, got %v", c.CreditLimit)
	}
}

// ── Balance + credit_limit warning scenario ──────────────────────────────────
// The warning is computed in the frontend, but we verify Balance returns correct value.

func TestBalance_CreditLimitWarningScenario(t *testing.T) {
	// Cliente: tope $100.000, deuda actual $80.000, venta de $30.000
	// deuda + venta = $110.000 > $100.000 → frontend should warn
	repo := &mockCustomerRepo{
		customer: existingCustomer(),
		balance:  80000,
	}
	svc := newCustomerSvc(repo)

	balance, err := svc.Balance(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	creditLimit := int64(100000)
	saleTotal := float64(30000)
	if balance+saleTotal <= float64(creditLimit) {
		t.Error("expected balance+sale to exceed credit limit")
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

// ── US-082 contract §4, T-8 (AC-7) ──────────────────────────────────────────

func TestCreatePayment_T8_OpenRegisterLookupFailureStoresNothing(t *testing.T) {
	for _, role := range []string{"admin", "cashier"} {
		t.Run(role, func(t *testing.T) {
			repo := &mockCustomerRepo{customer: existingCustomer(), balance: 50000}
			reg := &stubRegisterRepo{getOpenErr: errors.New("database is locked")}
			svc := newCustomerSvcWithRegister(repo, reg)

			_, err := svc.CreatePayment(ctxWithRole(role), 1, 20000, "cash", "")
			if err == nil {
				t.Fatal("expected an error when the open-register lookup fails")
			}
			if repo.createdPay != nil {
				t.Errorf("repo CreatePayment was called (amount %.0f); nothing must be stored", repo.createdPay.Amount)
			}
		})
	}
}

// ── US-082 contract §4, T-9 (AC-4, AC-5, AC-6, AC-9) ─────────────────────────

func TestCreatePayment_T9_CashierWithoutOpenRegisterRejected(t *testing.T) {
	repo := &mockCustomerRepo{customer: existingCustomer(), balance: 50000}
	svc := newCustomerSvcWithRegister(repo, &stubRegisterRepo{}) // GetOpen -> ErrNotFound

	_, err := svc.CreatePayment(cashierCtx(), 1, 20000, "cash", "")
	if err == nil {
		t.Fatal("expected an error for a cashier with no open register")
	}
	if repo.createdPay != nil {
		t.Error("repo CreatePayment was called; nothing must be stored")
	}

	// Same input as admin succeeds unattributed (AC-5), so the rejection above
	// is the role rule and not some other validation.
	repoAdmin := &mockCustomerRepo{customer: existingCustomer(), balance: 50000}
	svcAdmin := newCustomerSvcWithRegister(repoAdmin, &stubRegisterRepo{})
	p, err := svcAdmin.CreatePayment(adminCtx(), 1, 20000, "cash", "")
	if err != nil {
		t.Fatalf("admin with no open register: unexpected error: %v", err)
	}
	if p.RegisterID != nil {
		t.Errorf("admin abono register_id = %d, want nil", *p.RegisterID)
	}
	if p.Method != "cash" || p.Amount != 20000 {
		t.Errorf("admin abono = %q/%.0f, want cash/20000", p.Method, p.Amount)
	}
}

func TestCreatePayment_T9_CashierWithOpenRegisterLinksRegister(t *testing.T) {
	repo := &mockCustomerRepo{customer: existingCustomer(), balance: 50000}
	reg := &stubRegisterRepo{openRegister: &model.Register{ID: 7, Status: "open", OpeningAmount: 100000}}
	svc := newCustomerSvcWithRegister(repo, reg)

	p, err := svc.CreatePayment(cashierCtx(), 1, 20000, "transfer", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createdPay == nil {
		t.Fatal("repo CreatePayment was not called")
	}
	if repo.createdPay.RegisterID == nil || *repo.createdPay.RegisterID != 7 {
		t.Errorf("stored register_id = %v, want 7", repo.createdPay.RegisterID)
	}
	if repo.createdPay.Method != "transfer" {
		t.Errorf("stored method = %q, want transfer", repo.createdPay.Method)
	}
	if p.Amount != 20000 {
		t.Errorf("amount = %.0f, want 20000", p.Amount)
	}
}

func TestCreatePayment_T9_CashierOverBalanceRejected(t *testing.T) {
	// Balance $50,000, abono $60,000 -> over-balance error, nothing stored.
	repo := &mockCustomerRepo{customer: existingCustomer(), balance: 50000}
	reg := &stubRegisterRepo{openRegister: &model.Register{ID: 1, Status: "open", OpeningAmount: 100000}}
	svc := newCustomerSvcWithRegister(repo, reg)

	_, err := svc.CreatePayment(cashierCtx(), 1, 60000, "cash", "")
	if err == nil {
		t.Fatal("expected an over-balance error")
	}
	if repo.createdPay != nil {
		t.Error("repo CreatePayment was called; nothing must be stored")
	}
}

func TestCreatePayment_T9_CashierZeroAmountRejected(t *testing.T) {
	repo := &mockCustomerRepo{customer: existingCustomer(), balance: 50000}
	reg := &stubRegisterRepo{openRegister: &model.Register{ID: 1, Status: "open", OpeningAmount: 100000}}
	svc := newCustomerSvcWithRegister(repo, reg)

	_, err := svc.CreatePayment(cashierCtx(), 1, 0, "cash", "")
	if err == nil {
		t.Fatal("expected an error for amount 0")
	}
	if repo.createdPay != nil {
		t.Error("repo CreatePayment was called; nothing must be stored")
	}
}
