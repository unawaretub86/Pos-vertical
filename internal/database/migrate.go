package database

func (db *DB) migrate() error {
	_, err := db.Exec(schema)
	return err
}

const schema = `
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
