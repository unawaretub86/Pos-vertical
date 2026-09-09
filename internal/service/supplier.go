package service

import (
	"context"
	"errors"
	"strings"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

type SupplierService struct {
	repo repository.SupplierRepository
}

func NewSupplierService(repo repository.SupplierRepository) *SupplierService {
	return &SupplierService{repo: repo}
}

func (s *SupplierService) List(ctx context.Context) ([]*model.Supplier, error) {
	return s.repo.List(ctx)
}

func (s *SupplierService) GetByID(ctx context.Context, id int64) (*model.Supplier, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SupplierService) Create(ctx context.Context, name, phone, notes string) (*model.Supplier, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("el nombre es requerido")
	}
	return s.repo.Create(ctx, &model.Supplier{Name: name, Phone: strings.TrimSpace(phone), Notes: strings.TrimSpace(notes)})
}

func (s *SupplierService) Update(ctx context.Context, id int64, name, phone, notes string) (*model.Supplier, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("el nombre es requerido")
	}
	return s.repo.Update(ctx, &model.Supplier{ID: id, Name: name, Phone: strings.TrimSpace(phone), Notes: strings.TrimSpace(notes)})
}

func (s *SupplierService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
