# Backlog & Development Workflow

## Roles
- **Andrey** — Product Owner
- **Claude** — Tech Lead & Developer

## Definition of Done (DoD)
1. Code compiles without errors (`go build ./...`, `pnpm build` en `frontend/`)
2. Existing tests pass (`go test ./...`, `pnpm test`)
3. New code has tests where applicable
4. Code follows the Design System: tokens vigentes en `frontend/app/globals.css` (`var(--canvas)`, `var(--surface-2)`…) y reglas de frontend de `CLAUDE.md`
5. Mobile responsive (320px, 768px, 1024px)
6. No console errors
7. Sprint Document updated (task checked off)
8. **Si cambió una regla de negocio, `project-docs/spec/` se actualiza en el mismo commit.** Sin esto el spec envejece y vuelve a pasar lo de `PLAN.md`.

## Design System Reference
Tokens vigentes: `frontend/app/globals.css` (fuente de verdad). Specs originales de componentes y tipografía: `archive/2026-05-plan-mvp.md` → Fase 0.0 (histórico; si difiere de `globals.css`, manda `globals.css`).

## Dónde está cada cosa

| Pregunta | Documento |
|---|---|
| Stack, convenciones de código, trampas conocidas | [`../CLAUDE.md`](../CLAUDE.md) |
| Qué manda sobre qué en `project-docs/` | [README.md](README.md) |
| Reglas de negocio vigentes (fiado, caja, rentabilidad) | [spec/](spec/) |
| **Qué se construye después** | [roadmap.md](roadmap.md) — única fuente |

**Regla de verificación:** ninguna afirmación sobre el código entra en un plan sin `archivo:línea`. Los docs de investigación envejecen; el código no miente. Ver `CLAUDE.md` → Regla 1.

## Validación con el usuario (opcional por sprint)

Cuando un sprint introduce una interacción nueva —no una pantalla más de algo que ya existe— vale la pena una sección `## Validaciones Previas con Usuario` en el documento de sprint: qué mostrarle a Don Carlos o Laura, y qué pregunta concreta hacerle, antes de construir.

Aplica a formas de interactuar que el usuario no ha visto antes. No a features incrementales.

## Iteration Loop (9 Steps)
1. **Select** — Pick first unchecked US from Sprint Document
2. **Plan** — Output plan, wait for approval if needed
3. **Code** — Implement with TDD
4. **Self-Review** — Tests, lint, mobile, design system
5. **Notify** — Present to Andrey with completion template
6. **Wait** — For feedback
7. **Iterate** — If rejected, fix and re-present
8. **Finalize** — Mark task `[x]` in Sprint Document
9. **Next** — Ask "Ready for next task?"

## Progress

| Sprint | Focus | Status |
|--------|-------|--------|
| SPRINT-01 | Production Polish & Gaps | ✅ Completo |
| SPRINT-02 | Active Bug Fixes & Schema Migration | ✅ Completo |
| SPRINT-03 | v1.1 Polish: Barcode, CSV, Auto Backup | ✅ Completo |
| SPRINT-04 | Prod Ready: Binary Integration & Polish | ✅ Completo |
| SPRINT-05 | Mobile Responsive: Demo-Ready en Celular | ⬜ Pendiente |
| SPRINT-06 | Facturación Electrónica (Factus) | ✅ Completo |
| SPRINT-07 | Factus Sandbox + Diferenciadores | ⚠️ Deuda: US-034 pendiente |
| SPRINT-08 | Rentabilidad Real y Cartera Envejecida | ✅ Completo |
| SPRINT-09 | Historial de Cliente, Pedidos y Catálogo | ⬜ Pendiente |
| SPRINT-10 | Cobro Mixto, Auditoría y Gastos | ✅ Completo |
| SPRINT-11 | QA de Cálculos, Proveedores/Compras y Cierre de Caja | ✅ Completo |
| SPRINT-12 | Resumen WhatsApp al Dueño y Acceso LAN desde Celular | ⬜ Pendiente |
| SPRINT-13 | Mesas, Pedidos y Pantalla de Barra (módulo restaurante) | ⬜ Pendiente |
| SPRINT-14 | Asistente Local de Consultas (motor de intenciones offline) | 🔄 En progreso: US-059, US-060, US-061, US-063, US-064 y US-065 completas, US-062 pendiente |
