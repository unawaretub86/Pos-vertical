# PRD — POS Genérico MVP

## 1. Executive Summary

### Problem Statement

Small and medium businesses in Colombia (corner stores, food distributors, bike shops) rely on paper notebooks or expensive cloud-based POS systems ($30–80 USD/month) that require constant internet. They need a simple, offline tool that handles sales, inventory, and — critically — customer credit ("fiado"), which existing solutions handle poorly.

### Proposed Solution

A single-binary POS application (`pos.exe`) that runs entirely on the client's machine. No internet, no subscription, no installation complexity. The user opens a browser to `localhost:8080` and starts selling. SQLite stores everything in one file. Future versions add optional cloud sync for multi-branch and remote reporting.

### Success Criteria

| KPI | Target |
|-----|--------|
| Complete a sale (search → cart → payment → ticket) | < 30 seconds |
| First-time setup (install + configure business) | < 5 minutes |
| Application startup time | < 3 seconds |
| Cash register close/reconciliation | < 2 minutes |
| Credit portfolio query (who owes what) | < 5 seconds, single screen |

---

## 2. User Experience & Functionality

### User Personas

**Persona 1 — Don Carlos (Business Owner)**
- Owns a small food supply distributor in Bogotá
- Age 45–60, basic computer literacy (uses WhatsApp, Excel occasionally)
- Sells on credit to restaurant owners, tracks debts in a notebook
- Needs: simple sales, know who owes him money, print receipts

**Persona 2 — Laura (Cashier/Employee)**
- Works at the counter, processes 50–150 sales/day
- Age 20–35, comfortable with phones and basic apps
- Needs: fast product search, quick checkout, no complex menus

### User Stories & Acceptance Criteria

#### Initial Setup

| Story | Acceptance Criteria |
|-------|-------------------|
| As an owner, I want to configure my business info on first launch so that tickets show my brand | - Fields: business name, logo, address, NIT/CC, currency (COP default) - Data persists in SQLite - Configuration screen only appears on first run, accessible later from settings |

#### Products

| Story | Acceptance Criteria |
|-------|-------------------|
| As an owner, I want to create products with name, category, price, cost, and unit of measure | - CRUD operations for products - Fields: name, category, sale price, wholesale price (optional), cost, type (product/service), unit (unit/kg/lb/liter), minimum stock - Validation: name required, price > 0, cost >= 0 |
| As a cashier, I want to search products by name or code in < 1 second | - Search input with real-time filtering - Matches on product name or internal code - Results appear as user types (debounced 200ms) |
| As an owner, I want to manage custom categories | - CRUD for categories - Products reference category by ID - Deleting a category requires reassigning its products |

#### Point of Sale

| Story | Acceptance Criteria |
|-------|-------------------|
| As a cashier, I want to add products to a cart and complete a sale | - Search and add products to cart - Adjust quantity per line item - Remove items from cart - Display running total - Confirm sale and select payment method |
| As a cashier, I want to apply discounts per item or globally | - Discount field per line item (percentage or fixed amount) - Global discount field on the sale - Discounts reflected in real-time total |
| As a cashier, I want to process payment via cash, transfer, or card | - Cash: input amount tendered, display change - Transfer (Nequi/Daviplata): record reference number (optional) - Card: record as external dataphone payment - All methods record the payment type in the sale |
| As a cashier, I want to sell on credit ("fiado") to a registered customer | - Customer must be selected before marking as credit - Sale is recorded with status "credit" - Customer's debt balance increases by sale total - Receipt prints "CRÉDITO" clearly |
| As an owner, I want to modify the sale price at the register (with permission) | - Price override field per line item - Only users with admin role can modify prices - Override is logged for audit |

#### Cash Register

| Story | Acceptance Criteria |
|-------|-------------------|
| As a cashier, I want to open the register with a base amount | - Input opening cash amount - Timestamp recorded - Only one register open per session |
| As an owner, I want to close the register and reconcile | - System calculates expected cash (opening + cash sales − cash withdrawals) - Cashier inputs actual counted cash - Difference (surplus/shortage) displayed and recorded - Generates close summary: total sales by method, total transactions |
| As an owner, I want to see all register movements | - List of: sales, manual cash-in, manual cash-out - Filterable by current session - Each entry shows: timestamp, amount, type, description |

#### Customers & Credit

| Story | Acceptance Criteria |
|-------|-------------------|
| As an owner, I want to register customers | - CRUD: name (required), phone, address - Customer list searchable by name or phone |
| As a customer (via the owner), I want to make payments toward my debt | - Record partial or full payment ("abono") - Payment reduces customer balance - Receipt generated for payment |
| As an owner, I want to see the full credit portfolio | - Single screen showing: customer name, total debt, oldest unpaid sale date - Sortable by debt amount or age - Drill-down to individual customer: list of credit sales and payments with dates |

#### Inventory

| Story | Acceptance Criteria |
|-------|-------------------|
| As an owner, I want to see current stock levels | - List of products with current stock quantity - Highlight products below minimum stock (red/warning) |
| As an owner, I want to register incoming merchandise | - Manual stock entry: select product, input quantity, optional note - Stock increases immediately - Entry logged with timestamp |

#### Reports

| Story | Acceptance Criteria |
|-------|-------------------|
| As an owner, I want to see sales summaries | - Sales by day, week, month (selectable range) - Total revenue, total cost, gross margin - Breakdown by payment method |
| As an owner, I want to see best-selling products | - Ranked list by quantity sold or revenue - Filterable by date range |
| As an owner, I want to see pending debts | - Same as credit portfolio view - Exportable (print or future CSV) |
| As an owner, I want to see register history | - List of register open/close events - Each with: opening amount, expected close, actual close, difference |

#### Ticket Printing

| Story | Acceptance Criteria |
|-------|-------------------|
| As an owner, I want to print sale receipts on a thermal printer | - ESC/POS protocol support - Ticket includes: business name/logo, date, items with prices, totals, payment method, "CRÉDITO" if applicable - Configurable format (header text, footer text) |

#### Users & Roles

| Story | Acceptance Criteria |
|-------|-------------------|
| As an owner, I want to create user accounts with roles | - Two roles: admin, cashier - Admin: full access - Cashier: POS, own register, view own sales - Login with username + password (local auth) |

### Non-Goals (MVP)

- **No DIAN electronic invoicing** — complex regulatory project, separate effort
- **No barcode scanner support** — manual search is sufficient for MVP
- **No multi-branch** — requires cloud sync infrastructure
- **No supplier/purchase management** — manual stock entry covers MVP needs
- **No multiple price lists** — two prices (regular + wholesale) are enough
- **No unit conversion** (box → unit) — unnecessary complexity for MVP
- **No cloud sync** — MVP is 100% local
- **No mobile app** — responsive web in browser is sufficient
- **No complex tax rules** — basic IVA field on products only

---

## 3. Technical Specifications

### Architecture Overview

```
┌─────────────────────────────────────┐
│           pos.exe (Go binary)       │
│                                     │
│  ┌──────────┐    ┌───────────────┐  │
│  │ Chi Router│◄──►│ Static Files  │  │
│  │ REST API  │    │ (Next.js out) │  │
│  └─────┬────┘    └───────────────┘  │
│        │                            │
│  ┌─────▼────┐                       │
│  │ Service  │                       │
│  │ Layer    │                       │
│  └─────┬────┘                       │
│        │                            │
│  ┌─────▼────────┐                   │
│  │ Repository   │ ← interface       │
│  │ (SQLite impl)│                   │
│  └─────┬────────┘                   │
│        │                            │
│  ┌─────▼────┐                       │
│  │ data.db  │ (SQLite file)         │
│  └──────────┘                       │
└─────────────────────────────────────┘

User opens browser → localhost:8080
```

**Layers:**

| Layer | Responsibility |
|-------|---------------|
| **Handler** | HTTP request/response, input validation, auth middleware |
| **Service** | Business logic, transaction orchestration |
| **Repository** | Data access via interface (SQLite today, swappable for cloud sync later) |

### Data Model (Core Entities)

```
business_config    (singleton: name, logo, address, nit, currency, ticket_format)

users              (id, username, password_hash, role, created_at)

categories         (id, name, created_at)

products           (id, name, code, category_id, type[product|service],
                    sale_price, wholesale_price, cost, unit[unit|kg|lb|liter],
                    stock, min_stock, tax_rate, active, created_at)

customers          (id, name, phone, address, created_at)

sales              (id, user_id, customer_id, subtotal, discount, tax, total,
                    payment_method[cash|transfer|card|credit],
                    cash_tendered, change, reference, status, register_id,
                    created_at)

sale_items         (id, sale_id, product_id, quantity, unit_price, discount,
                    subtotal)

customer_payments  (id, customer_id, amount, note, created_at)

registers          (id, user_id, opening_amount, expected_close, actual_close,
                    difference, opened_at, closed_at, status[open|closed])

register_movements (id, register_id, type[sale|cash_in|cash_out], amount,
                    description, created_at)

stock_entries      (id, product_id, quantity, note, created_at)
```

### Integration Points

| System | Protocol | Details |
|--------|----------|---------|
| Thermal printer | ESC/POS over USB/Serial | Standard 58mm/80mm thermal printers |
| Browser | HTTP (localhost:8080) | REST API + static file serving |
| File system | SQLite file | Single `data.db` file, same directory as binary |

### API Design (REST)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/setup` | GET/PUT | Business configuration |
| `/api/auth/login` | POST | User authentication |
| `/api/users` | CRUD | User management (admin only) |
| `/api/categories` | CRUD | Category management |
| `/api/products` | CRUD | Product management |
| `/api/products/search?q=` | GET | Product search |
| `/api/products/low-stock` | GET | Products below min_stock |
| `/api/customers` | CRUD | Customer management |
| `/api/customers/:id/balance` | GET | Customer debt balance |
| `/api/customers/:id/payments` | GET/POST | Customer debt payments |
| `/api/sales` | GET/POST | Create and list sales |
| `/api/sales/:id/ticket` | GET | Generate ticket for printing |
| `/api/registers` | GET/POST | Open register |
| `/api/registers/:id/close` | POST | Close register with reconciliation |
| `/api/registers/:id/movements` | GET/POST | Register movements |
| `/api/stock-entries` | POST | Manual stock entry |
| `/api/reports/sales` | GET | Sales summary (params: from, to) |
| `/api/reports/top-products` | GET | Best-selling products |
| `/api/reports/portfolio` | GET | Credit portfolio summary |
| `/api/reports/registers` | GET | Register history |

### Security & Privacy

- **Authentication**: Local session-based auth (JWT or cookie-based token)
- **Password storage**: bcrypt hashed, never stored in plaintext
- **No external network calls**: Zero data leaves the machine in MVP
- **Data backup**: User manually copies `data.db` file (automatic backup is v2)
- **No PII transmitted**: All customer data stays local

### Frontend Pages

| Page | Route | Role |
|------|-------|------|
| Login | `/login` | All |
| Setup (first run) | `/setup` | Admin |
| Dashboard | `/` | All |
| Point of Sale | `/pos` | All |
| Products | `/products` | Admin |
| Categories | `/categories` | Admin |
| Customers | `/customers` | Admin |
| Customer Detail | `/customers/:id` | Admin |
| Credit Portfolio | `/portfolio` | Admin |
| Inventory | `/inventory` | Admin |
| Stock Entry | `/inventory/entry` | Admin |
| Register (current) | `/register` | All |
| Register History | `/registers` | Admin |
| Reports | `/reports` | Admin |
| Users | `/users` | Admin |
| Settings | `/settings` | Admin |

---

## 4. Risks & Roadmap

### Technical Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| SQLite concurrent write contention | Data corruption if multiple tabs/users write simultaneously | Enforce single-writer via WAL mode + mutex in Go; limit to one active register per session |
| ESC/POS printer compatibility | Not all thermal printers behave identically | Support the 2–3 most common brands (Epson TM series, generic 58mm); document supported models |
| Binary size bloat from embedded frontend | Large download for clients on slow connections | Tree-shake Next.js static export; target < 30 MB total binary |
| Data loss (no cloud backup) | Client loses PC = loses all data | Provide manual backup/restore (copy data.db); document backup procedure; cloud backup in v2 |
| Local auth is weak | Physical access to machine = access to data | Acceptable for MVP market; encrypted DB in v2 |

### Phased Rollout

**MVP (Phase 1) — 100% Local**
- All Priority 1–7 features from context doc
- Single binary distribution
- Manual backup via file copy
- Target: functional POS usable by a real business

**v1.1 — Polish**
- Barcode scanner support
- CSV export for reports
- Automatic local backup (scheduled copy of data.db)
- UX improvements based on first client feedback

**v2.0 — Cloud Hybrid**
- Optional cloud sync (subscription model)
- Multi-branch support
- Remote reporting (view sales from phone)
- Automatic cloud backup
- Supplier/purchase management

**v3.0 — Compliance & Scale**
- DIAN electronic invoicing
- Advanced price lists
- Unit conversion (box → unit)
- Mobile app

---

## 5. Currency & Locale

| Setting | Value |
|---------|-------|
| Default currency | COP (Colombian Peso) |
| Decimal places | 0 (no cents) |
| Thousands separator | `.` (Colombian format: $1.500) |
| Tax | Basic IVA field per product (percentage) |
| Language | Spanish (Colombia) |
