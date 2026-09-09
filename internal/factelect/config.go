package factelect

import (
	"context"
	"database/sql"
	"time"

	"github.com/andreyo/pos/internal/model"
)

// FiscalConfigRepository persists the business fiscal configuration.
type FiscalConfigRepository interface {
	Get(ctx context.Context) (*model.FiscalConfig, error)
	Update(ctx context.Context, cfg *model.FiscalConfig) error
}

type fiscalConfigSQLite struct {
	db *sql.DB
}

// NewFiscalConfigRepository creates a SQLite implementation.
func NewFiscalConfigRepository(db *sql.DB) FiscalConfigRepository {
	return &fiscalConfigSQLite{db: db}
}

func (r *fiscalConfigSQLite) Get(ctx context.Context) (*model.FiscalConfig, error) {
	var cfg model.FiscalConfig
	err := r.db.QueryRowContext(ctx, `
		SELECT id, enabled, nit, razon_social, direccion, ciiu,
		       resolucion_dian, COALESCE(prefijo,''), rango_desde, rango_hasta,
		       fecha_resolucion, fecha_vencimiento,
		       clave_tecnica, ambiente, software_id, software_pin,
		       provider, provider_client_id, provider_secret,
		       COALESCE(webhook_url,''),
		       COALESCE(numbering_range_id,0), COALESCE(municipality_id,149),
		       created_at, updated_at
		FROM fiscal_config WHERE id=1`,
	).Scan(
		&cfg.ID, &cfg.Enabled, &cfg.NIT, &cfg.RazonSocial, &cfg.Direccion, &cfg.CIIU,
		&cfg.ResolucionDIAN, &cfg.Prefijo, &cfg.RangoDesde, &cfg.RangoHasta,
		&cfg.FechaResolucion, &cfg.FechaVencimiento,
		&cfg.ClaveTecnica, &cfg.Ambiente, &cfg.SoftwareID, &cfg.SoftwarePIN,
		&cfg.Provider, &cfg.ProviderClientID, &cfg.ProviderSecret,
		&cfg.WebhookURL,
		&cfg.NumberingRangeID, &cfg.MunicipalityID,
		&cfg.CreatedAt, &cfg.UpdatedAt,
	)
	return &cfg, err
}

func (r *fiscalConfigSQLite) Update(ctx context.Context, cfg *model.FiscalConfig) error {
	cfg.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE fiscal_config SET
			enabled=?, nit=?, razon_social=?, direccion=?, ciiu=?,
			resolucion_dian=?, prefijo=?, rango_desde=?, rango_hasta=?,
			fecha_resolucion=?, fecha_vencimiento=?,
			clave_tecnica=?, ambiente=?, software_id=?, software_pin=?,
			provider=?, provider_client_id=?, provider_secret=?,
			webhook_url=?, numbering_range_id=?, municipality_id=?, updated_at=?
		WHERE id=1`,
		cfg.Enabled, cfg.NIT, cfg.RazonSocial, cfg.Direccion, cfg.CIIU,
		cfg.ResolucionDIAN, cfg.Prefijo, cfg.RangoDesde, cfg.RangoHasta,
		cfg.FechaResolucion, cfg.FechaVencimiento,
		cfg.ClaveTecnica, cfg.Ambiente, cfg.SoftwareID, cfg.SoftwarePIN,
		cfg.Provider, cfg.ProviderClientID, cfg.ProviderSecret,
		cfg.WebhookURL, cfg.NumberingRangeID, cfg.MunicipalityID, cfg.UpdatedAt,
	)
	return err
}
