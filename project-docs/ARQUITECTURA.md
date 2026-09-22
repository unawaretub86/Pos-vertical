# Arquitectura — POS Vertical

> **Tipo:** histórico · **Vigente:** NO · **Fecha:** 2026-04-17
> **No es fuente de verdad sobre el código actual.** Foto de la estructura en esa fecha. Para convenciones y arquitectura vigentes ver [`../CLAUDE.md`](../CLAUDE.md).


> Analizado el 2026-04-17. Fuentes: `internal/`, `frontend/`, `project-docs/PRD.md`, `go.mod`.

---

## 1. Estructura de Carpetas (estado actual)

```
Pos-vertical/                        ← raíz del monorepo
│
├── cmd/                             ← entrypoints Go
│   ├── pos/                         ← binario principal (main.go)
│   └── reset-password/              ← utilidad CLI para resetear contraseñas
│
├── internal/                        ← backend (paquetes privados Go)
│   ├── database/
│   │   ├── database.go              ← apertura de DB, PRAGMAs WAL + FK
│   │   ├── migrate.go               ← sistema de migraciones versionado (v1–v7)
│   │   └── backup.go                ← backup automático pre-migración
│   │
│   ├── model/
│   │   └── model.go                 ← structs Go: User, Product, Sale, Customer…
│   │
│   ├── repository/                  ← capa de datos (interface + impl SQLite)
│   │   ├── repository.go            ← interface Repository (contrato)
│   │   ├── user.go
│   │   ├── product.go
│   │   ├── category.go
│   │   ├── customer.go              ← balance on-the-fly incluido
│   │   ├── sale.go                  ← TX atómica: sale + items + stock + movement
│   │   ├── register.go
│   │   ├── report.go
│   │   ├── setup.go
│   │   ├── stock_entry.go
│   │   └── errors.go
│   │
│   ├── service/                     ← lógica de negocio (entre handler y repo)
│   │
│   ├── handler/                     ← HTTP handlers (Chi router)
│   │   ├── handler.go               ← router principal, agrupación de rutas
│   │   ├── auth.go
│   │   ├── products.go
│   │   ├── categories.go
│   │   ├── customer.go
│   │   ├── sale.go
│   │   ├── register.go
│   │   ├── report.go
│   │   ├── user.go
│   │   ├── dashboard.go
│   │   └── setup.go
│   │
│   └── middleware/                  ← JWT auth, role check
│
├── frontend/                        ← Next.js 14 (App Router, TypeScript)
│   ├── app/
│   │   ├── layout.tsx               ← root layout
│   │   ├── page.tsx                 ← Dashboard (/)
│   │   ├── globals.css
│   │   ├── login/
│   │   ├── setup/
│   │   ├── pos/                     ← Punto de Venta
│   │   ├── inventory/
│   │   ├── customers/
│   │   ├── portfolio/               ← Cartera de crédito
│   │   ├── reports/
│   │   ├── register/                ← Caja actual
│   │   ├── users/
│   │   ├── settings/
│   │   └── change-password/
│   │
│   ├── components/
│   │   ├── AppLayout.tsx            ← shell: sidebar + topbar
│   │   ├── Sidebar.tsx
│   │   ├── Topbar.tsx
│   │   ├── TicketPreview.tsx        ← vista previa de ticket térmico
│   │   ├── ConfirmDialog.tsx
│   │   ├── EmptyState.tsx
│   │   ├── LoadingSpinner.tsx
│   │   ├── Toast.tsx
│   │   └── ui/                     ← componentes shadcn/ui
│   │
│   ├── hooks/                       ← custom hooks React
│   ├── lib/                         ← helpers, API client
│   └── public/
│
├── web/                             ← build estático de Next.js embebido en el binario
├── scripts/                         ← scripts de build/deploy
├── project-docs/                    ← PRD, PLAN, SPRINTs, backlog
├── data.db                          ← SQLite (producción local)
├── pos.exe                          ← binario final compilado
├── go.mod                           ← github.com/andreyo/pos, go 1.25
├── go.sum
└── Makefile
```

### Stack técnico

| Capa | Tecnología |
|------|-----------|
| Backend | Go 1.25, Chi v5 (router), JWT (golang-jwt/jwt v5) |
| DB | SQLite via modernc.org/sqlite (puro Go, sin CGO) |
| Frontend | Next.js 14 (App Router), TypeScript, shadcn/ui |
| Distribución | Binario único `pos.exe` con frontend embebido en `/web` |

---

## 2. Esquema de DB (estado real — 7 migraciones aplicadas)

> El schema vive en `internal/database/migrate.go`. Se presentan las tablas con todas las columnas incluyendo las agregadas por migraciones v2–v7.

### Diagrama de relaciones

```
business_config (singleton)

users ──────────────────────────────────────────────────────┐
  │                                                          │
  ├── registers (1 usuario → N cajas)                       │
  │       │                                                  │
  │       ├── register_movements                             │
  │       └── sales ──────────────────────── voided_by → users
  │               │
  │               ├── sale_items → products → categories
  │               └── customer_id → customers
  │                                    │
  │                                    └── customer_payments
  │
  ├── audit_log
  └── (stock_entries → products)
```

### Tablas detalladas

#### `business_config` — Singleton de configuración del negocio
```sql
id                      INTEGER PRIMARY KEY CHECK (id = 1)
name                    TEXT NOT NULL DEFAULT ''
logo                    TEXT
address                 TEXT
nit                     TEXT
currency                TEXT NOT NULL DEFAULT 'COP'
ticket_header           TEXT
ticket_footer           TEXT
default_opening_amount  REAL NOT NULL DEFAULT 0    -- migración 4
allow_global_discount   INTEGER NOT NULL DEFAULT 0 -- migración 5
updated_at              DATETIME DEFAULT CURRENT_TIMESTAMP
```

#### `users` — Empleados / acceso al sistema
```sql
id            INTEGER PRIMARY KEY AUTOINCREMENT
username      TEXT NOT NULL UNIQUE
password_hash TEXT NOT NULL                        -- bcrypt
role          TEXT NOT NULL CHECK (role IN ('admin', 'cashier'))
active        INTEGER NOT NULL DEFAULT 1
created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
```

#### `categories` — Categorías de producto
```sql
id          INTEGER PRIMARY KEY AUTOINCREMENT
name        TEXT NOT NULL UNIQUE
description TEXT NOT NULL DEFAULT ''               -- fixup migración 1
created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
```

#### `products` — Catálogo de productos y servicios
```sql
id               INTEGER PRIMARY KEY AUTOINCREMENT
name             TEXT NOT NULL
code             TEXT
barcode          TEXT UNIQUE                        -- migración 7
category_id      INTEGER REFERENCES categories(id)
type             TEXT NOT NULL DEFAULT 'product'
                   CHECK (type IN ('product', 'service'))
sale_price       REAL NOT NULL
wholesale_price  REAL
cost             REAL NOT NULL DEFAULT 0
unit             TEXT NOT NULL DEFAULT 'unit'
                   CHECK (unit IN ('unit', 'kg', 'lb', 'liter'))
stock            REAL NOT NULL DEFAULT 0
min_stock        REAL NOT NULL DEFAULT 0
tax_rate         REAL NOT NULL DEFAULT 0
active           INTEGER NOT NULL DEFAULT 1
created_at       DATETIME DEFAULT CURRENT_TIMESTAMP

-- Índices
INDEX idx_products_active   ON products(active)
INDEX idx_products_category ON products(category_id)
UNIQUE INDEX idx_products_barcode ON products(barcode) WHERE barcode IS NOT NULL
```

#### `customers` — Clientes (incluye deudores de crédito)
```sql
id         INTEGER PRIMARY KEY AUTOINCREMENT
name       TEXT NOT NULL
phone      TEXT
address    TEXT
created_at DATETIME DEFAULT CURRENT_TIMESTAMP

-- ⚠️ NO existe campo `balance`.
-- El saldo se calcula on-the-fly:
-- SUM(sales.total WHERE payment_method='credit' AND status='completed')
-- - SUM(customer_payments.amount)
```

#### `registers` — Cajas / turnos de caja
```sql
id              INTEGER PRIMARY KEY AUTOINCREMENT
user_id         INTEGER NOT NULL REFERENCES users(id)
opening_amount  REAL NOT NULL
expected_close  REAL
actual_close    REAL
difference      REAL
status          TEXT NOT NULL DEFAULT 'open'
                  CHECK (status IN ('open', 'closed'))
opened_at       DATETIME DEFAULT CURRENT_TIMESTAMP
closed_at       DATETIME
```

#### `register_movements` — Movimientos dentro de una caja
```sql
id          INTEGER PRIMARY KEY AUTOINCREMENT
register_id INTEGER NOT NULL REFERENCES registers(id)
type        TEXT NOT NULL CHECK (type IN ('sale', 'cash_in', 'cash_out', 'opening'))
amount      REAL NOT NULL
description TEXT
created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
```

#### `sales` — Ventas (nunca se borran, solo se anulan)
```sql
id             INTEGER PRIMARY KEY AUTOINCREMENT
user_id        INTEGER NOT NULL REFERENCES users(id)
customer_id    INTEGER REFERENCES customers(id)
register_id    INTEGER REFERENCES registers(id)
subtotal       REAL NOT NULL
discount       REAL NOT NULL DEFAULT 0
tax            REAL NOT NULL DEFAULT 0
total          REAL NOT NULL
payment_method TEXT NOT NULL
                 CHECK (payment_method IN ('cash','transfer','card','credit'))
cash_tendered  REAL
change_amount  REAL
reference      TEXT
status         TEXT NOT NULL DEFAULT 'completed'   -- 'completed' | 'voided'
voided_by      INTEGER REFERENCES users(id)        -- migración 2
voided_at      DATETIME                            -- migración 2
created_at     DATETIME DEFAULT CURRENT_TIMESTAMP

-- Índices
INDEX idx_sales_created  ON sales(created_at)
INDEX idx_sales_register ON sales(register_id)
```

#### `sale_items` — Líneas de cada venta
```sql
id             INTEGER PRIMARY KEY AUTOINCREMENT
sale_id        INTEGER NOT NULL REFERENCES sales(id)
product_id     INTEGER NOT NULL REFERENCES products(id)
quantity       REAL NOT NULL
unit_price     REAL NOT NULL
original_price REAL NOT NULL DEFAULT 0   -- migración 6: precio antes del override
discount       REAL NOT NULL DEFAULT 0
subtotal       REAL NOT NULL
tax_rate       REAL NOT NULL DEFAULT 0   -- migración 3: IVA vigente al momento de la venta
```

#### `customer_payments` — Abonos a deuda (fiado)
```sql
id          INTEGER PRIMARY KEY AUTOINCREMENT
customer_id INTEGER NOT NULL REFERENCES customers(id)
amount      REAL NOT NULL
note        TEXT
created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
```

#### `stock_entries` — Ingresos de inventario (compras / reposición)
```sql
id         INTEGER PRIMARY KEY AUTOINCREMENT
product_id INTEGER NOT NULL REFERENCES products(id)
quantity   REAL NOT NULL
note       TEXT
created_at DATETIME DEFAULT CURRENT_TIMESTAMP
```

#### `audit_log` — Auditoría de acciones sensibles
```sql
id         INTEGER PRIMARY KEY AUTOINCREMENT
user_id    INTEGER NOT NULL REFERENCES users(id)
action     TEXT NOT NULL
entity     TEXT NOT NULL
entity_id  INTEGER NOT NULL
details    TEXT
created_at DATETIME DEFAULT CURRENT_TIMESTAMP
```

#### `schema_migrations` — Control de versiones de schema
```sql
version    INTEGER PRIMARY KEY
applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
-- Versiones aplicadas: 1, 2, 3, 4, 5, 6, 7
```

### Bugs conocidos — estado actual

| ID | Descripción | Severidad | Estado |
|----|-------------|-----------|--------|
| B2 | `expected_close` no suma `cash_in` | 🔴 Alta | ✅ Corregido — `service/register.go`: `expected = opening + cashSales + cashIn - cashOut` |
| B3 | `sale.Tax` sin recalcular en backend; `sale_items.tax_rate` nunca se escribía | 🔴 Alta | ✅ Corregido — `service/sale.go:59`: `item.TaxRate = product.TaxRate`; IVA recalculado y validado |
| B4 | Ventas `voided` aparecen en reportes | 🟡 Media | ✅ Corregido — `repository/report.go`: todos los queries filtran `WHERE s.status = 'completed'` |

---

## 3. Wireframes de Baja Fidelidad — 3 Flujos Principales

### Flujo 1: Checkout / Punto de Venta (`/pos`)

```
┌─────────────────────────────────────────────────────────────────────┐
│  POS  [🔍 Buscar producto o escanear código de barras............]  │
├──────────────────────────────┬──────────────────────────────────────┤
│   RESULTADOS DE BÚSQUEDA     │            CARRITO                   │
│                              │                                      │
│  ┌──────────────────────┐    │  ┌──────────────────────────────┐   │
│  │ 📦 Arroz x25kg       │    │  │ Producto         Cant  Precio│   │
│  │    $85.000  [+ Agreg]│    │  ├──────────────────────────────┤   │
│  └──────────────────────┘    │  │ Arroz x25kg       1   $85.000│   │
│  ┌──────────────────────┐    │  │   Desc: [  0  ]  Sub: $85.000│   │
│  │ 🔧 Aceite Motor 1L   │    │  ├──────────────────────────────┤   │
│  │    $32.000  [+ Agreg]│    │  │ Aceite Motor 1L   2   $64.000│   │
│  └──────────────────────┘    │  │   Desc: [  0  ]  Sub: $64.000│   │
│  ┌──────────────────────┐    │  ├──────────────────────────────┤   │
│  │ 🥤 Gaseosa 2L        │    │  │                              │   │
│  │    $8.500   [+ Agreg]│    │  │  Desc. global: [          0] │   │
│  └──────────────────────┘    │  │  IVA:                $12.654 │   │
│                              │  │  TOTAL:             $161.654  │   │
│  (scroll)                    │  └──────────────────────────────┘   │
│                              │                                      │
│                              │  Cliente: [Seleccionar cliente ▼]   │
│                              │                                      │
│                              │  ┌────────┐ ┌────────┐ ┌────────┐  │
│                              │  │Efectivo│ │Transf. │ │Tarjeta │  │
│                              │  └────────┘ └────────┘ └────────┘  │
│                              │  ┌────────┐                         │
│                              │  │Crédito │ ← requiere cliente     │
│                              │  └────────┘                         │
│                              │                                      │
│                              │  [🗑 Limpiar]    [✅ Confirmar]     │
└──────────────────────────────┴──────────────────────────────────────┘

── Modal de pago (Efectivo) ──────────────────────────────────────────
│  Total a cobrar:        $161.654                                    │
│  Efectivo recibido: [              200.000]                         │
│  Cambio:                 $38.346                                    │
│  [Cancelar]                           [✅ Finalizar venta]         │
─────────────────────────────────────────────────────────────────────
```

---

### Flujo 2: Cartera de Crédito (`/portfolio` → `/customers/:id`)

```
── Vista principal: Cartera (/portfolio) ─────────────────────────────
┌─────────────────────────────────────────────────────────────────────┐
│  💳 CARTERA DE CRÉDITO           [🔍 Buscar cliente...]            │
│                                                                     │
│  Total cartera: $4.820.000   Clientes deudores: 12                 │
│  ─────────────────────────────────────────────────────────────────  │
│  Cliente ↑↓         Deuda ↑↓     Venta más antigua    Acción      │
│  ─────────────────────────────────────────────────────────────────  │
│  👤 Rest. El Buen Sabor $1.200.000  hace 45 días  [Ver detalle]    │
│  👤 Carnes Rodríguez    $890.000    hace 12 días  [Ver detalle]    │
│  👤 Tienda La Esquina   $430.000    hace 3 días   [Ver detalle]    │
│  ─────────────────────────────────────────────────────────────────  │
│                               [< Ant]  Página 1/2  [Sig >]         │
└─────────────────────────────────────────────────────────────────────┘

── Detalle de cliente (/customers/:id) ───────────────────────────────
┌─────────────────────────────────────────────────────────────────────┐
│  ← Volver   👤 Rest. El Buen Sabor   Tel: 311-555-0123            │
│                                                                     │
│  ┌───────────────────────┐  ┌────────────────────────────────────┐ │
│  │  SALDO DEUDOR         │  │  REGISTRAR ABONO                   │ │
│  │  $1.200.000           │  │  Monto: [                        ] │ │
│  │  (45 días de deuda)   │  │  Nota:  [                        ] │ │
│  └───────────────────────┘  │  [💰 Registrar pago]              │ │
│                              └────────────────────────────────────┘ │
│  HISTORIAL                                                          │
│  ─────────────────────────────────────────────────────────────────  │
│  Fecha        Concepto                   Monto       Saldo         │
│  ─────────────────────────────────────────────────────────────────  │
│  2026-04-15   Venta #247 (crédito)     +$350.000   $1.200.000     │
│  2026-04-10   Abono                    -$200.000     $850.000      │
│  2026-04-08   Venta #231 (crédito)     +$890.000   $1.050.000     │
│  2026-03-30   Abono                    -$840.000     $160.000      │
│  ─────────────────────────────────────────────────────────────────  │
│                              [🖨 Imprimir estado de cuenta]        │
└─────────────────────────────────────────────────────────────────────┘
```

---

### Flujo 3: Apertura y Cierre de Caja (`/register`)

```
── Estado: Sin caja abierta ──────────────────────────────────────────
┌─────────────────────────────────────────────────────────────────────┐
│  🏧 CAJA                                                            │
│                                                                     │
│       No hay una caja abierta.                                      │
│       Para realizar ventas, debes abrir la caja primero.            │
│                                                                     │
│       Monto de apertura:                                            │
│       ┌──────────────────────────────────────────────┐             │
│       │  $  200.000                                   │             │
│       └──────────────────────────────────────────────┘             │
│                                                                     │
│                      [✅ Abrir caja]                                │
└─────────────────────────────────────────────────────────────────────┘

── Estado: Caja abierta (durante el día) ─────────────────────────────
┌─────────────────────────────────────────────────────────────────────┐
│  🏧 CAJA ABIERTA  desde 08:30 am    por: Laura                     │
│                                                                     │
│  Apertura:          $200.000                                        │
│  Ventas efectivo:   $850.000                                        │
│  Ingresos manual:    $50.000                                        │
│  Egresos manual:    -$20.000                                        │
│  ─────────────────────────────                                      │
│  Efectivo esperado: $1.080.000                                      │
│                                                                     │
│  MOVIMIENTOS DEL DÍA                                                │
│  ─────────────────────────────────────────────────────────────────  │
│  09:15  Venta #245  efectivo          +$85.000                     │
│  10:32  Venta #246  tarjeta            $32.000  (no suma a caja)   │
│  11:05  Ingreso manual               +$50.000   "vuelto cliente"   │
│  12:48  Venta #247  crédito           $350.000  (no suma a caja)   │
│  ─────────────────────────────────────────────────────────────────  │
│                                                                     │
│  [+ Registrar ingreso]  [- Registrar egreso]   [🔒 Cerrar caja]   │
└─────────────────────────────────────────────────────────────────────┘

── Modal: Cierre de caja ─────────────────────────────────────────────
┌─────────────────────────────────────────────────────────────────────┐
│  🔒 CIERRE DE CAJA                                                  │
│                                                                     │
│  Efectivo esperado:            $1.080.000                           │
│  Efectivo contado: [                    ]  ← ingresa el cajero     │
│  Diferencia:       (calculada al tipear)                           │
│                                                                     │
│  RESUMEN DE VENTAS                                                  │
│  Efectivo:   $850.000   │  Transferencia:  $120.000                │
│  Tarjeta:     $32.000   │  Crédito:        $350.000                │
│  Total ventas: $1.352.000    Nº transacciones: 18                  │
│                                                                     │
│  [Cancelar]                         [✅ Confirmar cierre]          │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Notas para próximos pasos

> **Separación backend/frontend:** Ya existe desde el inicio. El flujo de build es: `next build --export` → output a `/web` → Go embebe `/web` en el binario via `embed.FS` → un solo `pos.exe`, sin instalación.
