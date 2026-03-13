package repository

import (
	"database/sql"
)

type BusinessConfig struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Logo          string `json:"logo"`
	Address       string `json:"address"`
	NIT           string `json:"nit"`
	Currency      string `json:"currency"`
	TicketHeader  string `json:"ticket_header"`
	TicketFooter  string `json:"ticket_footer"`
	Configured    bool   `json:"configured"`
}

type SetupRepository struct {
	db *sql.DB
}

func NewSetupRepository(db *sql.DB) *SetupRepository {
	return &SetupRepository{db: db}
}

func (r *SetupRepository) Get() (*BusinessConfig, error) {
	row := r.db.QueryRow(`SELECT id, name, COALESCE(logo,''), COALESCE(address,''), COALESCE(nit,''), currency, COALESCE(ticket_header,''), COALESCE(ticket_footer,'') FROM business_config WHERE id = 1`)
	cfg := &BusinessConfig{}
	err := row.Scan(&cfg.ID, &cfg.Name, &cfg.Logo, &cfg.Address, &cfg.NIT, &cfg.Currency, &cfg.TicketHeader, &cfg.TicketFooter)
	if err == sql.ErrNoRows {
		cfg.Currency = "COP"
		cfg.Configured = false
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	cfg.Configured = cfg.Name != ""
	return cfg, nil
}

func (r *SetupRepository) Upsert(cfg *BusinessConfig) error {
	_, err := r.db.Exec(`
		INSERT INTO business_config (id, name, logo, address, nit, currency, ticket_header, ticket_footer, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, logo=excluded.logo, address=excluded.address,
			nit=excluded.nit, currency=excluded.currency,
			ticket_header=excluded.ticket_header, ticket_footer=excluded.ticket_footer,
			updated_at=CURRENT_TIMESTAMP
	`, cfg.Name, cfg.Logo, cfg.Address, cfg.NIT, cfg.Currency, cfg.TicketHeader, cfg.TicketFooter)
	return err
}
