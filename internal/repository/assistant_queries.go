package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// AssistantQueryRepository logs every question asked to the local assistant
// (US-061), understood or not. It is a vocabulary log, not chat history: it
// never persists the computed answer or any business figure (AC-16), only
// the raw text the user typed plus the engine's classification metadata.
type AssistantQueryRepository interface {
	// Create inserts one bitácora row. tier must already be mapped to
	// "none" when the engine did not classify the query (the CHECK
	// constraint rejects an empty string).
	Create(ctx context.Context, userID int64, query, intent, tier string, confidence float64) error
}

type assistantQueryRepo struct {
	db *sql.DB
}

// NewAssistantQueryRepository returns an AssistantQueryRepository backed by db.
func NewAssistantQueryRepository(db *sql.DB) AssistantQueryRepository {
	return &assistantQueryRepo{db: db}
}

func (r *assistantQueryRepo) Create(ctx context.Context, userID int64, query, intent, tier string, confidence float64) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO assistant_queries (user_id, query, intent, tier, confidence) VALUES (?, ?, ?, ?, ?)`,
		userID, query, intent, tier, confidence,
	)
	if err != nil {
		return fmt.Errorf("assistant query insert: %w", err)
	}
	return nil
}
