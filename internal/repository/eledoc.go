package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/andreyo/pos/internal/model"
	"github.com/google/uuid"
)

type eledocRepository struct {
	db *sql.DB
}

func NewElectronicDocRepository(db *sql.DB) ElectronicDocRepository {
	return &eledocRepository{db: db}
}

func (r *eledocRepository) Create(ctx context.Context, doc *model.ElectronicDocument) error {
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	now := time.Now()
	doc.CreatedAt = now
	doc.UpdatedAt = now
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO electronic_documents
			(id, sale_id, type, status, provider, provider_id, cude, number,
			 xml_url, pdf_url, dian_response, rejection_reason, retry_count,
			 issued_at, accepted_at, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		doc.ID, doc.SaleID, doc.Type, doc.Status, doc.Provider,
		nullStr(doc.ProviderID), nullStr(doc.CUDE), nullStr(doc.Number),
		nullStr(doc.XMLURL), nullStr(doc.PDFURL), nullStr(doc.DIANResponse),
		nullStr(doc.RejectionReason), doc.RetryCount,
		doc.IssuedAt, doc.AcceptedAt, doc.CreatedAt, doc.UpdatedAt,
	)
	return err
}

func (r *eledocRepository) GetBySaleID(ctx context.Context, saleID int64) (*model.ElectronicDocument, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, sale_id, type, status, provider, COALESCE(provider_id,''),
				COALESCE(cude,''), COALESCE(number,''), COALESCE(xml_url,''),
				COALESCE(pdf_url,''), COALESCE(dian_response,''),
				COALESCE(rejection_reason,''), retry_count,
				issued_at, accepted_at, created_at, updated_at
		 FROM electronic_documents WHERE sale_id = ?`, saleID)
	return scanDoc(row)
}

func (r *eledocRepository) GetByProviderID(ctx context.Context, providerID string) (*model.ElectronicDocument, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, sale_id, type, status, provider, COALESCE(provider_id,''),
				COALESCE(cude,''), COALESCE(number,''), COALESCE(xml_url,''),
				COALESCE(pdf_url,''), COALESCE(dian_response,''),
				COALESCE(rejection_reason,''), retry_count,
				issued_at, accepted_at, created_at, updated_at
		 FROM electronic_documents WHERE provider_id = ?`, providerID)
	return scanDoc(row)
}

func (r *eledocRepository) Update(ctx context.Context, doc *model.ElectronicDocument) error {
	doc.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE electronic_documents SET
			status=?, provider_id=?, cude=?, number=?, xml_url=?, pdf_url=?,
			dian_response=?, rejection_reason=?, retry_count=?,
			issued_at=?, accepted_at=?, updated_at=?
		WHERE id=?`,
		doc.Status, nullStr(doc.ProviderID), nullStr(doc.CUDE), nullStr(doc.Number),
		nullStr(doc.XMLURL), nullStr(doc.PDFURL), nullStr(doc.DIANResponse),
		nullStr(doc.RejectionReason), doc.RetryCount,
		doc.IssuedAt, doc.AcceptedAt, doc.UpdatedAt, doc.ID,
	)
	return err
}

func (r *eledocRepository) ListByStatus(ctx context.Context, status string, limit int) ([]*model.ElectronicDocument, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, sale_id, type, status, provider, COALESCE(provider_id,''),
			   COALESCE(cude,''), COALESCE(number,''), COALESCE(xml_url,''),
			   COALESCE(pdf_url,''), COALESCE(dian_response,''),
			   COALESCE(rejection_reason,''), retry_count,
			   issued_at, accepted_at, created_at, updated_at
		FROM electronic_documents WHERE status=? LIMIT ?`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []*model.ElectronicDocument
	for rows.Next() {
		doc, err := scanDoc(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

type docScanner interface {
	Scan(dest ...any) error
}

func scanDoc(s docScanner) (*model.ElectronicDocument, error) {
	var doc model.ElectronicDocument
	err := s.Scan(
		&doc.ID, &doc.SaleID, &doc.Type, &doc.Status, &doc.Provider,
		&doc.ProviderID, &doc.CUDE, &doc.Number, &doc.XMLURL, &doc.PDFURL,
		&doc.DIANResponse, &doc.RejectionReason, &doc.RetryCount,
		&doc.IssuedAt, &doc.AcceptedAt, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &doc, err
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
