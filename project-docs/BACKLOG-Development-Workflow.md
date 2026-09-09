# Backlog & Development Workflow

## Roles
- **Andrey** — Product Owner
- **Claude** — Tech Lead & Developer

## Definition of Done (DoD)
1. Code compiles without errors (`go build`, `npm run build`)
2. Existing tests pass (`go test ./...`)
3. New code has tests where applicable
4. Code follows Design System defined in PLAN.md (Phase 0.0)
5. Mobile responsive (320px, 768px, 1024px)
6. No console errors
7. Sprint Document updated (task checked off)

## Design System Reference
See `PLAN.md` → Fase 0.0 for full token definitions, component specs, and typography rules.

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
| SPRINT-10 | Cobro Mixto, Auditoría y Gastos | ⬜ Pendiente |
