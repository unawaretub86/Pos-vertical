# SPRINT-01 — Production Polish & Functional Gaps

**Objetivo:** Cerrar todos los pendientes funcionales del PLAN.md para que el sistema esté 100% listo para producción.

**Fecha inicio:** 2026-03-31

---

## Prioridad 1 — Bugs funcionales

### US-001: Fix user reactivation button
- [x] Agregar botón para reactivar usuario desactivado en la tabla de usuarios
- **Acceptance Criteria:**
  - Un usuario desactivado muestra un botón "Activar" visible en la tabla
  - Al hacer click, el usuario cambia a estado activo
  - Toast de confirmación
- **Ref:** PLAN.md línea 886

### US-002: Credit sale receipt prints "CRÉDITO"
- [x] El ticket/recibo de una venta a crédito debe mostrar claramente "CRÉDITO"
- **Acceptance Criteria:**
  - Ventas con `payment_method = 'credit'` muestran "CRÉDITO" destacado en el recibo
  - Visible tanto en print browser como en vista previa
- **Ref:** PLAN.md línea 555

---

## Prioridad 2 — Features POS core

### US-003: Discounts per item
- [x] Implementar descuento por ítem en el POS (% o monto fijo)
- **Acceptance Criteria:**
  - Campo de descuento por línea de ítem en el carrito
  - Soporta porcentaje o monto fijo
  - Total se recalcula en tiempo real
  - Backend valida descuentos
- **Ref:** PLAN.md línea 552

### US-004: Global discount on sale
- [x] Implementar descuento global configurable por admin
- **Acceptance Criteria:**
  - Campo de descuento global en la venta
  - Solo admin puede habilitar/deshabilitar desde configuración
  - Reflejado en total y en ticket
- **Ref:** PLAN.md línea 553

### US-005: Admin price override at POS
- [x] Permitir modificar precio en línea solo para admin
- **Acceptance Criteria:**
  - Campo editable de precio por ítem (solo si rol = admin)
  - Override se registra en log de auditoría
  - Cajero ve el precio pero no puede editarlo
- **Ref:** PLAN.md línea 554

---

## Prioridad 3 — UX improvements

### US-006: Register movements filter by session
- [x] Filtrar movimientos de caja por sesión actual
- **Acceptance Criteria:**
  - Por defecto muestra solo movimientos de la caja activa
  - Opción de ver histórico completo
- **Ref:** PLAN.md línea 624

### US-007: Voided sales filter ("Todos los estados")
- [x] Agregar botón/filtro para mostrar ventas anuladas
- **Acceptance Criteria:**
  - Por defecto ventas anuladas están ocultas
  - Botón "Todos los estados" muestra todas incluyendo anuladas
  - Ventas anuladas con badge visual diferenciado
- **Ref:** PLAN.md línea 671

### US-008: Settings improvements
- [x] Link a gestión de usuarios desde settings
- [x] Logo se actualiza dinámicamente en sidebar al cambiarlo
- **Ref:** PLAN.md líneas 961, 967

---

## Prioridad 4 — Testing

### US-009: Missing unit tests (3 cases)
- [x] Test: Venta con cantidad > stock disponible
- [x] Test: Eliminar categoría con productos asignados
- [x] Test: Abrir segunda caja con una ya abierta
- **Ref:** PLAN.md líneas 1028-1033

---

## Registro de progreso

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-001 | User reactivation bug | ✅ Completa | 2026-03-31 |
| US-002 | Credit receipt "CRÉDITO" | ✅ Completa | 2026-03-31 |
| US-003 | Discounts per item | ✅ Completa | 2026-03-31 |
| US-004 | Global discount | ✅ Completa | 2026-03-31 |
| US-005 | Admin price override | ✅ Completa | 2026-03-31 |
| US-006 | Register movements filter | ✅ Completa | 2026-03-31 |
| US-007 | Voided sales filter | ✅ Completa | 2026-04-09 |
| US-008 | Settings improvements | ✅ Completa | 2026-04-09 |
| US-009 | Missing unit tests | ✅ Completa | 2026-04-09 |
