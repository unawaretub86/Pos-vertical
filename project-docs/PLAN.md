# Plan de Desarrollo — POS Genérico MVP

## Convenciones del Checklist

- `[ ]` = Pendiente
- `[/]` = En progreso
- `[x]` = Completado
- **DoD** = Definition of Done (criterios para considerar el ítem terminado)

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
- [x] Configurar WAL mode + busy timeout para concurrencia
- [x] Seed de usuario admin por defecto (admin/admin)

**DoD:**
- [x] Al ejecutar el binario, se crea `data.db` si no existe
- [x] Todas las tablas se crean correctamente con FKs e índices
- [x] SQLite opera en WAL mode
- [x] Existe un usuario admin por defecto para primer acceso

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
- [x] Admin: Punto de Venta, Dashboard, Caja, Clientes, Inventario, Reportes, Configuración

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
- [ ] Aparecen como **category tabs** en el POS (`.cat-tab`):
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
- [ ] Las categorías aparecen correctamente como tabs en el POS

---

### 1.5 Productos

**Backend:**
- [x] `GET /api/products` — listar productos (con filtros: search, category_id, active)
- [x] `GET /api/products/:id` — detalle de producto
- [x] `POST /api/products` — crear producto
- [x] `PUT /api/products/:id` — actualizar producto
- [x] `DELETE /api/products/:id` — eliminar producto (hard delete; pendiente cambiar a soft delete)
- [x] `GET /api/products?search=q` — búsqueda integrada en listado (nombre o código, LIKE)
- [ ] `GET /api/products/low-stock` — endpoint dedicado para productos bajo stock mínimo
- [x] Campos: name, code, category_id, type (product/service), sale_price, wholesale_price, cost, unit (unit/kg/lb/liter), stock, min_stock, tax_rate, active

**Pendientes backend:**
- [ ] Cambiar DELETE a soft delete (`active = 0`) en lugar de hard delete
- [ ] Endpoint dedicado `GET /api/products/low-stock`

**Frontend — Pantalla Productos / Cards en POS:**

Cards de producto en el grid del POS (`.product-card`):
- [ ] Grid: `repeat(3, 1fr)`, gap `8px`, scroll vertical con scrollbar thin (4px)
- [ ] Card: `background: surface-1`, `border: 1px solid border`, `border-radius: 8px`, `padding: 12px`
- [ ] Hover: `border-color: border-emphasis`, `background: surface-2`, `transform: translateY(-1px)`
- [ ] Active (click): `transform: translateY(0)`, `opacity: 0.8`
- [ ] Nombre: `font-size: 13px`, `font-weight: 600`, `line-height: 1.3`
- [ ] Precio: `font-size: 17px`, `font-weight: 700`, `color: accent`, `.tnum`, `letter-spacing: -0.5px`
- [ ] Stock badge (`.stock-badge`): `font-size: 10px`, `font-weight: 600`, `padding: 2px 6px`, `border-radius: 4px`
  - Verde > 10: `.stock-ok` (success-dim/success)
  - Amarillo ≤ 10: `.stock-low` (accent-dim/accent)
  - Rojo ≤ 3: `.stock-out` (danger-dim/danger)

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
- [ ] Cards tienen hover effect y transform implementados (pendiente POS frontend)

---

### 1.6 Punto de Venta (POS)

**Backend:**
- [x] `POST /api/sales` — crear venta completa (header + items)
- [x] `GET /api/sales` — listar ventas (con filtros de fecha)
- [x] `GET /api/sales/:id` — detalle de venta

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
- [ ] Modificar precio en línea (solo admin) — pendiente
- [ ] Venta a crédito marca recibo como "CRÉDITO" — pendiente ticket

**DoD:**
- [x] Flujo completo (buscar → carrito → pago → ticket) en < 30 segundos
- [x] Stock se descuenta correctamente al completar venta
- [x] Servicios no descuentan stock
- [x] Venta a crédito requiere cliente seleccionado (validación con mensaje claro)
- [x] Cambio se calcula correctamente para pagos en efectivo
- [x] Descuentos se reflejan en el total en tiempo real
- [x] Montos en formato COP (sin decimales, separador de miles con punto)
- [x] El botón "Cobrar" muestra el monto exacto actualizado
- [x] Las category tabs filtran productos correctamente
- [x] Stock badges visible en cada card de producto
- [ ] Venta se asocia al registro de caja abierto — pendiente Fase 2

---

## Fase 2 — Prioridad 2: Control de Dinero

### 2.1 Caja Registradora

**Backend:**
- [x] `POST /api/registers` — abrir caja (monto base)
- [x] `GET /api/registers` — listar cajas (historial)
- [x] `GET /api/registers/:id` — detalle de caja
- [x] `POST /api/registers/:id/close` — cerrar caja con cuadre
- [x] `GET /api/registers/:id/movements` — movimientos de caja
- [x] `POST /api/registers/:id/movements` — entrada/salida manual de efectivo

**Frontend — Pantalla Caja (`/register`) — Fuente de verdad: `#screen-caja` en prototipo:**

Topbar:
- [ ] Título: "Caja" | Subtítulo: fecha + turno del usuario actual
- [ ] Acciones: "＋ Movimiento" + "Cerrar Caja" (btn-danger)

Banner de estado (`.caja-banner`):
- [ ] Abierta: `background: success-dim`, `border: 1px solid success`, `color: success`, ícono checkmark
- [ ] Cerrada: `background: danger-dim`, `border: 1px solid danger`, `color: danger`
- [ ] Texto: "Caja abierta · Inicio: X:XX AM · Turno de N horas"
- [ ] `border-radius: 8px`, `padding: 12px 16px`

Summary grid (`.caja-summary-grid`): `repeat(3, 1fr)`, gap `12px`
- [ ] Card "Base apertura" → monto en `stat-value`
- [ ] Card "Ventas efectivo" → suma ventas en efectivo
- [ ] Card "Total esperado" → apertura + ventas efectivo − salidas; valor en `color: accent`

Movimientos (`.section-card` + `.movement-item`):
- [ ] Cada movimiento: `padding: 10px 14px`, `border-bottom: 1px solid border`
- [ ] Punto indicador (`.movement-dot`): 8×8px, circle
  - Entrada (`.in`): `background: success`
  - Salida (`.out`): `background: danger`
- [ ] Descripción + timestamp en `text-muted 11px`
- [ ] Monto: `font-weight: 600`, `.tnum`
  - Entrada: `color: success`; Salida: `color: danger`
- [ ] Tipos de movimientos: apertura, ventas efectivo, retiros, entradas manuales
- [ ] Filtrable por sesión actual

Apertura:
- [ ] Formulario de monto base (input numérico COP)
- [ ] Solo una caja abierta por sesión
- [ ] Timestamp de apertura

Cierre:
- [ ] Input de monto real contado
- [ ] Cálculo automático: apertura + ventas efectivo − salidas = esperado
- [ ] Mostrar diferencia (sobrante/faltante) — verde si sobrante, rojo si faltante
- [ ] Botón cierre: full width, btn-danger grande, muestra monto esperado
- [ ] Resumen: total ventas por método, total transacciones

**DoD:**
- [ ] No se puede vender sin caja abierta
- [ ] No se puede abrir una segunda caja si hay una abierta
- [ ] El cuadre calcula correctamente el esperado
- [ ] La diferencia se registra y se muestra claramente
- [ ] Cierre de caja genera resumen completo
- [ ] Cierre de caja en < 2 minutos
- [ ] Puntos de color correcto (verde/rojo) en movimientos
- [ ] Banner de estado refleja correctamente si caja está abierta o cerrada

---

## Fase 3 — Prioridad 3: Clientes y Crédito (Diferenciador)

### 3.1 Clientes

**Backend:**
- [ ] `GET /api/customers` — listar clientes (búsqueda por nombre/teléfono)
- [ ] `POST /api/customers` — crear cliente
- [ ] `PUT /api/customers/:id` — actualizar cliente
- [ ] `DELETE /api/customers/:id` — eliminar cliente (solo si saldo = 0)
- [ ] `GET /api/customers/:id/balance` — saldo deudor del cliente

**Frontend — Pantalla Clientes/Cartera (`/customers`) — Fuente de verdad: `#screen-clientes` en prototipo:**

Topbar:
- [ ] Título: "Clientes · Cartera" | Subtítulo: "$X.XXX.XXX por cobrar en total"
- [ ] Acciones: "Exportar" (btn) + "＋ Nuevo cliente" (btn-accent)

Toolbar:
- [ ] Input búsqueda con ícono lupa (flex: 1)
- [ ] Botón "Ordenar por deuda" con ícono flecha
- [ ] Botón "Todos los estados" (filtro)

Tabla de clientes (`.table-wrap`):
- [ ] Headers: Cliente | Teléfono | Deuda más antigua | Saldo pendiente | Acciones
- [ ] Header de cliente contiene:
  - Nombre: `font-weight: 600`, `color: text-primary`
  - Dirección/info: `font-size: 11px`, `color: text-muted`
  - Barra de deuda (`.debt-bar`): full width, `height: 4px`, `background: surface-3`, `border-radius: 4px`
    - Fill (`.debt-bar-fill`): `background: danger`, `border-radius: 4px`, ancho proporcional al monto max
    - Transición: `0.3s ease`
- [ ] Antigüedad: `font-size: 12px`, `color: text-muted`
- [ ] Saldo: `right`, clase `.danger-amount` (`font-weight: 600`, `color: danger`, `.tnum`)
- [ ] Acciones por fila: "Detalle" (btn-sm) + "Abonar" (btn-sm btn-accent)
- [ ] Fila clickable (`.customer-row`): cursor pointer

**DoD:**
- [ ] CRUD funcional desde la UI
- [ ] Búsqueda por nombre y teléfono funciona
- [ ] No se puede eliminar cliente con deuda pendiente
- [ ] Balance del cliente se muestra en la lista
- [ ] Barra de deuda refleja proporción correcta
- [ ] Saldo total se muestra en el subtítulo del topbar

---

### 3.2 Pagos de Crédito (Abonos)

**Backend:**
- [ ] `GET /api/customers/:id/payments` — historial de pagos del cliente
- [ ] `POST /api/customers/:id/payments` — registrar abono

**Frontend — Detalle de Cliente (`/customers/:id`):**
- [ ] Lista de ventas a crédito del cliente
- [ ] Lista de abonos realizados
- [ ] Formulario para registrar abono (parcial o total)
- [ ] Input monto abono con validación (no > saldo)
- [ ] Generación de recibo de abono
- [ ] Toast "Abono registrado" al guardar

**DoD:**
- [ ] Abono reduce el saldo del cliente correctamente
- [ ] No se puede abonar más de lo que se debe
- [ ] Se genera recibo por cada abono
- [ ] Historial de abonos visible en detalle del cliente

---

### 3.3 Cartera de Crédito

**Backend:**
- [ ] `GET /api/reports/portfolio` — resumen de cartera

**Frontend — Vista Cartera (`/portfolio`):**
- [ ] Lista: nombre cliente, deuda total, fecha venta más antigua
- [ ] Ordenable por monto o antigüedad
- [ ] Drill-down a detalle del cliente
- [ ] Botón "Abonar" directo por fila

**DoD:**
- [ ] Vista de cartera carga en < 5 segundos
- [ ] Muestra todos los clientes con deuda > 0
- [ ] Se puede ordenar por deuda y por antigüedad
- [ ] Click en cliente lleva a su detalle con ventas y abonos
- [ ] Todo visible en una sola pantalla

---

## Fase 4 — Prioridad 4: Inventario

### 4.1 Control de Stock

**Backend:**
- [ ] `GET /api/products/low-stock` — productos bajo stock mínimo
- [ ] `POST /api/stock-entries` — entrada de inventario

**Frontend — Pantalla Inventario (`/inventory`) — Fuente de verdad: `#screen-inventario` en prototipo:**

Topbar:
- [ ] Título: "Inventario" | Subtítulo: resumen (ej. "3 productos bajo mínimo")
- [ ] Acciones: "＋ Entrada de inventario" (btn-accent)

Stats grid superior (`.inv-stats`): `repeat(3, 1fr)`, gap `12px`
- [ ] Card "Total productos activos"
- [ ] Card "Bajo stock mínimo" — valor en `color: danger`
- [ ] Card "Sin stock" — valor en `color: danger`

Tabla de productos:
- [ ] Columnas: Producto | Categoría | Stock actual | Stock mínimo | Estado | Acciones
- [ ] Fila resaltada con background `danger-dim` para productos bajo mínimo
- [ ] Badge de estado: verde (ok), amarillo (bajo), rojo (sin stock) — misma lógica que `.stock-badge`
- [ ] Acción: "＋ Entrada" por fila

Entrada de inventario (modal o subpantalla):
- [ ] Seleccionar producto (búsqueda)
- [ ] Input cantidad (numérico)
- [ ] Nota opcional
- [ ] Timestamp automático

**DoD:**
- [ ] Stock se actualiza inmediatamente al registrar entrada
- [ ] Alerta visual clara para productos bajo stock mínimo (fila resaltada + badge)
- [ ] Cada entrada queda registrada con timestamp y nota
- [ ] Stock se descuenta automáticamente con ventas (validado en Fase 1)
- [ ] Stats del topbar/summary muestran conteos correctos

---

## Fase 5 — Prioridad 5: Reportes

### 5.1 Reportes de Ventas

**Backend:**
- [ ] `GET /api/reports/sales` — resumen de ventas (params: from, to, group_by)
- [ ] `GET /api/reports/top-products` — productos más vendidos
- [ ] `GET /api/reports/registers` — historial de cajas

**Frontend — Pantalla Reportes (`/reports`) — Fuente de verdad: `#screen-reportes` en prototipo:**

Topbar:
- [ ] Título: "Reportes" | Subtítulo: rango de fechas activo
- [ ] Acciones: selector de rango (Día / Semana / Mes / Personalizado) + "Exportar"

Layout en grid `1fr 1fr`, gap `16px`:

Card "Ventas del período" (`.report-card`):
- [ ] Chart placeholder bar chart: `height: 120px`, `background: surface-2`, `border-radius: 6px`
  - Barras: `background: accent-dim`, `border-top: 2px solid accent`, `opacity: 0.7`
  - Barra actual/hoy: `opacity: 1`
- [ ] Total ingresos, total costo, margen bruto
- [ ] Desglose por método de pago

Card "Top Productos":
- [ ] Ranking por cantidad vendida o por ingreso
- [ ] Filtrable por rango de fecha
- [ ] Lista con posición, nombre, cantidad, ingreso

Card "Cartera":
- [ ] Resumen de deuda total
- [ ] Acceso a vista completa de cartera

Card "Historial de Cajas":
- [ ] Lista: fecha apertura/cierre, monto apertura, esperado, real, diferencia
- [ ] Diferencia en verde (sobrante) o rojo (faltante)
- [ ] Opción de imprimir

**DoD:**
- [ ] Reportes de ventas agregan correctamente por período
- [ ] Margen bruto = ingresos - costos (calculado correctamente)
- [ ] Top productos ordena correctamente
- [ ] Datos son consistentes con las ventas registradas
- [ ] Chart bars proporcionales a los datos reales

---

## Fase 6 — Prioridad 6: Tickets

### 6.1 Impresión de Tickets

**Backend:**
- [ ] `GET /api/sales/:id/ticket` — generar datos de ticket
- [ ] Soporte ESC/POS para impresoras térmicas (58mm/80mm)
- [ ] Contenido del ticket:
  - Nombre/logo del negocio
  - Fecha y hora
  - Items con cantidades y precios
  - Subtotal, descuentos, impuestos, total
  - Método de pago
  - **"CRÉDITO"** si es venta fiada (en grande, destacado)
  - Header/footer configurables

**Frontend:**
- [ ] Vista previa de ticket en pantalla (fallback si no hay impresora)
- [ ] Montos en formato COP sin decimales
- [ ] Configuración de formato en `/settings`
- [ ] Botón imprimir desde detalle de venta

**DoD:**
- [ ] Ticket se genera correctamente con todos los campos
- [ ] Impresión en impresora térmica funciona (Epson TM y genéricas 58mm)
- [ ] "CRÉDITO" aparece claramente en ventas fiadas
- [ ] Vista previa funciona como fallback
- [ ] Header y footer son configurables

---

## Fase 7 — Prioridad 7: Usuarios

### 7.1 Gestión de Usuarios

**Backend:**
- [ ] `GET /api/users` — listar usuarios (admin only)
- [ ] `POST /api/users` — crear usuario
- [ ] `PUT /api/users/:id` — actualizar usuario
- [ ] `DELETE /api/users/:id` — desactivar usuario

**Frontend — Pantalla Usuarios (`/users`):**
- [ ] Tabla de usuarios: username, rol, estado (activo/inactivo), fecha creación
- [ ] Formulario: username, password, rol (admin/cajero)
- [ ] Roles: admin (acceso total), cajero (POS + su caja + sus ventas)
- [ ] Badge de rol: misma lógica que sidebar (`.role-badge.admin` / `.role-badge.cashier`)
- [ ] Solo admin puede acceder a esta pantalla

**DoD:**
- [ ] Solo admin puede gestionar usuarios
- [ ] Cajero solo ve: POS, Caja
- [ ] Cajero no puede acceder a: productos, reportes, usuarios, settings
- [ ] Password se almacena hasheado (bcrypt)
- [ ] No se puede eliminar el último admin

---

## Fase 8 — UI/UX, Dashboard y Polish

### 8.1 Dashboard

**Frontend — Pantalla Dashboard (`/`) — Fuente de verdad: `#screen-dashboard` en prototipo:**

Topbar:
- [ ] Título dinámico: "Buenos días/tardes/noches, [nombre] ☀/🌙"
- [ ] Subtítulo: fecha actual formateada en español
- [ ] Acción: "Abrir POS" (btn-accent)

Banner estado de caja (`.caja-banner`):
- [ ] Abierta: verde, muestra hora apertura y monto base
- [ ] Cerrada: rojo, invita a abrir caja
- [ ] Botón "Ver caja" inline (btn-sm btn-success)

Stats grid — 4 tarjetas (`.stats-grid`, `repeat(4, 1fr)`):
- [ ] **Ventas hoy** — monto total del día; `.stat-delta.up/down` comparado con ayer
- [ ] **Transacciones** — número de ventas; `.stat-delta` vs ayer
- [ ] **Efectivo en caja** — monto actual en caja; `text-muted` "Actualizado hace X min"
- [ ] **Por cobrar (fiado)** — total deuda; `stat-value.danger`; N clientes
- [ ] `.stat-label`: 11px, uppercase, letter-spacing 0.5px, text-muted
- [ ] `.stat-value`: 22px, font-weight 700, .tnum, letter-spacing -0.5px
- [ ] `.stat-delta`: 11px, font-weight 600; `.up` (success) / `.down` (danger) con ícono chevron

Ventas recientes (`.table-wrap`):
- [ ] Headers: Hora | Cliente | Items | Total | Método | Estado
- [ ] Hora: `12px`, `text-muted`
- [ ] Total: clase `.amount.right` (font-weight 600, text-primary, .tnum)
- [ ] Método: `.badge` con color según tipo (efectivo/transfer/tarjeta/crédito)
- [ ] Estado: `.badge-ok` (success)
- [ ] Botón "Ver todas →" (btn-sm btn-ghost) en section-title

Acciones rápidas (`.quick-actions`):
- [ ] "Abrir POS" → navega a POS
- [ ] "Ver Cartera" → navega a Clientes
- [ ] "Ver Caja" → navega a Caja
- [ ] "Cerrar Caja" (btn-danger) → acción de cierre

**DoD:**
- [ ] Dashboard muestra información correcta del día actual
- [ ] Datos de ventas recientes se cargan al entrar al dashboard
- [ ] Banner de estado de caja refleja estado real
- [ ] Stats actualizadas correctamente
- [ ] Navegación desde acciones rápidas funciona

---

### 8.2 Pantalla de Configuración

**Frontend — Pantalla Configuración (`/settings`) — Fuente de verdad: `#screen-configuracion` en prototipo:**

Topbar:
- [ ] Título: "Configuración"

Secciones tipo config-row (`.config-row`):
- [ ] Cada fila: `padding: 12px 16px`, `background: surface-1`, `border: 1px solid border`, `border-radius: 8px`
- [ ] Left: label 13px font-weight 500 + sublabel 11px text-muted
- [ ] Right: control (input, toggle, botón)

Toggle (`.toggle`):
- [ ] 36×20px, `border-radius: 20px`, background: `surface-3` (off) / `accent` (on)
- [ ] Círculo interior: 14×14px blanco, `top: 2px left: 2px` → `left: 18px` (on)
- [ ] Transición: 150ms ease

Configuraciones a implementar:
- [ ] Nombre del negocio + dirección + NIT
- [ ] Logo del negocio (upload)
- [ ] Tema: claro / oscuro (sincronizado con el toggle del sidebar)
- [ ] Formato de ticket (header/footer)
- [ ] Configuración de impresora
- [ ] Formato de moneda (COP por defecto)
- [ ] Gestión de usuarios (link a `/users`)

**DoD:**
- [ ] Cambios en configuración se persisten correctamente
- [ ] Toggle de tema en settings sincroniza con el del sidebar
- [ ] Logo se actualiza en el sidebar al cambiarlo
- [ ] Configuración de ticket se refleja en la vista previa

---

### 8.3 UX General — Polish Global

- [ ] Formato COP en toda la aplicación ($1.500, sin decimales)
- [ ] Mensajes de confirmación para acciones destructivas (modal o confirm dialog)
- [ ] Sistema de toasts funcional:
  - Esquina inferior derecha
  - Auto-dismiss 3 segundos
  - Tipos: success (`.toast-dot.success`), error (danger color), info (`.toast-dot.info`)
  - Máximo 3 simultáneos
  - Animación: `toastIn 200ms ease-out` + `toastOut 200ms` a los 2.5s
- [ ] Loading states: skeleton/spinner en todas las llamadas API
- [ ] Empty states: ícono Lucide + texto descriptivo en tablas y listas vacías
- [ ] Manejo de errores con mensajes claros en español (`color: danger`)
- [ ] Scrollbars custom: `6px wide`, `track transparent`, `thumb border-emphasis`, `border-radius 6px`
- [ ] Scrollbars thin en paneles internos: `3–4px wide`
- [ ] Responsive: funcional en resolución 1366×768 (laptop típico) y >= 1280px

**DoD:**
- [ ] Dashboard muestra información correcta del día actual
- [ ] Navegación funciona correctamente según rol
- [ ] Aplicación es usable en resolución 1366×768
- [ ] Setup completo (instalar + configurar) en < 5 minutos
- [ ] Arranque de la aplicación en < 3 segundos
- [ ] Sin errores de consola en ninguna pantalla
- [ ] Transiciones de pantalla: 150ms ease-out, sin flash

---

## Fase 9 — Distribución y Entrega

### 9.1 Build y Empaquetado

- [ ] Script de build: `make build` o equivalente
  - Compila frontend (`next build && next export`)
  - Embebe estáticos en binario Go (`embed`)
  - Compila binario para Windows amd64
- [ ] Binario final: `pos.exe` (< 30 MB)
- [ ] Al ejecutar, abre automáticamente el navegador en `localhost:8080`
- [ ] Icono del ejecutable

### 9.2 Testing

- [ ] Tests unitarios para capa de servicio (Go)
- [ ] Tests de integración para handlers (Go, con SQLite en memoria)
- [ ] Tests de repositorio (Go, operaciones CRUD)
- [ ] Tests de componentes frontend (React Testing Library)
- [ ] Test E2E del flujo principal: login → POS → venta → ticket

**DoD:**
- [ ] Binario se genera sin errores en un solo comando
- [ ] Binario funciona en Windows 10/11 sin dependencias
- [ ] `pos.exe` < 30 MB
- [ ] Tests pasan al 100%
- [ ] Flujo E2E completo funciona

### 9.3 Documentación Mínima

- [ ] README con instrucciones de uso (para el cliente)
- [ ] Instrucciones de backup (copiar data.db)
- [ ] Lista de impresoras compatibles

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
Fase 0 (Fundación)     ████████░░░░░░░░░░░░  ~15%
Fase 1 (Core POS)      ████████████░░░░░░░░  ~30%
Fase 2 (Caja)          ██████░░░░░░░░░░░░░░  ~10%
Fase 3 (Crédito)       ██████░░░░░░░░░░░░░░  ~10%
Fase 4 (Inventario)    ████░░░░░░░░░░░░░░░░  ~5%
Fase 5 (Reportes)      ██████░░░░░░░░░░░░░░  ~10%
Fase 6 (Tickets)       ████░░░░░░░░░░░░░░░░  ~5%
Fase 7 (Usuarios)      ████░░░░░░░░░░░░░░░░  ~5%
Fase 8 (UI/UX)         ██████░░░░░░░░░░░░░░  ~5%
Fase 9 (Distribución)  ██████░░░░░░░░░░░░░░  ~5%
```

> **Nota:** Cada fase puede comenzar solo cuando la anterior tiene sus DoD cumplidos. La Fase 8 (UI/UX) se desarrolla en paralelo con las otras fases pero se pule al final. El `pos-prototype.html` es la **fuente de verdad visual** — cualquier decisión de UI debe validarse ahí primero.
