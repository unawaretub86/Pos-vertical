package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// stubRegisterRepo is a minimal stub for RegisterService tests. It is also the
// register dependency of CustomerService in customer_test.go (US-082).
type stubRegisterRepo struct {
	openRegister  *model.Register // nil means no open register
	getOpenErr    error           // when set, GetOpen returns it (US-082 T-8)
	created       *model.Register
	cashSales     float64
	cashIn        float64
	cashOut       float64
	cashAbonos    float64 // US-082
	cashAbonosErr error   // US-082 T-7: failing abono sum

	// Recorded by Close (US-082 T-7).
	closeCalled    bool
	closedExpected float64
	closedDiff     float64
	closedActual   float64
}

func (s *stubRegisterRepo) Create(_ context.Context, r *model.Register, mv *model.RegisterMovement) (*model.Register, error) {
	r.ID = 1
	s.created = r
	return r, nil
}
func (s *stubRegisterRepo) List(_ context.Context) ([]*model.Register, error) { return nil, nil }
func (s *stubRegisterRepo) GetByID(_ context.Context, _ int64) (*model.Register, error) {
	if s.openRegister != nil {
		return s.openRegister, nil
	}
	return &model.Register{ID: 1, Status: "open", OpeningAmount: 100}, nil
}
func (s *stubRegisterRepo) GetOpen(_ context.Context) (*model.Register, error) {
	if s.getOpenErr != nil {
		return nil, s.getOpenErr
	}
	if s.openRegister == nil {
		return nil, repository.ErrNotFound
	}
	return s.openRegister, nil
}
func (s *stubRegisterRepo) Close(_ context.Context, _ int64, actual, expected, difference float64) (*model.Register, error) {
	s.closeCalled = true
	s.closedActual = actual
	s.closedExpected = expected
	s.closedDiff = difference
	return &model.Register{Status: "closed"}, nil
}
func (s *stubRegisterRepo) AddMovement(_ context.Context, mv *model.RegisterMovement) (*model.RegisterMovement, error) {
	return mv, nil
}
func (s *stubRegisterRepo) ListMovements(_ context.Context, _ int64) ([]*model.RegisterMovement, error) {
	return nil, nil
}
func (s *stubRegisterRepo) SumCashSales(_ context.Context, _ int64) (float64, error) {
	return s.cashSales, nil
}
func (s *stubRegisterRepo) SumCashOut(_ context.Context, _ int64) (float64, error) {
	return s.cashOut, nil
}
func (s *stubRegisterRepo) SumCashIn(_ context.Context, _ int64) (float64, error) {
	return s.cashIn, nil
}
func (s *stubRegisterRepo) SumCashAbonos(_ context.Context, _ int64) (float64, error) {
	if s.cashAbonosErr != nil {
		return 0, s.cashAbonosErr
	}
	return s.cashAbonos, nil
}

func (s *stubRegisterRepo) SumByPaymentMethod(_ context.Context, _ int64) ([]repository.PaymentMethodTotal, error) {
	return nil, nil
}
func (s *stubRegisterRepo) CountOpenOrders(_ context.Context) (int, error) {
	return 0, nil
}

func TestRegisterService_Open_AlreadyOpen_ReturnsError(t *testing.T) {
	// A register is already open — opening a second one must fail.
	repo := &stubRegisterRepo{openRegister: &model.Register{ID: 1, Status: "open"}}
	svc := NewRegisterService(repo, nil, nil)

	_, err := svc.Open(context.Background(), 42, 100)
	if err == nil {
		t.Fatal("expected error when opening a second register")
	}
	if err != ErrRegisterAlreadyOpen {
		t.Errorf("expected ErrRegisterAlreadyOpen, got: %v", err)
	}
}

func TestRegisterService_Open_NoExistingRegister_Succeeds(t *testing.T) {
	repo := &stubRegisterRepo{} // GetOpen returns ErrNotFound
	svc := NewRegisterService(repo, nil, nil)

	reg, err := svc.Open(context.Background(), 42, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg == nil {
		t.Fatal("expected a register to be returned")
	}
}

func TestRegisterService_Summary_IncludesCashIn(t *testing.T) {
	// opening=100, cashSales=200, cashIn=50, cashOut=30
	// expected = 100 + 200 + 50 - 30 = 320
	repo := &stubRegisterRepo{
		openRegister: &model.Register{ID: 1, Status: "open", OpeningAmount: 100},
		cashSales:    200,
		cashIn:       50,
		cashOut:      30,
	}
	svc := NewRegisterService(repo, nil, nil)

	summary, err := svc.Summary(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Expected != 320 {
		t.Errorf("expected 320, got %.2f", summary.Expected)
	}
}

func TestRegisterService_Summary_NoCashIn_Unchanged(t *testing.T) {
	// opening=500, cashSales=100, cashIn=0, cashOut=50
	// expected = 500 + 100 + 0 - 50 = 550
	repo := &stubRegisterRepo{
		openRegister: &model.Register{ID: 1, Status: "open", OpeningAmount: 500},
		cashSales:    100,
		cashOut:      50,
	}
	svc := NewRegisterService(repo, nil, nil)

	summary, err := svc.Summary(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Expected != 550 {
		t.Errorf("expected 550, got %.2f", summary.Expected)
	}
}

func TestRegisterService_Close_ExpectedIncludesCashIn(t *testing.T) {
	// opening=100, cashSales=200, cashIn=50, cashOut=30 → expected=320
	// actual=320 → difference=0
	repo := &stubRegisterRepo{
		openRegister: &model.Register{ID: 1, Status: "open", OpeningAmount: 100},
		cashSales:    200,
		cashIn:       50,
		cashOut:      30,
	}
	svc := NewRegisterService(repo, nil, nil)

	summary, err := svc.Close(context.Background(), 1, 320)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Expected != 320 {
		t.Errorf("expected expected_close=320, got %.2f", summary.Expected)
	}
}

// ─── US-082 contract §4, T-7 (AC-10, AC-12) ──────────────────────────────────

func t7Repo() *stubRegisterRepo {
	return &stubRegisterRepo{
		openRegister: &model.Register{ID: 1, Status: "open", OpeningAmount: 100000},
		cashSales:    30000,
		cashAbonos:   20000,
		cashIn:       5000,
		cashOut:      15000,
	}
}

func TestRegisterService_T7_SummaryExpectedIncludesCashAbonos(t *testing.T) {
	// expected = 100,000 + 30,000 + 20,000 + 5,000 - 15,000 = 140,000
	svc := NewRegisterService(t7Repo(), nil, nil)

	summary, err := svc.Summary(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.CashAbonos != 20000 {
		t.Errorf("CashAbonos = %.0f, want 20000", summary.CashAbonos)
	}
	if summary.Expected != 140000 {
		t.Errorf("Expected = %.0f, want 140000", summary.Expected)
	}
}

func TestRegisterService_T7_CloseStoresExpectedWithCashAbonos(t *testing.T) {
	// expected 140,000; actual 145,000 -> difference +5,000
	repo := t7Repo()
	svc := NewRegisterService(repo, nil, nil)

	summary, err := svc.Close(context.Background(), 1, 145000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.closeCalled {
		t.Fatal("repo Close was not called")
	}
	if repo.closedExpected != 140000 {
		t.Errorf("stored expected_close = %.0f, want 140000", repo.closedExpected)
	}
	if repo.closedDiff != 5000 {
		t.Errorf("stored difference = %.0f, want 5000", repo.closedDiff)
	}
	if repo.closedActual != 145000 {
		t.Errorf("stored actual_close = %.0f, want 145000", repo.closedActual)
	}
	if summary.Expected != 140000 || summary.CashAbonos != 20000 {
		t.Errorf("summary expected/cash_abonos = %.0f/%.0f, want 140000/20000", summary.Expected, summary.CashAbonos)
	}
}

func TestRegisterService_T7_CloseFailsWhenAbonoSumFails(t *testing.T) {
	repo := t7Repo()
	repo.cashAbonosErr = errors.New("db error")
	svc := NewRegisterService(repo, nil, nil)

	if _, err := svc.Close(context.Background(), 1, 145000); err == nil {
		t.Fatal("Close returned nil error, want error when the abono sum fails")
	}
	if repo.closeCalled {
		t.Error("repo Close was called; the register must stay open when the abono sum fails")
	}
}

func TestRegisterService_T7_SummaryFailsWhenAbonoSumFails(t *testing.T) {
	repo := t7Repo()
	repo.cashAbonosErr = errors.New("db error")
	svc := NewRegisterService(repo, nil, nil)

	if _, err := svc.Summary(context.Background(), 1); err == nil {
		t.Fatal("Summary returned nil error, want error when the abono sum fails")
	}
}
