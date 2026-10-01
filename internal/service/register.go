package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreyo/pos/internal/mailer"
	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

var (
	ErrRegisterAlreadyOpen = errors.New("ya hay una caja abierta")
	ErrRegisterNotOpen     = errors.New("no hay una caja abierta")
	ErrRegisterClosed      = errors.New("la caja ya está cerrada")
)

// RegisterSummary aggregates the register state for display.
type RegisterSummary struct {
	Register        *model.Register              `json:"register"`
	CashSales       float64                      `json:"cash_sales"`
	CashIn          float64                      `json:"cash_in"`
	CashOut         float64                      `json:"cash_out"`
	Expected        float64                      `json:"expected"`
	Movements       []*model.RegisterMovement       `json:"movements"`
	ByPaymentMethod []repository.PaymentMethodTotal `json:"by_payment_method"`
	OpenOrdersCount int                             `json:"open_orders_count"`
}

// RegisterService handles business logic for cash registers.
type RegisterService struct {
	repo      repository.RegisterRepository
	userRepo  repository.UserRepository
	setupRepo *repository.SetupRepository
}

// NewRegisterService creates a new RegisterService.
func NewRegisterService(repo repository.RegisterRepository, userRepo repository.UserRepository, setupRepo *repository.SetupRepository) *RegisterService {
	return &RegisterService{repo: repo, userRepo: userRepo, setupRepo: setupRepo}
}

// Open creates a new register session. Fails if one is already open.
func (s *RegisterService) Open(ctx context.Context, userID int64, openingAmount float64) (*model.Register, error) {
	_, err := s.repo.GetOpen(ctx)
	if err == nil {
		return nil, ErrRegisterAlreadyOpen
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("check open register: %w", err)
	}

	reg, err := s.repo.Create(ctx, &model.Register{
		UserID:        userID,
		OpeningAmount: openingAmount,
	}, &model.RegisterMovement{
		Type:        "opening",
		Amount:      openingAmount,
		Description: "Apertura de caja",
	})
	if err != nil {
		return nil, fmt.Errorf("crear caja: %w", err)
	}

	return reg, nil
}

// GetOpen returns the current open register.
func (s *RegisterService) GetOpen(ctx context.Context) (*model.Register, error) {
	return s.repo.GetOpen(ctx)
}

// List returns all registers.
func (s *RegisterService) List(ctx context.Context) ([]*model.Register, error) {
	return s.repo.List(ctx)
}

// GetByID returns a register by ID.
func (s *RegisterService) GetByID(ctx context.Context, id int64) (*model.Register, error) {
	return s.repo.GetByID(ctx, id)
}

// Summary returns the full state of a register.
func (s *RegisterService) Summary(ctx context.Context, id int64) (*RegisterSummary, error) {
	reg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	cashSales, err := s.repo.SumCashSales(ctx, id)
	if err != nil {
		return nil, err
	}
	cashIn, err := s.repo.SumCashIn(ctx, id)
	if err != nil {
		return nil, err
	}
	cashOut, err := s.repo.SumCashOut(ctx, id)
	if err != nil {
		return nil, err
	}
	movements, err := s.repo.ListMovements(ctx, id)
	if err != nil {
		return nil, err
	}
	byMethod, err := s.repo.SumByPaymentMethod(ctx, id)
	if err != nil {
		return nil, err
	}
	openOrders, _ := s.repo.CountOpenOrders(ctx)
	expected := reg.OpeningAmount + cashSales + cashIn - cashOut
	return &RegisterSummary{
		Register:        reg,
		CashSales:       cashSales,
		CashIn:          cashIn,
		CashOut:         cashOut,
		Expected:        expected,
		Movements:       movements,
		ByPaymentMethod: byMethod,
		OpenOrdersCount: openOrders,
	}, nil
}

// Close closes the register with the actual cash counted.
func (s *RegisterService) Close(ctx context.Context, id int64, actualClose float64) (*RegisterSummary, error) {
	reg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if reg.Status == "closed" {
		return nil, ErrRegisterClosed
	}

	cashSales, _ := s.repo.SumCashSales(ctx, id)
	cashIn, _ := s.repo.SumCashIn(ctx, id)
	cashOut, _ := s.repo.SumCashOut(ctx, id)
	expected := reg.OpeningAmount + cashSales + cashIn - cashOut
	difference := actualClose - expected

	closed, err := s.repo.Close(ctx, id, actualClose, expected, difference)
	if err != nil {
		return nil, err
	}
	movements, _ := s.repo.ListMovements(ctx, id)
	byMethod, _ := s.repo.SumByPaymentMethod(ctx, id)

	summary := &RegisterSummary{
		Register:        closed,
		CashSales:       cashSales,
		CashIn:          cashIn,
		CashOut:         cashOut,
		Expected:        expected,
		Movements:       movements,
		ByPaymentMethod: byMethod,
	}

	if s.setupRepo != nil {
		if cfg, err := s.setupRepo.Get(); err == nil && cfg.OwnerEmail != "" && cfg.OwnerEmailAppPassword != "" {
			cashierName := "Desconocido"
			if s.userRepo != nil {
				if user, err := s.userRepo.GetByID(context.Background(), closed.UserID); err == nil {
					cashierName = user.FullName
					if cashierName == "" {
						cashierName = user.Username
					}
				}
			}
			
			pmLines := make([]mailer.PaymentLine, len(byMethod))
			for i, m := range byMethod {
				pmLines[i] = mailer.PaymentLine{Method: m.Method, Total: m.Total}
			}

			closedAt := time.Now()
			if closed.ClosedAt != nil {
				closedAt = *closed.ClosedAt
			}

			go func(d mailer.RegisterSummaryData, smtpCfg mailer.SMTPConfig) {
				subject, body := mailer.BuildRegisterSummaryEmail(d)
				_ = mailer.Send(smtpCfg, subject, body)
			}(mailer.RegisterSummaryData{
				BusinessName:    cfg.Name,
				Cashier:         cashierName,
				ClosedAt:        closedAt,
				ByPaymentMethod: pmLines,
				CashIn:          cashIn,
				CashOut:         cashOut,
				Expected:        expected,
				ActualClose:     actualClose,
				Difference:      difference,
			}, mailer.SMTPConfig{
				Email:       cfg.OwnerEmail,
				AppPassword: cfg.OwnerEmailAppPassword,
			})
		}
	}

	return summary, nil
}

// AddMovement adds a manual cash_in or cash_out movement.
func (s *RegisterService) AddMovement(ctx context.Context, registerID int64, mvType string, amount float64, description string) (*model.RegisterMovement, error) {
	reg, err := s.repo.GetByID(ctx, registerID)
	if err != nil {
		return nil, err
	}
	if reg.Status == "closed" {
		return nil, ErrRegisterClosed
	}
	return s.repo.AddMovement(ctx, &model.RegisterMovement{
		RegisterID:  registerID,
		Type:        mvType,
		Amount:      amount,
		Description: description,
	})
}

// ListMovements returns all movements for a register.
func (s *RegisterService) ListMovements(ctx context.Context, registerID int64) ([]*model.RegisterMovement, error) {
	return s.repo.ListMovements(ctx, registerID)
}
