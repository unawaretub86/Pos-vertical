package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/andreyo/pos/internal/model"
)

// execer is satisfied by both *sql.DB and *sql.Tx, letting InsertAuditLog be
// called either standalone or as part of an existing transaction.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// InsertAuditLog writes a single audit_log row. Kept as a free function
// (rather than a full repository) since every event site only ever needs
// this one write.
func InsertAuditLog(ctx context.Context, ex execer, userID int64, action, entity string, entityID int64, details string) error {
	_, err := ex.ExecContext(ctx,
		`INSERT INTO audit_log (user_id, action, entity, entity_id, details) VALUES (?, ?, ?, ?, ?)`,
		userID, action, entity, entityID, details,
	)
	if err != nil {
		return fmt.Errorf("audit_log insert: %w", err)
	}
	return nil
}

// AuditFilter controls listing behavior for GET /api/audit.
type AuditFilter struct {
	UserID *int64
	Action string
	From   *time.Time
	To     *time.Time
}

// AuditRepository reads audit_log entries.
type AuditRepository interface {
	List(ctx context.Context, filter AuditFilter) ([]*model.AuditLogEntry, error)
}

type auditRepo struct {
	db *sql.DB
}

// NewAuditRepository returns a SQLite-backed AuditRepository.
func NewAuditRepository(db *sql.DB) AuditRepository {
	return &auditRepo{db: db}
}

func (r *auditRepo) List(ctx context.Context, filter AuditFilter) ([]*model.AuditLogEntry, error) {
	query := `
		SELECT a.id, a.user_id, u.username, a.action, a.entity, a.entity_id, COALESCE(a.details, ''), a.created_at
		FROM audit_log a
		JOIN users u ON u.id = a.user_id
		WHERE 1=1`
	args := []any{}

	if filter.UserID != nil {
		query += " AND a.user_id = ?"
		args = append(args, *filter.UserID)
	}
	if filter.Action != "" {
		query += " AND a.action = ?"
		args = append(args, filter.Action)
	}
	if filter.From != nil {
		query += " AND a.created_at >= ?"
		args = append(args, *filter.From)
	}
	if filter.To != nil {
		query += " AND a.created_at <= ?"
		args = append(args, *filter.To)
	}
	query += " ORDER BY a.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("audit list: %w", err)
	}
	defer rows.Close()

	var entries []*model.AuditLogEntry
	for rows.Next() {
		e := &model.AuditLogEntry{}
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &e.Action, &e.Entity, &e.EntityID, &e.Details, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("audit scan: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
