// Package service contains business logic for the POS system.
package service

import (
	"context"
	"errors"
	"strings"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// CategoryService handles category business logic.
type CategoryService struct {
	repo repository.CategoryRepository
}

// NewCategoryService creates a CategoryService backed by the given repository.
func NewCategoryService(repo repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) Create(ctx context.Context, name string) (*model.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("el nombre es requerido")
	}
	return s.repo.Create(ctx, name)
}

func (s *CategoryService) List(ctx context.Context) ([]*model.Category, error) {
	return s.repo.List(ctx)
}

func (s *CategoryService) GetByID(ctx context.Context, id int64) (*model.Category, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CategoryService) Update(ctx context.Context, id int64, name string) (*model.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("el nombre es requerido")
	}
	return s.repo.Update(ctx, id, name)
}

func (s *CategoryService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
