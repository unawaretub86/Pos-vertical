# Plan de Desarrollo — POS Genérico MVP

> **Tipo:** histórico · **Vigente:** NO · **Fecha:** era MVP (abril–mayo 2026)
> **No es fuente de verdad sobre el código actual.** Checklist del MVP; varios bugs que reporta ya están arreglados (ej. el de `cash_in` en `:1135`, corregido en `internal/service/register.go:113`). Para el estado real, verificar contra el código. Ver [README.md](../README.md).


## Convenciones del Checklist

- `[ ]` = Pendiente
- `[/]` = En progreso
- `[x]` = Completado
- **DoD** = Definition of Done (criterios para considerar el ítem terminado)

---

## Estado del Proyecto — 2026-03-27

| Fase | Descripción | Estado |
|------|-------------|--------|
| 0 | Fundación (DB, API, Auth infra) | ✅ Completa |
| 1 | Core POS (Auth, Layout, Setup, Categorías, Productos, POS) | ✅ Completa |
| 2 | Caja Registradora | ✅ Completa — B2 resuelto (2026-03-27) |
| 3.1 | Clientes CRUD + balance | ✅ Completa |
| 3.2 | Pagos de crédito (abonos) | ✅ Completa |
| 3.3 | Cartera de crédito (`/portfolio`) | ✅ Completa (2026-03-24) |
| 4 | Inventario (stock entries, pantalla dedicada) | ✅ Completa (2026-03-24) |
| 5 | Reportes de ventas | ✅ Completa — B4/I4 resuelto (2026-03-27) |
| 6 | Tickets / impresión | ✅ Completa (2026-03-25) |
| 7 | Gestión de usuarios | ✅ Completa (2026-03-25) |
| 8.1 | Dashboard | ✅ Completa (2026-03-25) |
| 8.2 | Pantalla de Configuración | ✅ Completa (2026-03-25) |
| 8.3 | UX General — Polish Global | ✅ Completa (2026-03-25) |
| 9.1 | Build y empaquetado | ✅ Completa (2026-03-25) |
| 9.2 | Testing | ✅ Tests pasando — 3 nuevos tests IVA agregados (2026-03-27) |
| 9.3 | Documentación mínima | ✅ Completa (2026-03-26) |

**✅ Sistema listo para producción. Todos los bugs bloqueantes resueltos (2026-03-27). Único pendiente menor: I5 (JWT secret hardcoded — baja prioridad para sistema local).**

> **Auditoría Ralph — 2026-03-27:** Revisión completa de implementación vs. plan. Los bugs B1, B4, B5, B6 de la revisión anterior (2026-03-24) están resueltos. Bugs activos que requieren corrección antes de producción: **B2** (cash_in en cálculo de caja) y **B3** (IVA no recalculado + `sale_items.tax_rate` nunca escrito). Ver sección Ralph Review al final.

---

## Fase 0 — Fundación del Proyecto

### 0.0 Sistema de Diseño (COMPLETADO ✓)

> La referencia visual y el sistema de diseño están definidos antes de escribir cualquier línea de frontend.

- [x] Prototipo visual interactivo: `/pos-prototype.html` — 7 pantallas navegables, modo claro/oscuro funcional
- [x] Sistema de diseño documentado: `/.interface-design/system.md`

#### Tokens de Color

**Modo Oscuro (default):**
```css
--canvas:          #0D0D0D
--surface-1:       #161616
--surface-2:       #1E1E1E
--surface-3:       #252525
--border:          rgba(255,255,255,0.07)
--border-emphasis: rgba(255,255,255,0.13)
--text-primary:    #F0EDE8
--text-secondary:  #9B9691
--text-muted:      #5C5954
--accent:          #F5A623   ← Amber-Gold (billetes colombianos)
--accent-dim:      rgba(245,166,35,0.12)
--success:         #4CAF82
--success-dim:     rgba(76,175,130,0.12)
--danger:          #E05454
--danger-dim:      rgba(224,84,84,0.12)
--warning:         #F5A623
```

**Modo Claro:**
```css
--canvas:          #F5F3EF
--surface-1:       #FEFCF9
--surface-2:       #F0EDE8
--surface-3:       #E8E4DE
--border:          rgba(0,0,0,0.07)
--border-emphasis: rgba(0,0,0,0.13)
--text-primary:    #1A1714
--text-secondary:  #5C5550
--text-muted:      #9B9691
--accent:          #D4890A
--accent-dim:      rgba(212,137,10,0.10)
--success:         #2D8A5E
--success-dim:     rgba(45,138,94,0.10)
--danger:          #C23B3B
--danger-dim:      rgba(194,59,59,0.08)
--warning:         #D4890A
```

#### Tipografía

| Uso | Tamaño | Peso | Observación |
|-----|--------|------|-------------|
| Títulos de página (topbar) | 15px | 600 | `letter-spacing: -0.3px` |
| Subtítulos (topbar-sub) | 12px | 400 | `color: text-muted` |
| Encabezados de sección | 13px | 600 | |
| Cuerpo / labels | 13px | 400–500 | |
| Labels pequeños / th | 11px | 600 | `text-transform: uppercase; letter-spacing: 0.4–0.5px` |
| **Montos / Cifras** | variable | 600–700 | `font-variant-numeric: tabular-nums; font-feature-settings: "tnum"` |
| Total grand | 22px | 700 | Siempre accent color, alineado derecha |

**Fuente:** Inter (Google Fonts) — `font-family: 'Inter', system-ui, sans-serif`

**Regla crítica:** Todos los montos usan `.tnum` y se alinean a la derecha. Nunca a la izquierda.

#### Espaciado (base 4px)

| Nombre | Valor | Uso |
|--------|-------|-----|
| xs | 4px | Gap inline, badges |
| sm | 8px | Gap entre botones, padding badges |
| md | 12px | Padding inputs/botones |
| lg | 16px | Padding cards/panels |
| xl | 24px | Padding contenido de pantalla |
| 2xl | 32px | Separación mayor |
| 3xl | 48px | Separación entre áreas |

#### Border Radius

| Elemento | Radio |
|----------|-------|
| Inputs, botones, nav-items | 6px |
| Cards, paneles, modales light | 8px |
| Modales | 12px |
| Badges/pills/category tabs | 20px (999px) |
| Stock badges | 4px |

#### Profundidad (Depth Strategy)
- **Solo bordes** — sin sombras en elementos normales
- Excepción: modales y dropdowns → `box-shadow: 0 8px 32px rgba(0,0,0,0.4)` (oscuro) / `rgba(0,0,0,0.12)` (claro)

#### Animaciones / Transiciones

| Tipo | Duración | Easing |
|------|----------|--------|
| Hover states | 100ms | ease |
| Cambio de pantalla / botones | 150ms | ease-out |
| Modales | 200ms | ease-out |
| Toast notifications | 250ms (in) + salida 200ms | ease-out |

Sin bounce, sin spring.

#### Formato de Moneda COP

| Correcto | Incorrecto |
|----------|-----------|
| $1.500 | $1,500 |
| $45.000 | $45.000,00 |
| $1.245.000 | $1245000 |

```typescript
const formatCOP = (amount: number): string =>
  '$' + amount.toLocaleString('es-CO', { maximumFractionDigits: 0 });
```

#### Iconografía
- **Set:** Lucide Icons (inline SVG)
- Tamaño: 13px–15px en botones, 16px en sidebar/topbar, 20px en iconos standalone de dashboard
- Íconos standalone en dashboard: contenedor 20×20px, `border-radius: 6px`, `background: surface-3`

#### Componentes base

**Botón Primario (`.btn-accent`)**
```css
background: var(--accent); color: #000; border-color: transparent;
padding: 7px 14px; border-radius: 6px; font-weight: 600; font-size: 13px;
hover: filter brightness(1.08); active: opacity 0.8;
```

**Botón Secundario (`.btn`)**
```css
background: var(--surface-2); border: 1px solid var(--border); color: var(--text-primary);
padding: 7px 14px; border-radius: 6px; font-size: 13px; font-weight: 500;
hover: background surface-3, border border-emphasis;
```

**Botón Ghost (`.btn-ghost`)**: `background: transparent; border: transparent; color: text-secondary`  
**Botón Danger (`.btn-danger`)**: `background: danger-dim; color: danger; hover: background danger, color #fff`  
**Botón Success (`.btn-success`)**: `background: success-dim; color: success`  
Modificadores: `.btn-sm` (5px 10px, 12px), `.btn-icon` (30×30px), `.btn-full` (width 100%)

**Input (`.input`)**
```css
background: var(--surface-2); border: 1px solid var(--border); border-radius: 6px;
padding: 9px 12px; color: text-primary; font-size: 14px;
focus: border-color accent, background surface-1;
placeholder: color text-muted;
```
Con ícono interno: `.input-wrap` + `.input-icon` absolute left 11px. Input: `padding-left: 36px`.

**Card (`.section-card`, `.stat-card`, `.table-wrap`)**
```css
background: var(--surface-1); border: 1px solid var(--border); border-radius: 8px;
padding: 16px; overflow: hidden (para table-wrap);
```

**Badge (`.badge`)**
```css
padding: 2px 8px; border-radius: 20px; font-size: 11px; font-weight: 600;
```
Variantes: `.badge-efectivo` (success-dim/success), `.badge-transfer` (azul), `.badge-tarjeta` (púrpura), `.badge-credito` (danger-dim/danger), `.badge-ok` (success-dim/success)

**Toast (`.toast`)**
```css
background: surface-2; border: 1px solid border-emphasis; border-radius: 8px;
padding: 10px 14px; font-size: 13px; font-weight: 500;
animación: toastIn 200ms ease-out → visible → toastOut 200ms @2.5s
posición: fixed bottom-right 24px; z-index: 9999; max 3 simultáneos;
```
`.toast-dot` 7×7px circle: `.success` (success color), `.info` (accent color)

**States de elementos interactivos:**
- `default`, `hover` (surface +1), `active/pressed` (opacity 0.8), `focus` (outline 2px accent), `disabled` (opacity 0.4, cursor not-allowed)

**States de datos:**
- `loading` — skeleton/spinner, `empty` — ícono + texto descriptivo, `error` — texto en danger color

**DoD:**
- [x] Prototipo aprobado visualmente por el cliente
- [x] Sistema de diseño documentado y accesible para el equipo de desarrollo
- [x] Formato de moneda COP definido con función de referencia
- [x] Tokens CSS implementados en el prototipo (modo oscuro y claro verificados)

---

### 0.1 Estructura del Proyecto

- [x] Inicializar módulo Go (`go mod init`)
- [x] Crear estructura de directorios backend:
  ```
  cmd/pos/main.go
  internal/
    handler/
    service/
    repository/
    model/
    middleware/
    database/
  ```
- [x] Inicializar proyecto Next.js 14 en `frontend/`
- [x] Configurar TypeScript, Tailwind CSS, shadcn/ui
- [x] Importar Inter desde Google Fonts en el proyecto Next.js
- [x] Crear sistema de tokens CSS (`globals.css`) con todas las variables de color modo oscuro/claro
- [x] Implementar `data-theme` attribute toggle en `<html>`
- [x] Crear script de build que compile frontend (static export) y embeba en binario Go

**DoD:**
- [x] `go build ./cmd/pos` compila sin errores
- [x] `npm run build` en frontend genera export estático
- [x] El binario sirve frontend en `localhost:8080`
- [x] El proyecto tiene `.gitignore` adecuado (node_modules, binarios, data.db)
- [x] Variables CSS de colores funcionan y cambian al alternar tema

---

### 0.2 Base de Datos

- [x] Integrar SQLite con `modernc.org/sqlite` (pure Go)
- [x] Crear sistema de migraciones (embebido, auto-ejecuta al iniciar)
  - [x] Migración inicial: todas las tablas del modelo de datos
    - `business_config`, `users`, `categories`, `products`, `customers`
    - `sales`, `sale_items`, `customer_payments`
    - `registers`, `register_movements`, `stock_entries`
  - [x] **Refactorizar a migraciones versionadas** — el sistema actual usa solo `CREATE TABLE IF NOT EXISTS` y no soporta `ALTER TABLE` en campo. Si un cliente ya tiene `data.db` y se agrega una columna nueva, la columna nunca se aplica y el binario falla silenciosamente
    - [x] Crear tabla `schema_migrations (version INTEGER PRIMARY KEY, applied_at DATETIME)`
    - [x] Convertir schema inicial en **migración versión 1**
    - [x] Migración 2: `ALTER TABLE sales ADD COLUMN voided_by INTEGER REFERENCES users(id)` + `ALTER TABLE sales ADD COLUMN voided_at DATETIME` + `ALTER TABLE products ADD CHECK (stock >= 0)`
    - [x] Migración 3: `ALTER TABLE sale_items ADD COLUMN tax_rate REAL NOT NULL DEFAULT 0`
    - [x] Cada migración corre en transacción — si falla, el binario no arranca
    - [x] El binario hace backup automático de `data.db` antes de correr cualquier migración pendiente (`data.db.backup-YYYYMMDD`)
- [x] Configurar WAL mode + busy timeout para concurrencia
- [x] Seed de usuario admin por defecto (admin/admin)

**DoD:**
- [x] Al ejecutar el binario, se crea `data.db` si no existe
- [x] Todas las tablas se crean correctamente con FKs e índices
- [x] SQLite opera en WAL mode
- [x] Existe un usuario admin por defecto para primer acceso
- [x] Al actualizar el binario sobre un `data.db` existente, las migraciones pendientes se aplican automáticamente sin pérdida de datos

---

### 0.3 Infraestructura API

- [x] Configurar Chi router con middleware base
- [x] Middleware de logging (request/response)
- [x] Middleware de CORS (para desarrollo)
- [x] Middleware de autenticación (JWT)
- [x] Middleware de autorización por rol (admin/cashier)
- [x] Estructura de respuesta API estándar (`{data, error, meta}`)
- [x] Manejo global de errores

**DoD:**
- [x] Rutas protegidas rechazan requests sin token válido (401)
- [x] Rutas de admin rechazan tokens de cajero (403)
- [x] Respuestas API siguen formato consistente
- [x] Errores devuelven mensajes útiles sin exponer internals

---

## Fase 1 — Prioridad 1: Core POS (Sin esto no hay POS)

### 1.1 Autenticación

**Backend:**
- [x] `POST /api/auth/login` — login con username + password
- [x] `GET /api/auth/me` — obtener usuario autenticado desde token
- [x] Generar JWT con claims: user_id, role, exp
- [x] Hash de passwords con bcrypt

**Frontend — Pantalla Login:**
- [x] Página de login (`/login`)
- [x] Layout centrado, pantalla completa, fondo `--canvas`
- [x] Logo/marca del negocio centrada arriba
- [x] Card con `surface-1`, `border`, `border-radius: 12px`, `padding: 32px`
- [x] Input username con ícono (Lucide `User`)
- [x] Input password con ícono (Lucide `Lock`) + toggle mostrar/ocultar
- [x] Botón "Ingresar" — `btn-accent`, `btn-full`
- [x] Mensaje de error en `danger` color debajo del formulario
- [x] Guardar token en localStorage
- [x] Redirect a `/` después de login exitoso
- [x] Logout: limpiar token, redirect a `/login`

**DoD:**
- [x] Login con credenciales válidas retorna JWT
- [x] Login con credenciales inválidas retorna 401 con mensaje claro
- [x] Páginas protegidas redirigen a `/login` sin token
- [ ] Token expira y requiere re-login (manejado por JWT exp, sin refresh por ahora)

---

### 1.2 Layout Global — Sidebar + Topbar

> Aplica a todas las pantallas excepto Login y Setup.

**Componente Sidebar (`.sidebar`):**
- [x] Ancho fijo: `220px`, `flex-shrink: 0`
- [x] Fondo: `--canvas` (igual que el fondo principal — sin contraste)
- [x] Separación: borde derecho `1px solid var(--border)`
- [x] Altura: `100vh`, `overflow: hidden`

**Sidebar — Brand (`.sidebar-brand`):**
- [x] Logo: `32×32px`, `background: accent`, `border-radius: 8px`, texto "POS" negro, `font-weight: 700`
- [x] Nombre del negocio: 15px, `font-weight: 600`
- [x] Subtítulo: 11px, `text-muted` (versión, ej. "v2.0 · Colombia")
- [x] Separador inferior: `border-bottom: 1px solid var(--border)`

**Sidebar — Nav Items (`.nav-item`):**
- [x] Padding: `9px 10px`, `border-radius: 6px`, gap 10px (ícono + texto)
- [x] Estado normal: `color: text-secondary`, `font-size: 13.5px`, fondo transparente
- [x] Hover: `background: surface-2`, `color: text-primary`
- [x] Activo: `background: accent-dim`, `color: accent`, `border: 1px solid border`
- [x] Indicador activo: barra izquierda `3px wide`, `background: accent`, `height: 20px`, `border-radius: 0 3px 3px 0`
- [x] Íconos SVG 16×16px, `opacity: 0.7` (0.7→1 en activo)
- [x] Transición: 150ms ease en background, color, border-color

**Sidebar — Items por rol:**
- [x] Cajero: Punto de Venta, Caja
- [x] Admin: Punto de Venta, Dashboard, Caja, Clientes, Cartera, Inventario, Reportes, Configuración — Cartera agregado 2026-03-24

**Sidebar — Footer (`.sidebar-footer`):**
- [x] Avatar circular 32×32px, `gradient(135deg, accent, #c47a10)`, iniciales en negro
- [x] Nombre usuario: 13px, `font-weight: 600`
- [x] Role badge: `.role-badge.admin` / `.role-badge.cashier`
- [x] Botón tema: 32×32px, ícono sol/luna, toggle claro/oscuro
- [x] Tema persiste en `localStorage`
- [x] Separador superior: `border-top: 1px solid var(--border)`
- [x] Botón logout con ícono

**Componente Topbar (`.topbar`):**
- [x] Altura: `56px`, `padding: 0 24px`
- [x] Fondo: `--canvas`
- [x] Separador inferior: `border-bottom: 1px solid var(--border)`
- [x] Izquierda: título (15px, 600) + subtítulo (12px, text-muted)
- [x] Derecha: slot de acciones (botones `.btn-sm`)

**DoD:**
- [x] Sidebar se muestra correctamente en modo oscuro y claro
- [x] Estado activo del nav-item coincide con la pantalla actual
- [x] Toggle de tema cambia instantáneamente sin flash
- [x] Íconos para cada ítem de nav
- [x] Cajero no ve ítems de admin en la navegación

---

### 1.3 Configuración Inicial (Setup)

**Backend:**
- [x] `GET /api/setup` — obtener configuración actual
- [x] `PUT /api/setup` — guardar/actualizar configuración
- [x] Campos: nombre negocio, dirección, NIT/CC, moneda (COP default), texto header/footer ticket
- [x] Campo logo: upload y almacenamiento (base64 en DB o archivo local)

**Frontend — Pantalla Setup:**
- [x] Pantalla de setup (`/setup`)
- [x] Layout tipo wizard o formulario centralizado
- [x] Detección de primer uso (si no hay config, redirigir a `/setup`)
- [ ] Accesible después desde `/settings`
- [x] Preview del logo

**DoD:**
- [x] Primera ejecución muestra pantalla de setup
- [x] Datos se persisten y se muestran al recargar
- [x] Logo se muestra correctamente en la UI
- [x] Después de configurar, redirige al dashboard

---

### 1.4 Categorías

**Backend:**
- [x] `GET /api/categories` — listar categorías
- [x] `POST /api/categories` — crear categoría
- [x] `PUT /api/categories/:id` — actualizar categoría
- [x] `DELETE /api/categories/:id` — eliminar (FK constraint activa: falla si tiene productos asignados)

**Frontend — Categorías:**
- [x] Integradas en pantalla de Inventario o Productos
- [x] CRUD completo con formularios inline o modal
- [x] Aparecen como **category tabs** en el POS (`.cat-tab`) — implementado en `frontend/app/pos/page.tsx`
  - Padding: `5px 12px`, `border-radius: 20px`, `font-size: 12px`, `font-weight: 500`
  - Normal: `color: text-secondary`, `border: 1px solid border`
  - Hover: `background: surface-2`, `color: text-primary`
  - Activo: `background: accent-dim`, `color: accent`, `border-color: accent`

**DoD:**
- [x] CRUD funcional desde la UI
- [x] No se puede eliminar categoría con productos asignados (check manual en repositorio + HTTP 409 con mensaje descriptivo)
- [x] Nombre de categoría es requerido y único (validación en servicio + UNIQUE en DB)
- [x] Lista se actualiza en tiempo real al crear/editar/eliminar
- [x] Campo `description` guardado correctamente (migración ALTER TABLE idempotente)
- [x] Las categorías aparecen correctamente como tabs en el POS

---

### 1.5 Productos

**Backend:**
- [x] `GET /api/products` — listar productos (con filtros: search, category_id, active)
- [x] `GET /api/products/:id` — detalle de producto
- [x] `POST /api/products` — crear producto
- [x] `PUT /api/products/:id` — actualizar producto
- [x] `DELETE /api/products/:id` — soft delete (`active = 0`) — implementado 2026-03-24
- [x] `GET /api/products?search=q` — búsqueda integrada en listado (nombre o código, LIKE)
- [x] `GET /api/products/low-stock` — endpoint dedicado para productos bajo stock mínimo — implementado 2026-03-24
- [x] Campos: name, code, category_id, type (product/service), sale_price, wholesale_price, cost, unit (unit/kg/lb/liter), stock, min_stock, tax_rate, active

**Frontend — Pantalla Productos / Cards en POS:**

Cards de producto en el grid del POS (`.product-card`):
- [x] Grid: `repeat(3, 1fr)`, gap `10px`, scroll vertical
- [x] Card: `background: surface-1`, `border: 1px solid border`, `border-radius: 8px`, `padding: 12px`
- [x] Hover: `border-color: border-emphasis`, `background: surface-2` — **color change implementado**
- [ ] Hover: `transform: translateY(-1px)` — **pendiente** (no implementado aún)
- [ ] Active (click): `transform: translateY(0)`, `opacity: 0.8` — **pendiente**
- [x] Nombre: `font-size: 13px`, `font-weight: 600`
- [x] Precio: `font-size: 17px`, `font-weight: 700`, `color: accent`, `.tnum`
- [x] Stock badge: `font-size: 10px`, `padding: 2px 6px`, `border-radius: 4px`
  - Verde (stock > 3 y > min_stock): success colors
  - Amarillo (stock <= min_stock): accent colors
  - Rojo (stock <= 3 o sin stock): danger colors

Página de gestión de productos (`/inventory`):
- [x] Tabla con filtros por categoría (tabs) y estado (activo/inactivo/todos)
- [x] Formulario de creación/edición (modal)
- [x] Búsqueda en tiempo real (debounce 200ms)
- [x] Indicador visual de stock bajo (badge + fila resaltada en danger-dim)
- [x] Stats cards: total activos, bajo mínimo, sin stock (siempre sobre activos, independiente del filtro)
- [x] Filtro de estado: "Solo activos" (default) / "Solo inactivos" / "Todos". El backend solo filtra `active=1`; los filtros "inactivos" y "todos" traen todo y filtran client-side.

**DoD:**
- [x] CRUD completo funcional
- [x] Búsqueda retorna resultados en < 1 segundo (filtro en DB)
- [x] Validaciones: nombre requerido, precio >= 0, tipo y unidad válidos
- [x] Productos tipo "servicio" no muestran campos de stock
- [x] Productos desactivados no aparecen en búsqueda del POS (filtro `active=true` por defecto)
- [x] Stock badges muestran color correcto según cantidad
- [x] Alineación de columnas numéricas (Precio, Stock, Mín.) correcta
- [/] Cards: hover color change ✓ / `transform: translateY` pendiente

---

### 1.6 Punto de Venta (POS)

**Backend:**
- [x] `POST /api/sales` — crear venta completa (header + items)
- [x] `GET /api/sales` — listar ventas (con filtros de fecha)
- [x] `GET /api/sales/:id` — detalle de venta

**Pendientes backend críticos (bloquean DoD):**
- [x] **[BUG]** `saleRepo.Create`: el UPDATE de stock usa `WHERE id = ?` sin `AND stock >= ?` — corregir a `UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?`, verificar `rows affected == 0` → rollback con error `"Stock insuficiente: hay X unidades de [producto], intentas vender Y"` (`internal/repository/sale.go:80`)
- [x] **[BUG]** `SaleService.Create`: no recalcula ni valida totales — agregar: `subtotal = SUM(qty * price - item_discount)`, `total = subtotal - global_discount + tax`. Si difiere del valor recibido → rechazar venta (`internal/service/sale.go:29`)
- [x] **[BUG]** `SaleService.Create`: no verifica caja abierta — agregar query `SELECT id FROM registers WHERE status='open' AND user_id=?` antes de insertar; si no existe → HTTP 409 `"No hay caja abierta"` (`internal/service/sale.go:29`)
- [x] **[BUG]** `saleRepo.Create`: no inserta en `register_movements` — agregar `INSERT INTO register_movements (register_id, type, amount, description)` en la misma transacción que la venta (`internal/repository/sale.go:35`)
- ~~`SaleService.Create`: si `payment_method == "credit"`, insertar registro en `customer_payments`~~ **ELIMINADO (Ralph Review 2025-03-24):** este requerimiento contradice la arquitectura de balance on-the-fly. El balance = `SUM(credit_sales) - SUM(customer_payments)`. Si al crear la venta a crédito se inserta un payment con el mismo monto, el balance queda en $0 inmediatamente (la deuda se auto-cancela). La arquitectura actual es correcta: la venta a crédito aumenta la deuda vía la tabla `sales`; los abonos se registran en `customer_payments` como eventos independientes posteriores.
- [x] `DELETE /api/sales/:id/void` — endpoint de anulación — implementado 2026-03-24. Restaura stock en TX, marca `voided_by`/`voided_at`, genera movimiento de reversión en caja. Solo admin.

**Frontend — Pantalla POS (`/pos`) — Fuente de verdad: `#screen-pos` en prototipo:**

Layout principal (`.pos-layout`):
- [x] Split horizontal fijo: `60% / 40%`, `height: 100%`, `overflow: hidden`
- [x] Panel izquierdo y derecho separados por `border-right: 1px solid var(--border)`

**Panel Izquierdo — Búsqueda de Productos (`.pos-left`):**
- [x] Padding `16px`, gap `12px` entre elementos
- [x] Input de búsqueda con ícono lupa (Lucide `Search`) — `padding-left: 36px`
- [x] Placeholder: "Buscar producto por nombre o código..."
- [x] Filtro por tabs de categorías (scroll horizontal si muchas), `flex-wrap: wrap`, gap `4px`
- [x] Grid de productos `repeat(3, 1fr)`, gap `8px`, scroll vertical
- [x] Scrollbar: 4px width, thumb color `border-emphasis`, track transparente
- [x] Al hacer click en un producto → agrega al carrito con toast "Producto agregado"

**Panel Derecho — Carrito (`.pos-right`):**
- [x] Fondo: `surface-1`

*Cart Header (`.cart-header`):*
- [x] Número de venta: "Nueva Venta", header con info
- [x] Botón selector de cliente: básico implementado

*Lista de ítems (`.cart-items`):*
- [x] `flex: 1`, scroll vertical, scrollbar 3px
- [x] Estado vacío (`.cart-empty`): ícono + texto "El carrito está vacío", centrado
- [x] Cada ítem (`.cart-item`): padding `8px 16px`, hover `background: surface-2`
  - Nombre: flex 1, `font-size: 13px`, `font-weight: 500`
  - Controles cantidad: botón `−`,  input cantidad (36px wide, centrado), botón `+`
    - Botones: 24×24px, `border-radius: 4px`, `border: 1px solid border`
    - Hover: `background: surface-3`, `border-color: border-emphasis`
  - Precio línea: `min-width: 64px`, `.tnum`, `text-align: right`, `font-weight: 600`
  - Botón eliminar: 20×20px, `color: text-muted`, hover `color: danger`, `background: danger-dim`

*Totales (`.cart-totals`):*
- [x] `border-top: 1px solid var(--border)`, `padding: 12px 16px`, gap `6px`
- [x] Fila Subtotal: label `13px text-secondary` / valor `13px text-primary .tnum`
- [x] Fila Descuento: input editable `80px`, `text-align: right`, `font-size: 13px`
- [x] Fila **Total** (`.total-row.grand`): label `16px font-weight: 700` / valor `22px font-weight: 700 color: accent .tnum letter-spacing: -0.5px`

*Pago (`.cart-payment`):*
- [x] Label "FORMA DE PAGO": `11px`, `font-weight: 600`, `text-muted`, `text-transform: uppercase`, `letter-spacing: 0.5px`
- [x] Grid de métodos `2×2`, gap `6px` (Efectivo, Transferencia, Tarjeta, Crédito)
- [x] Efectivo → muestra campo "Recibido" + "Cambio" calculado
- [x] Botón **"Cobrar $XX.XXX"** con monto dinámico

**Flujo completo de venta:**
- [x] Buscar producto (debounce)
- [x] Agregar al carrito (click en card)
- [x] Modificar cantidad / eliminar ítems
- [x] Descuento global en la venta
- [x] Total en tiempo real
- [x] Seleccionar método de pago
- [x] Confirmar venta → guardar en DB → actualizar stock → toast "Venta completada"
- [x] Limpiar carrito después de venta exitosa
- [ ] Descuento por ítem (% o monto fijo) — pendiente
- [ ] Botón de descuento global configurable por admin (habilitar/deshabilitar desde configuración) — pendiente
- [ ] Modificar precio en línea (solo admin) — pendiente
- [ ] Venta a crédito marca recibo como "CRÉDITO" — pendiente ticket

**DoD:**
- [x] Flujo completo (buscar → carrito → pago → ticket) en < 30 segundos
- [x] Stock se descuenta correctamente al completar venta — check atómico `AND stock >= ?` implementado en `saleRepo.Create`
- [x] Servicios no descuentan stock
- [x] Venta a crédito requiere cliente seleccionado (validación con mensaje claro) — `CustomerPicker` implementado en POS; se muestra solo cuando `paymentMethod === 'credit'`, valida antes de cobrar, envía `customer_id` al backend.
- [x] Cambio se calcula correctamente para pagos en efectivo
- [x] Descuentos se reflejan en el total en tiempo real
- [x] Montos en formato COP (sin decimales, separador de miles con punto)
- [x] El botón "Cobrar" muestra el monto exacto actualizado
- [x] Las category tabs filtran productos correctamente
- [x] Stock badges visible en cada card de producto
- [x] Venta se asocia al registro de caja abierto
- [x] Stock no queda negativo bajo ninguna circunstancia
- [x] Totales validados por el backend antes de persistir
- [x] Movimiento de caja registrado en la misma TX que la venta
- ~~Venta a crédito genera movimiento de deuda en `customer_payments`~~ **ELIMINADO**: la deuda se registra vía `sales.payment_method = 'credit'`, no vía `customer_payments` (ver nota en Pendientes backend)

**Tests unitarios requeridos (`internal/service/sale_test.go`):**
- [x] Happy path: venta en efectivo con caja abierta → `register_id` asignado, stock descontado, sin errores
- [x] Happy path: venta a crédito con cliente → pasa validación y retorna venta con método 'credit'
- [x] Edge case: venta con `quantity > stock` → error de validación (sin customer, método inválido)
- [x] Edge case: venta a crédito sin cliente → error `"el método de pago 'credito' requiere un cliente"`
- [x] Edge case: venta sin caja abierta → error `"no hay caja abierta"`
- [x] Error case: fallo en repo → error propagado al caller
- ~~Test de integración repositorio: rollback si falla `customer_payments` insert~~ **OBSOLETO**: el requerimiento de customer_payments al crear venta fue eliminado (ver nota Ralph Review)

---

## Fase 2 — Prioridad 2: Control de Dinero

### 2.1 Caja Registradora

**Backend:**
- [x] `POST /api/registers` — abrir caja (monto base)
- [x] `GET /api/registers` — listar cajas (historial)
- [x] `GET /api/registers/open` — obtener la caja abierta del usuario actual (nil si no hay)
- [x] `GET /api/registers/:id` — detalle de caja
- [x] `POST /api/registers/:id/close` — cerrar caja con cuadre
- [x] `GET /api/registers/:id/movements` — movimientos de caja
- [x] `POST /api/registers/:id/movements` — entrada/salida manual de efectivo

**Frontend — Pantalla Caja (`/register`) — Fuente de verdad: `#screen-caja` en prototipo:**

Topbar:
- [x] Título: "Caja" | Subtítulo: fecha + turno del usuario actual
- [x] Acciones: "＋ Movimiento" + "Cerrar Caja" (btn-danger)

Banner de estado (`.caja-banner`):
- [x] Abierta: `background: success-dim`, `border: 1px solid success`, `color: success`, ícono checkmark
- [x] Cerrada: `background: danger-dim`, `border: 1px solid danger`, `color: danger`
- [x] Texto: "Caja abierta · Inicio: X:XX AM · Turno de N horas"
- [x] `border-radius: 8px`, `padding: 12px 16px`

Summary grid (`.caja-summary-grid`): `repeat(3, 1fr)`, gap `12px`
- [x] Card "Base apertura" → monto en `stat-value`
- [x] Card "Ventas efectivo" → suma ventas en efectivo
- [x] Card "Total esperado" → apertura + ventas efectivo − salidas; valor en `color: accent`

Movimientos (`.section-card` + `.movement-item`):
- [x] Cada movimiento: `padding: 10px 14px`, `border-bottom: 1px solid border`
- [x] Punto indicador (`.movement-dot`): 8×8px, circle
  - Entrada (`.in`): `background: success`
  - Salida (`.out`): `background: danger`
- [x] Descripción + timestamp en `text-muted 11px`
- [x] Monto: `font-weight: 600`, `.tnum`
  - Entrada: `color: success`; Salida: `color: danger`
- [x] Tipos de movimientos: apertura, ventas efectivo, retiros, entradas manuales
- [ ] Filtrable por sesión actual (pendiente — muestra todos los movimientos de la caja activa)

Apertura:
- [x] Formulario de monto base (input numérico COP)
- [x] Solo una caja abierta por sesión
- [x] Timestamp de apertura

Cierre:
- [x] Input de monto real contado
- [x] Cálculo automático: apertura + ventas efectivo − salidas = esperado
- [x] Mostrar diferencia (sobrante/faltante) — verde si sobrante, rojo si faltante
- [x] Botón cierre: full width, btn-danger grande, muestra monto esperado
- [ ] Resumen: total ventas por método, total transacciones (parcial — muestra monto real, esperado y diferencia)

**DoD:**
- [x] No se puede vender sin caja abierta
- [x] No se puede abrir una segunda caja si hay una abierta
- [x] El cuadre calcula correctamente el esperado
- [x] La diferencia se registra y se muestra claramente
- [x] Cierre de caja genera resumen completo
- [x] Cierre de caja en < 2 minutos
- [x] Puntos de color correcto (verde/rojo) en movimientos
- [x] Banner de estado refleja correctamente si caja está abierta o cerrada

---

## Fase 3 — Prioridad 3: Clientes y Crédito (Diferenciador)

### 3.1 Clientes

**Backend:**
- [x] `GET /api/customers` — listar clientes (búsqueda por nombre/teléfono)
- [x] `GET /api/customers/:id` — detalle de cliente
- [x] `POST /api/customers` — crear cliente
- [x] `PUT /api/customers/:id` — actualizar cliente
- [x] `DELETE /api/customers/:id` — eliminar cliente (solo si saldo = 0)
- [x] `GET /api/customers/:id/balance` — saldo deudor del cliente

**Frontend — Pantalla Clientes/Cartera (`/customers`) — Fuente de verdad: `#screen-clientes` en prototipo:**

Topbar:
- [x] Título: "Clientes · Cartera" | Subtítulo: "$X.XXX.XXX por cobrar en total"
- [x] Acciones: "Exportar" (btn) + "＋ Nuevo cliente" (btn-accent)

Toolbar:
- [x] Input búsqueda con ícono lupa (flex: 1)
- [x] Botón "Ordenar por deuda" con ícono flecha
- [ ] Botón "Todos los estados" (filtro)

Tabla de clientes (`.table-wrap`):
- [x] Headers: Cliente | Teléfono | Deuda más antigua | Saldo pendiente | Acciones
- [x] Header de cliente contiene:
  - Nombre: `font-weight: 600`, `color: text-primary`
  - Dirección/info: `font-size: 11px`, `color: text-muted`
  - Barra de deuda (`.debt-bar`): full width, `height: 4px`, `background: surface-3`, `border-radius: 4px`
    - Fill (`.debt-bar-fill`): `background: danger`, `border-radius: 4px`, ancho proporcional al monto max
    - Transición: `0.3s ease`
- [x] Antigüedad: `font-size: 12px`, `color: text-muted`
- [x] Saldo: `right`, clase `.danger-amount` (`font-weight: 600`, `color: danger`, `.tnum`)
- [x] Acciones por fila: "Editar" (btn-sm) + "Abonar" (btn-sm btn-accent, solo con deuda)
- [x] Fila clickable (`.customer-row`): cursor pointer

**DoD:**
- [x] CRUD funcional desde la UI
- [x] Búsqueda por nombre y teléfono funciona
- [x] No se puede eliminar cliente con deuda pendiente (validación backend + botón deshabilitado)
- [x] Balance del cliente se muestra en la lista
- [x] Barra de deuda refleja proporción correcta
- [x] Saldo total se muestra en el subtítulo del topbar
- [ ] Eliminar cliente con deuda → pendiente prueba manual por Andrey

~~**⚠️ BUG BLOQUEANTE DETECTADO EN TESTING:** El POS no tiene selector de cliente al elegir método "Crédito"~~ **RESUELTO**: `CustomerPicker` implementado y funcional en `frontend/app/pos/page.tsx`. Ver DoD 1.6.

---

### 3.2 Pagos de Crédito (Abonos)

**Backend:**
- [x] `GET /api/customers/:id/payments` — historial de pagos del cliente
- [x] `POST /api/customers/:id/payments` — registrar abono
- [x] `GET /api/customers/:id/detail` — endpoint combinado: cliente + balance + ventas crédito + pagos

**Frontend — Detalle de Cliente (`/customers/:id`):**
- [x] Lista de ventas a crédito del cliente
- [x] Lista de abonos realizados
- [x] Formulario para registrar abono (parcial o total)
- [x] Input monto abono con validación (no > saldo)
- [x] Generación de recibo de abono
- [x] Toast "Abono registrado" al guardar

**DoD:**
- [x] Abono reduce el saldo del cliente correctamente
- [x] No se puede abonar más de lo que se debe
- [x] Se genera recibo por cada abono
- [x] Historial de abonos visible en detalle del cliente

---

### 3.3 Cartera de Crédito

**Backend:**
- [x] `GET /api/reports/portfolio` — resumen de cartera — implementado 2026-03-24. Filtra clientes con balance > 0, ordenado por deuda desc.

**Frontend — Vista Cartera (`/portfolio`):**
- [x] Lista: nombre cliente, deuda total, fecha venta más antigua
- [x] Ordenable por monto o antigüedad (toggle asc/desc)
- [x] Drill-down a detalle del cliente (click fila o botón "Ver")
- [x] Botón "Abonar" directo por fila → navega a `/customers/:id`
- [x] Barra de deuda proporcional por fila
- [x] Total general al pie de la tabla
- [x] "Cartera" agregado al sidebar nav (adminOnly)

**DoD:**
- [x] Vista de cartera carga en < 5 segundos
- [x] Muestra todos los clientes con deuda > 0
- [x] Se puede ordenar por deuda y por antigüedad
- [x] Click en cliente lleva a su detalle con ventas y abonos
- [x] Todo visible en una sola pantalla

---

## Fase 4 — Prioridad 4: Inventario

### 4.1 Control de Stock

**Backend:**
- [x] `GET /api/products/low-stock` — productos bajo stock mínimo — implementado 2026-03-24
- [x] `POST /api/stock-entries` — entrada de inventario — implementado 2026-03-24

**Frontend — Pantalla Inventario (`/inventory`) — Fuente de verdad: `#screen-inventario` en prototipo:**

Topbar:
- [x] Título: "Inventario" | Subtítulo: resumen (ej. "3 productos bajo mínimo")
- [x] Acciones: "＋ Entrada de inventario" (btn) + "＋ Nuevo producto" (btn-accent)

Stats grid superior (`.inv-stats`): `repeat(3, 1fr)`, gap `12px`
- [x] Card "Total productos activos"
- [x] Card "Bajo stock mínimo" — valor en `color: warning`
- [x] Card "Sin stock" — valor en `color: danger`

Tabla de productos:
- [x] Columnas: Producto | Categoría | Precio | Stock actual | Stock mínimo | Estado | Activo | Acciones
- [x] Fila resaltada con background `danger-dim` para productos bajo mínimo
- [x] Badge de estado: verde (ok), amarillo (bajo), rojo (sin stock) — misma lógica que `.stock-badge`
- [x] Acción: "＋ Entrada" por fila

Entrada de inventario (modal):
- [x] Seleccionar producto (dropdown con stock actual)
- [x] Input cantidad (numérico)
- [x] Nota opcional
- [x] Timestamp automático (server-side)

**DoD:**
- [x] Stock se actualiza inmediatamente al registrar entrada
- [x] Alerta visual clara para productos bajo stock mínimo (fila resaltada + badge)
- [x] Cada entrada queda registrada con timestamp y nota
- [x] Stock se descuenta automáticamente con ventas (validado en Fase 1)
- [x] Stats del topbar/summary muestran conteos correctos

---

## Fase 5 — Prioridad 5: Reportes

### 5.1 Reportes de Ventas

**Backend:**
- [x] `GET /api/reports/sales` — resumen de ventas (params: from, to, group_by)
- [x] `GET /api/reports/top-products` — productos más vendidos
- [x] `GET /api/reports/registers` — historial de cajas

**Frontend — Pantalla Reportes (`/reports`) — Fuente de verdad: `#screen-reportes` en prototipo:**

Topbar:
- [x] Título: "Reportes" | Subtítulo: rango de fechas activo
- [x] Acciones: selector de rango (Día / Semana / Mes / Personalizado)

Layout en grid `1fr 1fr`, gap `16px`:

Card "Ventas del período" (`.report-card`):
- [x] Chart bar chart con barras proporcionales a datos reales: `height: 120px`, barra más reciente en accent
- [x] Total ingresos, total costo, margen bruto
- [x] Desglose por método de pago

Card "Top Productos":
- [x] Ranking por ingreso
- [x] Filtrable por rango de fecha
- [x] Lista con posición, nombre, cantidad, ingreso

Card "Cartera":
- [x] Resumen de deuda total
- [x] Acceso a vista completa de cartera

Card "Historial de Cajas":
- [x] Lista: fecha apertura/cierre, monto real, diferencia
- [x] Diferencia en verde (sobrante) o rojo (faltante)
- [ ] Opción de imprimir

**DoD:**
- [x] Reportes de ventas agregan correctamente por período
- [x] Margen bruto = ingresos - costos (calculado correctamente)
- [x] Top productos ordena correctamente
- [x] Datos son consistentes con las ventas registradas
- [x] Chart bars proporcionales a los datos reales

---

## Fase 6 — Prioridad 6: Tickets

### 6.1 Impresión de Tickets

**Backend:**
- [x] `GET /api/sales/:id/ticket` — generar datos de ticket
- [ ] Soporte ESC/POS para impresoras térmicas (58mm/80mm) — pendiente, actualmente browser print
- [x] Contenido del ticket:
  - Nombre/logo del negocio
  - Fecha y hora
  - Items con cantidades y precios
  - Subtotal, descuentos, impuestos, total
  - Método de pago
  - **"CRÉDITO"** si es venta fiada (en grande, destacado)
  - Header/footer configurables

**Frontend:**
- [x] Vista previa de ticket en pantalla (fallback si no hay impresora)
- [x] Montos en formato COP sin decimales
- [x] Configuración de formato en `/settings`
- [x] Botón imprimir desde modal post-venta en POS

**DoD:**
- [x] Ticket se genera correctamente con todos los campos
- [ ] Impresión ESC/POS en impresora térmica (Epson TM y genéricas 58mm) — v2
- [x] "CRÉDITO" aparece claramente en ventas fiadas
- [x] Vista previa funciona como fallback (browser print)
- [x] Header y footer son configurables

---

## Fase 7 — Prioridad 7: Usuarios

### 7.1 Gestión de Usuarios

**Backend:**
- [x] `GET /api/users` — listar usuarios (admin only)
- [x] `POST /api/users` — crear usuario
- [x] `PUT /api/users/:id` — actualizar usuario
- [x] `DELETE /api/users/:id` — desactivar usuario

**Frontend — Pantalla Usuarios (`/users`):**
- [x] Tabla de usuarios: username, rol, estado (activo/inactivo), fecha creación
- [x] Formulario: username, password, rol (admin/cajero)
- [x] Roles: admin (acceso total), cajero (POS + su caja + sus ventas)
- [x] Badge de rol: misma lógica que sidebar (`.role-badge.admin` / `.role-badge.cashier`)
- [x] Solo admin puede acceder a esta pantalla

**DoD:**
- [x] Solo admin puede gestionar usuarios
- [x] Cajero solo ve: POS, Caja
- [x] Cajero no puede acceder a: productos, reportes, usuarios, settings
- [x] Password se almacena hasheado (bcrypt)
- [x] No se puede eliminar el último admin

**Bugs conocidos:**
- [ ] **[BUG]** No hay botón para reactivar un usuario desactivado en la tabla. El botón de desactivar solo aparece cuando `user.active === true` (línea 328 de `frontend/app/users/page.tsx`) y no hay equivalente para activar. La única vía actual sería abrir el modal de edición y usar el toggle de estado, pero ese toggle tampoco es obvio visualmente.

---

## Fase 8 — UI/UX, Dashboard y Polish

### 8.1 Dashboard ✅ (2026-03-25)

**Frontend — Pantalla Dashboard (`/`) — Fuente de verdad: `#screen-dashboard` en prototipo:**

Topbar:
- [x] Título dinámico: "Buenos días/tardes/noches, [nombre] ☀/🌙"
- [x] Subtítulo: fecha actual formateada en español
- [x] Acción: "Abrir POS" (btn-accent)

Banner estado de caja (`.caja-banner`):
- [x] Abierta: verde, muestra hora apertura y monto base
- [x] Cerrada: rojo, invita a abrir caja
- [x] Botón "Ver caja" inline (btn-sm btn-success)

Stats grid — 4 tarjetas (`.stats-grid`, `repeat(4, 1fr)`):
- [x] **Ventas hoy** — monto total del día; `.stat-delta.up/down` comparado con ayer
- [x] **Transacciones** — número de ventas; `.stat-delta` vs ayer
- [x] **Efectivo en caja** — monto actual en caja; `text-muted` "Actualizado hace X min"
- [x] **Por cobrar (fiado)** — total deuda; `stat-value.danger`; N clientes
- [x] `.stat-label`: 11px, uppercase, letter-spacing 0.5px, text-muted
- [x] `.stat-value`: 22px, font-weight 700, .tnum, letter-spacing -0.5px
- [x] `.stat-delta`: 11px, font-weight 600; `.up` (success) / `.down` (danger) con ícono chevron

Ventas recientes (`.table-wrap`):
- [x] Headers: Hora | Cliente | Items | Total | Método | Estado
- [x] Hora: `12px`, `text-muted`
- [x] Total: clase `.amount.right` (font-weight 600, text-primary, .tnum)
- [x] Método: `.badge` con color según tipo (efectivo/transfer/tarjeta/crédito)
- [x] Estado: `.badge-ok` (success)
- [x] Botón "Ver todas →" (btn-sm btn-ghost) en section-title

Acciones rápidas (`.quick-actions`):
- [x] "Abrir POS" → navega a POS
- [x] "Ver Cartera" → navega a Clientes
- [x] "Ver Caja" → navega a Caja
- [ ] "Cerrar Caja" (btn-danger) → acción de cierre (omitido — el cierre se hace desde /register)

**DoD:**
- [x] Dashboard muestra información correcta del día actual
- [x] Datos de ventas recientes se cargan al entrar al dashboard
- [x] Banner de estado de caja refleja estado real
- [x] Stats actualizadas correctamente
- [x] Navegación desde acciones rápidas funciona

---

### 8.2 Pantalla de Configuración ✅ (2026-03-25)

**Frontend — Pantalla Configuración (`/settings`) — Fuente de verdad: `#screen-configuracion` en prototipo:**

Topbar:
- [x] Título: "Configuración"

Secciones tipo config-row (`.config-row`):
- [x] Cada fila: `padding: 12px 16px`, `background: surface-1`, `border: 1px solid border`, `border-radius: 8px`
- [x] Left: label 13px font-weight 500 + sublabel 11px text-muted
- [x] Right: control (input, toggle, botón)

Toggle (`.toggle`): usado en sidebar (theme), CSS definido en globals.css
- [x] 36×20px, `border-radius: 20px`, background: `surface-3` (off) / `accent` (on)
- [x] Transición: 150ms ease

Configuraciones implementadas:
- [x] Nombre del negocio + dirección + NIT
- [x] Logo del negocio (upload, base64)
- [x] Formato de ticket (header/footer)
- [x] **Monto base por defecto para apertura de caja** — pre-llena al abrir caja
- [ ] Tema: claro / oscuro en settings (existe en sidebar, no duplicado en settings)
- [ ] Configuración de impresora (out of scope MVP — impresora se configura a nivel OS)
- [ ] Link a gestión de usuarios desde settings

**DoD:**
- [x] Cambios en configuración se persisten correctamente (`PUT /setup`)
- [x] Logo se guarda y carga correctamente
- [x] Monto base pre-llena el campo al abrir caja
- [ ] Logo se actualiza en el sidebar dinámicamente al cambiarlo (requiere evento/context)

---

### 8.3 UX General — Polish Global

- [x] Formato COP en toda la aplicación — centralizado en `lib/format.ts`, eliminados duplicados en 9 archivos
- [x] Mensajes de confirmación para acciones destructivas — `components/ConfirmDialog.tsx`, reemplazó `window.confirm()`
- [x] Sistema de toasts funcional — ya existía (`components/Toast.tsx`), toastIn/toastOut en globals.css
- [x] Loading states: `components/LoadingSpinner.tsx` + `SkeletonRow` + `SkeletonCard` (animación skeleton-pulse)
- [x] Empty states: `components/EmptyState.tsx` aplicado en customers y portfolio
- [x] Manejo de errores con mensajes claros en español — toast tipo error en todas las páginas existentes
- [x] Scrollbars custom: 6px, track transparent, thumb border-emphasis, border-radius 6px (ya estaban en globals.css)
- [x] Scrollbars thin: clase `.scrollbar-thin` (3px) agregada a globals.css
- [x] Animaciones: `spin`, `skeleton-pulse`, `fadeIn`, `modalIn` agregadas a globals.css
- [x] Responsive: media query `@media (max-width: 1279px)` agregada a globals.css

**DoD:**
- [x] Dashboard muestra información correcta del día actual
- [x] Navegación funciona correctamente según rol
- [x] Aplicación es usable en resolución 1366×768
- [x] Setup completo (instalar + configurar) en < 5 minutos
- [x] Sin errores de consola en ninguna pantalla
- [x] Transiciones de pantalla: 150ms ease-out, sin flash

---

## Fase 9 — Distribución y Entrega

### 9.1 Build y Empaquetado ✅

- [x] Script de build: `make build` — compila frontend + copia estáticos + compila Go
- [x] Binario final: `pos.exe` (~18 MB, cumple < 30 MB)
- [x] Frontend embebido en binario Go (`embed`)
- [x] Al ejecutar, abre automáticamente el navegador en `localhost:8080`
- [x] Icono del ejecutable (`.ico`) — `goversioninfo` + `versioninfo.json` en `cmd/pos/`
- [x] Cross-compile para Windows desde cualquier OS (`GOOS=windows GOARCH=amd64`) — `make build-windows`

### 9.2 Testing 🟡

**Implementado — 46 tests unitarios (service layer), todos pasando:**

- [x] Tests unitarios `CategoryService` (7 tests) — Create, Update, Delete, GetByID
- [x] Tests unitarios `ProductService` (10 tests) — validación, Create, Update
- [x] Tests unitarios `CustomerService` / `CustomerPayments` (10 tests) — pagos, validaciones
- [x] Tests unitarios `SaleService` (18 tests) — flujos de venta, crédito, caja

**Pendiente:**
- [ ] Tests de integración para handlers (Go, con SQLite en memoria)
- [ ] Tests de repositorio (Go, operaciones CRUD)
- [ ] Tests de componentes frontend (React Testing Library)
- [ ] Test E2E del flujo principal: login → POS → venta → ticket

**Cobertura mínima requerida — Service Layer (Go):**

| Módulo | Caso | Test existente | Estado |
|--------|------|----------------|--------|
| `SaleService` | Venta efectivo, caja abierta | `TestSaleService_Create_AssignsOpenRegisterID` | ✅ |
| `SaleService` | Venta crédito con cliente | `TestSaleService_Create_CreditWithCustomer_Succeeds` | ✅ |
| `SaleService` | Venta crédito sin cliente | `TestSaleService_Create_CreditWithoutCustomer_ReturnsError` | ✅ |
| `SaleService` | Venta sin caja abierta | `TestSaleService_Create_NoOpenRegister_ReturnsError` | ✅ |
| `SaleService` | Cantidad > stock disponible | — | 🔲 |
| `CustomerPayments` | Abono parcial a deuda | `TestCreatePayment_OK` | ✅ |
| `CustomerPayments` | Abono mayor al saldo | `TestCreatePayment_ExceedsBalance` | ✅ |
| `ProductService` | Crear producto con precio negativo | `TestValidateProduct_NegativeSalePrice` | ✅ |
| `CategoryService` | Eliminar categoría con productos asignados | — | 🔲 |
| `RegisterService` | Abrir segunda caja con una ya abierta | — | 🔲 |

**DoD:**
- [x] Binario se genera sin errores (`make build`)
- [x] `pos.exe` < 30 MB
- [x] Binario auto-abre navegador al ejecutar
- [ ] Tests de los 3 casos pendientes de la tabla de cobertura
- [ ] Tests de integración de handlers
- [ ] Flujo E2E completo funciona

### 9.3 Documentación Mínima

- [x] README con instrucciones de uso (para el cliente) → `MANUAL-USUARIO.md`
- [x] Instrucciones de backup (copiar data.db) → incluido en `MANUAL-USUARIO.md`
- [x] Lista de impresoras compatibles → incluido en `MANUAL-USUARIO.md`

---

## Resumen de Criterios de Aceptación Globales

| Criterio | Objetivo |
|----------|----------|
| Venta completa (buscar → carrito → pago → ticket) | < 30 segundos |
| Setup inicial (instalar + configurar) | < 5 minutos |
| Arranque de aplicación | < 3 segundos |
| Cierre de caja | < 2 minutos |
| Consulta de cartera | < 5 segundos, una pantalla |
| Tamaño del binario | < 30 MB |
| Dependencias externas | 0 (solo un navegador) |
| Conexión a internet requerida | No |
| Resolución mínima soportada | 1366×768 |
| Temas | Claro y Oscuro, persistidos en localStorage |
| Fuente | Inter (Google Fonts) |
| Iconografía | Lucide Icons (SVG inline) |
| Formato de moneda | COP, sin decimales, separador de miles con punto |

---

## Guía de Pantallas — Referencia Visual Rápida

| Pantalla | Ruta | ID Prototipo | Rol |
|----------|------|--------------|-----|
| Login | `/login` | — | Todos |
| Setup inicial | `/setup` | — | Admin (primer uso) |
| Punto de Venta | `/pos` | `#screen-pos` | Cajero + Admin |
| Dashboard | `/` | `#screen-dashboard` | Admin |
| Caja | `/register` | `#screen-caja` | Cajero + Admin |
| Clientes / Cartera | `/customers` | `#screen-clientes` | Admin |
| Inventario | `/inventory` | `#screen-inventario` | Admin |
| Reportes | `/reports` | `#screen-reportes` | Admin |
| Configuración | `/settings` | `#screen-configuracion` | Admin |
| Usuarios | `/users` | — | Admin |

---

## Orden de Implementación Recomendado

```
Fase 0 (Fundación)     ████████████████████  ✅ 100%
Fase 1 (Core POS)      ████████████████████  ✅ 100%
Fase 2 (Caja)          ████████████████████  ✅ 100%
Fase 3 (Crédito)       ████████████████████  ✅ 100%
Fase 4 (Inventario)    ████████████████████  ✅ 100%
Fase 5 (Reportes)      ████████████████████  ✅ 100%
Fase 6 (Tickets)       ████████████████████  ✅ 100%
Fase 7 (Usuarios)      ████████████████████  ✅ 100%
Fase 8.1 (Dashboard)   ████████████████████  ✅ 100%
Fase 8.2 (Settings)    ████████████████░░░░  ~85% (falta logo live-update)
Fase 8.3 (Polish)      ████████████████████  ✅ 100% (2026-03-25)
Fase 9.1 (Build)       ████████████████████  ✅ 100% (2026-03-25)
Fase 9.2 (Testing)     ████████████░░░░░░░░  ~60% (service tests ✓, faltan 3 casos + integración)
Fase 9.3 (Docs)        ░░░░░░░░░░░░░░░░░░░░  🔲 0%
```

> **Nota:** Cada fase puede comenzar solo cuando la anterior tiene sus DoD cumplidos. La Fase 8 (UI/UX) se desarrolla en paralelo con las otras fases pero se pule al final. El `pos-prototype.html` es la **fuente de verdad visual** — cualquier decisión de UI debe validarse ahí primero.

---

## Ralph Review — Auditoría de Integridad (2025-03-24)

> Revisión crítica del plan y código actual enfocada en manejo de dinero, integridad de datos y viabilidad para uso real. Esta sección documenta hallazgos, correcciones pendientes y decisiones diferidas.

### Decisiones de contexto MVP

| Pregunta | Decisión | Impacto |
|----------|----------|---------|
| ¿Cuántos cajeros operan simultáneamente? | **1 solo cajero para MVP**. El register es global (no por usuario) y esto es aceptable | El service `GetOpen` retorna la primera caja abierta sin filtrar por `user_id`. No cambiar para MVP |
| ¿Cómo se manejan devoluciones parciales? | **Decisión diferida**. Definir al implementar `VoidSale` — sentarse a diseñar si anular + recrear es suficiente o se necesita `ReturnItems` parcial | Afecta el diseño de la anulación. No empezar VoidSale sin resolver esto |
| ¿Cómo se aplica precio mayorista? | **Decisión diferida**. Agregar flag `is_wholesale` (o similar) al cliente para que el POS aplique `wholesale_price` automáticamente. Definir la UX al implementar | El campo `wholesale_price` existe en productos pero no hay mecanismo de activación. Resolver cuando se trabaje la funcionalidad de clientes en POS |

### 🔴 Bugs y correcciones bloqueantes

Estos ítems deben corregirse **antes de que el sistema maneje dinero real**.

#### ~~B1. Abonos de crédito sin implementar (`customer_payments` sin endpoints)~~ ✅ RESUELTO (2026-03-27)

**Verificado en código:** `handler/customer.go` implementa `listCustomerPayments`, `createCustomerPayment`, `getCustomerBalance`, y `getCustomerDetail`. El service y repository correspondientes también están implementados con validaciones `amount > 0` y `amount <= balance`. El endpoint `POST /api/customers/:id/payments` existe y funciona.

#### B2. `cash_in` no incluido en cálculo de caja esperada

**Ubicación:** `internal/service/register.go:99`
**Problema:** `expected = opening + cashSales - cashOut`. Falta `+ cashIn`. Si se registra un movimiento de tipo `cash_in` (e.g., Don Carlos mete cambio), al cerrar caja el sistema muestra un sobrante ficticio.
**Corrección:**
1. Agregar `SumCashIn(ctx, registerID) (float64, error)` al `RegisterRepository`
2. Query: `SELECT COALESCE(SUM(amount), 0) FROM register_movements WHERE register_id = ? AND type = 'cash_in'`
3. Actualizar fórmula en `Summary()` y `Close()`: `expected = opening + cashSales + cashIn - cashOut`

#### B3. IVA/Tax no se recalcula en el backend — **CONFIRMADO BLOQUEANTE** (IVA obligatorio)

**Ubicación:** `internal/service/sale.go:52` + `internal/repository/sale.go:65-69`
**Contexto:** Confirmado (2026-03-27) que los negocios target **sí usan IVA**. Este bug es bloqueante para producción.
**Problema (dos partes):**
1. `sale.Tax` viene del frontend sin recálculo en el backend — si el frontend envía `tax: 0` en un producto con IVA 19%, la venta se registra sin impuesto y la contabilidad queda incorrecta.
2. `sale_items.tax_rate` (columna agregada en migración 3) **nunca se escribe** — el INSERT en `repository/sale.go:65` no incluye la columna, por lo que siempre queda en `DEFAULT 0`. La data histórica de IVA por ítem es inútil para auditoría contable y futura integración DIAN.

**Corrección:**
1. En `saleRepo.Create`, al consultar el producto para el stock check, leer también `tax_rate`
2. Guardar `sale_items.tax_rate = product.tax_rate` — incluirlo en el INSERT de `sale_items`
3. En `SaleService.Create`, calcular `item_tax = item.Subtotal * product.tax_rate` por ítem
4. `sale.Tax = SUM(item_tax)` — recalculado server-side, no aceptado del frontend
5. Recalcular `sale.Total = subtotal - discount + tax`

**Test cases requeridos:**
- Venta con producto IVA 19% ($100.000) → `sale_items.tax_rate = 0.19`, `sale.tax = $19.000`, total = $119.000
- Frontend envía `tax: 0` en producto con IVA → backend recalcula y registra IVA real
- Producto tipo `service` con `tax_rate = 0` → `sale_items.tax_rate = 0`, sin tax

#### ~~B4. Anulación de ventas no implementada~~ ✅ RESUELTO (2026-03-27)

**Verificado en código:** `handler/sale.go:60` implementa `voidSale`. Llama a `h.sales.Void()` en el service, que delega a `repo.Void()`. Las columnas `voided_by` y `voided_at` se agregan en migración 2. El endpoint es `POST /api/sales/:id/void` (solo admin).

#### ~~B5. Selector de cliente falta en POS para ventas a crédito~~ ✅ RESUELTO (2026-03-27)

**Verificado:** `CustomerPicker` implementado y funcional en `frontend/app/pos/page.tsx`. Se activa cuando `paymentMethod === 'credit'`, valida antes de cobrar, envía `customer_id` al backend. Confirmado funcionando en testing de desarrollo.

#### ~~B6. Entrada de inventario sin implementar~~ ✅ RESUELTO (2026-03-27)

**Verificado en código:** `POST /api/stock-entries` existe en handler/products.go. Implementado en service y repository. `UPDATE products SET stock = stock + ?` + INSERT en `stock_entries` en transacción atómica.

### 🟡 Correcciones importantes (antes de producción)

#### I1. Backup no hace WAL checkpoint

**Ubicación:** `internal/database/migrate.go:94`
**Problema:** El comentario dice "checkpoints the WAL first" pero el código no ejecuta `PRAGMA wal_checkpoint(TRUNCATE)`. La copia de `data.db` puede ser inconsistente (datos en `data.db-wal` no se incluyen).
**Corrección:** Agregar `db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")` antes de copiar en `backupDB()`.

#### I2. Error de apertura de caja silenciada

**Ubicación:** `internal/service/register.go:56`
**Problema:** `_, _ = s.repo.AddMovement(...)` — si falla el insert del movimiento de apertura, la caja se abre pero el movimiento no queda registrado.
**Corrección:** Manejar el error. Si `AddMovement` falla, la apertura debe fallar también (o al mínimo logear el error).

#### I3. Hard delete en productos

**Ubicación:** `internal/repository/` — `DELETE FROM products WHERE id = ?`
**Problema:** Si el producto tiene `sale_items` referenciándolo, el FK constraint falla. Además, pierde el historial del producto.
**Corrección:** Cambiar a soft delete: `UPDATE products SET active = 0 WHERE id = ?`. Ya marcado como pendiente en el PLAN (1.5).

#### I4. `SaleFilter` no filtra por status

**Ubicación:** `internal/repository/sale.go:121`
**Problema:** `SELECT ... FROM sales WHERE 1=1` sin filtro de status. Cuando existan ventas anuladas, aparecerán en listados y reportes inflando totales.
**Corrección:** Agregar `AND status = 'completed'` por defecto en `SaleFilter`. Opcionalmente, parámetro `IncludeVoided bool` para vistas de auditoría.

#### I5. JWT secret hardcoded

**Ubicación:** `internal/middleware/middleware.go:22`
**Problema:** Default `"pos-secret-key-change-in-prod"`. Para un sistema local el riesgo es bajo, pero cualquier persona con el binario puede generar tokens.
**Corrección:** Generar un secret random al primer inicio y persistirlo en `business_config` o archivo local. Leer desde ahí en arranques posteriores.

### 🟢 Notas de arquitectura (correctas, documentar para referencia)

- **Balance on-the-fly**: `SUM(credit_sales) - SUM(customer_payments)` es la fuente de verdad. No existe campo `balance` en `customers`. Esto es correcto y se debe mantener.
- **Register global para MVP**: Con 1 cajero, `GetOpen` sin filtro de `user_id` es aceptable. Si en el futuro se soportan múltiples cajeros, agregar `user_id` al filtro y permitir múltiples cajas abiertas.
- **Stock check atómico**: `UPDATE ... WHERE stock >= ?` + verificar `rows affected` es correcto y está implementado.
- **Transacciones en ventas**: sale + items + stock update + register_movement en una TX es correcto.

### Decisiones diferidas (resolver en la fase correspondiente)

| Decisión | Fase | Contexto |
|----------|------|----------|
| ¿Anulación total o devolución parcial? | Al implementar VoidSale (B4) | Don Carlos puede necesitar devolver 3 de 10 productos. Definir si anular+recrear es suficiente o necesita `ReturnItems`. Sentarse a diseñar juntos |
| ¿Cómo se activa precio mayorista? | Al implementar selector de cliente en POS (B5) | Agregar flag/tipo al cliente (`is_wholesale` o `customer_type`). Cuando el cliente es mayorista y está seleccionado en POS, aplicar `wholesale_price` automáticamente. Diseñar la UX juntos |

### Test cases requeridos post-corrección

| Caso | Resultado esperado |
|------|--------------------|
| Registrar abono parcial $50.000 a cliente con deuda $200.000 | Balance = $150.000, abono en `customer_payments` |
| Registrar abono = saldo total | Balance = $0, botón "Abonar" desaparece |
| Registrar abono > saldo | Error de validación, no se inserta |
| Agregar `cash_in` $200.000 → cerrar caja | Expected incluye los $200.000, diferencia correcta |
| Venta con producto IVA 19% ($100.000) | `sale_items.tax_rate = 0.19`, `sale.tax = $19.000`, total = $119.000 |
| Frontend envía `tax: 0` en producto con IVA | Backend recalcula y registra el IVA real |
| Anular venta en efectivo | Stock restaurado, status `voided`, `voided_by` registrado |
| Anular venta a crédito | Balance del cliente se reduce (venta ya no cuenta) |
| Anular venta después de 24h | Error "fuera de ventana de anulación" |
| Cajero intenta anular | Error "solo admin puede anular" |
| Backup antes de migración | Copia completa incluyendo datos de WAL |
| Registrar entrada de inventario | Stock aumenta, entry registrado en `stock_entries` |

---

## Ralph Review — Auditoría de Implementación (2026-03-27)

> Segunda auditoría. Revisión completa de código real vs. plan. Foco en verificar el estado de los bugs anteriores y nuevos hallazgos.

### Resumen ejecutivo

Los bugs B1, B4, B5, B6 de la auditoría anterior están correctamente implementados. La arquitectura general (handler/service/repository, migraciones versionadas, transacciones atómicas en ventas) es sólida. El sistema **no está listo para manejar dinero real** hasta resolver B2 y B3.

El sistema sigue en fase de desarrollo — no hay cliente real todavía.

### Bugs activos confirmados (pendientes de corrección)

#### B2. `cash_in` no incluido en cálculo de caja esperada — 🔴 ACTIVO

**Ubicación:** `internal/service/register.go:99` y `:121`
```go
expected := reg.OpeningAmount + cashSales - cashOut  // falta + cashIn
```
**Impacto real:** Si Laura hace un `cash_in` (p.ej., Don Carlos le pasa cambio de $200.000 para el turno), al cerrar caja el sistema muestra un sobrante ficticio de $200.000. En el primer día de uso real van a pensar que el sistema está roto o que alguien tomó plata.

**Corrección requerida:**
1. Agregar `SumCashIn(ctx, registerID) (float64, error)` al `RegisterRepository`
2. Query: `SELECT COALESCE(SUM(amount), 0) FROM register_movements WHERE register_id = ? AND type = 'cash_in'`
3. Actualizar fórmula en `Summary()` y `Close()`: `expected = opening + cashSales + cashIn - cashOut`

#### B3. IVA no recalculado + `sale_items.tax_rate` nunca escrito — 🔴 ACTIVO (IVA obligatorio)

Ver detalle completo en sección anterior. Resumen ejecutivo:
- `service/sale.go:52` acepta `sale.Tax` del frontend sin validarlo
- `repository/sale.go:65` INSERT de `sale_items` no incluye `tax_rate` — la columna siempre queda en 0
- Con IVA obligatorio en el negocio target, esto es un bloqueante legal y contable

#### B4 (antes I4). Ventas anuladas aparecen en listados — 🟡 ACTIVO

**Ubicación:** `internal/handler/sale.go:46` + `internal/repository/sale.go`
```go
filter := repository.SaleFilter{}  // sin status filter
```
**Impacto:** El listado de ventas y potencialmente los reportes incluyen ventas `voided`. Cuando se empiecen a anular ventas, los totales del dashboard y reportes van a inflarse con montos de ventas canceladas.

**Corrección:** Agregar `status = 'completed'` como default en `SaleFilter`. Parámetro opcional `IncludeVoided bool` para vistas de auditoría.

### Hallazgos adicionales (🟡 antes de producción)

#### I1. Backup no hace WAL checkpoint — 🟡 ACTIVO

Documentado en auditoría anterior, sigue sin corregir. `backupDB()` en `database/migrate.go` no ejecuta `PRAGMA wal_checkpoint(TRUNCATE)` antes de copiar. El backup puede ser inconsistente.

#### I2. Error de apertura de caja silenciado — 🟡 ACTIVO

Documentado en auditoría anterior, sigue sin corregir. `internal/service/register.go:56`: `_, _ = s.repo.AddMovement(...)` — si falla el insert del movimiento de apertura, la caja queda abierta pero sin el movimiento registrado.

### Estado de pendientes por prioridad

| Bug | Descripción | Prioridad | Estado |
|-----|-------------|-----------|--------|
| B2 | `cash_in` en cálculo de caja esperada | 🔴 Blocker | ✅ RESUELTO (2026-03-27) |
| B3 | IVA no recalculado + `tax_rate` no escrito | 🔴 Blocker (IVA obligatorio) | ✅ RESUELTO (2026-03-27) |
| B4/I4 | Voided sales en listados y reportes | 🟡 Antes de producción | ✅ RESUELTO (2026-03-27) |
| I1 | WAL checkpoint antes de backup | 🟡 Antes de producción | ✅ RESUELTO (2026-03-27) |
| I2 | Error silenciado en apertura de caja | 🟡 Antes de producción | ✅ RESUELTO (2026-03-27) |
| I5 | JWT secret hardcoded | 🟡 Baja prioridad (sistema local) | Pendiente |
