# SPRINT-10 — Cobro Mixto, Auditoría y Gastos del Negocio

**Objetivo:** Cerrar el gap de caja diaria que más nota un tendero el primer día (no poder cobrar mitad efectivo/mitad Nequi), darle al dueño visibilidad de qué hizo cada empleado, y mostrar la ganancia neta real (no solo bruta) descontando gastos del negocio.

**Fecha inicio:** (por definir)

**Referencias:** `project-docs/better-finds-pos.md` (F1, I1, F2, F4), `project-docs/SPRINT-09-Historial-Pedidos-Catalogo.md`

**Deuda técnica entrante:**
- US-034 — Factus sandbox validation (abierta desde SPRINT-07, requiere acceso externo a Factus) — **sigue abierta**, no bloqueante, sin acción en este sprint.
- SPRINT-05 — Mobile Responsive (marcado ⬜ Pendiente en la tabla de progreso del backlog) — **sigue abierta**, no relacionada con este sprint.
- US-042c, US-043, US-044, US-045 (SPRINT-09 completo, todavía ⬜ Pendiente según `BACKLOG-Development-Workflow.md`) — **sigue abierta**. Este sprint (10) no reemplaza al 09; es una inserción priorizada por hallazgos nuevos de investigación de mercado. El dueño del backlog decide el orden real de ejecución entre SPRINT-09 y SPRINT-10.

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-046 | Pago mixto en el cobro | [ ] Pendiente | — |
| US-047 | Pantalla de auditoría | [ ] Pendiente | — |
| US-048 | Venta en espera | [ ] Pendiente | — |
| US-049 | Gastos del negocio | [ ] Pendiente | — |

---

## Orden de implementación recomendado

| Orden | US | Razón |
|-------|----|-------|
| **1** | US-048 | Cero backend, cero schema change. Sube momentum del sprint sin riesgo. |
| **2** | US-047 | La tabla `audit_log` ya existe (migration6) — solo falta escribir más eventos y exponer un endpoint de lectura. Sin migración nueva. |
| **3** | US-046 | Requiere migración 12 (nueva tabla `sale_payments` + recreación de `sales` para ampliar el CHECK de `payment_method`). Se hace después de las dos anteriores para no mezclar una migración de schema con cambios puramente de UI/lectura. |
| **4** | US-049 | Migración 12 ya está en curso por US-046; se agrega la tabla `expenses` en la misma migración para no abrir una migración 13 solo por esto. |

---

## US-048 — Venta en espera (F2)

**Como** Laura (cajera),
**quiero** poder "parquear" el carrito de un cliente que salió a buscar plata y atender a otro sin perder lo que llevaba,
**para** no hacer esperar a nadie ni perder una venta en hora pico.

**Esfuerzo estimado:** 1 día. **Impacto:** Alto. Dolor diario en hora pico, cero riesgo técnico.

**Estado verificado:** El carrito del POS vive en memoria/estado de React en `frontend/app/pos/page.tsx`; no hay backend para carritos ni tabla relacionada en `internal/database/migrate.go`. No existe ningún mecanismo de "parking" hoy.

**Contexto técnico:** 100% frontend, sin backend. Persistir el carrito activo y una lista de carritos "en espera" en `localStorage` desde `frontend/app/pos/page.tsx`. No hay migración ni endpoint nuevo.

**Scope — qué NO entra:**
- Sincronizar carritos en espera entre distintas cajas/dispositivos (eso es del alcance de E1, sync multi-caja, no de este sprint).
- Expirar o borrar automáticamente carritos viejos — quedan hasta que el cajero los retome o los descarte manualmente.

**Acceptance Criteria:**

**Parquear:**
- [ ] Botón "Dejar en espera" visible en el POS mientras el carrito tiene al menos 1 ítem
- [ ] Al presionarlo, pide un nombre corto (ej. "Pedro" o "Cliente 1") y guarda: nombre, hora, ítems, cliente seleccionado (si había), descuento aplicado
- [ ] El carrito activo se vacía después de parquear, listo para la siguiente venta
- [ ] Si no se ingresa nombre, usa "Venta N" autoincremental

**Retomar:**
- [ ] Un indicador "En espera (N)" siempre visible en el POS cuando hay ≥1 carrito parqueado
- [ ] Al abrir la lista, se ve nombre, hora ("hace 8 min") y total de cada carrito en espera
- [ ] Al seleccionar uno, se carga completo en el carrito activo (ítems, cliente, descuento) y se quita de la lista de espera
- [ ] Si el carrito activo actual tiene ítems al momento de retomar uno en espera, se pide confirmación (no se pierde silenciosamente)
- [ ] Botón "Descartar" por carrito en espera, con confirmación

**Persistencia y bordes:**
- [ ] Los carritos en espera sobreviven a un refresh de página (persisten en `localStorage`)
- [ ] Si algún producto del carrito parqueado fue desactivado o eliminado mientras estaba en espera, se muestra con aviso "Ya no disponible" al retomarlo, sin romper el resto del carrito
- [ ] Responsive: la lista de espera es usable en 320px (modal o drawer, no tabla ancha)

**DoD:**
- [ ] Sin cambios de backend, sin migración
- [ ] Estado de "en espera" vive en `localStorage` bajo una key dedicada (ej. `pos.parkedCarts`), serializado como array de objetos
- [ ] Test manual: parquear 2 carritos con nombres distintos, retomar el segundo, confirmar que el primero sigue intacto en la lista
- [ ] Test manual: parquear un carrito, recargar la página, confirmar que sigue apareciendo en "En espera"
- [ ] Build limpio (`npm run build`)

---

## US-047 — Pantalla de auditoría (I1)

**Como** Don Carlos (dueño),
**quiero** ver quién anuló una venta, quién dio un descuento, quién sacó plata de caja o cambió el precio de un producto,
**para** detectar hurto interno o errores del personal sin tener que preguntar uno por uno.

**Esfuerzo estimado:** 1 día. **Impacto:** Alto. La tabla ya existe — es casi puro cableado.

**Estado verificado:** `audit_log` existe desde migration6 (`internal/database/migrate.go:373-380`) con columnas `user_id, action, entity, entity_id, details, created_at`. Hoy solo se escribe un evento: `price_override` al vender con precio distinto al de catálogo (`internal/repository/sale.go:79`). No hay escritura para anulación de venta, descuento, salida de caja ni cambio de precio de producto. No existe endpoint `GET /audit` ni página en el frontend — confirmado por ausencia en `internal/handler/handler.go` y en `frontend/app`.

**Contexto técnico:** Sin migración nueva (la tabla ya existe). Cambios:
- `internal/repository/sale.go` — `Void()`: agregar insert en `audit_log` con `action='void'`, `entity='sale'`.
- `internal/handler/sale.go` o servicio de venta — al aplicar descuento manual (fuera de `allow_global_discount`), agregar `action='discount'`.
- `internal/repository/register.go` / handler de movimientos — al registrar `cash_out`, agregar `action='cash_out'`, `entity='register_movement'`.
- `internal/handler/products.go` (`updateProduct`) — si cambia `sale_price` o `cost` respecto al valor anterior, agregar `action='product_price_change'`, `entity='product'`, `details` con `{from, to}`.
- Nuevo endpoint `GET /api/audit` (grupo admin, mismo patrón que `/api/users` en `internal/handler/handler.go:170-175`) con filtros opcionales `?user_id=&from=&to=&action=`.
- Nueva página `frontend/app/settings` (sección) o página propia `frontend/app/audit/page.tsx` — solo visible para `role=admin`.

**Acceptance Criteria:**

**Escritura de eventos (backend):**
- [ ] Anular una venta (`DELETE /sales/{id}/void`) crea una fila `audit_log` con `action='void'`, `entity='sale'`, `entity_id=<sale_id>`, `details` con el motivo si se capturó uno
- [ ] Aplicar un descuento manual en una venta crea `action='discount'` con `details` incluyendo el % o monto
- [ ] Registrar una salida de caja (`cash_out`) crea `action='cash_out'`, `entity='register_movement'`
- [ ] Cambiar `sale_price` o `cost` de un producto vía `PUT /products/{id}` crea `action='product_price_change'` con `details={"field":"sale_price","from":10000,"to":12000}`
- [ ] Si el campo actualizado no cambió de valor, no se crea entrada (evita ruido de "guardar sin cambios")
- [ ] Los eventos existentes de `price_override` (venta con precio distinto al de catálogo) siguen funcionando sin cambios

**Vista de auditoría (frontend, solo admin):**
- [ ] Página listada solo en el menú si `role === 'admin'`
- [ ] Tabla con columnas: fecha/hora, usuario, acción, entidad afectada, detalle
- [ ] Filtro por usuario (dropdown de usuarios activos e inactivos)
- [ ] Filtro por rango de fechas
- [ ] Filtro por tipo de acción (dropdown: void, discount, cash_out, product_price_change, price_override)
- [ ] Sin resultados con los filtros aplicados → mensaje "Sin eventos en este rango"
- [ ] Un cajero (`role='cashier'`) que intente acceder a la URL directamente recibe 403/redirect, no ve datos
- [ ] Responsive: tabla con scroll horizontal en 320px

**DoD:**
- [ ] `GET /api/audit` (grupo admin, igual patrón que `/api/users`) → `[]{id, user_id, username, action, entity, entity_id, details, created_at}` (join con `users` para el nombre)
- [ ] Query params opcionales: `user_id`, `action`, `from`, `to` — todos combinables con AND
- [ ] Nuevo `AuditRepo` o extensión de repo existente con método `List(ctx, filter)`
- [ ] Test: anular una venta → aparece exactamente 1 fila con `action='void'` y el `entity_id` correcto
- [ ] Test: `PUT /products/{id}` cambiando solo el nombre (no precio/costo) → no crea fila de auditoría
- [ ] Test: `GET /api/audit?action=cash_out` → retorna solo filas de ese tipo
- [ ] Build limpio (`go build`, `npm run build`)

---

## US-046 — Pago mixto en el cobro (F1)

**Como** Laura (cajera),
**quiero** cobrar una venta combinando varios métodos de pago (por ejemplo $30.000 efectivo + $20.000 Nequi + el resto a crédito),
**para** no rechazar ventas cuando el cliente no tiene el monto completo en un solo medio.

**Esfuerzo estimado:** 2.5 días. **Impacto:** Alto. Gap #1 de la ronda 2 — un tendero lo nota el primer día.

**Estado verificado:** `sales.payment_method` es una sola columna `TEXT NOT NULL CHECK (payment_method IN ('cash', 'transfer', 'card', 'credit'))` (`internal/database/migrate.go:214`). No existe tabla de pagos por venta. El insert de venta (`internal/repository/sale.go:46-50`) escribe un único `payment_method` por venta. `internal/database/migrate.go:266-267` documenta que SQLite no permite `ALTER TABLE` para modificar un `CHECK` existente — el precedente ya usado en el proyecto (`stock >= 0` en `products`) es validar esa regla en la capa de aplicación en vez de en el schema.

**Contexto técnico:**
- **Migración 12** (nueva, comparte migración con US-049 — ver más abajo):
  - Recrear `sales` con el patrón estándar de SQLite (`CREATE TABLE sales_new ...` con el mismo schema pero `CHECK (payment_method IN ('cash','transfer','card','credit','mixed'))`, copiar filas con `INSERT INTO sales_new SELECT * FROM sales`, `DROP TABLE sales`, `ALTER TABLE sales_new RENAME TO sales`, recrear índices `idx_sales_register` etc.). Es la única forma de ampliar el `CHECK` sin desactivar la validación a nivel de aplicación para todo el resto de valores.
  - Nueva tabla:
    ```sql
    CREATE TABLE IF NOT EXISTS sale_payments (
        id         INTEGER PRIMARY KEY AUTOINCREMENT,
        sale_id    INTEGER NOT NULL REFERENCES sales(id),
        method     TEXT NOT NULL CHECK (method IN ('cash', 'transfer', 'card', 'credit')),
        amount     REAL NOT NULL CHECK (amount > 0),
        reference  TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    CREATE INDEX IF NOT EXISTS idx_sale_payments_sale ON sale_payments(sale_id);
    ```
- `internal/repository/sale.go` — `Create()`: si la venta trae más de un pago, insertar `sales.payment_method='mixed'` y una fila en `sale_payments` por cada método; si trae exactamente uno, mantener el comportamiento actual (columna directa) **y además** insertar la fila equivalente en `sale_payments`, para que todo reporte pueda sumar siempre desde `sale_payments` sin ramas especiales.
- `internal/service/sale.go` — validar que `SUM(sale_payments.amount) == sale.Total` (tolerancia de redondeo ±$1 COP). Si no cuadra, rechazar con 400 antes de tocar stock/caja.
- **Reportes que se rompen y cómo se arreglan:**
  - `internal/repository/report.go` — cualquier `GROUP BY payment_method` sobre `sales` debe migrar a `GROUP BY sale_payments.method` (una venta `mixed` de $50k con $30k efectivo + $20k transferencia debe sumar $30k a la fila "cash" y $20k a "transfer" del reporte, no $50k a "mixed").
  - Cierre de caja (`internal/handler/register.go`, `internal/repository/register.go`) — el cálculo de `expected_close` (efectivo esperado) debe sumar solo `sale_payments.amount WHERE method='cash'`, no el total de ventas con `payment_method='cash'` (que ahora también deja fuera el efectivo de una venta mixta).
- `internal/handler/sale.go` — `createSale`: el payload gana `payments: [{method, amount, reference}]`, reemplazando el uso directo de `payment_method` en el body (se mantiene `payment_method` de solo lectura en la respuesta, derivado).
- Frontend `frontend/app/pos/page.tsx` — modal de cobro gana selector de "agregar otro método de pago" con línea por método y monto, mostrando el restante a cubrir en tiempo real.

**Regla de negocio:**
```
suma_pagos = Σ payments[].amount
si suma_pagos != sale.total (tolerancia ±$1 COP) → rechazar venta, HTTP 400
si algún payments[].method == 'credit' → requiere customer_id (igual que hoy con payment_method='credit' puro)
si len(payments) == 1 → sales.payment_method = payments[0].method
si len(payments) > 1  → sales.payment_method = 'mixed'
```

**Scope — qué NO entra:**
- Integración real con pasarela Nequi/Wompi (sigue siendo un campo/registro manual, sin API — igual que hoy con `transfer`).
- Pago mixto en devoluciones/notas crédito (v2, cuando exista F3 devolución parcial).

**Acceptance Criteria:**

**Modal de cobro:**
- [ ] Por defecto el modal se comporta igual que hoy (un solo método, un solo monto = total)
- [ ] Botón "Agregar otro método de pago" agrega una segunda línea método+monto
- [ ] El sistema muestra en todo momento "Falta cubrir: $X" mientras la suma de líneas sea menor al total
- [ ] No se puede confirmar el cobro si `suma_pagos != total`
- [ ] Se puede combinar hasta 3 métodos en una misma venta (cash + transfer + credit, por ejemplo)
- [ ] Si uno de los métodos es `credit` y no hay cliente seleccionado, se bloquea con el mismo mensaje que usa hoy la venta 100% a crédito
- [ ] El ticket (`getTicket`) muestra el desglose por método cuando la venta es mixta: "Efectivo: $30.000 · Nequi: $20.000 · Fiado: $10.000"

**Reportes y cierre de caja:**
- [ ] El reporte de ventas por método de pago (`GET /reports/sales`) suma correctamente cuando hay ventas mixtas (no duplica ni ignora montos)
- [ ] El cierre de caja calcula `expected_close` sumando solo la porción en efectivo de cada venta, incluidas las mixtas
- [ ] Responsive: el modal de cobro con múltiples líneas es usable en 320px

**DoD:**
- [ ] Migración 12: tabla `sale_payments` + recreación de `sales` (CHECK ampliado a `mixed`) — ver SQL exacto en Contexto técnico
- [ ] `POST /sales` (autenticado) — body gana `payments: [{method, amount, reference}]` en vez de `payment_method` suelto; respuesta sigue incluyendo `payment_method` (derivado: `'mixed'` o el único método)
- [ ] `GET /reports/sales` y el cálculo de `expected_close` en cierre de caja actualizados para leer de `sale_payments`
- [ ] Test: venta de $60.000 con pagos `[{cash,30000},{transfer,20000},{credit,10000}]` → `sales.payment_method='mixed'`, 3 filas en `sale_payments`, cliente queda con `+10000` de saldo
- [ ] Test: venta con un solo pago `[{cash,15000}]` sobre total $15.000 → se comporta igual que antes (`payment_method='cash'`), sin romper flujo existente
- [ ] Test: pagos que suman $59.999 sobre total $60.000 (fuera de tolerancia) → rechazada con 400
- [ ] Test: cierre de caja con 1 venta mixta ($30k cash de $60k total) + 1 venta 100% cash ($20k) → `expected_close` cash = $50.000, no $80.000
- [ ] Build limpio (`go build`, `npm run build`)

---

## US-049 — Gastos del negocio (F4)

**Como** Don Carlos (dueño),
**quiero** registrar los gastos del negocio (luz, domiciliario, arriendo) categorizados,
**para** ver la ganancia neta real, no solo la ganancia bruta de ventas.

**Esfuerzo estimado:** 1.5 días. **Impacto:** Alto. Sin esto, la ganancia bruta que ya muestra el dashboard es una cifra que no refleja lo que realmente le queda al dueño.

**Estado verificado:** `register_movements` solo tiene `type CHECK IN ('sale', 'cash_in', 'cash_out', 'opening')` (`internal/database/migrate.go:245-251`), sin categoría ni relación a un reporte de utilidad. No existe tabla `expenses`. El dashboard de ganancia bruta vive en `internal/handler/dashboard.go` / `getDashboardToday` — no descuenta gastos.

**Contexto técnico:**
- **Migración 12** (misma migración que US-046):
  ```sql
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
  ```
  `register_id` es nullable a propósito: un gasto (ej. arriendo mensual) puede registrarse sin caja abierta en ese momento.
- Cuando un gasto se registra con caja abierta, además insertar en `register_movements` con `type='cash_out'` y `description='Gasto: <categoria> — <note>'`, para que el arqueo de caja (US existente de cierre) lo siga sumando como salida de efectivo sin duplicar lógica de caja.
- Nuevo endpoint `POST /api/expenses` (autenticado) y `GET /api/expenses` (autenticado, filtros `from/to/category`).
- `internal/handler/dashboard.go` — `getDashboardToday` gana `net_profit = gross_profit - Σ expenses.amount WHERE created_at::date = today`.
- Nueva sección en `frontend/app/reports/page.tsx` (o pestaña dedicada): registrar gasto + listado + card "Ganancia neta" junto a la bruta existente.

**Regla de negocio:**
```
ganancia_bruta_hoy = (existente en dashboard, sin cambios)
gastos_hoy         = SUM(expenses.amount) WHERE date(created_at) = date('now')
ganancia_neta_hoy  = ganancia_bruta_hoy - gastos_hoy   -- puede ser negativa, se muestra tal cual (en rojo)
```

**Scope — qué NO entra:**
- Gastos recurrentes/programados (ej. "arriendo automático cada mes 1") — se registra manual cada vez.
- Aprobación de gastos por rol (todo usuario autenticado puede registrar) — permisos granulares quedan para I4 (fuera de este sprint).

**Acceptance Criteria:**

**Registrar gasto:**
- [ ] Formulario con: categoría (dropdown fijo: Servicios, Arriendo, Nómina, Proveedor, Otro), monto, nota opcional
- [ ] Monto debe ser mayor a 0, valida en cliente y servidor
- [ ] Si hay una caja abierta del usuario actual, el gasto se asocia a esa caja automáticamente (sin pedirlo)
- [ ] Si no hay caja abierta, el gasto se guarda igual (`register_id = null`), sin bloquear el registro
- [ ] Confirmación visual al guardar ("Gasto registrado: $80.000 — Servicios")

**Listado y reporte:**
- [ ] Lista de gastos del día/rango con categoría, monto, nota, usuario, hora
- [ ] Filtro por categoría y por rango de fechas
- [ ] Total de gastos del período visible arriba de la lista
- [ ] Dashboard muestra "Ganancia neta" junto a "Ganancia bruta" existente, con el signo correcto si los gastos superan la ganancia bruta (número en rojo, no oculta el negativo)
- [ ] Vacío: sin gastos en el rango → "Sin gastos registrados en este período"
- [ ] Responsive: formulario y lista usables en 320px

**Caja:**
- [ ] Un gasto registrado con caja abierta aparece también en el detalle de movimientos de esa caja (`GET /registers/{id}/movements`) como `cash_out`
- [ ] El cierre de caja (`expected_close`) descuenta los gastos igual que cualquier otro `cash_out`, sin doble conteo

**DoD:**
- [ ] Migración 12 (compartida con US-046): tabla `expenses` — ver SQL exacto en Contexto técnico
- [ ] `POST /api/expenses` (autenticado) → `{category, amount, note}` → crea gasto y, si hay caja abierta, movimiento `cash_out` asociado
- [ ] `GET /api/expenses?from=&to=&category=` (autenticado) → lista + total
- [ ] `GET /api/dashboard/today` extiende la respuesta con `net_profit` (además del `gross_profit` existente)
- [ ] Test: gasto de $80.000 (Servicios) con caja abierta → aparece en `register_movements` como `cash_out` y en `expected_close` del cierre
- [ ] Test: ganancia bruta $200.000 hoy, gastos $80.000 + $40.000 hoy → `net_profit = 80.000`
- [ ] Test: gasto con monto 0 o negativo → rechazado con 400
- [ ] Build limpio (`go build`, `npm run build`)

---

## Criterio de éxito del sprint

- Laura cobra $60.000 combinando $30.000 efectivo + $20.000 Nequi + $10.000 fiado en una sola venta, sin trucos ni ventas separadas.
- Don Carlos abre "Auditoría" y en segundos ve que fue Laura quien anuló la venta #453 y a qué hora.
- En hora pico, Laura deja el carrito de un cliente en espera, atiende a otro, y retoma el primero sin haber perdido nada.
- Don Carlos ve al final del día que vendió $200.000 en bruto pero, tras $120.000 de gastos (luz + domiciliario), la ganancia neta real fue $80.000 — no una cifra inflada.

---

## Notas técnicas para el dev

- **US-046 y US-049 comparten la migración 12** — hacerla en un solo archivo/función `migration12` con ambos bloques SQL, para no forzar una migración 13 solo por los gastos.
- La recreación de `sales` en US-046 es la parte más riesgosa del sprint: hacerla primero dentro de la migración, en una transacción, y correr toda la suite de tests de `internal/service/sale_test.go` y `internal/repository` antes de tocar cualquier otro archivo. Si algo en `report.go` o `register.go` sigue leyendo `sales.payment_method` directo después de este cambio, hay que encontrarlo con `grep -rn "payment_method" internal/` — no asumir que se encontraron todos los usos listados en este documento.
- **US-047**: no crear un paquete de auditoría nuevo si un método `Insert` simple sobre la tabla ya existente alcanza — evitar sobre-ingeniería en algo que hoy es una sola escritura por evento.
- Todas las queries de fecha (`date(created_at) = date('now')`, rangos `from/to`) deben usar las mismas funciones SQLite (`date()`, `julianday()`) que ya usa `internal/repository/report.go`, por consistencia de zona horaria (sin TZ explícito, todo en hora local del servidor — igual que el resto del proyecto).

---

## Roadmap — qué queda del doc fuente

| Sprint | Features (ID fuente) | Por qué ahí |
|--------|----------------------|-------------|
| SPRINT-11 | G1 (proveedores + compras), F5+F6 (arqueo denominaciones + ticket de cierre) | G1 es 3 días solo — desbloquea US-044 (SPRINT-09) y no cabía en el presupuesto de este sprint sin exceder 7 días. F5+F6 son rápidas (1.5 días juntas) y profesionalizan el cierre que ya se tocó en US-046/US-049. |
| SPRINT-12 | I2 (resumen WhatsApp al dueño) + K1 (acceso LAN desde celular) | Combo de 1 día documentado en el bloque K del doc fuente — cubre "ver ventas en celular" sin romper el posicionamiento offline. |
| SPRINT-13 | H2 (apartados/layaway), G2 (kardex + ajustes de inventario) | Extienden el diferenciador de fiado y cumplen el "requisito 2026" de kardex, pero el propio doc fuente los marca como pedidos del mes 2, no del día 1. |
| Sin fecha | F7+I4 (PIN rápido + permisos granulares), G3–G7 (vencimientos, combos, etiquetas, historial de precio, foto), H3–H5 (domicilios, impuesto bolsas, cotizaciones), E1 (sync multi-caja), E2 (asistente IA local), scoring de cliente fiado | Necesarios cuando el negocio crece a >1 empleado o multi-caja, o son features de nicho (farmacia/vencimientos, ferretería/combos). No hay señal de que bloqueen la venta al primer cliente. |
| Con partner externo (no ejecutable en solitario) | H1 (recargas/pago de servicios), capital embebido | Requieren alianza con fintech regulada o agregador (Movilred/Puntored/SuRed) — fuera del alcance de "ejecutable en solitario". |
