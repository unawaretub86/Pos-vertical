# Contract — US-XXX: <name>

> **Sprint:** `project-docs/sprints/SPRINT-XX-*.md` · **Level:** M | L
> **Worktree:** <absolute path> · **Branch:** us/US-XXX · **Date:** YYYY-MM-DD
> **Language:** English only (CLAUDE.md Rule 0).

## 1. Goal

One or two sentences: what changes for the user (or for the dev, if it is a refactor) and why.

## 2. Verified state

What exists TODAY, with `file:line`. No "according to the doc".

| Claim | Evidence |
|---|---|
| The dashboard computes daily sales with inline SQL | `internal/handler/dashboard.go:NN` |

If the sprint claims something the code contradicts, record it here as a **DISCREPANCY** and raise it at GATE 1.

## 3. Acceptance criteria (EARS)

Format: `WHEN <trigger>, the system SHALL <response>` · `WHILE <state>, the system SHALL …` · `IF <unwanted condition>, THEN the system SHALL …`

- **AC-1** WHEN … , the system SHALL … .
- **AC-2** IF … , THEN the system SHALL … .

## 4. Required tests

Every AC has at least one test. Concrete numbers, never vague assertions (CLAUDE.md, Tests).

| ID | Covers | File | Case (input → expected output) |
|---|---|---|---|
| T-1 | AC-1 | `internal/service/x_test.go` | 3 payments of $30,000 + $20,000 + $10,000 → balance 60,000 |
| T-2 | AC-2 | `internal/handler/x_test.go` | one sale at 23:59:59 and another at 00:00:00 → `[from, to)` range counts only the first |

Fixture: `setupTestDB(t)` with `database.New(":memory:")` and seeding via `INSERT` (`internal/handler/dashboard_test.go:14`).

## 5. Files to touch

| File | Change | Who |
|---|---|---|
| `internal/repository/x.go` | new method `Y` | implementer |
| `internal/repository/x_test.go` | T-1 | test-author |

**Migration:** none | migration N (check the last number in `internal/database/migrate.go` before writing it).

## 6. Spec delta

Which rule in `project-docs/spec/` changes. If none changes, write **"No spec change"** and why.

- `spec/domain/<file>.md` § <section>: before «…» → after «…».

## 7. Out of scope

What looks related but does NOT go in. Whatever deserves doing goes to `project-docs/roadmap.md` at close.

## 8. Questions for the PO (GATE 1)

- [ ] …
