# SPRINT-11 — QA de Cálculos, Proveedores/Compras y Cierre de Caja Profesional

**Objetivo:** Cerrar la brecha de confianza en los cálculos de dinero que salió a la luz probando SPRINT-10 (IVA nunca se cobraba, dato de `tax_rate` corrupto, carrito vendía sin stock), dar al negocio un flujo real de compras a proveedor que actualice costos, y profesionalizar el cierre de caja con conteo de denominaciones y tirilla imprimible.

**Fecha inicio:** (por definir)

**Referencias:** `project-docs/better-finds-pos.md` (G1, F5, F6), `project-docs/SPRINT-10-Cobro-Mixto-Auditoria-Gastos.md` (incidente de QA que origina US-050), `qa-checklist.md`

**Deuda técnica entrante:**
- US-034 — Factus sandbox validation (abierta desde SPRINT-07, requiere acceso externo a Factus) — **sigue abierta**, no bloqueante, sin acción en este sprint.
- SPRINT-05 — Mobile Responsive (marcado ⬜ Pendiente en la tabla de progreso del backlog) — **sigue abierta**, no relacionada con este sprint.
- US-042c, US-043, US-044, US-045 (SPRINT-09, ⬜ Pendiente) — **sigue abierta**. US-044 (sugerencia de pedido) depende de G1 (US-051 de este sprint) para tener a quién sugerir el pedido — queda desbloqueada al cierre de este sprint, pero su implementación en sí no entra aquí.
- US-046, US-049 (SPRINT-10) — según el estado del repo al momento de escribir esto, US-046 (pago mixto) está en desarrollo/testing y US-049 (gastos) sigue pendiente. **Este sprint no reemplaza al 10** — si SPRINT-10 sigue sin cerrar cuando arranque este, termínalo primero; el orden real lo decide el dueño del backlog.

**Incidente que origina US-050 (contexto para el dev):** durante el testing manual de US-046 se encontraron en cadena, en producción, 4 bugs de correctitud que ningún test existente atrapó: (1) el checkout del POS nunca calculaba ni cobraba IVA — el campo `tax` iba hardcodeado en `0`, cualquier producto con `tax_rate > 0` rompía el cobro o, si la validación del backend no existiera, habría cobrado de menos; (2) el formulario de inventario guardaba/cargaba `tax_rate` como número crudo en vez de convertir %↔fracción, corrompiendo el dato de productos reales (19 guardado en vez de 0.19 — un error de 100x que además viaja a la factura DIAN); (3) la pantalla de auditoría (`frontend/app/audit/page.tsx`) tenía un loop infinito de render (referencia de objeto inestable en dependencia de `useEffect`) que la dejaba en "Cargando…" para siempre; (4) el carrito del POS dejaba agregar más unidades de las que había en stock. Los 4 ya están corregidos (rama `feat--3`, sesión del 2026-09-08), pero ninguno tenía cobertura de test antes — de ahí la prioridad de US-050.

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-050 | QA: cobertura de cálculos financieros + CI | [x] Completo | 2026-09-08 |
| US-051 | Arqueo con conteo de denominaciones + ticket de cierre (F5+F6) | [ ] Pendiente | — |
| US-052 | Proveedores y compras (G1) | [ ] Pendiente | — |

---

## Orden de implementación recomendado

| Orden | US | Razón |
|-------|----|-------|
| **1** | US-050 | Cero riesgo de schema, cero dependencias. Es la respuesta directa al incidente de SPRINT-10 — se hace primero para que US-051 y US-052 ya nazcan con el hábito de test de dinero en su lugar (los DoD de ambas referencian los patrones que US-050 deja establecidos). |
| **2** | US-051 | Sin migración nueva (ver Contexto técnico) — UI + reuso de `TicketPreview`. Sube momentum antes de tocar schema. |
| **3** | US-052 | Requiere migración 13 (`suppliers`, `purchases`, `purchase_items`) y toca `products.cost`. Se deja al final por ser la más riesgosa y la única con cambio de schema, mismo criterio que SPRINT-10 aplicó a US-046. |

---

## US-050 — QA: cobertura de cálculos financieros + CI

**Como** Andrey (dueño del producto y único developer),
**quiero** que los cálculos de dinero (subtotal, IVA, descuentos, pagos mixtos, saldo de cartera, cierre de caja) tengan test automatizado que corra antes de cada push, y que el checklist manual sepa cuáles de sus ítems ya están cubiertos por esos tests,
**para** no volver a enterarme de un bug de cobro por un cliente real en vez de por un test que corre en segundos.

**Esfuerzo estimado:** 1 día. **Impacto:** Alto — es la garantía de que el resto del roadmap (G1 con costo promedio ponderado, F5 con suma de denominaciones) no repite el mismo patrón de bug.

**Estado verificado:**
- Backend: 11 archivos `*_test.go` existen (`internal/service`, `internal/repository`, `internal/handler`, `internal/factelect`); `go test ./...` corre y pasa. Los tests de dinero existentes cubren pago único, pago mixto y (desde el fix de esta sesión) mixto + IVA (`internal/service/sale_mixed_test.go`), pero **no existían antes del incidente** — no había ningún test que combinara `tax_rate > 0` con nada.
- Frontend: cero infraestructura de test antes de esta sesión (`grep` sobre `frontend/package.json` no tenía ningún test runner). Se agregó `vitest` como devDependency y `frontend/lib/saleCalc.ts` (subtotal/IVA/total/balance de pagos, extraído de `frontend/app/pos/page.tsx`) con `frontend/lib/saleCalc.test.ts` (11 tests). Es el único módulo de frontend con test hoy.
- CI: no existe `.github/workflows/` en el repo — hoy nada corre `go test` ni `npm test` automáticamente antes de un push o PR, aunque el remoto (`github.com/unawaretub86/Pos-vertical`) sí es GitHub.
- `qa-checklist.md` (156 líneas, 11 secciones) es 100% manual — ningún ítem indica si ya tiene cobertura automatizada.

**Contexto técnico:**
- No hay migración ni endpoint nuevo — es tooling y tests.
- **CI**: `.github/workflows/test.yml` — un solo job, corre en `push` y `pull_request` sobre cualquier rama: `go build ./...`, `go test ./...`, y (en el mismo job o uno separado) `cd frontend && npm ci && npm run build && npm test`. Sin matriz de OS/versión — un solo runner `ubuntu-latest` es suficiente para un repo solo (nota: `internal/database` usa `modernc.org/sqlite`, driver puro Go sin CGO, así que corre igual en Linux que en Windows — confirmar que no hay ninguna dependencia con `mattn/go-sqlite3` con build tags que solo compilen en Windows antes de asumir esto).
- **Cobertura backend a agregar/confirmar** (que no exista ya, ver estado verificado de cada uno antes de escribir):
  - `internal/service/sale_test.go` o `sale_mixed_test.go`: al menos un caso con `tax_rate > 0` Y descuento global simultáneos (combinación no probada hoy).
  - `internal/repository`: un test de `Balance`/`TotalPortfolioBalance` con una venta que tenga IVA + fiado parcial (mixed + credit + tax a la vez) — el caso más complejo de todos y el que más fácil se rompe si alguien toca `sale_payments`.
  - `internal/service/register_test.go`: un caso de cierre con ventas mixtas + gastos (`cash_out`) + IVA combinados, una vez exista US-049 (si SPRINT-10 ya cerró, si no, dejar el TODO explícito).
- **Cobertura frontend a agregar**: `frontend/lib/saleCalc.ts` ya cubre subtotal/tax/total/payment-balance. Extraer de igual forma (si no existe ya un módulo puro) el cálculo de cambio (`change = cashReceived - total`) y el chequeo de tope de crédito (`customerBalance + cartTotal > creditLimit`) que hoy viven inline en `pos/page.tsx` sin test.
- **Checklist**: en `qa-checklist.md`, agregar una marca `🤖` al lado de cada ítem de las secciones 3 (Ventas), 4 (Clientes/fiado) y 6 (Reportes) que ya tenga test automatizado equivalente, con el nombre del test entre paréntesis. No mover ítems de sección ni reescribir el archivo — es una anotación, no una reestructuración.

**Regla de negocio:** N/A (tooling).

**Scope — qué NO entra:**
- Tests end-to-end con navegador real (Playwright/Cypress) — no hay entorno con acceso a red para descargar binarios de navegador en el ambiente de desarrollo actual (confirmado en la sesión que originó este sprint). Si eso cambia, es un US aparte.
- Cobertura del 100% del código — el objetivo es cerrar los caminos de dinero que hoy no tienen ningún test, no perseguir un número de cobertura.
- Un pipeline de deploy/release — CI aquí es solo "que no rompa", no build-and-publish de `pos.exe`.

**Acceptance Criteria:**

**CI:**
- [x] Un push o PR a cualquier rama dispara el workflow y se ve en la pestaña Actions de GitHub. (`.github/workflows/test.yml`, `on: push, pull_request` sin filtro de rama — pendiente de confirmación visual en Actions tras el primer push, no verificable localmente.)
- [ ] El workflow falla (rojo) si `go test ./...` falla — probarlo rompiendo un test a propósito en una rama de prueba y confirmando que el check se pone en rojo. **(requiere push a GitHub, no ejecutable localmente — pendiente para Andrey)**
- [ ] El workflow falla si `npm test` (vitest) falla. **(idem, pendiente de verificación en GitHub)**
- [ ] El workflow falla si `npm run build` (Next.js) falla. **(idem, pendiente de verificación en GitHub)**
- [x] Tiempo total del workflow razonable para uso diario (referencia: <5 min) — backend `go test ./...` <1s, frontend `npm test` ~0.3s + `npm run build` ~15s localmente; total esperado muy por debajo de 5 min.

**Cobertura backend:**
- [x] Existe un test de venta con `tax_rate > 0` + descuento global simultáneos, con el número exacto verificado a mano en el test (`TestSaleService_Create_TaxPlusGlobalDiscount`, `internal/service/sale_mixed_test.go`).
- [x] Existe un test de saldo de cartera (`Balance`) sobre una venta mixta que incluye una porción a crédito y tiene IVA (`TestSaleService_Create_MixedPayment_WithTax`, ya existente desde el fix del incidente — confirmado, no duplicado).

**Cobertura frontend:**
- [x] `frontend/lib/saleCalc.ts` cubre cambio (`calcChange`) y validación de tope de crédito (`exceedsCreditLimit`) con test — ambas funciones extraídas y wireadas en `frontend/app/pos/page.tsx` reemplazando la lógica inline.
- [x] `npm test` corre en menos de 10 segundos (~0.3s, 18 tests, sin jsdom).

**Checklist:**
- [x] `qa-checklist.md` tiene las secciones 3, 4 y 6 anotadas con 🤖 + nombre de test por ítem cubierto (parcial o total), resto marcado como manual.

**DoD:**
- [x] `.github/workflows/test.yml` creado — corriendo en GitHub Actions pendiente de confirmar tras push (no verificable en este entorno sin acceso a GitHub Actions).
- [x] `go build ./...` y `go test ./...` limpios.
- [x] `cd frontend && npm run build && npm test` limpios.
- [x] Test: venta con `tax_rate=0.19` + descuento global de $5.000 sobre subtotal $50.000 → total 54500 calculado a mano en el test.
- [x] Test: venta mixta ($30k cash + $20k transfer... en este caso $20k cash + $39.5k credit) sobre un producto con IVA → saldo de cartera del cliente es exactamente la porción `credit` ($39.500), no el total ($59.500).
- [x] `qa-checklist.md` actualizado.

---

## US-051 — Arqueo con conteo de denominaciones + ticket de cierre (F5+F6)

**Como** Laura (cajera),
**quiero** contar los billetes y monedas por denominación al cerrar caja en vez de escribir un solo número, y que salga una tirilla imprimible con el resumen del turno,
**para** cuadrar caja sin calculadora aparte y tener un comprobante firmable si algo no cuadra.

**Esfuerzo estimado:** 1.5 días (0.5 arqueo + 1 ticket, según estimado del doc fuente). **Impacto:** Medio-alto. Profesionaliza un flujo que ya existe pero es rudimentario (`getRegistersReport`), sin tocar schema.

**Estado verificado:** el cierre de caja hoy (`internal/handler/register.go`, `closeRegister`) recibe un único `actual_close` numérico desde el frontend (confirmado contra el payload usado en verificación manual de SPRINT-10: `{"actual_close":50000}`). No hay grid de denominaciones en ningún componente de `frontend/app/register`. No existe ticket de cierre — `frontend/components/TicketPreview.tsx` solo tiene la plantilla de venta (`TicketData` con `items`, `payments`, etc.), sin una variante de resumen de caja.

**Contexto técnico:**
- **Sin migración nueva.** El grid de denominaciones es una calculadora puramente frontend: `frontend/app/register/page.tsx` (o el componente de modal de cierre que ya exista ahí) gana una tabla de denominaciones COP (`$100.000, $50.000, $20.000, $10.000, $5.000, $2.000, $1.000, $500, $200, $100, $50`) con un input de cantidad por fila; la suma se calcula en el cliente y llena el campo `actual_close` que ya se envía a `POST /registers/{id}/close` — el backend no cambia.
- El desglose de denominaciones **no se persiste** (el doc fuente lo describe como calculadora, no como auditoría) — si Andrey quiere guardarlo para auditoría futura, es un US aparte con su propia tabla; no inflar el scope de este.
- **Ticket de cierre**: nuevo componente `frontend/components/CloseTicketPreview.tsx`, mismo patrón visual que `TicketPreview.tsx` (franja de 80mm, `formatCOP`, secciones con `border-top: dashed`). Los datos ya existen en la respuesta de `POST /registers/{id}/close` y `GET /registers/{id}` (confirmado: `register`, `cash_sales`, `cash_in`, `cash_out`, `expected`, `movements` — ver verificación manual de SPRINT-10). El ticket muestra: fecha/hora apertura y cierre, cajero, ventas por método de pago (reusar `GET /reports/sales?from=&to=` filtrado a la ventana de la caja, ya corregido en SPRINT-10 para leer de `sale_payments`), total entradas/salidas, `expected` vs `actual_close`, diferencia.
- No hay endpoint nuevo — todo el dato ya existe; es composición de UI.

**Regla de negocio:**
```
suma_denominaciones = Σ (cantidad_billete_N × valor_N)
actual_close = suma_denominaciones   (reemplaza el input manual de un solo número, mismo campo)
diferencia = actual_close - expected_close   (cálculo ya existente en el backend, sin cambios)
```

**Scope — qué NO entra:**
- Persistir el desglose de denominaciones en base de datos (ver Contexto técnico).
- Impresión física ESC/POS — el ticket es HTML/preview igual que el de venta hoy, imprimir por diálogo del navegador. Cuando llegue soporte ESC/POS real (mencionado como pendiente en el doc fuente), sale gratis para ambos tickets a la vez.

**Acceptance Criteria:**

**Arqueo:**
- [ ] El modal de cierre de caja muestra un input de cantidad por cada denominación COP estándar (billetes y monedas vigentes).
- [ ] La suma se actualiza en tiempo real mientras se escribe.
- [ ] El total sumado reemplaza (o pre-llena, editable) el campo que hoy se manda como `actual_close` — el cajero puede seguir corrigiendo a mano si prefiere.
- [ ] Responsive: usable en 320px (columna, no tabla ancha con 11 denominaciones en fila).

**Ticket de cierre:**
- [ ] Al cerrar caja exitosamente, se muestra un preview de ticket con: negocio, cajero, apertura/cierre, ventas por método (cash/transfer/card/credit/mixed desglosado), entradas, salidas, esperado, real, diferencia.
- [ ] Botón de imprimir (mismo mecanismo que el ticket de venta).
- [ ] Diferencia negativa se muestra en rojo, positiva o cero en el color normal (mismo criterio visual que "Ganancia neta" de US-049).
- [ ] Responsive en 320px.

**DoD:**
- [ ] Sin migración, sin endpoint nuevo — confirmar con `git diff` que `internal/` no cambió si el AC se cumple solo con frontend.
- [ ] Test (vitest, `frontend/lib/`): función pura de suma de denominaciones — 3 casos incluido uno con cantidad 0 en varias filas.
- [ ] Test manual: cerrar caja con denominaciones que suman $123.450 (número no redondo) → `actual_close` enviado es exactamente $123.450, sin errores de redondeo flotante.
- [ ] `npm run build` limpio.

---

## US-052 — Proveedores y compras (G1)

**Como** Don Carlos (dueño),
**quiero** registrar la factura de compra a un proveedor (qué compré, a quién, a qué costo) y que eso actualice el costo y el stock del producto automáticamente,
**para** que la ganancia que me muestra el sistema use el costo real de la última compra, no un número que metí a mano y quedó viejo.

**Esfuerzo estimado:** 3 días. **Impacto:** Alto — desbloquea US-044 (sugerencia de pedido, SPRINT-09) que hoy no tiene a quién sugerirle un pedido, y corrige márgenes que dependen de `products.cost`.

**Estado verificado:** `stock_entries` (`internal/database/migrate.go:253-259`) solo tiene `product_id, quantity, note, created_at` — sin costo, sin proveedor. No existe tabla `suppliers` ni `purchases` en ninguna migración (`grep -rn "supplier\|proveedor\|purchase" internal/` no encuentra ninguna, solo una coincidencia irrelevante: la categoría `'proveedor'` del enum de `expenses.category` de SPRINT-10, que es un gasto operativo, no una compra de mercancía). No hay endpoint `/suppliers` ni `/purchases` en `internal/handler/handler.go`. No hay página `frontend/app/suppliers` ni `frontend/app/purchases`.

**Contexto técnico:**
- **Migración 13** (nueva):
  ```sql
  CREATE TABLE IF NOT EXISTS suppliers (
      id         INTEGER PRIMARY KEY AUTOINCREMENT,
      name       TEXT NOT NULL,
      phone      TEXT,
      notes      TEXT,
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );

  CREATE TABLE IF NOT EXISTS purchases (
      id          INTEGER PRIMARY KEY AUTOINCREMENT,
      supplier_id INTEGER NOT NULL REFERENCES suppliers(id),
      user_id     INTEGER NOT NULL REFERENCES users(id),
      reference   TEXT,
      total       REAL NOT NULL,
      created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
  );

  CREATE TABLE IF NOT EXISTS purchase_items (
      id          INTEGER PRIMARY KEY AUTOINCREMENT,
      purchase_id INTEGER NOT NULL REFERENCES purchases(id),
      product_id  INTEGER NOT NULL REFERENCES products(id),
      quantity    REAL NOT NULL CHECK (quantity > 0),
      unit_cost   REAL NOT NULL CHECK (unit_cost >= 0),
      subtotal    REAL NOT NULL
  );
  CREATE INDEX IF NOT EXISTS idx_purchases_supplier ON purchases(supplier_id);
  CREATE INDEX IF NOT EXISTS idx_purchase_items_purchase ON purchase_items(purchase_id);
  CREATE INDEX IF NOT EXISTS idx_purchase_items_product ON purchase_items(product_id);
  ```
- `internal/repository/purchase.go` (nuevo) — `Create()` en una transacción: inserta `purchases` + `purchase_items`, por cada item hace `UPDATE products SET stock = stock + ?, cost = ? WHERE id = ?` (ver Regla de negocio para qué costo exactamente), y agrega un `stock_entries` con `note` referenciando la compra (reusa la tabla existente para que el historial de movimientos de stock siga siendo una sola fuente).
- `internal/service/purchase.go` (nuevo) — valida `quantity > 0`, `unit_cost >= 0`, `supplier_id` existe.
- Nuevo endpoint `POST /api/purchases` (grupo autenticado, igual patrón que `/api/sales`) y `GET /api/purchases` (filtros `supplier_id`, `from`, `to`).
- Nuevo `SupplierRepository` + `POST /api/suppliers`, `GET /api/suppliers`, `PUT /api/suppliers/{id}`, `DELETE /api/suppliers/{id}` — mismo patrón CRUD que `internal/repository/customer.go` sin el saldo (los proveedores no manejan cartera en este sprint).
- Nueva página `frontend/app/suppliers/page.tsx` (CRUD simple, mismo patrón que `frontend/app/customers/page.tsx`) y `frontend/app/purchases/page.tsx` (o una sección dentro de inventario) para registrar una compra: proveedor, buscar/agregar productos con cantidad y costo unitario, guardar.

**Regla de negocio:**
```
Al registrar una compra, por cada item:
  products.stock += purchase_item.quantity
  products.cost   = purchase_item.unit_cost      -- último costo (simple), NO promedio ponderado

purchase_item.subtotal = purchase_item.quantity × purchase_item.unit_cost
purchase.total          = Σ purchase_item.subtotal
```
> Nota de scope: el doc fuente menciona "último costo o promedio ponderado" como alternativas. Este sprint implementa **último costo** (más simple, sin necesidad de rastrear historial ponderado). Promedio ponderado queda como mejora futura si Don Carlos lo pide — anotarlo en Notas técnicas, no bloquear este US por eso.

**Scope — qué NO entra:**
- Cartera/cuentas por pagar a proveedores (plata que el negocio le debe al proveedor) — eso es manejo de flujo de caja del lado de compras, simétrico a la cartera de clientes pero con su propio alcance; v2.
- Órdenes de compra (pedir antes de recibir) — este US es solo registrar lo que ya llegó. "Sugerencia de pedido" (US-044) es el siguiente paso una vez este exista.
- Costo promedio ponderado (ver nota arriba).

**Acceptance Criteria:**

**Proveedores:**
- [ ] CRUD de proveedores (nombre, teléfono, notas) desde `frontend/app/suppliers`.
- [ ] No se puede eliminar un proveedor con compras registradas (mismo criterio que categorías con productos asociados).

**Compras:**
- [ ] Formulario de compra: seleccionar proveedor, agregar productos con cantidad y costo unitario, ver total en tiempo real.
- [ ] Al guardar: `products.stock` sube exactamente la cantidad comprada por producto.
- [ ] Al guardar: `products.cost` queda igual al `unit_cost` de esa compra (último costo).
- [ ] La compra aparece en `stock_entries` con referencia a la compra en la nota.
- [ ] Listado de compras con filtro por proveedor y rango de fechas.
- [ ] Sin resultados con los filtros aplicados → mensaje vacío, no tabla en blanco ni error.
- [ ] Responsive: formulario usable en 320px.

**DoD:**
- [ ] Migración 13: `suppliers`, `purchases`, `purchase_items` — SQL exacto en Contexto técnico.
- [ ] `POST /api/purchases` → `{supplier_id, reference, items: [{product_id, quantity, unit_cost}]}` → crea compra, actualiza stock y costo.
- [ ] `GET /api/purchases?supplier_id=&from=&to=`.
- [ ] `POST /api/suppliers`, `GET /api/suppliers`, `PUT /api/suppliers/{id}`, `DELETE /api/suppliers/{id}`.
- [ ] Test: compra de 10 unidades a costo $8.000 de un producto que tenía stock 5 y costo $7.500 → stock final 15, costo final $8.000 (no promedio).
- [ ] Test: compra con `quantity=0` o `unit_cost` negativo → rechazada con 400.
- [ ] Test: eliminar proveedor con compras existentes → rechazado.
- [ ] Build limpio (`go build`, `npm run build`).

---

## Criterio de éxito del sprint

- Un push a GitHub corre los tests de dinero automáticamente y se pone en rojo si alguien rompe un cálculo — el bug de IVA de SPRINT-10 no habría llegado a Andrey probando manualmente, lo habría atrapado el CI en segundos.
- Laura cuenta los billetes del cajón, el sistema suma solo, y al cerrar imprime una tirilla que puede firmar y archivar.
- Don Carlos registra la factura de su proveedor de lácteos, el stock y el costo de sus productos se actualizan solos, y la ganancia que ve en el dashboard ya no usa un costo desactualizado.

---

## Notas técnicas para el dev

- **US-050 primero, sin excepción** — si por presión de tiempo hay que recortar el sprint, F5+F6 y G1 pueden esperar una semana más; la deuda de QA no debería crecer otra vez antes de tocar schema nuevo (migración 13 de US-052).
- **US-051 y US-052 no comparten nada técnicamente** (a diferencia de US-046/US-049 en SPRINT-10) — se pueden hacer en cualquier orden entre sí si algo bloquea a una; el orden sugerido es solo por riesgo (schema al final).
- Costo promedio ponderado (mencionado en el doc fuente para G1) queda fuera de este sprint — ver nota de scope en US-052. Si se necesita, es un US propio que migra `products.cost` de "último costo" a un cálculo derivado.
- El desglose de denominaciones de US-051 es intencionalmente efímero (no persiste) — si en el futuro se pide para auditoría, es un US nuevo con su propia tabla (`register_denominations`), no una extensión de este.
- Antes de escribir el workflow de CI, confirmar que ningún paquete Go del proyecto depende de `mattn/go-sqlite3` (CGO) en vez de `modernc.org/sqlite` (puro Go) — si lo hace, el runner de GitHub Actions necesita un paso extra de toolchain C que no está contemplado en el estimado de 1 día de US-050.

---

## Roadmap — qué queda del doc fuente

| Sprint | Features (ID fuente) | Por qué ahí |
|--------|----------------------|-------------|
| SPRINT-12 | I2 (resumen WhatsApp al dueño) + K1 (acceso LAN desde celular) | Combo de 1 día documentado en el bloque K del doc fuente — cubre "ver ventas en celular" sin romper el posicionamiento offline. |
| SPRINT-13 | H2 (apartados/layaway), G2 (kardex + ajustes de inventario) | Extienden el diferenciador de fiado y cumplen el "requisito 2026" de kardex, pero el propio doc fuente los marca como pedidos del mes 2, no del día 1. |
| Sin fecha | F7+I4 (PIN rápido + permisos granulares), G3–G7 (vencimientos, combos, etiquetas, historial de precio, foto), H3–H5 (domicilios, impuesto bolsas, cotizaciones), E1 (sync multi-caja), E2 (asistente IA local), scoring de cliente fiado | Necesarios cuando el negocio crece a >1 empleado o multi-caja, o son features de nicho (farmacia/vencimientos, ferretería/combos). No hay señal de que bloqueen la venta al primer cliente. |
| Con partner externo (no ejecutable en solitario) | H1 (recargas/pago de servicios), capital embebido | Requieren alianza con fintech regulada o agregador (Movilred/Puntored/SuRed) — fuera del alcance de "ejecutable en solitario". |
