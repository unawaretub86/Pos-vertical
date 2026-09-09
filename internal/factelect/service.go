package factelect

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
	"github.com/google/uuid"
)

// FacElectService orchestrates electronic document emission.
type FacElectService struct {
	adapter    SaleAdapter
	client     *Client
	eledocRepo repository.ElectronicDocRepository
	fiscalRepo FiscalConfigRepository
}

func NewFacElectService(
	adapter SaleAdapter,
	client *Client,
	eledocRepo repository.ElectronicDocRepository,
	fiscalRepo FiscalConfigRepository,
) *FacElectService {
	return &FacElectService{
		adapter:    adapter,
		client:     client,
		eledocRepo: eledocRepo,
		fiscalRepo: fiscalRepo,
	}
}

// EmitFromSale emits a DEE for the given sale. Call in a goroutine (async).
// Does not block sale confirmation.
func (s *FacElectService) EmitFromSale(ctx context.Context, saleID int64, buyerEmail string) {
	cfg, err := s.fiscalRepo.Get(ctx)
	if err != nil || !cfg.Enabled {
		return
	}

	doc := &model.ElectronicDocument{
		ID:       uuid.New().String(),
		SaleID:   saleID,
		Type:     "doc_equivalente",
		Status:   "draft",
		Provider: cfg.Provider,
	}
	if err := s.eledocRepo.Create(ctx, doc); err != nil {
		slog.Error("factelect: crear draft", "sale_id", saleID, "err", err)
		return
	}

	req, err := s.adapter.ToBillRequest(ctx, saleID, buyerEmail)
	if err != nil {
		slog.Error("factelect: construir BillRequest", "sale_id", saleID, "err", err)
		return
	}

	resp, err := s.client.CreateBill(ctx, req)
	if err != nil {
		slog.Error("factelect: CreateBill falló", "sale_id", saleID, "err", err)
		// doc stays in 'draft' → poller will retry
		return
	}

	now := time.Now()
	doc.Status = "pending"
	doc.ProviderID = resp.ProviderID
	doc.Number = resp.Number
	doc.IssuedAt = &now
	if err := s.eledocRepo.Update(ctx, doc); err != nil {
		slog.Error("factelect: actualizar a pending", "sale_id", saleID, "err", err)
	}
}

// CreateCreditNote emits a credit note referencing an accepted DEE.
func (s *FacElectService) CreateCreditNote(ctx context.Context, saleID int64, motive string) (*model.ElectronicDocument, error) {
	orig, err := s.eledocRepo.GetBySaleID(ctx, saleID)
	if err != nil {
		return nil, fmt.Errorf("factelect: doc original no encontrado: %w", err)
	}

	req, err := s.adapter.ToBillRequest(ctx, saleID, "")
	if err != nil {
		return nil, err
	}
	cnReq := &CreditNoteRequest{
		OriginalCUDE: orig.CUDE,
		OriginalID:   orig.ProviderID,
		Motive:       motive,
		Issuer:       req.Issuer,
		Buyer:        req.Buyer,
		Items:        req.Items,
		Totals:       req.Totals,
		IssuedAt:     time.Now(),
	}

	resp, err := s.client.CreateCreditNote(ctx, cnReq)
	if err != nil {
		return nil, fmt.Errorf("factus nota crédito rechazada: %w", err)
	}

	cfg, _ := s.fiscalRepo.Get(ctx)
	now := time.Now()
	noteDoc := &model.ElectronicDocument{
		ID:         uuid.New().String(),
		SaleID:     saleID,
		Type:       "nota_credito",
		Status:     "pending",
		Provider:   cfg.Provider,
		ProviderID: resp.ProviderID,
		Number:     resp.Number,
		IssuedAt:   &now,
	}
	if err := s.eledocRepo.Create(ctx, noteDoc); err != nil {
		return nil, err
	}
	return noteDoc, nil
}

// GetDocBySaleID returns the electronic document for a sale, or nil if none exists.
func (s *FacElectService) GetDocBySaleID(ctx context.Context, saleID int64) (*model.ElectronicDocument, error) {
	doc, err := s.eledocRepo.GetBySaleID(ctx, saleID)
	if err == repository.ErrNotFound {
		return nil, nil
	}
	return doc, err
}
