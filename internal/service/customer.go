package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// ErrCustomerHasBalance is returned when deleting a customer that still owes money.
var ErrCustomerHasBalance = errors.New("no se puede eliminar un cliente con saldo pendiente")

type CustomerService struct {
	repo repository.CustomerRepository
}

func NewCustomerService(repo repository.CustomerRepository) *CustomerService {
	return &CustomerService{repo: repo}
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

func (s *CustomerService) CreatePayment(ctx context.Context, customerID int64, amount float64, note string) (*model.CustomerPayment, error) {
	if amount <= 0 {
		return nil, errors.New("el monto del abono debe ser mayor a cero")
	}

	// Verify customer exists
	if _, err := s.repo.GetByID(ctx, customerID); err != nil {
		return nil, errors.New("cliente no encontrado")
	}

	// Validate amount does not exceed current balance
	balance, err := s.repo.Balance(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if amount > balance {
		return nil, fmt.Errorf("el pago ($%.0f) supera la deuda actual ($%.0f)", amount, balance)
	}

	return s.repo.CreatePayment(ctx, &model.CustomerPayment{
		CustomerID: customerID,
		Amount:     amount,
		Note:       strings.TrimSpace(note),
	})
}

func (s *CustomerService) ListCreditSales(ctx context.Context, customerID int64) ([]*model.Sale, error) {
	return s.repo.ListCreditSales(ctx, customerID)
}
