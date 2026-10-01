# CLAUDE.md — Pos-vertical

Local, offline-first POS shipped as a single executable, for shops and distributors in Colombia. The differentiator is **fiado** (credit extended to customers), which competitors handle poorly. No cloud, no monthly fee, no internet.

**Andrey** is Product Owner and decides scope and priority. Claude is Tech Lead and developer.

---

## Rule 0 — Everything is written in English. NEVER in Spanish.

All planning and documentation artifacts are written in **English**, always, with no exceptions:

- `project-docs/` in full: `roadmap.md`, `workflow.md`, `README.md`, `spec/`, `sprints/`, `gaps.md`, design docs, ADRs.
- Per-US pipeline artifacts: `changes/<US>/contract.md`, `qa.md`, test plans, acceptance criteria, gate questions and decisions.
- Agent and command definitions (`.claude/agents/`, `.claude/commands/`), this `CLAUDE.md`, commit messages, PR descriptions, code comments, identifiers.
- Chat replies from Claude about planning, design or status, and any new document, table row or note anyone adds.

Do not write a "Spanish version", a bilingual doc, or a Spanish aside. If you touch an existing file that still contains Spanish prose, translate the part you touch in the same commit.

**The only Spanish allowed** is *product content*, because the product is used in Colombia:
- UI strings shown to end users (labels, buttons, toasts, error messages, ticket text, WhatsApp/email message bodies).
- Colombian domain terms that have no clean English equivalent (`fiado`, `tendero`, `abono`, `cierre de caja`). Introduce each with an English gloss the first time it appears in a doc; the canonical glossary is `project-docs/spec/domain/glosario.md`.
- Verbatim quotes of what a user types or says (e.g. assistant test phrases like `"cuánto vendí ayer"`), because they are test data.

Dated historical documents (`project-docs/archive/`, `project-docs/research/`, and sprints already closed before 2026-10-01) are frozen records and are not retranslated; they stay as written.

## Rule 1 — Verify before asserting

**No claim about the code enters a plan, a doc or an answer without a `file:line` that backs it.**

Docs in `project-docs/` age faster than code. If a doc says "X does not exist", `grep` before repeating it. If it says "Y is missing", verify it is still missing.

This is not ceremony: while planning SPRINT-14 this rule caught four false claims inherited from old docs, among them "there are no backend tests" (there are 13 `_test.go` files).

Prefer *"I verified that…"* over *"according to the doc…"*.

## Rule 2 — If behavior changes, the spec changes

`project-docs/spec/` is the only folder that must always be true. A commit that changes a business rule updates the spec in the same commit. Without this, `spec/` becomes another historical doc within six months.

## Rule 3 — One roadmap

`project-docs/roadmap.md` is the only place where upcoming work is recorded. Sprint documents **do not reserve future sprint numbers** and do not carry their own roadmap table. (They used to, which led SPRINT-13 and SPRINT-14 to both claim SPRINT-15.)

---

## Stack

**Backend** — Go 1.25, module `github.com/andreyo/pos`
- Router: `chi/v5` · Auth: `golang-jwt/v5` · DB: `modernc.org/sqlite`
- **`modernc.org/sqlite` is pure-Go SQLite, no CGO.** That is what lets us build an `.exe` without a C toolchain. Do not introduce CGO dependencies without an explicit PO decision.
- Binaries: `cmd/pos` (the product) and `cmd/reset-password` (CLI utility)

**Frontend** — Next.js 16.1.6 (App Router), React 19.2.3, TypeScript
- Package manager: **pnpm**, not npm
- Tests: vitest · Styles: Tailwind v4 + `@base-ui/react` + shadcn

## Commands

```bash
go build ./...          # build backend
go test ./...           # backend tests

cd frontend
pnpm build              # build frontend
pnpm test               # vitest
pnpm lint               # eslint
```

CI (`.github/workflows/test.yml`) runs `go build`, `go test`, `pnpm build` and `pnpm test` on every push. It **does not run `pnpm lint`**, which currently fails with 41 pre-existing problems in `frontend/` (roadmap R-35). Still run lint locally and do not add new problems.

## Architecture

Three layers, in this dependency order:

```
internal/repository/   SQL. Public interface + private struct + NewXRepository(db *sql.DB)
internal/service/      Business rules. Receives repositories through interfaces.
internal/handler/      HTTP. No SQL, no business logic.
```

- Everything is wired in `handler.New` (`internal/handler/handler.go:38`).
- Routes are registered in `Routes()` of the same file. Groups using `appmw.RequireAdmin` are admin-only — dashboard, reports and anything that exposes sales, profit or receivables live there.
- Migrations: numbered slice in `internal/database/migrate.go`. **Never edit an already-applied migration**; always add the next one.
- Errors wrapped with `%w`, never swallowed.
- SQL with `?` parameters, never concatenated.

**Frontend:**
- `frontend/app/layout.tsx` is the **root** layout: it also wraps `/login`, `/setup` and `/onboarding`. Anything that requires a session does NOT go there.
- `frontend/components/AppLayout.tsx` is the **authenticated** container. Anything only logged-in users see goes there.
- Currency: `formatCOP` from `frontend/lib/format.ts`. HTTP: `api` from `frontend/lib/api.ts`.
- Colors via CSS variables (`var(--canvas)`, `var(--surface-2)`, `var(--border)`, `var(--text-primary)`). No hardcoded colors, or the dark theme breaks.
- Mobile-first: everything works at 320px.

## Tests

Established pattern: `setupTestDB(t)` with `database.New(":memory:")` and seeding through literal `INSERT`s — see `internal/handler/dashboard_test.go:14`. Follow it; do not invent another fixture.

Tests carry concrete numbers, not vague assertions: *"3 payments of $30k + $20k + $10k → total 60,000, remainder 0"*.

## Conventions

- **Language: English for everything except end-user product text** (see Rule 0). Code, variable names, comments, docs, specs, sprints, roadmap, commits: English. UI strings shown to end users: Spanish.
- Currency: Colombian pesos, no decimals, dot as thousands separator → `$1.234.567` (this is the UI display format; docs may write plain numbers like `1,234,567`).
- Date ranges: **always half-open `[from, to)`**. Never `BETWEEN`, which is inclusive at both ends and double-counts midnight sales.
- Time zone: server local time. The POS runs on the shop's machine.
- Deleting historically referenced entities: logical (`active = 0`), never a physical `DELETE`.

## Workflow

Sprints live in `project-docs/sprints/SPRINT-XX-*.md`. The iteration loop and the progress table are in `project-docs/workflow.md`: take the first unchecked US, implement it against its acceptance criteria, mark it `[x]` when done.

Before planning a feature, read `project-docs/README.md` to know which document rules over which. **Not everything in `project-docs/` is current** — several documents are dated research or MVP history, and are marked as such in their header.

## Known traps

- `project-docs/archive/2026-05-plan-mvp.md` (formerly `PLAN.md`) and `project-docs/archive/2026-04-17-arquitectura.md` (formerly `ARQUITECTURA.md`) describe the April–May 2026 code. They are history, not current state. Example: `2026-05-plan-mvp.md:1135` reports a `cash_in` bug in the register close that is **already fixed** (`internal/service/register.go:113`).
- Iterating a Go `map` is random by design. If order matters (rule evaluation, priorities), use an ordered slice or tests will fail intermittently.
- The documents in `project-docs/research/` (formerly `find-pos.md`, `finds-betters.md`, `better-pos.md`, `better-finds-pos.md`, `facElect-finds.md`) are dated market research, not a code inventory. Their "this does not exist" claims are verified before use.
