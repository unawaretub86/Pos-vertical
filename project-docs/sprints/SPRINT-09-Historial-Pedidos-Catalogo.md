# SPRINT-09 — Historial de Cliente, Pedidos Inteligentes y Catálogo

**Objetivo:** Darle al tendero contexto del cliente en el momento de cobrar, automatizar la decisión de qué pedir al proveedor, y permitirle compartir su catálogo por WhatsApp sin salir del sistema.

**Fecha inicio:** (por definir)

**Referencias:** `project-docs/find-pos.md`, `project-docs/SPRINT-08-Rentabilidad-Cartera.md`

**Deuda técnica entrante:**
- US-034 — Factus sandbox validation (sigue abierto de SPRINT-07, requiere acceso externo)
- US-042 — Velocidad de venta / días restantes (pospuesta de SPRINT-08 por falta de historial suficiente — **entra en este sprint** como US-042c)

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-043 | Historial del cliente en POS al cobrar | [ ] Pendiente | — |
| US-044 | Sugerencia de pedido al proveedor | [ ] Pendiente | — |
| US-045 | Catálogo compartible por WhatsApp | [ ] Pendiente | — |
| US-042c | Dashboard: stock con días restantes (velocidad de venta) | [ ] Pendiente | — |

---

## Orden de implementación recomendado

| Orden | US | Razón |
|-------|----|-------|
| **1** | US-042c | Query sobre `sale_items` que ya existe. Sin schema change. Cierra deuda de SPRINT-08. |
| **2** | US-043 | Misma query base que US-042c — reutiliza el patrón de agregación sobre ventas. Sin schema change. |
| **3** | US-044 | Extiende la lógica de velocidad de venta de US-042c al módulo de inventario. Nuevo endpoint. |
| **4** | US-045 | Requiere nueva columna en settings + endpoint público sin JWT. Va al final para no bloquear el sprint. |

---

## US-042c — Dashboard: stock con días restantes (velocidad de venta)

**Como** Don Carlos (dueño),
**quiero** ver no solo qué productos están bajo el mínimo, sino cuántos días me quedan de cada uno al ritmo actual de ventas,
**para** priorizar qué pedir antes de quedarme sin stock.

**Esfuerzo estimado:** 0.5 días. **Impacto:** Medio-alto. Mejora directa del panel "Reponer pronto" de US-042b.

**Contexto técnico:** US-042b ya muestra productos con `stock < min_stock`. Esta US agrega `days_remaining = stock / avg_daily_sales_7d` a cada ítem. Si `avg_daily_sales_7d = 0` (producto sin ventas en 7 días), no se muestra el campo (no divide entre cero).

**Regla de días restantes:**
- `avg_daily = SUM(quantity) / 7` sobre `sale_items JOIN sales` donde `sales.created_at >= now - 7 days` y `sales.status = 'completed'`
- `days_remaining = ROUND(stock / avg_daily)` — solo si `avg_daily > 0`
- Si `avg_daily = 0` → no mostrar días (producto sin movimiento reciente)

**Acceptance Criteria:**
- [ ] Panel "Reponer pronto" en dashboard muestra `days_remaining` junto al déficit de stock
- [ ] Formato: `Arroz Diana · 3 unid / mín 10 · **2 días restantes**`
- [ ] Si `avg_daily = 0`, mostrar solo `3 unid / mín 10` sin días (no mostrar "∞ días" ni "—")
- [ ] Ordenado por `days_remaining ASC` (los más críticos primero). Productos sin datos de velocidad van al final
- [ ] Si el producto está bajo mínimo pero tiene velocidad = 0 → aparece en el panel sin días restantes
- [ ] Limitado a top 5. Sin cambio de este límite
- [ ] Responsive: funciona en 320px

**DoD:**
- [ ] `GET /api/dashboard/today` extiende `low_stock []{ product_id, name, stock, min_stock, avg_daily, days_remaining }` — `avg_daily` y `days_remaining` son `null` si no hay ventas en 7 días
- [ ] Query no falla si no hay ventas en los últimos 7 días (LEFT JOIN)
- [ ] Test: producto con 5 unidades, promedio 2.5/día → `days_remaining = 2`; producto con 0 ventas en 7 días → `days_remaining = null`
- [ ] Build limpio (`go build`, `npm run build`)

---

## US-043 — Historial del cliente en POS al cobrar

**Como** Laura (cajera) o Don Carlos,
**quiero** ver al seleccionar un cliente en el carrito: su última visita, lo que suele comprar, y si tiene deuda,
**para** saber con quién estoy tratando antes de cerrar la venta y evitar fiar a quien ya debe demasiado.

**Esfuerzo estimado:** 2 días. **Impacto:** Alto. Ningún POS low-cost colombiano muestra esto en el flujo de cobro.

**Contexto técnico:** Toda la data necesaria existe en `sales JOIN sale_items JOIN products WHERE customer_id = ?`. El balance ya se calcula en `customerRepo.Balance()`. No hay schema change. Nuevo endpoint `GET /api/customers/:id/pos-summary`.

**Qué muestra el panel en el POS:**
```
┌─ Pedro Ramírez ──────────────────────────────┐
│ Última visita: hace 3 días                    │
│ Lo de siempre: Arroz · Aceite · Leche         │
│ ⚠ Debe: $80.000 (22 días)                     │
└──────────────────────────────────────────────┘
```

**Acceptance Criteria:**

**Panel en carrito (aparece al seleccionar cliente):**
- [ ] Al seleccionar un cliente en el carrito del POS, aparece panel compacto debajo del selector
- [ ] Muestra "Última visita: hace X días" (basado en la venta más reciente del cliente, cualquier método de pago)
- [ ] Si nunca ha comprado: "Cliente nuevo — sin historial"
- [ ] Muestra "Lo de siempre:" con los top 3 productos más comprados en frecuencia (por número de transacciones, no por cantidad)
- [ ] Si tiene menos de 3 productos distintos en historial → muestra los que hay (1 o 2)
- [ ] Si no tiene ningún historial de compras → no muestra sección "Lo de siempre"
- [ ] Si `balance > 0`: muestra badge de deuda: `⚠ Debe: $80.000 (22 días)` con color amarillo o rojo según antigüedad (regla de SPRINT-08: < 7d gris, 7–14d amarillo, > 14d rojo)
- [ ] Si `balance = 0`: no muestra sección de deuda (no mostrar "Sin deuda" — simplemente no aparece)
- [ ] Si `balance > credit_limit` (y `credit_limit != null`): muestra aviso adicional: `"Superó tope de crédito ($150.000)"`
- [ ] Panel no bloquea la venta — es informativo únicamente
- [ ] Panel se puede cerrar/minimizar con un clic (para no ocupar espacio durante cobro rápido)
- [ ] Al deseleccionar el cliente: el panel desaparece

**Acciones rápidas desde el panel:**
- [ ] Botón "Agregar lo de siempre" agrega los top 3 productos al carrito de una vez (si tienen stock)
- [ ] Si alguno de los top 3 no tiene stock → se omite y se muestra toast: `"Arroz no tiene stock — no se agregó"`
- [ ] Botón solo aparece si el cliente tiene historial

**DoD:**
- [ ] `GET /api/customers/:id/pos-summary` retorna: `{ last_visit_at, top_products []{product_id, name, times_bought}, balance, days_overdue, credit_limit }`
- [ ] `last_visit_at` = fecha de la venta más reciente del cliente (cualquier método)
- [ ] `top_products` = top 3 por `COUNT(DISTINCT sale_id)` en `sale_items JOIN sales WHERE customer_id = ? AND status = 'completed'`
- [ ] Endpoint retorna 200 con campos nulos si cliente no tiene historial (no 404)
- [ ] Test: cliente con 5 ventas y 8 productos distintos → retorna exactamente 3 productos ordenados por frecuencia descendente
- [ ] Test: cliente nuevo sin ventas → `last_visit_at = null`, `top_products = []`, `balance = 0`
- [ ] Panel visible en 320px (stack vertical, texto reducido si necesario)
- [ ] La llamada al endpoint se hace solo cuando se selecciona un cliente (no en cada keystroke)

---

## US-044 — Sugerencia de pedido al proveedor

**Como** Don Carlos (dueño),
**quiero** ver una lista de qué debo pedir al proveedor esta semana, con la cantidad sugerida basada en mis ventas recientes,
**para** no quedarm sin stock de los productos que más rotan sin tener que revisar el inventario producto por producto.

**Esfuerzo estimado:** 2 días. **Impacto:** Alto. Completamente diferenciador — ningún POS low-cost colombiano lo hace.

**Contexto técnico:** Extiende la lógica de velocidad de venta de US-042c. Query sobre `sale_items JOIN sales`. No hay schema change. Nuevo endpoint `GET /api/inventory/reorder-suggestions`. Nueva vista en el módulo de inventario.

**Lógica de sugerencia:**
```
avg_daily_7d = SUM(quantity_sold) / 7   -- últimos 7 días
days_remaining = stock / avg_daily_7d
suggested_order = CEIL(avg_daily_7d * 14) - stock   -- para tener 2 semanas de stock
```

**Criterio de aparición en la lista:**
- Producto con `active = true`
- `avg_daily_7d > 0` (tiene ventas recientes — si no vende no hay sugerencia)
- `days_remaining < 14` (le quedan menos de 2 semanas al ritmo actual)

**Acceptance Criteria:**

**Vista de sugerencias (sección en inventario o tab separado):**
- [ ] Nueva sección "Pedido sugerido" accesible desde el módulo de inventario (botón o tab)
- [ ] Muestra lista de productos que cumplen el criterio: `avg_daily > 0` Y `days_remaining < 14`
- [ ] Por producto: nombre, stock actual, promedio diario, días restantes, cantidad sugerida a pedir
- [ ] Formato: `Arroz Diana · 5 unid · 2.5/día · **2 días** → Pedir 30 unid`
- [ ] Ordenado por `days_remaining ASC` (más urgente primero)
- [ ] Badge de urgencia por días: `≤ 2 días` rojo, `3–7 días` amarillo, `8–13 días` gris
- [ ] Si no hay productos que cumplan el criterio: panel vacío con mensaje "Todo en orden — ningún producto requiere pedido esta semana"
- [ ] Botón "Copiar lista" genera texto plano con la lista para pegar en WhatsApp al proveedor:
  ```
  Pedido sugerido — [fecha]
  - Arroz Diana: 30 unid
  - Aceite Girasol: 12 unid
  - Leche Alpina: 24 unid
  ```
- [ ] Sin límite de items (mostrar todos los que cumplen criterio — es una lista de trabajo, no un widget)

**DoD:**
- [ ] `GET /api/inventory/reorder-suggestions` retorna `[]{product_id, name, stock, avg_daily, days_remaining, suggested_order}` ordenado por `days_remaining ASC`
- [ ] Query usa ventana de 7 días exactos desde `now() - 7 days`
- [ ] Productos con `active = false` excluidos
- [ ] Productos con `avg_daily = 0` excluidos (sin ventas en últimos 7 días)
- [ ] `suggested_order` mínimo 1 (si el cálculo da 0.something → redondear arriba)
- [ ] Test: producto con 35 unidades en stock, promedio 5/día → `days_remaining = 7`, `suggested_order = 35` (para llegar a 70 = 14 días)
- [ ] Test: producto con 20 unidades, promedio 1/día, `days_remaining = 20` → no aparece en lista (> 14 días)
- [ ] Test: producto inactivo con ventas → no aparece
- [ ] Responsive: tabla con scroll horizontal en 320px si columnas no caben

---

## US-045 — Catálogo compartible por WhatsApp

**Como** Don Carlos (dueño),
**quiero** generar un link con mis productos y precios para enviárselo a clientes por WhatsApp,
**para** que puedan ver qué tengo y a qué precio sin tener que llamarme.

**Esfuerzo estimado:** 2.5 días. **Impacto:** Medio-alto. Extensión natural del botón WhatsApp de US-035.

**Contexto técnico:** Requiere nuevo campo `catalog_token TEXT` y `catalog_enabled BOOLEAN DEFAULT 0` en tabla `settings` (o nueva tabla `catalog_config`). El endpoint público `/public/catalog/:token` no lleva JWT middleware. La página del catálogo es estática en Next.js (`/catalog/[token]/page.tsx`) que llama al backend. Sin fotos en MVP.

**Scope MVP — qué NO entra:**
- Fotos de productos (complejidad de storage + upload)
- Carrito de compras en el catálogo (es solo consulta)
- Personalización de colores o logo
- Selección granular de qué productos aparecen (MVP: todos los activos con precio > 0)

**Acceptance Criteria:**

**Configuración (en Settings):**
- [ ] Nueva sección "Catálogo público" en la página de ajustes
- [ ] Toggle para activar/desactivar el catálogo (default: desactivado)
- [ ] Al activar por primera vez: se genera un token UUID aleatorio y se guarda en DB
- [ ] Muestra el link generado: `http://localhost:PORT/catalogo/[token]` (o la IP del servidor si está configurada)
- [ ] Botón "Copiar link" copia al portapapeles
- [ ] Botón "Compartir por WhatsApp" abre `https://wa.me/?text=...` con el link
- [ ] Si el catálogo está desactivado: el endpoint público retorna 404 (el link deja de funcionar)
- [ ] Botón "Regenerar link" genera nuevo token (invalida el anterior) — con confirmación previa

**Catálogo público (`/catalogo/:token`):**
- [ ] Página accesible sin login — renderizada desde el frontend
- [ ] Muestra: nombre del negocio (de `settings.business_name`), lista de productos activos con precio > 0
- [ ] Por producto: nombre, precio COP formateado (`$12.000`), unidad de medida
- [ ] Si `wholesale_price` existe → no se muestra (precio mayorista es privado)
- [ ] Ordenado por categoría (si existe), luego alfabético
- [ ] Si el token no existe o catálogo está desactivado: página de "catálogo no disponible"
- [ ] Sin paginación en MVP (lista completa)
- [ ] Responsive: funciona en mobile (principal canal de uso es celular del cliente)
- [ ] Sin navbar de la app (es una página pública, no la app del tendero)

**DoD:**
- [ ] Nueva migración SQL: `ALTER TABLE settings ADD COLUMN catalog_token TEXT`, `ADD COLUMN catalog_enabled INTEGER DEFAULT 0`
- [ ] `GET /public/catalog/:token` — sin JWT middleware — retorna `{ business_name, products []{name, sale_price, unit, category} }` si `catalog_enabled = 1` y token coincide
- [ ] `PUT /api/settings/catalog` — con JWT — activa/desactiva y regenera token
- [ ] Token generado con `crypto/rand` (UUID v4 o 32 hex chars aleatorios)
- [ ] Test: catálogo habilitado con token válido → 200 con productos activos
- [ ] Test: token inválido → 404
- [ ] Test: catálogo desactivado con token válido → 404
- [ ] Test: productos inactivos o con precio = 0 → no aparecen en catálogo
- [ ] Build limpio

---

## Criterio de éxito del sprint

- Laura selecciona a "Pedro Ramírez" en el POS y ve en 1 segundo que debe $80k desde hace 22 días y suele llevar arroz, aceite y leche
- Don Carlos abre "Pedido sugerido" y tiene en 10 segundos una lista lista para copiar y enviar al proveedor por WhatsApp
- Don Carlos genera un link de catálogo y se lo manda a un cliente nuevo sin necesitar mostrarle el sistema
- Los paneles del dashboard (US-042b) ahora muestran días restantes para los productos con ventas recientes

---

## Notas técnicas para el dev

- **US-043 y US-044** comparten la misma sub-query de `avg_daily_sales_7d` — extraerla como CTE o función helper para no duplicar SQL
- **US-045** endpoint público debe registrarse en el router **antes** del middleware JWT, no dentro del grupo autenticado
- **Catálogo token:** usar `database/crypto` (no `math/rand`) para generación del token
- **US-042c** es una extensión del campo `low_stock` existente en `dashboard/today` — agregar `avg_daily` y `days_remaining` como campos nullable, no rompe clientes que ya leen el campo
- La query de velocidad de venta debe usar `julianday('now') - julianday(sales.created_at) <= 7` para compatibilidad con SQLite
