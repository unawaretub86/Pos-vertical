package service

import (
	"context"
	"errors"
	"strings"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

type DiningTableService struct {
	repo repository.DiningTableRepository
}

func NewDiningTableService(repo repository.DiningTableRepository) *DiningTableService {
	return &DiningTableService{repo: repo}
}

func (s *DiningTableService) List(ctx context.Context) ([]*model.TableWithState, error) {
	tables, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	lines, err := s.repo.OpenOrderLines(ctx)
	if err != nil {
		return nil, err
	}
	// open_total is the same IVA-inclusive total as GET /orders/{id}.
	totals := orderTotals(lines)
	for _, t := range tables {
		if t.OpenOrderID != nil {
			t.OpenTotal = totals[*t.OpenOrderID]
		}
	}
	return tables, nil
}

func (s *DiningTableService) Create(ctx context.Context, name, zone string, seats, sortOrder int) (*model.DiningTable, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("el nombre es requerido")
	}
	return s.repo.Create(ctx, &model.DiningTable{Name: name, Zone: strings.TrimSpace(zone), Seats: seats, SortOrder: sortOrder})
}

func (s *DiningTableService) Update(ctx context.Context, id int64, name, zone string, seats, sortOrder int) (*model.DiningTable, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("el nombre es requerido")
	}
	return s.repo.Update(ctx, &model.DiningTable{ID: id, Name: name, Zone: strings.TrimSpace(zone), Seats: seats, SortOrder: sortOrder})
}

func (s *DiningTableService) Deactivate(ctx context.Context, id int64) error {
	return s.repo.Deactivate(ctx, id)
}
