# SPRINT-05 — Mobile Responsive: Demo-Ready en Celular

**Objetivo:** El sistema se puede demostrar y usar en un celular (320px–430px). El bloqueador actual es el sidebar fijo de 220px que aplasta el contenido en pantallas pequeñas. Al final de este sprint, un prospecto puede ver el POS en su propio celular.

**Fecha inicio:** —

**Contexto de uso mobile:**
- Demo a prospectos en celular del vendedor o del cliente
- Cajero en tablet/celular pequeño haciendo ventas
- Admin revisando dashboard o cartera desde el celular

---

## Problemas identificados (iPhone 14 Pro Max — 430px)

1. **Sidebar fijo 220px** — ocupa ~50% del ancho, el contenido queda inutilizable
2. **Textos cortados** — "TRANSACCI...", "Todos los est..." en dashboard
3. **No hay menú hamburger** — no hay forma de navegar en mobile
4. **AppLayout no tiene breakpoints** — `display: flex` siempre lateral, nunca apilado

---

## Prioridad 1 — Navegación mobile

### US-022: Sidebar colapsable con drawer en mobile

- [x] Convertir el sidebar en un drawer deslizante en pantallas < 768px
- **Acceptance Criteria:**
  - En pantallas ≥ 768px: sidebar fijo lateral de 220px (comportamiento actual, sin cambios)
  - En pantallas < 768px: sidebar oculto por defecto, se abre con botón hamburger (☰)
  - El botón hamburger aparece en la esquina superior izquierda en mobile, visible en todas las páginas
  - Al abrir el drawer, aparece un overlay semitransparente detrás; tocarlo cierra el drawer
  - El drawer tiene el mismo contenido que el sidebar actual (nav items, usuario, logout, theme toggle)
  - Al navegar a una página, el drawer se cierra automáticamente
  - Animación de entrada/salida suave (transform translateX), sin librerías externas
  - La detección de breakpoint se hace con `window.matchMedia` o `useMediaQuery` hook, no con CSS puro (para evitar flash de layout)
- **Archivos a modificar:**
  - `frontend/components/Sidebar.tsx` — agregar estado open/closed y lógica de overlay
  - `frontend/components/AppLayout.tsx` — pasar estado del drawer si es necesario, ajustar layout en mobile
- **No romper:** comportamiento desktop actual (220px fijo, sin cambios visibles en ≥ 768px)

---

## Prioridad 2 — Dashboard mobile

### US-023: Dashboard legible en 320px–430px

- [x] Corregir textos cortados y layout del dashboard en mobile
- **Problemas específicos (vistos en screenshot):**
  - Card "TRANSACCI..." — título cortado, necesita truncar con ellipsis o abreviar
  - "Ventas recientes — Todos los est..." — botón/texto cortado
  - Cards de stats apiladas verticalmente en mobile (hoy ocupan row fijo que se desborda)
  - Topbar con demasiada info apilada en 430px
- **Acceptance Criteria:**
  - En mobile (< 768px): las cards de stats (`Ventas hoy`, `Transacciones`) se apilan en columna o en grid 2×2 sin desbordarse
  - Ningún texto se corta con `...` excepto nombres de clientes/productos (comportamiento esperado)
  - El botón "Todos los..." en ventas recientes tiene texto completo o se abrevia intencionalmente ("Ver todos")
  - La Topbar en mobile muestra solo lo esencial (saludo + botón POS), sin desbordarse
  - Ventas recientes muestra máximo 3 filas en mobile para no requerir scroll excesivo
- **Archivos a modificar:**
  - `frontend/app/page.tsx` — ajustes de layout y texto en secciones del dashboard
  - `frontend/components/Topbar.tsx` — simplificar en mobile si aplica

---

## Prioridad 3 — POS usable en mobile

### US-024: Punto de venta funcional en celular

- [x] El flujo de venta completo debe poder hacerse en mobile (320px–430px)
- **Contexto:** El cajero puede estar en un local pequeño usando el celular como terminal
- **Acceptance Criteria:**
  - El buscador de productos ocupa el ancho completo en mobile
  - Los ítems del carrito son legibles (nombre, cantidad, precio) sin desbordarse
  - El resumen de totales (subtotal, descuento, IVA, total) es visible sin scroll horizontal
  - El botón "Cobrar" es prominente y fácil de tocar (mínimo 48px de altura)
  - El modal de pago (selección de método, monto recibido, cambio) funciona en mobile sin que el teclado tape los campos críticos
  - El modal del ticket post-venta se puede cerrar fácilmente en mobile
- **Archivos a modificar:**
  - `frontend/app/pos/page.tsx` — ajustes de layout específicos para mobile

---

## Prioridad 4 — Páginas secundarias

### US-025: Clientes, Cartera e Inventario legibles en mobile

- [x] Las páginas más usadas por admin deben ser navegables en mobile
- **Scope:** Solo legibilidad y navegación básica — no se requiere edición en mobile
- **Acceptance Criteria:**
  - `/customers`: lista de clientes con nombre, teléfono y saldo visible sin scroll horizontal
  - `/portfolio`: tabla de cartera legible; columnas se reducen en mobile (mostrar nombre + deuda, ocultar fecha)
  - `/inventory`: lista de productos con nombre, stock y precio visible; acciones (editar/agregar stock) accesibles
  - Ninguna tabla requiere scroll horizontal en 320px
  - Las tablas en mobile pueden mostrar menos columnas que en desktop (columnas secundarias se ocultan)
- **Archivos a modificar:**
  - `frontend/app/customers/page.tsx`
  - `frontend/app/portfolio/page.tsx`
  - `frontend/app/inventory/page.tsx`

---

## Fuera del scope de este sprint

- Páginas de admin pesado (Usuarios, Configuración, Reportes) — se usan en desktop
- Instalación como PWA / app móvil
- Soporte para pantallas < 320px

---

## Criterio de éxito del sprint

Al terminar, Andrey puede abrir el sistema en su celular, crear una venta de prueba completa (buscar producto → agregar al carrito → cobrar en efectivo → ver ticket), y mostrarle eso a un prospecto en su celular sin vergüenza.

---

## Registro de progreso

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-022 | Sidebar drawer en mobile | ✅ Completo | 2026-04-09 |
| US-023 | Dashboard mobile | ✅ Completo | 2026-04-09 |
| US-024 | POS funcional en mobile | ✅ Completo | 2026-04-09 |
| US-025 | Clientes/Cartera/Inventario mobile | ✅ Completo | 2026-04-10 |
