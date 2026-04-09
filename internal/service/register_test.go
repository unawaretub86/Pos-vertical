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
}

func (s *stubRegisterRepo) Create(_ context.Context, r *model.Register) (*model.Register, error) {
	r.ID = 1
	s.created = r
	return r, nil
}
func (s *stubRegisterRepo) List(_ context.Context) ([]*model.Register, error) { return nil, nil }
func (s *stubRegisterRepo) GetByID(_ context.Context, _ int64) (*model.Register, error) {
	return nil, nil
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
func (s *stubRegisterRepo) SumCashSales(_ context.Context, _ int64) (float64, error) { return 0, nil }
func (s *stubRegisterRepo) SumCashOut(_ context.Context, _ int64) (float64, error)   { return 0, nil }
func (s *stubRegisterRepo) SumCashIn(_ context.Context, _ int64) (float64, error)    { return 0, nil }

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
