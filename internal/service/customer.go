package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// ErrCustomerHasBalance is returned when deleting a customer that still owes money.
var ErrCustomerHasBalance = errors.New("no se puede eliminar un cliente con saldo pendiente")

// ErrInvalidAbono matches (errors.Is) every rejected abono input: amount,
// method, unknown customer, amount above the balance. The handler maps it to
// 400; any other CreatePayment error is internal (500).
var ErrInvalidAbono = errors.New("abono inválido")

// abonoInputError carries the user-facing message of a rejected abono input
// and matches ErrInvalidAbono.
type abonoInputError struct{ msg string }

func (e *abonoInputError) Error() string        { return e.msg }
func (e *abonoInputError) Is(target error) bool { return target == ErrInvalidAbono }

// ErrInvalidAbonoMethod is returned when an abono's payment method is not
// cash, transfer or card (US-082).
var ErrInvalidAbonoMethod error = &abonoInputError{"método de pago inválido: use efectivo, transferencia o tarjeta"}

// ErrAbonoNeedsOpenRegister is returned when a non-admin registers an abono
// while no register is open (US-082, PO Q-1).
var ErrAbonoNeedsOpenRegister = errors.New("Debe haber una caja abierta para registrar un abono")

type CustomerService struct {
	repo         repository.CustomerRepository
	registerRepo repository.RegisterRepository
}

func NewCustomerService(repo repository.CustomerRepository, registerRepo repository.RegisterRepository) *CustomerService {
	return &CustomerService{repo: repo, registerRepo: registerRepo}
}

func (s *CustomerService) List(ctx context.Context, search string) ([]*model.CustomerWithBalance, error) {
	return s.repo.List(ctx, search)
}

func (s *CustomerService) GetByID(ctx context.Context, id int64) (*model.Customer, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CustomerService) Create(ctx context.Context, name, phone, address, notes string, creditLimit *int64) (*model.Customer, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("el nombre es requerido")
	}
	return s.repo.Create(ctx, &model.Customer{
		Name:        name,
		Phone:       strings.TrimSpace(phone),
		Address:     strings.TrimSpace(address),
		Notes:       strings.TrimSpace(notes),
		CreditLimit: creditLimit,
	})
}

func (s *CustomerService) Update(ctx context.Context, id int64, name, phone, address, notes string, creditLimit *int64) (*model.Customer, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("el nombre es requerido")
	}
	return s.repo.Update(ctx, &model.Customer{
		ID:          id,
		Name:        name,
		Phone:       strings.TrimSpace(phone),
		Address:     strings.TrimSpace(address),
		Notes:       strings.TrimSpace(notes),
		CreditLimit: creditLimit,
	})
}

func (s *CustomerService) Delete(ctx context.Context, id int64) error {
	balance, err := s.repo.Balance(ctx, id)
	if err != nil {
		return err
	}
	if balance > 0 {
		return ErrCustomerHasBalance
	}
	return s.repo.Delete(ctx, id)
}

func (s *CustomerService) Balance(ctx context.Context, id int64) (float64, error) {
	return s.repo.Balance(ctx, id)
}

func (s *CustomerService) TotalPortfolioBalance(ctx context.Context) (float64, error) {
	return s.repo.TotalPortfolioBalance(ctx)
}

func (s *CustomerService) ListPayments(ctx context.Context, customerID int64) ([]*model.CustomerPayment, error) {
	return s.repo.ListPayments(ctx, customerID)
}

// CreatePayment registers an abono. method defaults to cash. The abono is
// linked to the open register when there is one; with no open register only an
// admin may register it, and it stays unattributed (register_id NULL).
func (s *CustomerService) CreatePayment(ctx context.Context, customerID int64, amount float64, method, note string) (*model.CustomerPayment, error) {
	if amount <= 0 {
		return nil, &abonoInputError{"el monto del abono debe ser mayor a cero"}
	}
	method = strings.TrimSpace(method)
	if method == "" {
		method = "cash"
	}
	if method != "cash" && method != "transfer" && method != "card" {
		return nil, ErrInvalidAbonoMethod
	}

	var registerID *int64
	reg, err := s.registerRepo.GetOpen(ctx)
	switch {
	case err == nil:
		registerID = &reg.ID
	case errors.Is(err, repository.ErrNotFound):
		claims, _ := ctx.Value(appmw.ClaimsKey).(*model.Claims)
		if claims == nil || claims.Role != "admin" {
			return nil, ErrAbonoNeedsOpenRegister
		}
	default:
		return nil, fmt.Errorf("check open register: %w", err)
	}

	// Verify customer exists
	if _, err := s.repo.GetByID(ctx, customerID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, &abonoInputError{"cliente no encontrado"}
		}
		return nil, fmt.Errorf("get customer: %w", err)
	}

	// Validate amount does not exceed current balance
	balance, err := s.repo.Balance(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("customer balance: %w", err)
	}
	if amount > balance {
		return nil, &abonoInputError{fmt.Sprintf("el pago ($%.0f) supera la deuda actual ($%.0f)", amount, balance)}
	}

	return s.repo.CreatePayment(ctx, &model.CustomerPayment{
		CustomerID: customerID,
		Amount:     amount,
		Method:     method,
		RegisterID: registerID,
		Note:       strings.TrimSpace(note),
	})
}

func (s *CustomerService) ListCreditSales(ctx context.Context, customerID int64) ([]*model.Sale, error) {
	return s.repo.ListCreditSales(ctx, customerID)
}

func (s *CustomerService) GetPOSSummary(ctx context.Context, customerID int64) (*model.CustomerPosSummary, error) {
	return s.repo.GetPOSSummary(ctx, customerID)
}
