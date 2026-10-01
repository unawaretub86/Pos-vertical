# project-docs — Index

> **Updated:** 2026-10-01

**Read this before believing any document in this folder.** Not all of them are current: some specs must always be true, some research is dated months ago, and some is MVP history. They used to be all mixed together and written in the same present tense, and that produced plans with false claims about the code.

Each document opens with a status stamp that says which is which.

> **Language rule (`../CLAUDE.md` → Rule 0):** every document in this folder is written in **English**, never Spanish. Only end-user UI strings and verbatim user phrases used as test data may be Spanish. Dated historical documents (`archive/`, `research/`, sprints closed before 2026-10-01) are frozen records and keep their original language and file names.

---

## The four types

| Type | Rule | Trust for |
|---|---|---|
| **spec** | Must always be true. Changes in the same commit that changes the behavior. | Business rules, product decisions |
| **current** | Kept up to date. | What comes next, how we work |
| **sprint** | Unit of change. Frozen when closed. | What was built and with which criteria |
| **research** / **historical** | Was true on a date. **Never** a source of truth about the current code. | Context, ideas, why something was decided |

---

## What rules over what

| Question | Document |
|---|---|
| What is the product? What is it NOT? | [spec/product.md](spec/product.md) |
| Who do I write this US for? Who may see this? | [spec/personas.md](spec/personas.md) |
| How is a customer's balance computed? | [spec/domain/fiado.md](spec/domain/fiado.md) |
| How is the register reconciled? Which expense categories exist? | [spec/domain/caja.md](spec/domain/caja.md) |
| How are profit and margin computed? | [spec/domain/rentabilidad.md](spec/domain/rentabilidad.md) |
| How does invoicing fit with "offline"? | [spec/domain/facturacion-dian.md](spec/domain/facturacion-dian.md) |
| How do you say this in Colombian? | [spec/domain/glosario.md](spec/domain/glosario.md) |
| What is restaurant mode? How is a table's state derived? | [spec/domain/restaurante.md](spec/domain/restaurante.md) |
| How does the server bind and serve the frontend? | [spec/domain/servidor.md](spec/domain/servidor.md) |
| **What do we build next?** | [roadmap.md](roadmap.md) — single source |
| How is a sprint worked? | [workflow.md](workflow.md) |
| What is the stack, the code conventions and the language rule? | [`../CLAUDE.md`](../CLAUDE.md) |

## Sprints

`sprints/SPRINT-XX-*.md`, from 01 to 18. They are units of change: when closed they stay frozen as a record of what was built and with which acceptance criteria.

The status of each one is in the progress table of [workflow.md](workflow.md). Open debt is consolidated in [roadmap.md](roadmap.md) and [gaps.md](gaps.md).

> Sprints live in `project-docs/sprints/`. The `ralph-workflow` and `task-planner` skills look for them there (updated 2026-09-23).

## Market research

Dated, read-only, **never a source of truth about what exists in the code**. Their "this does not exist" claims are verified with `grep` before being used. These files are in Spanish (frozen, pre-Rule 0).

| Current file | Former name | Date | What it covers |
|---|---|---|---|
| [`2026-05-11-gaps-vs-competidores.md`](research/2026-05-11-gaps-vs-competidores.md) | `better-pos.md` | 2026-05-11 | First gaps detected against competitors |
| [`2026-05-11-features-pedidas-por-tendero.md`](research/2026-05-11-features-pedidas-por-tendero.md) | `finds-betters.md` | 2026-05-11 | Features the shopkeeper asks for |
| [`2026-05-verticales-y-rentabilidad.md`](research/2026-05-verticales-y-rentabilidad.md) | `find-pos.md` | ~2026-05 | Verticals, profitability, US-040 to US-048 |
| [`2026-09-08-investigacion-mercado.md`](research/2026-09-08-investigacion-mercado.md) | `better-finds-pos.md` | 2026-09-08 | **The most recent and complete.** Blocks A–K: worldwide and local competition, quick wins, offline LAN multi-register (E1), local assistant (E2) |
| [`2026-05-dian-proveedores.md`](research/2026-05-dian-proveedores.md) | `facElect-finds.md` | 2026-05 | DIAN and invoicing provider research |

The former names were similar and indistinguishable from each other. If you only read one of these documents, make it **[`2026-09-08-investigacion-mercado.md`](research/2026-09-08-investigacion-mercado.md)** (formerly `better-finds-pos.md`).

## Design and architecture

| Current file | Former name | Status |
|---|---|---|
| [`2026-09-20-mesas-pedidos-kds.md`](design/2026-09-20-mesas-pedidos-kds.md) | `DESIGN-Mesas-Pedidos-KDS.md` | Approved design of the restaurant module (2026-09-20), input to SPRINT-13 |
| [`tosdd-plan.md`](tosdd-plan.md) | — | **Proposal** (2026-09-23): agentic orchestration with SDD (4 agents, PO gates, US-059 pilot). The pipeline runs through `.claude/commands/us.md` |
| [`pipeline-bench.md`](pipeline-bench.md) | — | **Current.** Benchmark of each run of the `/us` pipeline: tokens, time, stops, defects found and adjustments applied |
| [`facturacion-electronica.md`](reference/facturacion-electronica.md) | `fix-gaps.md` | Electronic invoicing implementation plan. **The name was misleading**: it is not about generic "gaps". Implementation reference, May 2026 |
| [`2026-04-17-arquitectura.md`](archive/2026-04-17-arquitectura.md) | `ARQUITECTURA.md` | Snapshot of the code as of 2026-04-17. Historical |

## Historical

| Current file | Former name | Why it is not current |
|---|---|---|
| [`2026-05-plan-mvp.md`](archive/2026-05-plan-mvp.md) | `PLAN.md` | MVP-era checklist (1,298 lines). Describes April–May 2026 code. Concrete example: at `:1134` it reports a `cash_in` bug in the register close **that is already fixed** (`internal/service/register.go:113`) |
| [`2026-05-prd-mvp.md`](archive/2026-05-prd-mvp.md) | `PRD.md` | MVP PRD. The personas are still current and were extracted to `spec/personas.md`; the rest is history |
| [`2026-05-mvp-context.md`](archive/2026-05-mvp-context.md) | `mvp-context.md` | Initial vision. What is current was extracted to `spec/product.md` |

---

## How to add a document

1. **Is it a rule that must always be true?** → `spec/`, and it is updated with the code.
2. **Is it what comes next?** → do not create a document: it goes to `roadmap.md`.
3. **Is it a unit of work?** → `sprints/SPRINT-XX-Topic.md`.
4. **Is it research?** → dated name, `research` stamp, and it is not cited as the state of the code.

Every new document opens with its stamp, **in English**:

```markdown
> **Type:** spec · **Current:** yes · **Rules over:** <topic>
> **Updated:** YYYY-MM-DD
```
