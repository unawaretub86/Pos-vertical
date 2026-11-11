package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"time"
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
	CatalogEnabled       bool    `json:"catalog_enabled"`
	CatalogToken         string  `json:"catalog_token"`
	RestaurantMode       bool    `json:"restaurant_mode"`
	LanAccessEnabled     bool    `json:"lan_access_enabled"`
	PrintMethod          string  `json:"print_method"`
	PaperWidthMM         int     `json:"paper_width_mm"`
	PrinterHost          string  `json:"printer_host"`
	PrinterPort          int     `json:"printer_port"`
	PrinterName          string  `json:"printer_name"`
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
		       COALESCE(owner_email, ''), COALESCE(owner_email_app_password, ''),
		       COALESCE(catalog_enabled, 0), COALESCE(catalog_token, ''),
		       COALESCE(restaurant_mode, 0), COALESCE(lan_access_enabled, 0),
		       print_method, paper_width_mm, printer_host, printer_port, printer_name
		FROM business_config WHERE id = 1`)
	cfg := &BusinessConfig{}
	err := row.Scan(&cfg.ID, &cfg.Name, &cfg.Logo, &cfg.Address, &cfg.NIT,
		&cfg.CIIU, &cfg.Currency, &cfg.TicketHeader, &cfg.TicketFooter, &cfg.DefaultOpeningAmount, &cfg.AllowGlobalDiscount,
		&cfg.OwnerEmail, &cfg.OwnerEmailAppPassword, &cfg.CatalogEnabled, &cfg.CatalogToken,
		&cfg.RestaurantMode, &cfg.LanAccessEnabled,
		&cfg.PrintMethod, &cfg.PaperWidthMM, &cfg.PrinterHost, &cfg.PrinterPort, &cfg.PrinterName)
	if err == sql.ErrNoRows {
		cfg.Currency = "COP"
		cfg.PrintMethod = "browser"
		cfg.PaperWidthMM = 58
		cfg.PrinterPort = 9100
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
		INSERT INTO business_config (id, name, logo, address, nit, currency, ticket_header, ticket_footer, default_opening_amount, allow_global_discount, owner_email, owner_email_app_password, catalog_enabled, catalog_token, restaurant_mode, lan_access_enabled, print_method, paper_width_mm, printer_host, printer_port, printer_name, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, logo=excluded.logo, address=excluded.address,
			nit=excluded.nit, currency=excluded.currency,
			ticket_header=excluded.ticket_header, ticket_footer=excluded.ticket_footer,
			default_opening_amount=excluded.default_opening_amount,
			allow_global_discount=excluded.allow_global_discount,
			owner_email=excluded.owner_email,
			owner_email_app_password=CASE WHEN excluded.owner_email_app_password = '' THEN business_config.owner_email_app_password ELSE excluded.owner_email_app_password END,
			catalog_enabled=excluded.catalog_enabled,
			catalog_token=excluded.catalog_token,
			restaurant_mode=excluded.restaurant_mode,
			lan_access_enabled=excluded.lan_access_enabled,
			print_method=excluded.print_method,
			paper_width_mm=excluded.paper_width_mm,
			printer_host=excluded.printer_host,
			printer_port=excluded.printer_port,
			printer_name=excluded.printer_name,
			updated_at=CURRENT_TIMESTAMP
	`, cfg.Name, cfg.Logo, cfg.Address, cfg.NIT, cfg.Currency,
		cfg.TicketHeader, cfg.TicketFooter, cfg.DefaultOpeningAmount, cfg.AllowGlobalDiscount, cfg.OwnerEmail, cfg.OwnerEmailAppPassword,
		cfg.CatalogEnabled, cfg.CatalogToken, cfg.RestaurantMode, cfg.LanAccessEnabled,
		cfg.PrintMethod, cfg.PaperWidthMM, cfg.PrinterHost, cfg.PrinterPort, cfg.PrinterName)
	return err
}

func (r *SetupRepository) GetOrGenerateJWTSecret() ([]byte, error) {
	var secret string
	err := r.db.QueryRow(`SELECT COALESCE(jwt_secret, '') FROM business_config WHERE id = 1`).Scan(&secret)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if secret != "" {
		return []byte(secret), nil
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	secret = base64.StdEncoding.EncodeToString(b)

	_, err = r.db.Exec(`
		INSERT INTO business_config (id, jwt_secret) VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET jwt_secret = excluded.jwt_secret
	`, secret)
	if err != nil {
		return nil, err
	}
	return []byte(secret), nil
}

// backupTimeLayout is the UTC text layout of the backup timestamps (US-098 convention).
const backupTimeLayout = "2006-01-02 15:04:05"

// BackupConfig is the backup folder setting and the report of the last run.
// It is kept apart from BusinessConfig so PUT /setup can never overwrite it.
type BackupConfig struct {
	Dir           string
	LastAttemptAt *time.Time
	LastSuccessAt *time.Time
	LastOK        bool
	FellBack      bool
	LastError     string
	LastFile      string
	EffectiveDir  string
}

func parseBackupTime(s sql.NullString) *time.Time {
	if !s.Valid || s.String == "" {
		return nil
	}
	t, err := time.ParseInLocation(backupTimeLayout, s.String, time.UTC)
	if err != nil {
		return nil
	}
	return &t
}

// GetBackupConfig reads the backup folder and the last-run report. A missing
// business_config row yields the initial values.
func (r *SetupRepository) GetBackupConfig() (*BackupConfig, error) {
	var attempt, success sql.NullString
	cfg := &BackupConfig{}
	err := r.db.QueryRow(`
		SELECT backup_dir, backup_last_attempt_at, backup_last_success_at, backup_last_ok,
		       backup_fell_back, backup_last_error, backup_last_file, backup_effective_dir
		FROM business_config WHERE id = 1`).Scan(&cfg.Dir, &attempt, &success, &cfg.LastOK,
		&cfg.FellBack, &cfg.LastError, &cfg.LastFile, &cfg.EffectiveDir)
	if err == sql.ErrNoRows {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get backup config: %w", err)
	}
	cfg.LastAttemptAt = parseBackupTime(attempt)
	cfg.LastSuccessAt = parseBackupTime(success)
	return cfg, nil
}

// SaveBackupDir stores the backup folder ("" = default) without touching any
// other business_config field.
func (r *SetupRepository) SaveBackupDir(dir string) error {
	_, err := r.db.Exec(`
		INSERT INTO business_config (id, backup_dir) VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET backup_dir = excluded.backup_dir
	`, dir)
	if err != nil {
		return fmt.Errorf("save backup dir: %w", err)
	}
	return nil
}

// BackupDir returns the configured backup folder; "" means the default (also
// when it cannot be read, so a backup is never skipped for a settings error).
// It makes SetupRepository a database.BackupSource.
func (r *SetupRepository) BackupDir() string {
	cfg, err := r.GetBackupConfig()
	if err != nil {
		log.Printf("[backup] could not read backup folder, using default: %v", err)
		return ""
	}
	return cfg.Dir
}

// RecordBackup persists the result of a backup attempt. Only a successful
// attempt updates last_success_at and last_file.
func (r *SetupRepository) RecordBackup(at time.Time, ok, fellBack bool, errMsg, effectiveDir, file string) error {
	ts := at.UTC().Format(backupTimeLayout)
	var successAt, lastFile any // nil keeps the previous value through COALESCE
	if ok {
		successAt, lastFile = ts, file
	}
	_, err := r.db.Exec(`
		INSERT INTO business_config (id, backup_last_attempt_at, backup_last_success_at, backup_last_ok,
			backup_fell_back, backup_last_error, backup_last_file, backup_effective_dir)
		VALUES (1, ?, ?, ?, ?, ?, COALESCE(?, ''), ?)
		ON CONFLICT(id) DO UPDATE SET
			backup_last_attempt_at = excluded.backup_last_attempt_at,
			backup_last_success_at = COALESCE(?, business_config.backup_last_success_at),
			backup_last_ok = excluded.backup_last_ok,
			backup_fell_back = excluded.backup_fell_back,
			backup_last_error = excluded.backup_last_error,
			backup_last_file = COALESCE(?, business_config.backup_last_file),
			backup_effective_dir = excluded.backup_effective_dir
	`, ts, successAt, ok, fellBack, errMsg, lastFile, effectiveDir, successAt, lastFile)
	if err != nil {
		return fmt.Errorf("record backup: %w", err)
	}
	return nil
}
