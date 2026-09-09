# SPRINT-06 — Facturación Electrónica MVP

**Objetivo:** Implementar `internal/factelect` para emitir documentos equivalentes DIAN vía Factus.
El paquete es autónomo y desacoplable — cualquier vertical (taller, etc.) puede importarlo e implementar `SaleAdapter`.

**Fecha inicio:** 2026-05-05

**Referencia de arquitectura:** `project-docs/fix-gaps.md`

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-026 | Migration v8 + modelos nuevos | ✅ Completo | 2026-05-05 |
| US-027 | Repository layer (eledoc + errors) | ✅ Completo | 2026-05-05 |
| US-028 | Package factelect core (types, mapper, adapter, config) | ✅ Completo | 2026-05-05 |
| US-029 | Factus HTTP client | ✅ Completo | 2026-05-05 |
| US-030 | FacElectService + Poller | ✅ Completo | 2026-05-05 |
| US-031 | PosSaleAdapter | ✅ Completo | 2026-05-05 |
| US-032 | Handler wiring (sale.go, fiscal.go, handler.go) | ✅ Completo | 2026-05-05 |

---

## Criterio de éxito

`go build ./...` sin errores. `fiscal_config.enabled = false` → sistema opera igual que hoy.
`fiscal_config.enabled = true` → `POST /sales` dispara emisión async, poller actualiza estado.
