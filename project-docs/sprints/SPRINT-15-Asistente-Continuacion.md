# SPRINT-15 — Assistant Continuation and UTC Debt

**Goal:** Complete the assistant use case by adding inventory lookup and "quincena" (half-month pay period) periods, and fix the UTC/Local date bug that causes silent collapses in reports (detected in QA and in the roadmap).

**Start date:** (to be defined)

**References:** `project-docs/roadmap.md`, US-061 QA.

**Incoming technical debt:**
- US-034 (Factus), SPRINT-05 (Mobile), SPRINT-09, SPRINT-12 and SPRINT-13 (`dining_tables`) **remain open** and parked.
- R-06 (Make `to` half-open in `ExpenseRepository.List`) **remains open**.

---

## Tasks

| US | Name | Status | Date |
|----|------|--------|------|
| US-066 | UTC/Local bugfix: fix the SalesSummary collapse (R-48) and the date offset (R-05) | [ ] Pending | — |
| US-067 | Assistant: PRODUCT_LOOKUP intent ("¿cuánto queda de arroz?") (R-02) | [ ] Pending | — |
| US-068 | Assistant: time reference "la quincena" (R-03) | [ ] Pending | — |
| US-069 | UI fixes: 404 Expenses, Setup permissions, static params and CSS scroll | [ ] Pending | — |

**Total estimated effort: 4 days** (short sprint, ideal for cleaning the post-QA debt before jumping to tables/invoicing).

---

## Recommended implementation order

| Order | US | Reason |
|-------|----|--------|
| **1** | US-066 | The date bug affects the reports, the whole base of calculations and the date ranges. Solving it first removes ambiguity. |
| **2** | US-067 | Widens the assistant's usefulness for a hyper-frequent use case. Requires interacting with the DB. |
| **3** | US-068 | Adds a new period ("quincena") that requires isolated adjustments in `period.go`. |
| **4** | US-069 | Bundle of Interface and Permissions fixes (blockers for UX/Deployment). |

---

## US-066 — UTC/Local bugfix and SalesSummary collapse

**As** a developer,
**I want** SQLite to store and process dates assuming a single format and time zone (Local vs UTC),
**so that** there are no discrepancies between `dashboard.go` and the reports (the 19:00 bug in Colombia) and they do not collapse into a single bucket `period=""`.

**Estimated effort:** 1 day. **Impact:** High. Fixes two critical report defects and the base range calculations (R-05, R-48).

**Verified state:**
- `internal/database/database.go:14`: The SQLite connection is opened without specifying a date format (modernc.org/sqlite supports pragma and querystring args).
- `internal/repository/report.go:113-127`: `SalesSummary` uses `COALESCE(strftime(...), '')`, causing a silent collapse of sales from multiple days into a single bucket if the fixture inserts them as a literal `time.Time`.
- `internal/repository/metrics.go:335`: Aging excludes dates if they are null or wrongly parsed.

**Business rule:**
```text
All SQLite timestamps (including CURRENT_TIMESTAMP) must be interpreted in the local time zone.
Remove the COALESCE in report.go after fixing SQLite parsing from the connection.
```

**Acceptance Criteria:**
- [ ] Reproduce the bug (R-05) with a test using a timestamp past 19:00 (UTC-5), checking that it does not wrongly move to another day.
- [ ] Add the necessary configuration to the DSN in `internal/database/database.go:14` (e.g., parseTime, locales or whatever the `modernc.org/sqlite` driver requires).
- [ ] Validate that 3 sales on 3 different days produce exactly 3 groups in `SalesSummary`, not 1 empty one (R-48).
- [ ] The `dashboard_test.go` and `metrics_test.go` tests pass with the change.

**DoD:**
- [ ] No migration.
- [ ] `internal/database/database.go` and `internal/repository/report.go` adjusted.
- [ ] Test with the D-5 assertion written.
- [ ] Clean build (`go build ./...`).

---

## US-067 — Assistant: PRODUCT_LOOKUP intent

**As** a cashier or owner,
**I want** to ask the assistant "¿cuánto queda de X?" or "¿precio de X?",
**so that** I can check inventory quickly without searching the catalog visually.

**Estimated effort:** 1.5 days. **Impact:** Medium-High. Lets the user query their primary data (inventory). Requires similarity search over the catalog.

**Verified state:**
- `internal/assistant/engine.go:13-25`: `PRODUCT_LOOKUP` does not exist among the current Intents.
- `ProductRepository.Search(ctx, query)` exists to search for the product the user mentions.

**Technical context:**
New Intent `IntentProductLookup = "PRODUCT_LOOKUP"`. Modifies `engine.go` (regex/fuzzy bank). Entity extraction: isolate the direct object ("arroz" in "cuanto arroz queda"), then pass it to `ProductRepository.Search`.

**Acceptance Criteria:**
- [ ] The intent resolves correctly for "cuánto [x] queda", "precio de la [x]", "tengo [x]", etc.
- [ ] If `Search` finds multiple matches (e.g. 5 kinds of rice), the answer lists the price/stock of the first 3 and says "...y 2 opciones más".
- [ ] If `Search` finds nothing, it answers explicitly that the product was not found in inventory.
- [ ] Any user (`admin` and `cashier`) can use this Intent (modifying the permissions layer of US-061 if needed).

**DoD:**
- [ ] Touches `assistant.go`, `engine.go`.
- [ ] New unit test cases in `engine_test.go` for the word-bank fixtures.
- [ ] Clean build.

---

## US-068 — Assistant: time reference "la quincena"

**As** the owner,
**I want** to ask "cuánto fiaron esta quincena",
**so that** I can see my real collection period in Colombia (from the 1st to the 15th, or from the 16th to month end).

**Estimated effort:** 0.5 day. **Impact:** Medium. A key reference heavily used in the local business context.

**Verified state:**
- `internal/assistant/period.go:21`: `temporalWords` does not include "quincena".
- `internal/assistant/period.go`: there is no range calculation for "esta quincena" nor "la quincena pasada".

**Business rule:**
```text
Quincena 1 (Q1): day 1 to day 15 of the month. Range [1, 16)
Quincena 2 (Q2): day 16 to the last day of the month. Range [16, 1st of next month)

"Esta quincena" evaluates which quincena we are in today, and returns that range.
"Quincena pasada" returns the immediately preceding quincena.
```

**Acceptance Criteria:**
- [ ] "quincena" added to `temporalWords`.
- [ ] Resolve "esta quincena" for today = day 10 → range from the 1st to the 15th of the current month.
- [ ] Resolve "esta quincena" for today = day 20 → range from the 16th to the end of the current month.
- [ ] Resolve "quincena pasada" correctly across a month boundary (e.g. today April 4 -> last quincena is March 16-31).
- [ ] Tests in `period_test.go` validated.

**DoD:**
- [ ] `period.go` modified.
- [ ] Fixtures in `period_test.go` for the 4 month boundaries.
- [ ] Clean build.

---

## US-069 — UI fixes: 404 Expenses, Setup permissions, static params and CSS scroll

**As** a user and administrator,
**I want** the application interface to have no routing errors and to restrict access,
**so that** I can operate smoothly and without technical failures.

**Estimated effort:** 1 day. **Impact:** High (QA fixes).

**Verified state:**
1. Creating an expense (POST `/expenses`) in `/reports` returns 404 in the Next.js environment.
2. Failing build: `Page "/customers/[id]/page" is missing param "/customers/[id]" in "generateStaticParams()"`.
3. The label `Ventas del período` in reports is not descriptive.
4. The cashier (`cashier`) can enter `GET /setup` because the backend does not require `RequireAdmin`.
5. In `/settings`, the store configuration has no scroll (the main container lacks `overflow-y-auto`).

**Acceptance Criteria:**
- [ ] Expense creation works (resolve the 404 by checking the API client and CORS or trailing slashes).
- [ ] The static build (`output: export`) succeeds by adding `generateStaticParams()` returning `[]` in `app/customers/[id]/page.tsx`.
- [ ] The sales title in the report is clear and descriptive (e.g.: "Total vendido en el periodo").
- [ ] A user with the `cashier` role cannot access `/setup`. `handler.go` protects `GET /setup` with `RequireAdmin` and the frontend kicks them out or does not show them the tab.
- [ ] The container in `/settings` has functional vertical scroll (`overflow-y-auto`) so all options are visible.

**DoD:**
- [ ] `next build` finishes correctly with no "missing param" errors.
- [ ] Each fix is verified manually.
