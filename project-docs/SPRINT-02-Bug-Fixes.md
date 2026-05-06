# SPRINT-02 — Active Bug Fixes & Schema Migration

**Objetivo:** Cerrar todos los bugs activos documentados en el PRD antes del primer despliegue en cliente real.

**Fecha inicio:** 2026-04-09

---

## Prioridad 1 — Bugs financieros críticos

### US-010: Fix cash_in not included in expected close calculation
- [x] Corregir cálculo de cierre de caja para incluir movimientos `cash_in`
- **Bug:** `register.go:99` calcula `expected = opening + cashSales - cashOut` sin `+ cashIn`. Un depósito manual genera sobrante ficticio al cerrar.
- **Acceptance Criteria:**
  - `expected_close = opening_amount + cash_sales + cash_in - cash_out`
  - Test unitario: caja con depósito manual → diferencia = 0 si conteo coincide
  - Test unitario: caja sin movimientos extra → comportamiento previo intacto
- **Ref:** PRD §4 Risks — Ralph Review B2

### US-011: Backend tax recalculation (IVA)
- [x] El backend debe recalcular `sale.Tax` desde los ítems; `sale_items.tax_rate` debe guardarse al crear la venta
- **Bug:** `sale.Tax` se acepta del frontend sin validación. `sale_items.tax_rate` (migración 3) existe en schema pero nunca se escribe — siempre queda DEFAULT 0.
- **Acceptance Criteria:**
  - Al crear venta, el backend calcula `tax = SUM(item.quantity * item.unit_price * product.tax_rate / 100)` por cada ítem
  - Si `sale.Tax` del frontend difiere del calculado en más de $1 COP → venta rechazada (400)
  - `sale_items.tax_rate` se inserta con el valor de `products.tax_rate` vigente al momento de la venta
  - `sale.Total` recalculado como `subtotal - discount + tax`
  - Tests: venta con IVA 19%, venta con IVA 0%, venta rechazada por tax manipulado
- **Ref:** PRD §4 Risks — Ralph Review B3

---

## Prioridad 2 — UX crítico

### US-012: Customer selector on credit sale (POS)
- [x] Agregar selector de cliente en el flujo de pago cuando se elige "Crédito"
- **Bug:** El POS no muestra selector de cliente al elegir método de pago "Crédito". Sin `customer_id`, la venta a crédito es rechazada por el backend — funcionalidad inaccesible desde la UI.
- **Acceptance Criteria:**
  - Al seleccionar "Crédito" en el modal de pago, aparece un campo de búsqueda de cliente
  - Búsqueda en tiempo real por nombre o teléfono
  - Cliente seleccionado se muestra con nombre y saldo actual
  - El botón "Confirmar venta" queda deshabilitado si no hay cliente seleccionado
  - El `customer_id` se envía al backend al crear la venta
  - Mobile responsive (320px)
- **Ref:** PRD §2 — Ralph Review B5

---

## Prioridad 3 — Integridad de reportes

### US-013: Exclude voided sales from reports and totals
- [x] Filtrar ventas `voided` de todos los reportes y listados por defecto
- **Bug:** `SaleFilter` no filtra por status — ventas anuladas aparecen en totales de reportes, dashboard y listado de ventas.
- **Acceptance Criteria:**
  - `GET /api/sales` por defecto retorna solo `status = 'completed'`
  - `GET /api/reports/sales` excluye ventas `voided` de todos los totales
  - `GET /api/reports/top-products` excluye ítems de ventas `voided`
  - Listado de ventas en frontend muestra badge "ANULADA" para ventas voided (ya existente) pero no las cuenta en totales
  - Tests para cada endpoint verificando exclusión de voided
- **Ref:** PRD §4 Risks

---

## Prioridad 4 — Infraestructura

### US-014: Versioned migration system
- [x] Refactorizar `database/migrate.go` para usar migraciones numeradas con tabla `schema_migrations`
- **Bug:** El sistema actual usa `CREATE TABLE IF NOT EXISTS` sin versioning — no puede aplicar cambios de schema en campo (clientes con `data.db` existente).
- **Acceptance Criteria:**
  - Tabla `schema_migrations (version INTEGER PRIMARY KEY, applied_at DATETIME)` creada en primera ejecución
  - Migración 1: schema inicial completo (estado actual)
  - Migración 2: `voided_by`, `voided_at` en `sales` + `CHECK (stock >= 0)` en `products`
  - Migración 3: `tax_rate` en `sale_items`
  - Migración 4: `barcode` en `products`
  - Cada migración corre en transacción — si falla, rollback y el binario no arranca
  - Binario hace backup automático de `data.db` antes de correr migraciones pendientes
  - Tests: DB vacía aplica todas las migraciones; DB en versión 2 aplica solo 3 y 4
- **Ref:** PRD §3.5 Migraciones

---

## Registro de progreso

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-010 | Fix cash_in in expected close | ✅ Completa | 2026-04-09 |
| US-011 | Backend IVA recalculation | ✅ Completa | 2026-04-09 |
| US-012 | Customer selector on credit POS | ✅ Completa | 2026-04-09 |
| US-013 | Exclude voided from reports | ✅ Completa | 2026-04-09 |
| US-014 | Versioned migration system | ✅ Completa | 2026-04-09 |
