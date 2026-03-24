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
| As a cashier, I want to add products to a cart and complete a sale | - Search and add products to cart - Adjust quantity per line item - Remove items from cart - Display running total - Confirm sale and select payment method - **A sale cannot be created without an active open register: returns HTTP 409 with message "No hay caja abierta"** ✅ *[CORREGIDO]* - **Selling a quantity greater than available stock is blocked at the service layer with an atomic DB check: `UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?`, if rows affected = 0 the transaction rolls back and returns error "Stock insuficiente"** ✅ *[CORREGIDO]* - **Esta restricción de stock aplica ÚNICAMENTE a productos de tipo `product`. Los de tipo `service` no tienen stock y nunca se bloquean** - **El backend recalcula y valida subtotales: `subtotal = SUM(items[i].quantity * items[i].unit_price - items[i].discount)`, `total = subtotal - discount + tax`. Si el total enviado por el frontend difiere del calculado por el backend en más de $1 COP, la venta es rechazada** ✅ *[CORREGIDO para subtotal]* ⚠️ *[BUG ACTIVO: `sale.Tax` viene del frontend sin recálculo — el backend no calcula IVA desde `product.tax_rate`. `sale_items.tax_rate` nunca se escribe. Ver Ralph Review B3]* - **Crear una venta inserta un registro en `register_movements` (tipo `sale`, monto = total) dentro de la misma transacción atómica** ✅ *[CORREGIDO]* |
| As a cashier, I want to apply discounts per item or globally | - Discount field per line item (percentage or fixed amount) - Global discount field on the sale - Discounts reflected in real-time total |
| As a cashier, I want to process payment via cash, transfer, or card | - Cash: input amount tendered, display change - Transfer (Nequi/Daviplata): record reference number (optional) - Card: record as external dataphone payment - All methods record the payment type in the sale |
| As a cashier, I want to sell on credit ("fiado") to a registered customer | - Customer must be selected before marking as credit - Sale is recorded with `payment_method = "credit"` - **La venta se registra en el momento de la entrega de la mercancía, no al momento del pago. Esto es correcto para futura integración con facturación electrónica DIAN (la factura se emite al momento de entrega)** - **La deuda del cliente se registra implícitamente: la venta con `payment_method = 'credit'` aumenta el balance calculado on-the-fly (`SUM(credit_sales) - SUM(customer_payments)`). NO se inserta en `customer_payments` al crear la venta** (ver modelo de balance abajo) - El saldo deudor aumenta inmediatamente (calculado on-the-fly, ver modelo de balance) - Receipt prints "CRÉDITO" clearly - Los pagos posteriores (abonos) son eventos independientes en `customer_payments` ⚠️ *[NO IMPLEMENTADO: endpoints de `customer_payments` pendientes — ver Ralph Review B1]* - ⚠️ *[BUG ACTIVO: el frontend POS no tiene selector de cliente al elegir "Crédito" — sin `customer_id` la venta a crédito es imposible desde la UI. Ver Ralph Review B5]* |
| As an owner, I want to void a completed sale | - **Solo el rol admin puede anular ventas** - **Solo se puede anular dentro de las 24 horas posteriores a la creación** - La venta cambia status a `voided` — **nunca se borra de la DB** - Si la venta tenía ítems de tipo `product`, el stock se **restaura** en la misma transacción atómica (update status + restore stock). Si el restore falla, la anulación hace rollback completo - Si la venta era `credit`, el saldo del cliente se recalcula automáticamente (la venta anulada deja de sumarse al balance) - Se genera un registro de auditoría con: quién anuló, cuándo, y el ID de la venta original - **No existe cancelación real (borrado). Toda anulación deja rastro. Esto es requerimiento legal para futura compatibilidad con DIAN** |
| As an owner, I want to modify the sale price at the register (with permission) | - Price override field per line item - Only users with admin role can modify prices - Override is logged for audit |

#### Cash Register

| Story | Acceptance Criteria |
|-------|-------------------|
| As a cashier, I want to open the register with a base amount | - Input opening cash amount - Timestamp recorded - Only one register open per session |
| As an owner, I want to close the register and reconcile | - System calculates expected cash (opening + cash sales − cash withdrawals) - Cashier inputs actual counted cash - Difference (surplus/shortage) displayed and recorded - Generates close summary: total sales by method, total transactions - **El cierre de caja y el registro del movimiento de cierre deben ejecutarse en una sola transacción atómica. Si el proceso cae en medio del cierre, el estado de la caja queda consistente al reiniciar** |
| As an owner, I want to see all register movements | - List of: sales, manual cash-in, manual cash-out - Filterable by current session - Each entry shows: timestamp, amount, type, description |

#### Customers & Credit

| Story | Acceptance Criteria |
|-------|-------------------|
| As an owner, I want to register customers | - CRUD: name (required), phone, address - Customer list searchable by name or phone |
| As a customer (via the owner), I want to make payments toward my debt | - Record partial or full payment ("abono") - Payment reduces customer balance - Cannot record payment greater than current balance (returns validation error) - Receipt generated for payment - **⚠️ NO IMPLEMENTADO: `customer_payments` no tiene endpoints en el handler ni lógica en el service. Sin esto los clientes acumulan deuda pero nunca pueden pagarla.** |
| As an owner, I want to see the full credit portfolio | - Single screen showing: customer name, total debt, oldest unpaid sale date - Sortable by debt amount or age - Drill-down to individual customer: list of credit sales and payments with dates - **⚠️ NO IMPLEMENTADO: `/api/customers/:id/balance` no existe. El balance on-the-fly (`SUM sales - SUM payments`) no está implementado en ningún repositorio.** |

> **Modelo de balance del cliente (decisión de arquitectura):** El saldo deudor de un cliente es **calculado on-the-fly**, nunca almacenado como campo en `customers`. La fórmula es: `SUM(sales.total WHERE customer_id = X AND payment_method = 'credit') − SUM(customer_payments.amount WHERE customer_id = X)`. Esta es la fuente de verdad. No existe campo `balance` en la tabla `customers`.

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
                    cash_tendered, change, reference,
                    status[completed|voided],  -- 'voided' = anulada, nunca borrada
                    voided_by, voided_at,      -- auditoría de anulación
                    register_id, created_at)
                    -- ⚠️ COLUMNAS FALTANTES EN SCHEMA ACTUAL: voided_by y voided_at no existen.
                    -- Pendiente agregar via migración 2. Sin estas columnas la anulación no puede implementarse.

sale_items         (id, sale_id, product_id, quantity, unit_price, discount,
                    subtotal, tax_rate)
                    -- ⚠️ COLUMNA FALTANTE EN SCHEMA ACTUAL: tax_rate no existe en la tabla.
                    -- Pendiente agregar via migración 3.
                    -- tax_rate: el porcentaje de IVA vigente AL MOMENTO de la venta,
                    -- copiado desde products.tax_rate. Necesario para auditoría contable
                    -- y futura facturación electrónica DIAN.

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
| `/api/sales/:id/void` | POST | Anular venta (admin only, max 24h) |
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

## 3.5 Transaccionalidad, ACID e Integridad de Datos

> Este sistema maneja dinero. Toda operación que modifique estado financiero debe ser atómica, consistente, aislada y durable. Las decisiones de esta sección son no-negociables.

### Propiedades ACID en SQLite

SQLite en modo WAL (Write-Ahead Logging) garantiza ACID para operaciones single-process. Esta es la configuración requerida. El archivo `data.db` debe abrirse con:

```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;   -- NO está activo por default en SQLite, debe activarse explícitamente en cada conexión
```

### Operaciones que requieren transacción atómica obligatoria

| Operación | Qué entra en la transacción |
|-----------|----------------------------|
| **Crear venta** | INSERT en `sales` + INSERT en `sale_items` (todos) + UPDATE de stock en `products` (solo tipo `product`) + registro de movimiento en `register_movements` |
| **Anular venta** | UPDATE `sales.status = 'voided'` + UPDATE stock (restore) + INSERT en log de auditoría |
| **Cerrar caja** | UPDATE `registers.status = 'closed'` + INSERT en `register_movements` (movimiento de cierre) |
| **Pago de crédito** | INSERT en `customer_payments` (atómico por sí solo, pero validar que el monto no supere el saldo antes de insertar) |

### Reglas de validación del backend (service layer)

Estas validaciones ocurren **antes** de abrir cualquier transacción:

1. **Stock suficiente**: Para cada ítem de tipo `product`, verificar `stock >= quantity`. El check final se hace con `UPDATE ... WHERE stock >= ?` y verificando `rows affected`.
2. **Totales consistentes**: El backend recalcula `subtotal`, `tax` y `total` desde los ítems. Si difieren de los valores enviados por el frontend en más de $1 COP (tolerancia por redondeo), la venta es rechazada.
3. **Caja abierta**: No se puede crear una venta sin un `register_id` válido con `status = 'open'`.
4. **Cliente requerido para crédito**: `payment_method = 'credit'` requiere `customer_id` no nulo.

### Política de anulación de ventas

- **Quién puede anular**: Solo rol `admin`.
- **Ventana de tiempo**: Solo ventas creadas hace menos de 24 horas.
- **Qué pasa con el stock**: Se restaura en la misma transacción. Si falla el restore, la anulación no se completa.
- **Qué pasa con crédito**: La venta anulada deja de contar en el balance del cliente (el balance es calculado on-the-fly, incluye solo ventas `completed`).
- **Registro de auditoría**: `sales.voided_by` (user_id) y `sales.voided_at` (timestamp) se guardan siempre.
- **Borrado real**: Prohibido. Ninguna venta se borra de la base de datos.

### Migraciones de schema

El sistema de migraciones debe usar versioning numérico. No se permite modificar el schema de forma ad-hoc. Cada cambio de schema requiere una migración numerada. Esto es crítico para actualizaciones en campo (el cliente ya tiene `data.db` con datos reales).

**⚠️ ESTADO ACTUAL: `database/migrate.go` usa `CREATE TABLE IF NOT EXISTS` sin versioning — no soporta cambios de schema en campo. Debe refactorizarse antes del primer despliegue real.**

#### Tabla requerida (agregar al schema base):
```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version  INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

#### Reglas de trabajo (no negociables):
1. El schema actual se convierte en **migración versión 1**
2. Cada cambio de schema futuro = nueva migración numerada (`2`, `3`, `4`...)
3. **Nunca modificar migraciones anteriores** — solo agregar al final
4. Cada migración corre dentro de una transacción. Si falla → rollback y el binario no arranca
5. El binario hace **backup automático de `data.db`** antes de correr cualquier migración pendiente

#### Migraciones planificadas:

| Versión | Contenido |
|---------|-----------|
| 1 | Schema inicial completo (estado actual) |
| 2 | `ALTER TABLE sales ADD COLUMN voided_by INTEGER REFERENCES users(id)` + `ALTER TABLE sales ADD COLUMN voided_at DATETIME` + `ALTER TABLE products ADD CHECK (stock >= 0)` |
| 3 | `ALTER TABLE sale_items ADD COLUMN tax_rate REAL NOT NULL DEFAULT 0` |

#### Proceso de actualización en campo (MVP — manual):
```
1. Cerrar el POS (browser + proceso pos.exe)
2. Copiar data.db → data.db.backup-YYYYMMDD   ← SIEMPRE antes de actualizar
3. Reemplazar pos.exe con la nueva versión
4. Abrir pos.exe — corre las migraciones pendientes automáticamente al arrancar
5. Si falla: borrar data.db y renombrar el backup
```

> **v1.1**: El binario hace el backup automático en el paso 2 antes de migrar, eliminando el error humano.

### Preparación para Facturación Electrónica DIAN (futura)

Las siguientes decisiones de arquitectura se toman hoy para no tener que reescribir el modelo cuando llegue la facturación electrónica:

| Decisión | Razón |
|----------|-------|
| `sale_items.tax_rate` guarda el IVA al momento de la venta | La DIAN requiere el tax rate exacto por ítem, no el rate actual del producto |
| Las ventas nunca se borran (`voided` en lugar de `DELETE`) | La DIAN no acepta facturas que "desaparecen"; la anulación genera nota crédito |
| La venta crédito se registra al momento de entrega | La DIAN exige que la factura se emita al momento de entrega del bien/servicio |
| `sales.status` tiene valor `voided` desde MVP | Cuando llegue facturación electrónica, `voided` se mapea a nota crédito electrónica |
| `NIT` está en `business_config` desde el primer día | Requerido para cualquier documento fiscal |

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
| Schema sin versioning | Una actualización del binario puede romper un `data.db` existente en campo | Implementar tabla `schema_version` con migraciones numeradas antes del primer despliegue en cliente real |
| Stock negativo sin restricción DB | Un bug en el backend podría decrementar stock por debajo de 0 | ✅ CORREGIDO: check `WHERE stock >= ?` implementado en `repository/sale.go`. `CHECK (stock >= 0)` en schema pendiente como defensa secundaria |
| Totales calculados solo en frontend | Un bug o request manipulado registra montos incorrectos | ✅ PARCIAL: subtotal recalculado en backend. ⚠️ **BUG ACTIVO: `sale.Tax` no se recalcula — viene del frontend sin validación. `sale_items.tax_rate` nunca se escribe. Ver Ralph Review B3** |
| Ventas sin caja abierta | Una venta creada sin register_id válido rompe la reconciliación | ✅ CORREGIDO: service verifica caja abierta antes de crear venta |
| Movimientos de caja no registrados en ventas | La reconciliación al cierre no incluye las ventas individuales | ✅ CORREGIDO: INSERT en `register_movements` en la misma TX que la venta |
| `cash_in` no incluido en expected de caja | Movimientos `cash_in` existen pero no se suman en el cálculo de caja esperada | **⚠️ BUG ACTIVO: `register.go:99` calcula `expected = opening + cashSales - cashOut` sin `+ cashIn`. Un depósito manual de cambio genera sobrante ficticio al cerrar. Ver Ralph Review B2** |
| IVA/tax no recalculado en backend | El impuesto se acepta del frontend sin validación | **⚠️ BUG ACTIVO: `sale.Tax` viene del frontend tal cual. `sale_items.tax_rate` (migración 3) existe pero nunca se escribe. Ver Ralph Review B3** |
| Anulación de ventas no implementada | El endpoint `/api/sales/:id/void` no existe y el schema no tiene las columnas necesarias | Implementar VoidSale en repo+service+handler. Agregar `voided_by` y `voided_at` via migración 2 |
| `customer_payments` sin endpoints | Los clientes acumulan deuda pero no se pueden registrar abonos | **⚠️ BLOQUEANTE: Implementar handler+service+repo para `POST /api/customers/:id/payments` y `GET /api/customers/:id/payments`. El endpoint `/api/customers/:id/balance` ya existe. Ver Ralph Review B1** |
| Ventas anuladas en reportes | `SaleFilter` no filtra por status — ventas `voided` aparecen en totales y reportes | Agregar filtro `status = 'completed'` por defecto en List y en todos los reports |

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
