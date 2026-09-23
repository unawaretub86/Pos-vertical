# SPRINT-03 — v1.1 Polish: Barcode, CSV Export & Auto Backup

**Objetivo:** Implementar las features de v1.1 definidas en el PRD: barcode scanner, CSV export de reportes y backup automático local.

**Fecha inicio:** 2026-04-09

---

## Prioridad 1 — Barcode

### US-015: Barcode field on products
- [x] Agregar campo `barcode` en producto (CRUD) y endpoint de lookup por barcode
- **Acceptance Criteria:**
  - Campo `barcode` opcional en formulario de creación/edición de producto
  - Único: no dos productos comparten el mismo barcode
  - Puede escribirse manualmente o con escáner USB mientras el campo tiene foco
  - `GET /api/products/barcode/:barcode` retorna el producto o 404
  - Schema: `barcode` ya existe en migración 4 (aplicada)
- **Ref:** PRD §2 Products, migración 4

### US-016: Barcode scanner at POS
- [x] Detectar entrada de escáner de barcode en el campo de búsqueda del POS
- **Acceptance Criteria:**
  - El escáner USB envía el barcode seguido de Enter (comportamiento estándar HID)
  - Si el input termina en Enter con texto presente → intentar lookup por barcode primero (`/api/products/barcode/:code`)
  - Si hay match de barcode → producto agregado al carrito inmediatamente, input limpiado
  - Si no hay match de barcode → el texto queda en el campo y corre búsqueda normal por nombre
  - No pasos extra para el cajero (Laura)
  - Mobile responsive (320px)
- **Ref:** PRD §2 Point of Sale

---

## Prioridad 2 — CSV Export

### US-017: CSV export for reports
- [x] Agregar botón de exportar CSV en la página de reportes
- **Acceptance Criteria:**
  - Botón "Exportar CSV" en `/reports` descarga el resumen de ventas filtrado
  - Columnas: fecha, total_revenue, total_cost, gross_margin, total_count
  - Respeta el rango de fechas y agrupación seleccionados
  - Encoding UTF-8 con BOM (para compatibilidad con Excel en Colombia)
  - Nombre de archivo: `reporte-ventas-YYYY-MM-DD.csv`
  - Generado en el frontend (no requiere nuevo endpoint)
- **Ref:** PRD §4 v1.1

---

## Prioridad 3 — Auto Backup

### US-018: Automatic local backup (scheduled)
- [x] El binario hace una copia automática de `data.db` diariamente
- **Acceptance Criteria:**
  - Al arrancar y cada 24h, el binario copia `data.db` → `data.db.backup-YYYYMMDD`
  - Solo se mantienen los últimos 7 backups (elimina el más antiguo si hay más de 7)
  - Si el directorio no es escribible, loguea el error y continúa sin crashear
  - No interfiere con el WAL ni con operaciones en curso (checkpoint antes de copiar)
- **Ref:** PRD §4 v1.1

---

## Registro de progreso

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-015 | Barcode field on products | ✅ Completa | 2026-04-09 |
| US-016 | Barcode scanner at POS | ✅ Completa | 2026-04-09 |
| US-017 | CSV export for reports | ✅ Completa | 2026-04-09 |
| US-018 | Auto backup scheduled | ✅ Completa | 2026-04-09 |
