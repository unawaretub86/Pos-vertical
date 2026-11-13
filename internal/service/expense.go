package service

import (
	"context"
	"errors"
	"time"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

var (
	ErrInvalidExpenseAmount   = errors.New("el monto debe ser mayor a 0")
	ErrInvalidExpenseCategory = errors.New("categoría inválida")
)

var validExpenseCategories = map[string]bool{
	"servicios": true, "arriendo": true, "nomina": true, "proveedor": true, "otro": true,
}

// ExpenseService records business expenses and, when a register is open,
// mirrors them as a cash_out register movement so the closing arqueo picks
// them up without duplicating cash-flow logic.
type ExpenseService struct {
	repo      repository.ExpenseRepository
	registers *RegisterService
}

// NewExpenseService creates a new ExpenseService.
func NewExpenseService(repo repository.ExpenseRepository, registers *RegisterService) *ExpenseService {
	return &ExpenseService{repo: repo, registers: registers}
}

// Create records an expense. If a register is currently open, it is
// associated automatically and a matching cash_out movement is recorded.
func (s *ExpenseService) Create(ctx context.Context, userID int64, category string, amount float64, note string) (*model.Expense, error) {
	if amount <= 0 {
		return nil, ErrInvalidExpenseAmount
	}
	if !IsWholePeso(amount) {
		return nil, ErrNotWholePeso
	}
	if !validExpenseCategories[category] {
		return nil, ErrInvalidExpenseCategory
	}

	expense := &model.Expense{
		UserID:   userID,
		Category: category,
		Amount:   amount,
		Note:     note,
	}

	reg, err := s.registers.GetOpen(ctx)
	if err == nil && reg != nil {
		expense.RegisterID = &reg.ID
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	created, err := s.repo.Create(ctx, expense)
	if err != nil {
		return nil, err
	}

	if expense.RegisterID != nil {
		if _, err := s.registers.AddMovement(ctx, *expense.RegisterID, "cash_out", amount, "Gasto: "+category+describeNote(note)); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func describeNote(note string) string {
	if note == "" {
		return ""
	}
	return " — " + note
}

// List returns expenses matching the filter.
func (s *ExpenseService) List(ctx context.Context, filter repository.ExpenseFilter) ([]*model.Expense, error) {
	return s.repo.List(ctx, filter)
}

// SumRange returns the total expenses in [from, to).
func (s *ExpenseService) SumRange(ctx context.Context, from, to time.Time) (float64, error) {
	return s.repo.SumRange(ctx, from, to)
}
