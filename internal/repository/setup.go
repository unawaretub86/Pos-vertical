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
	OwnerEmail           string  `json:"owner_email"`
	OwnerEmailAppPassword string `json:"owner_email_app_password"`
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
		       COALESCE(default_opening_amount, 0), COALESCE(allow_global_discount, 0),
		       COALESCE(owner_email, ''), COALESCE(owner_email_app_password, '')
		FROM business_config WHERE id = 1`)
	cfg := &BusinessConfig{}
	err := row.Scan(&cfg.ID, &cfg.Name, &cfg.Logo, &cfg.Address, &cfg.NIT,
		&cfg.CIIU, &cfg.Currency, &cfg.TicketHeader, &cfg.TicketFooter, &cfg.DefaultOpeningAmount, &cfg.AllowGlobalDiscount,
		&cfg.OwnerEmail, &cfg.OwnerEmailAppPassword)
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

// IsSetupCompleted reports whether the first-run onboarding wizard has been
// completed. Returns false if business_config has no row yet (fresh DB).
func (r *SetupRepository) IsSetupCompleted() (bool, error) {
	var completed bool
	err := r.db.QueryRow(`SELECT COALESCE(setup_completed, 0) FROM business_config WHERE id = 1`).Scan(&completed)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return completed, err
}

// CompleteOnboarding sets the minimal business fields captured by the wizard
// and marks setup_completed = true. It only touches name/nit/currency/
// setup_completed, leaving any other existing config (logo, address, ticket
// header/footer) untouched.
func (r *SetupRepository) CompleteOnboarding(name, nit string) error {
	_, err := r.db.Exec(`
		INSERT INTO business_config (id, name, nit, currency, setup_completed, updated_at)
		VALUES (1, ?, ?, 'COP', 1, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, nit=excluded.nit, currency=excluded.currency,
			setup_completed=1, updated_at=CURRENT_TIMESTAMP
	`, name, nit)
	return err
}

func (r *SetupRepository) Upsert(cfg *BusinessConfig) error {
	_, err := r.db.Exec(`
		INSERT INTO business_config (id, name, logo, address, nit, currency, ticket_header, ticket_footer, default_opening_amount, allow_global_discount, owner_email, owner_email_app_password, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, logo=excluded.logo, address=excluded.address,
			nit=excluded.nit, currency=excluded.currency,
			ticket_header=excluded.ticket_header, ticket_footer=excluded.ticket_footer,
			default_opening_amount=excluded.default_opening_amount,
			allow_global_discount=excluded.allow_global_discount,
			owner_email=excluded.owner_email,
			owner_email_app_password=excluded.owner_email_app_password,
			updated_at=CURRENT_TIMESTAMP
	`, cfg.Name, cfg.Logo, cfg.Address, cfg.NIT, cfg.Currency,
		cfg.TicketHeader, cfg.TicketFooter, cfg.DefaultOpeningAmount, cfg.AllowGlobalDiscount, cfg.OwnerEmail, cfg.OwnerEmailAppPassword)
	return err
}
