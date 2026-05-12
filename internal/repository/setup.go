package repository

import (
	"database/sql"
)

type BusinessConfig struct {
	ID                   int     `json:"id"`
	Name                 string  `json:"name"`
	Logo                 string  `json:"logo"`
	Address              string  `json:"address"`
	NIT                  string  `json:"nit"`
	CIIU                 string  `json:"ciiu"`
	Currency             string  `json:"currency"`
	TicketHeader         string  `json:"ticket_header"`
	TicketFooter         string  `json:"ticket_footer"`
	DefaultOpeningAmount float64 `json:"default_opening_amount"`
	AllowGlobalDiscount  bool    `json:"allow_global_discount"`
	Configured           bool    `json:"configured"`
}

type SetupRepository struct {
	db *sql.DB
}

func NewSetupRepository(db *sql.DB) *SetupRepository {
	return &SetupRepository{db: db}
}

func (r *SetupRepository) Get() (*BusinessConfig, error) {
	row := r.db.QueryRow(`
		SELECT id, name, COALESCE(logo,''), COALESCE(address,''), COALESCE(nit,''),
		       COALESCE(ciiu,''), currency, COALESCE(ticket_header,''), COALESCE(ticket_footer,''),
		       COALESCE(default_opening_amount, 0), COALESCE(allow_global_discount, 0)
		FROM business_config WHERE id = 1`)
	cfg := &BusinessConfig{}
	err := row.Scan(&cfg.ID, &cfg.Name, &cfg.Logo, &cfg.Address, &cfg.NIT,
		&cfg.CIIU, &cfg.Currency, &cfg.TicketHeader, &cfg.TicketFooter, &cfg.DefaultOpeningAmount, &cfg.AllowGlobalDiscount)
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
		INSERT INTO business_config (id, name, logo, address, nit, currency, ticket_header, ticket_footer, default_opening_amount, allow_global_discount, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, logo=excluded.logo, address=excluded.address,
			nit=excluded.nit, currency=excluded.currency,
			ticket_header=excluded.ticket_header, ticket_footer=excluded.ticket_footer,
			default_opening_amount=excluded.default_opening_amount,
			allow_global_discount=excluded.allow_global_discount,
			updated_at=CURRENT_TIMESTAMP
	`, cfg.Name, cfg.Logo, cfg.Address, cfg.NIT, cfg.Currency,
		cfg.TicketHeader, cfg.TicketFooter, cfg.DefaultOpeningAmount, cfg.AllowGlobalDiscount)
	return err
}
