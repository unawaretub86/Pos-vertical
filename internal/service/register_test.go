package service

import (
	"context"
	"testing"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// stubRegisterRepo is a minimal stub for RegisterService tests.
type stubRegisterRepo struct {
	openRegister *model.Register // nil means no open register
	created      *model.Register
	cashSales    float64
	cashIn       float64
	cashOut      float64
}

func (s *stubRegisterRepo) Create(_ context.Context, r *model.Register) (*model.Register, error) {
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
	if s.openRegister == nil {
		return nil, repository.ErrNotFound
	}
	return s.openRegister, nil
}
func (s *stubRegisterRepo) Close(_ context.Context, _ int64, _, _, _ float64) (*model.Register, error) {
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

func (s *stubRegisterRepo) SumByPaymentMethod(_ context.Context, _ int64) ([]repository.PaymentMethodTotal, error) {
	return nil, nil
}


func TestRegisterService_Open_AlreadyOpen_ReturnsError(t *testing.T) {
	// A register is already open — opening a second one must fail.
	repo := &stubRegisterRepo{openRegister: &model.Register{ID: 1, Status: "open"}}
	svc := NewRegisterService(repo)

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
	svc := NewRegisterService(repo)

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
	svc := NewRegisterService(repo)

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
	svc := NewRegisterService(repo)

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
	svc := NewRegisterService(repo)

	summary, err := svc.Close(context.Background(), 1, 320)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Expected != 320 {
		t.Errorf("expected expected_close=320, got %.2f", summary.Expected)
	}
}
