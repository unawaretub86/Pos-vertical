package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

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

// Opening balance (deuda anterior, US-092) rules.
const (
	// maxOpeningBalance is the largest opening balance accepted (PO Q-5).
	maxOpeningBalance = 100_000_000
	// openingBalanceDateLayout is the calendar date the admin picks.
	openingBalanceDateLayout = "2006-01-02"
)

// ErrInvalidOpeningBalance matches (errors.Is) every rejected opening balance
// input: amount and debt date. The handler maps it to 400.
var ErrInvalidOpeningBalance = errors.New("deuda anterior inválida")

// openingBalanceInputError carries the user-facing message of a rejected
// opening balance input and matches ErrInvalidOpeningBalance.
type openingBalanceInputError struct{ msg string }

func (e *openingBalanceInputError) Error() string        { return e.msg }
func (e *openingBalanceInputError) Is(target error) bool { return target == ErrInvalidOpeningBalance }

// ErrOpeningCustomerNotFound is returned when the customer of an opening
// balance does not exist or was deleted (active = 0).
var ErrOpeningCustomerNotFound = errors.New("cliente no encontrado")

// ErrNoOpeningBalance is returned when correcting the opening balance of a
// customer that has none.
var ErrNoOpeningBalance = errors.New("el cliente no tiene deuda anterior")

// validateOpeningBalance checks the amount (whole pesos from 1 to
// maxOpeningBalance) and the debt date (YYYY-MM-DD, not after today in server
// local time), and returns the trimmed date.
func validateOpeningBalance(amount float64, debtDate string) (string, error) {
	if amount != math.Trunc(amount) || amount < 1 || amount > maxOpeningBalance {
		return "", &openingBalanceInputError{"el monto debe ser un valor entero entre $1 y $100.000.000"}
	}
	debtDate = strings.TrimSpace(debtDate)
	d, err := time.ParseInLocation(openingBalanceDateLayout, debtDate, time.Local)
	if err != nil {
		return "", &openingBalanceInputError{"fecha de la deuda inválida"}
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if d.After(today) {
		return "", &openingBalanceInputError{"la fecha de la deuda no puede ser futura"}
	}
	return debtDate, nil
}

// requireActiveCustomer returns ErrOpeningCustomerNotFound unless the customer
// exists with active = 1.
func (s *CustomerService) requireActiveCustomer(ctx context.Context, customerID int64) error {
	active, err := s.repo.IsActive(ctx, customerID)
	if err != nil {
		return fmt.Errorf("check customer: %w", err)
	}
	if !active {
		return ErrOpeningCustomerNotFound
	}
	return nil
}

// GetOpeningBalance returns the customer's opening balance, or nil when there
// is none.
func (s *CustomerService) GetOpeningBalance(ctx context.Context, customerID int64) (*model.CustomerOpeningBalance, error) {
	ob, err := s.repo.GetOpeningBalance(ctx, customerID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil
	}
	return ob, err
}

// CreateOpeningBalance records the customer's opening balance on behalf of
// userID. The customer must be active; a second one for the same customer is
// repository.ErrOpeningBalanceExists.
func (s *CustomerService) CreateOpeningBalance(ctx context.Context, customerID, userID int64, amount float64, debtDate, note string) (*model.CustomerOpeningBalance, error) {
	if err := s.requireActiveCustomer(ctx, customerID); err != nil {
		return nil, err
	}
	debtDate, err := validateOpeningBalance(amount, debtDate)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateOpeningBalance(ctx, &model.CustomerOpeningBalance{
		CustomerID: customerID,
		Amount:     amount,
		DebtDate:   debtDate,
		Note:       strings.TrimSpace(note),
		CreatedBy:  userID,
	})
}

// UpdateOpeningBalance corrects the customer's opening balance on behalf of
// userID. The same amount and date rules as CreateOpeningBalance apply; a
// customer with none is ErrNoOpeningBalance.
func (s *CustomerService) UpdateOpeningBalance(ctx context.Context, customerID, userID int64, amount float64, debtDate, note string) (*model.CustomerOpeningBalance, error) {
	if err := s.requireActiveCustomer(ctx, customerID); err != nil {
		return nil, err
	}
	debtDate, err := validateOpeningBalance(amount, debtDate)
	if err != nil {
		return nil, err
	}
	ob, err := s.repo.UpdateOpeningBalance(ctx, &model.CustomerOpeningBalance{
		CustomerID: customerID,
		Amount:     amount,
		DebtDate:   debtDate,
		Note:       strings.TrimSpace(note),
	}, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNoOpeningBalance
	}
	return ob, err
}
