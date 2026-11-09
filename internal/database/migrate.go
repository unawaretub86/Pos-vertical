package database

import (
	"context"
	"fmt"
	"os"
	"time"
)

// migration represents a single versioned database migration.
type migration struct {
	version int
	up      string   // SQL executed inside a transaction; failure stops the binary
	fixup   []string // SQL run after commit with errors silently ignored (idempotent ops)
}

var dbMigrations = []migration{
	{
		version: 1,
		up:      migration1,
		// For databases that pre-date the description column (created before this project added it).
		fixup: []string{
			`ALTER TABLE categories ADD COLUMN description TEXT NOT NULL DEFAULT ''`,
		},
	},
	{version: 2, up: migration2},
	{version: 3, up: migration3},
	{version: 4, up: migration4},
	{version: 5, up: migration5},
	{version: 6, up: migration6},
	{version: 7, up: migration7},
	{version: 8, up: migration8},
	{version: 9, up: migration9},
	{version: 10, up: migration10},
	{version: 11, up: migration11},
	{version: 12, up: migration12},
	{version: 13, up: migration13},
	{version: 14, up: migration14},
	{version: 15, up: migration15},
	{version: 16, up: migration16},
	{version: 17, up: migration17},
	{version: 18, up: migration18},
	{version: 19, up: migration19},
	{version: 20, up: migration20},
	{version: 21, up: migration21},
	{version: 22, up: migration22},
}

// migrate creates the schema_migrations table, detects pending migrations,
// backs up the database if needed, and runs each migration in a transaction.
// Any failure causes the binary to refuse to start.
func (db *DB) migrate() error {
	// The tracking table must always exist before we query it.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var current int
	_ = db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current)

	var pending []migration
	for _, m := range dbMigrations {
		if m.version > current {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		return nil
	}

	// Checkpoint the WAL before backup so the copy includes all committed data.
	// Skip backup for in-memory databases (used in tests).
	if db.path != ":memory:" {
		_, _ = db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
		if err := db.backupBeforeMigration(); err != nil {
			return fmt.Errorf("backup before migration: %w", err)
		}
	}

	for _, m := range pending {
		if err := db.runMigration(m); err != nil {
			return fmt.Errorf("migration v%d: %w", m.version, err)
		}
	}
	return nil
}

func (db *DB) runMigration(m migration) error {
	ctx := context.Background()
	// Migrations that rebuild a table (DROP TABLE + rename) fail with enforced
	// foreign keys while child rows exist, and PRAGMA foreign_keys is a no-op
	// inside a transaction. So pin one connection, switch them off for the
	// migration and restore them before the connection returns to the pool.
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`) //nolint:errcheck

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, m.up); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, m.version); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	// Same connection: with ":memory:" every pooled connection is its own database.
	for _, stmt := range m.fixup {
		_, _ = conn.ExecContext(ctx, stmt)
	}
	return nil
}

// backupBeforeMigration creates a transationally safe copy of the SQLite file.
// If the file does not exist yet (first run) it is a no-op.
func (db *DB) backupBeforeMigration() error {
	if _, err := os.Stat(db.path); os.IsNotExist(err) {
		return nil
	}
	dst := db.path + ".backup-" + time.Now().Format("20060102")
	_ = os.Remove(dst)
	_, err := db.Exec("VACUUM INTO ?", dst)
	return err
}

// ── Migration SQL ─────────────────────────────────────────────────────────────

// migration1 is the full initial schema. All tables use CREATE TABLE IF NOT
// EXISTS so it is safe to re-run on databases that already have the schema.
const migration1 = `
CREATE TABLE IF NOT EXISTS business_config (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	name TEXT NOT NULL DEFAULT '',
	logo TEXT,
	address TEXT,
	nit TEXT,
	currency TEXT NOT NULL DEFAULT 'COP',
	ticket_header TEXT,
	ticket_footer TEXT,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	role TEXT NOT NULL CHECK (role IN ('admin', 'cashier')),
	active INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	description TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	code TEXT,
	category_id INTEGER REFERENCES categories(id),
	type TEXT NOT NULL DEFAULT 'product' CHECK (type IN ('product', 'service')),
	sale_price REAL NOT NULL,
	wholesale_price REAL,
	cost REAL NOT NULL DEFAULT 0,
	unit TEXT NOT NULL DEFAULT 'unit' CHECK (unit IN ('unit', 'kg', 'lb', 'liter')),
	stock REAL NOT NULL DEFAULT 0,
	min_stock REAL NOT NULL DEFAULT 0,
	tax_rate REAL NOT NULL DEFAULT 0,
	active INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_products_active ON products(active);
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category_id);

CREATE TABLE IF NOT EXISTS customers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	phone TEXT,
	address TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS registers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL REFERENCES users(id),
	opening_amount REAL NOT NULL,
	expected_close REAL,
	actual_close REAL,
	difference REAL,
	status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
	opened_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	closed_at DATETIME
);

CREATE TABLE IF NOT EXISTS sales (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL REFERENCES users(id),
	customer_id INTEGER REFERENCES customers(id),
	register_id INTEGER REFERENCES registers(id),
	subtotal REAL NOT NULL,
	discount REAL NOT NULL DEFAULT 0,
	tax REAL NOT NULL DEFAULT 0,
	total REAL NOT NULL,
	payment_method TEXT NOT NULL CHECK (payment_method IN ('cash', 'transfer', 'card', 'credit')),
	cash_tendered REAL,
	change_amount REAL,
	reference TEXT,
	status TEXT NOT NULL DEFAULT 'completed',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sales_created ON sales(created_at);
CREATE INDEX IF NOT EXISTS idx_sales_register ON sales(register_id);

CREATE TABLE IF NOT EXISTS sale_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	sale_id INTEGER NOT NULL REFERENCES sales(id),
	product_id INTEGER NOT NULL REFERENCES products(id),
	quantity REAL NOT NULL,
	unit_price REAL NOT NULL,
	discount REAL NOT NULL DEFAULT 0,
	subtotal REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS customer_payments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	customer_id INTEGER NOT NULL REFERENCES customers(id),
	amount REAL NOT NULL,
	note TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS register_movements (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	register_id INTEGER NOT NULL REFERENCES registers(id),
	type TEXT NOT NULL CHECK (type IN ('sale', 'cash_in', 'cash_out', 'opening')),
	amount REAL NOT NULL,
	description TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS stock_entries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	product_id INTEGER NOT NULL REFERENCES products(id),
	quantity REAL NOT NULL,
	note TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Seed default admin user (password: admin)
INSERT OR IGNORE INTO users (id, username, password_hash, role)
VALUES (1, 'admin', '$2a$10$v1/OGJ.TMlKg77OMe0jkx.2ZxE32lp/ZDwyVlXcuawUqmt8RPpL/C', 'admin');
`

// migration2 adds void support to sales.
// Note: SQLite does not support adding CHECK constraints via ALTER TABLE;
// the CHECK (stock >= 0) on products is enforced at the application layer.
const migration2 = `
ALTER TABLE sales ADD COLUMN voided_by INTEGER REFERENCES users(id);
ALTER TABLE sales ADD COLUMN voided_at DATETIME;
`

// migration3 adds per-item tax tracking to sale_items.
const migration3 = `
ALTER TABLE sale_items ADD COLUMN tax_rate REAL NOT NULL DEFAULT 0;
`

// migration4 adds optional default opening amount to business_config.
const migration4 = `
ALTER TABLE business_config ADD COLUMN default_opening_amount REAL NOT NULL DEFAULT 0;
`

// migration5 adds global discount toggle to business_config.
const migration5 = `
ALTER TABLE business_config ADD COLUMN allow_global_discount INTEGER NOT NULL DEFAULT 0;
`

// migration7 adds the barcode column to products (was missing from initial schema).
const migration7 = `
ALTER TABLE products ADD COLUMN barcode TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_products_barcode ON products(barcode) WHERE barcode IS NOT NULL;
`

const migration8 = `
ALTER TABLE business_config ADD COLUMN ciiu TEXT;

ALTER TABLE customers ADD COLUMN email TEXT;
ALTER TABLE customers ADD COLUMN document_type TEXT DEFAULT 'CC';
ALTER TABLE customers ADD COLUMN document_number TEXT;
ALTER TABLE customers ADD COLUMN tax_regime TEXT DEFAULT 'simplified';

CREATE TABLE IF NOT EXISTS fiscal_config (
    id                  INTEGER PRIMARY KEY CHECK (id = 1),
    enabled             INTEGER NOT NULL DEFAULT 0,
    nit                 TEXT NOT NULL DEFAULT '',
    razon_social        TEXT NOT NULL DEFAULT '',
    direccion           TEXT NOT NULL DEFAULT '',
    ciiu                TEXT NOT NULL DEFAULT '',
    resolucion_dian     TEXT NOT NULL DEFAULT '',
    prefijo             TEXT DEFAULT '',
    rango_desde         INTEGER NOT NULL DEFAULT 0,
    rango_hasta         INTEGER NOT NULL DEFAULT 0,
    fecha_resolucion    DATE,
    fecha_vencimiento   DATE,
    clave_tecnica       TEXT NOT NULL DEFAULT '',
    ambiente            TEXT NOT NULL DEFAULT 'test',
    software_id         TEXT NOT NULL DEFAULT '',
    software_pin        TEXT NOT NULL DEFAULT '',
    provider            TEXT NOT NULL DEFAULT 'factus',
    provider_client_id  TEXT NOT NULL DEFAULT '',
    provider_secret     TEXT NOT NULL DEFAULT '',
    webhook_url         TEXT,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT OR IGNORE INTO fiscal_config (id) VALUES (1);

CREATE TABLE IF NOT EXISTS electronic_documents (
    id               TEXT PRIMARY KEY,
    sale_id          INTEGER NOT NULL REFERENCES sales(id),
    type             TEXT NOT NULL DEFAULT 'doc_equivalente',
    status           TEXT NOT NULL DEFAULT 'draft',
    provider         TEXT NOT NULL DEFAULT 'factus',
    provider_id      TEXT,
    cude             TEXT,
    number           TEXT,
    xml_url          TEXT,
    pdf_url          TEXT,
    dian_response    TEXT,
    rejection_reason TEXT,
    retry_count      INTEGER NOT NULL DEFAULT 0,
    issued_at        DATETIME,
    accepted_at      DATETIME,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_eledocs_sale_id ON electronic_documents(sale_id);
CREATE INDEX IF NOT EXISTS idx_eledocs_status ON electronic_documents(status);
CREATE INDEX IF NOT EXISTS idx_eledocs_provider_id ON electronic_documents(provider_id);
`

const migration9 = `
ALTER TABLE fiscal_config ADD COLUMN numbering_range_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE fiscal_config ADD COLUMN municipality_id INTEGER NOT NULL DEFAULT 149;
`

const migration10 = `
ALTER TABLE customers ADD COLUMN notes TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN credit_limit INTEGER;
`

// migration11 adds first-run onboarding support: a completion flag on
// business_config and a full_name column on users (set by the wizard).
const migration11 = `
ALTER TABLE business_config ADD COLUMN setup_completed BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN full_name TEXT NOT NULL DEFAULT '';
`

// migration6 adds price override tracking to sale_items and creates the audit_log table.
const migration6 = `
ALTER TABLE sale_items ADD COLUMN original_price REAL NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS audit_log (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id    INTEGER NOT NULL REFERENCES users(id),
	action     TEXT NOT NULL,
	entity     TEXT NOT NULL,
	entity_id  INTEGER NOT NULL,
	details    TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

// migration12 adds mixed-payment support (sale_payments, sales.payment_method
// CHECK widened to include 'mixed') and business expenses tracking. SQLite
// cannot ALTER an existing CHECK constraint, so sales is recreated following
// the same pattern already used in this project for CHECK changes.
const migration12 = `
CREATE TABLE sales_new (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL REFERENCES users(id),
	customer_id INTEGER REFERENCES customers(id),
	register_id INTEGER REFERENCES registers(id),
	subtotal REAL NOT NULL,
	discount REAL NOT NULL DEFAULT 0,
	tax REAL NOT NULL DEFAULT 0,
	total REAL NOT NULL,
	payment_method TEXT NOT NULL CHECK (payment_method IN ('cash', 'transfer', 'card', 'credit', 'mixed')),
	cash_tendered REAL,
	change_amount REAL,
	reference TEXT,
	status TEXT NOT NULL DEFAULT 'completed',
	voided_by INTEGER REFERENCES users(id),
	voided_at DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO sales_new (id, user_id, customer_id, register_id, subtotal, discount, tax, total, payment_method, cash_tendered, change_amount, reference, status, voided_by, voided_at, created_at)
SELECT id, user_id, customer_id, register_id, subtotal, discount, tax, total, payment_method, cash_tendered, change_amount, reference, status, voided_by, voided_at, created_at FROM sales;

DROP TABLE sales;
ALTER TABLE sales_new RENAME TO sales;

CREATE INDEX IF NOT EXISTS idx_sales_created ON sales(created_at);
CREATE INDEX IF NOT EXISTS idx_sales_register ON sales(register_id);

CREATE TABLE IF NOT EXISTS sale_payments (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	sale_id    INTEGER NOT NULL REFERENCES sales(id),
	method     TEXT NOT NULL CHECK (method IN ('cash', 'transfer', 'card', 'credit')),
	amount     REAL NOT NULL CHECK (amount > 0),
	reference  TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_sale_payments_sale ON sale_payments(sale_id);

-- Backfill sale_payments for pre-existing sales so history is not lost.
INSERT INTO sale_payments (sale_id, method, amount, created_at)
SELECT id, payment_method, total, created_at FROM sales WHERE total > 0;

CREATE TABLE IF NOT EXISTS expenses (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	register_id INTEGER REFERENCES registers(id),
	user_id     INTEGER NOT NULL REFERENCES users(id),
	category    TEXT NOT NULL CHECK (category IN ('servicios','arriendo','nomina','proveedor','otro')),
	amount      REAL NOT NULL CHECK (amount > 0),
	note        TEXT,
	created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_expenses_register ON expenses(register_id);
`

// migration13 adds suppliers and purchases (US-052). products.cost is
// updated to the purchase's unit_cost (last-cost, not weighted average) by
// the repository layer, not by this schema.
const migration13 = `
CREATE TABLE IF NOT EXISTS suppliers (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT NOT NULL,
	phone      TEXT,
	notes      TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS purchases (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	supplier_id INTEGER NOT NULL REFERENCES suppliers(id),
	user_id     INTEGER NOT NULL REFERENCES users(id),
	reference   TEXT,
	total       REAL NOT NULL,
	created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS purchase_items (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	purchase_id INTEGER NOT NULL REFERENCES purchases(id),
	product_id  INTEGER NOT NULL REFERENCES products(id),
	quantity    REAL NOT NULL CHECK (quantity > 0),
	unit_cost   REAL NOT NULL CHECK (unit_cost >= 0),
	subtotal    REAL NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_purchases_supplier ON purchases(supplier_id);
CREATE INDEX IF NOT EXISTS idx_purchase_items_purchase ON purchase_items(purchase_id);
CREATE INDEX IF NOT EXISTS idx_purchase_items_product ON purchase_items(product_id);
`

// migration14 adds the owner's Gmail address + app password to business_config,
// used to auto-send the register-close summary email (US-053).
const migration14 = `
ALTER TABLE business_config ADD COLUMN owner_email TEXT;
ALTER TABLE business_config ADD COLUMN owner_email_app_password TEXT;
`

// migration15 creates assistant_queries: the local assistant's query log
// (US-061). It is NOT a chat history — it never stores the computed answer
// nor any business figure, only the raw question and the engine's
// classification metadata, so the store owner's vocabulary can be reviewed
// later without duplicating sales/portfolio data.
const migration15 = `
CREATE TABLE IF NOT EXISTS assistant_queries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL REFERENCES users(id),
	query TEXT NOT NULL,
	intent TEXT NOT NULL,
	tier TEXT NOT NULL CHECK (tier IN ('regex','fuzzy','none')),
	confidence REAL NOT NULL DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_assistant_queries_intent ON assistant_queries(intent);
`

// migration16 adds the auto-generated JWT secret to business_config (US-070).
// This ensures that sessions cannot be forged using a hardcoded secret.
const migration16 = `
ALTER TABLE business_config ADD COLUMN jwt_secret TEXT NOT NULL DEFAULT '';
`

// migration17 adds the public catalog feature (US-045).
const migration17 = `
ALTER TABLE business_config ADD COLUMN catalog_enabled BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE business_config ADD COLUMN catalog_token TEXT;
`

// migration18 adds the restaurant module: tables, orders and order items (US-055).
const migration18 = `
CREATE TABLE IF NOT EXISTS dining_tables (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	zone TEXT NOT NULL DEFAULT '',
	seats INTEGER NOT NULL DEFAULT 0,
	sort_order INTEGER NOT NULL DEFAULT 0,
	active INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS orders (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	type TEXT NOT NULL CHECK (type IN ('table','takeaway')),
	table_id INTEGER REFERENCES dining_tables(id),
	turn_number INTEGER,
	customer_name TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','closed','cancelled')),
	user_id INTEGER NOT NULL REFERENCES users(id),
	sale_id INTEGER REFERENCES sales(id),
	note TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	closed_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_table ON orders(table_id);
CREATE INDEX IF NOT EXISTS idx_orders_updated ON orders(updated_at);
CREATE TABLE IF NOT EXISTS order_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	order_id INTEGER NOT NULL REFERENCES orders(id),
	product_id INTEGER NOT NULL REFERENCES products(id),
	quantity REAL NOT NULL,
	unit_price REAL NOT NULL,
	note TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending'
		CHECK (status IN ('pending','preparing','ready','delivered','void')),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	ready_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_status ON order_items(status);
ALTER TABLE business_config ADD COLUMN restaurant_mode INTEGER NOT NULL DEFAULT 0;
ALTER TABLE business_config ADD COLUMN lan_access_enabled INTEGER NOT NULL DEFAULT 0;
`

// migration19 adds logical deletion to customers. Customers are
// referenced by sales and payments, so deleting one sets active = 0 instead of
// removing the row. Existing customers stay active.
const migration19 = `
ALTER TABLE customers ADD COLUMN active INTEGER NOT NULL DEFAULT 1;
`

// migration20 adds the thermal printer settings (US-080). Existing shops keep
// printing through the browser until an admin picks an ESC/POS method.
const migration20 = `
ALTER TABLE business_config ADD COLUMN print_method TEXT NOT NULL DEFAULT 'browser';
ALTER TABLE business_config ADD COLUMN paper_width_mm INTEGER NOT NULL DEFAULT 58;
ALTER TABLE business_config ADD COLUMN printer_host TEXT NOT NULL DEFAULT '';
ALTER TABLE business_config ADD COLUMN printer_port INTEGER NOT NULL DEFAULT 9100;
ALTER TABLE business_config ADD COLUMN printer_name TEXT NOT NULL DEFAULT '';
`

// migration21 records how each abono was paid and which register received it
// (US-082). Existing abonos become cash with no register, so no balance and no
// past register close changes.
const migration21 = `
ALTER TABLE customer_payments ADD COLUMN method TEXT NOT NULL DEFAULT 'cash' CHECK (method IN ('cash','transfer','card'));
ALTER TABLE customer_payments ADD COLUMN register_id INTEGER NULL REFERENCES registers(id);
CREATE INDEX IF NOT EXISTS idx_customer_payments_register ON customer_payments(register_id);
`

// migration22 adds the opening fiado balance (deuda anterior) carried over from
// the paper notebook at go-live (US-092): at most one row per customer, added to
// the customer balance. debt_date is UTC text "YYYY-MM-DD HH:MM:SS" for 00:00
// server-local time of the picked date, the layout of MIN(sales.created_at).
const migration22 = `
CREATE TABLE IF NOT EXISTS customer_opening_balances (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	customer_id INTEGER NOT NULL UNIQUE REFERENCES customers(id),
	amount      REAL NOT NULL CHECK (amount > 0),
	debt_date   TEXT NOT NULL,
	note        TEXT NOT NULL DEFAULT '',
	created_by  INTEGER NOT NULL REFERENCES users(id),
	created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at  DATETIME
);
`
