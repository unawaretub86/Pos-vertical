package factelect

import (
	"context"
	"log/slog"
	"time"
)

const maxRetries = 10

// PollPending queries Factus for all docs with status='pending'.
// Call on a 30s ticker from main.
func (s *FacElectService) PollPending(ctx context.Context) {
	docs, err := s.eledocRepo.ListByStatus(ctx, "pending", 50)
	if err != nil {
		slog.Error("factelect: listar pendientes", "err", err)
		return
	}
	for _, doc := range docs {
		if doc.RetryCount >= maxRetries {
			doc.Status = "rejected"
			doc.RejectionReason = "máximo de reintentos alcanzado"
			_ = s.eledocRepo.Update(ctx, doc)
			continue
		}
		status, err := s.client.GetBillStatus(ctx, doc.ProviderID)
		if err != nil {
			slog.Warn("factelect: consultar estado", "provider_id", doc.ProviderID, "err", err)
			doc.RetryCount++
			_ = s.eledocRepo.Update(ctx, doc)
			continue
		}
		switch status.Status {
		case "accepted":
			now := time.Now()
			doc.Status = "accepted"
			doc.CUDE = status.CUDE
			doc.PDFURL = status.PDFURL
			doc.XMLURL = status.XMLURL
			doc.AcceptedAt = &now
		case "rejected":
			doc.Status = "rejected"
			doc.RejectionReason = status.RejectionReason
			doc.DIANResponse = status.DIANResponse
		default:
			doc.RetryCount++
		}
		if err := s.eledocRepo.Update(ctx, doc); err != nil {
			slog.Error("factelect: actualizar doc", "id", doc.ID, "err", err)
		}
	}
}

// StartPoller launches the polling job in the background.
// Cancel ctx to stop it.
func (s *FacElectService) StartPoller(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.PollPending(ctx)
			}
		}
	}()
}
