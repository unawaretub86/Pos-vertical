package service

import (
	"context"

	"github.com/andreyo/pos/internal/model"
)

type kdsRepo interface {
	GetActiveOrders(ctx context.Context) (model.KDSResponse, error)
}

type KDSService struct {
	repo kdsRepo
}

func NewKDSService(repo kdsRepo) *KDSService {
	return &KDSService{repo: repo}
}

func (s *KDSService) GetActiveOrders(ctx context.Context) (model.KDSResponse, error) {
	return s.repo.GetActiveOrders(ctx)
}
