package service

import (
	"context"
	"errors"
	"strings"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

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

func (s *CustomerService) Create(ctx context.Context, name, phone, address string) (*model.Customer, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("el nombre es requerido")
	}
	return s.repo.Create(ctx, &model.Customer{
		Name:    name,
		Phone:   strings.TrimSpace(phone),
		Address: strings.TrimSpace(address),
	})
}

func (s *CustomerService) Update(ctx context.Context, id int64, name, phone, address string) (*model.Customer, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("el nombre es requerido")
	}
	return s.repo.Update(ctx, &model.Customer{
		ID:      id,
		Name:    name,
		Phone:   strings.TrimSpace(phone),
		Address: strings.TrimSpace(address),
	})
}

func (s *CustomerService) Delete(ctx context.Context, id int64) error {
	balance, err := s.repo.Balance(ctx, id)
	if err != nil {
		return err
	}
	if balance > 0 {
		return errors.New("no se puede eliminar un cliente con saldo pendiente")
	}
	return s.repo.Delete(ctx, id)
}

func (s *CustomerService) Balance(ctx context.Context, id int64) (float64, error) {
	return s.repo.Balance(ctx, id)
}

func (s *CustomerService) TotalPortfolioBalance(ctx context.Context) (float64, error) {
	return s.repo.TotalPortfolioBalance(ctx)
}
