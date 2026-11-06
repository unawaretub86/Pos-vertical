# Backlog & Development Workflow

> **Language rule:** every document in `project-docs/` is written in English, never Spanish (`../CLAUDE.md` → Rule 0). Only end-user UI strings and verbatim test phrases may be Spanish.

## Roles
- **Andrey** — Product Owner
- **Claude** — Tech Lead & Developer

## Definition of Done (DoD)
1. Code compiles without errors (`go build ./...`, `pnpm build` in `frontend/`)
2. Existing tests pass (`go test ./...`, `pnpm test`)
3. New code has tests where applicable
4. Code follows the Design System: current tokens in `frontend/app/globals.css` (`var(--canvas)`, `var(--surface-2)`…) and the frontend rules in `CLAUDE.md`
5. Mobile responsive (320px, 768px, 1024px)
6. No console errors
7. Sprint Document updated (task checked off)
8. **If a business rule changed, `project-docs/spec/` is updated in the same commit.** Without this the spec ages and the `PLAN.md` situation repeats.
9. **All docs written or touched are in English** (Rule 0 in `CLAUDE.md`).

## Design System Reference
Current tokens: `frontend/app/globals.css` (source of truth). Original component and typography specs: `archive/2026-05-plan-mvp.md` → Phase 0.0 (historical; if it differs from `globals.css`, `globals.css` wins).

## Where everything lives

| Question | Document |
|---|---|
| Stack, code conventions, known traps, language rule | [`../CLAUDE.md`](../CLAUDE.md) |
| What rules over what in `project-docs/` | [README.md](README.md) |
| Current business rules (fiado, register, profitability) | [spec/](spec/) |
| **What gets built next** | [roadmap.md](roadmap.md) — single source |

**Verification rule:** no claim about the code enters a plan without `file:line`. Research docs age; the code does not lie. See `CLAUDE.md` → Rule 1.

## User validation (optional per sprint)

When a sprint introduces a new interaction — not just one more screen of something that already exists — it is worth adding a `## Prior User Validation` section to the sprint document: what to show Don Carlos or Laura, and which concrete question to ask, before building.

It applies to ways of interacting that the user has not seen before. Not to incremental features.

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
| SPRINT-01 | Production Polish & Gaps | ✅ Complete |
| SPRINT-02 | Active Bug Fixes & Schema Migration | ✅ Complete |
| SPRINT-03 | v1.1 Polish: Barcode, CSV, Auto Backup | ✅ Complete |
| SPRINT-04 | Prod Ready: Binary Integration & Polish | ✅ Complete |
| SPRINT-05 | Mobile Responsive: Demo-Ready on Phone | ⬜ Pending |
| SPRINT-06 | Electronic Invoicing (Factus) | ✅ Complete |
| SPRINT-07 | Factus Sandbox + Differentiators | ✅ Complete (US-034 finished in SPRINT-17) |
| SPRINT-08 | Real Profitability and Aging Receivables | ✅ Complete |
| SPRINT-09 | Customer History, Orders and Catalog | ⬜ Pending |
| SPRINT-10 | Mixed Payment, Audit and Expenses | ✅ Complete |
| SPRINT-11 | Calculation QA, Suppliers/Purchases and Register Close | ✅ Complete |
| SPRINT-12 | WhatsApp Summary to Owner and LAN Access from Phone | ⬜ Pending |
| SPRINT-13 | Tables, Orders and Bar Screen (restaurant module) | ✅ Complete |
| SPRINT-14 | Local Query Assistant (offline intent engine) | ✅ Complete in code and tests (US-059 to US-065). **Pending: PO manual validation of US-062 (M-1..M-9: visual, theme, 320 px, keyboard on a real phone)** |
| SPRINT-15 | Assistant Continuation and UTC Debt (R-02, R-03, R-05, R-48) | ⬜ Pending |
| SPRINT-16 | Security and Production (JWT, Tx, Backup) | ✅ Complete |
| SPRINT-17 | Factus Sandbox, WhatsApp and Responsive | ✅ Complete except US-074 (responsive 320px): criteria unchecked, requires manual verification on a phone/emulator |
| SPRINT-18 | Production Stabilization | 🔄 In progress: US-075..US-080 done in code (manual PO checks pending, see `gaps.md`; US-080 needs real printers); US-081 pending |
| SPRINT-19 | Fiado Collection (cash-correct abonos, R-61, US-043, quick abono, aging chips) | 🔄 In progress: US-082 done in code (manual PO checks pending, see `gaps.md`); US-083, US-043, US-084, US-085 pending |
