# SPRINT-20 — Go-Live for Client #1: Install, Back Up, Certify the Money Flows

**Goal:** Package, harden, validate and install the production build in the first client's shop: no data loss in SQLite, a register close that matches to the peso, a working *fiado* (store credit) and *abono* (partial payment toward a fiado balance) flow, and continuous operation with electronic invoicing (DIAN) turned off.

**Start date:** 2026-11-09 (planned)

**Origin:** PO planning draft of 2026-11-07 (`planificaci_n_del_sprint_lanzamiento_mvp_pos.md`, Spanish, untracked; replaced by this document). Every item was verified against the code on 2026-11-07 before being kept; the corrections are listed below. Scope approved by the PO on 2026-11-07 ("use all recommended"). References: `project-docs/spec/domain/caja.md`, `project-docs/spec/domain/fiado.md`, `project-docs/spec/domain/servidor.md`.

**Draft items already covered by the code (not planned again):**
- SP-01 SQLite hardening: WAL, `busy_timeout(5000)` and `foreign_keys(1)` are set on every pooled connection (`internal/database/database.go:23`); migrations run at startup on an empty file (`internal/database/database.go:33`). `synchronous=NORMAL` is deliberately not added: the default `FULL` is the safer choice for a shop PC that loses power.
- SP-03 money formulas: balance = completed credit sales − *abonos* (`internal/repository/customer.go:120-128`), so a voided credit sale drops out of the balance; only cash *abonos* reach expected cash (`internal/repository/register.go:226`, US-082); voids no longer double-subtract (US-088). Remaining: one end-to-end certification test (US-091).
- SP-04 single binary: `out/` is embedded (`web/static.go:14`) and served on the API port (`cmd/pos/main.go:72`). Remaining: F5 on every route in the binary (US-091).

**Draft corrections:**
- SP-02 assumed `eledocRepo = nil` when DIAN is off. It is never nil: it is always wired (`internal/handler/handler.go:60`, `:70`). What is nil when DIAN is off is the `factelect` service (`internal/handler/handler.go:73-79`). The handlers guard it (`internal/handler/sale.go:50`, `:127`, `internal/handler/order.go:242`, `internal/handler/fiscal.go:42`, `:60`); US-090 certifies this end to end.
- SP-05 assumed backups do not exist. A daily in-process `VACUUM INTO` backup exists (`internal/database/backup.go:16-48`), but it keeps 7 copies (`internal/database/backup.go:12`) next to the database file. US-089 extends it.
- SP-07 assumed a catalog import exists. It does not: SPRINT-03 US-017 is CSV **export** (`frontend/app/reports/page.tsx:227`). US-094 proposes one.

**Manual items (PO, not code):** SP-06 printer and scanner certification and SP-08 on-site go-live are recorded in `gaps.md` → "Pending PO validations" (rows `SPRINT-20 G-1..G-3`). US-087 M-1 and US-080 P-1..P-4 already cover the printer.

**Out of scope:** DIAN e-invoicing in production, cloud backup, Linux/`systemd` install (the client PC is Windows).

---

## Tasks

| US | Name | Status | Date |
|----|------|--------|------|
| US-089 | Start on Windows logon, and daily backups to a configurable folder with 30-day retention | [ ] | |
| US-090 | Certify that every sale flow works with DIAN turned off | [x] | 2026-11-07 |
| US-091 | End-to-end certification of register close and *fiado* math, and F5 on every route of the binary | [x] | 2026-11-08 |
| US-092 | Opening *fiado* balance per customer (moving debts from the paper notebook) | [ ] | |
| US-093 | Reset tool: clear test transactions, keep master data | [ ] | |
| US-094 | Product catalog import from CSV | [ ] | |
| US-095 | Admin void button for a sale in the UI | [ ] | |

**Total estimated effort: ~6.5 days**

---

## Recommended implementation order

| Order | US | Reason |
|-------|----|--------|
| **1** | US-090 | Small audit, level M. Confirms the base the other tests run on. |
| **2** | US-091 | Certifies the money flows before real money goes through them. |
| **3** | US-095 | Lets the admin undo a wrong charge at the counter; the G-2 dry run voids a sale through it. Its money effect is certified by US-091. |
| **4** | US-092 | Needed before loading the client's existing debts. |
| **5** | US-094 | Needed before loading the client's catalog (if approved). |
| **6** | US-093 | Run after the dry run, right before go-live. |
| **7** | US-089 | Last code before install; tested on the real install path. |

---

## US-089 — Start on Windows logon, and daily backups to a configurable folder with 30-day retention

**As** Don Carlos (owner),
**I want** the POS to come back by itself after a power cut and to keep a month of daily backups in a folder I choose,
**so that** a reboot or a dead disk does not stop the shop or lose the *fiado* ledger.

**Estimated effort:** 1 day. **Impact:** High.

**Today:** backups run at startup and every 24 h, written as `<db>.backup-YYYYMMDD` next to the database, keeping 7 (`internal/database/backup.go:12`, `:16-48`, started at `cmd/pos/main.go:61`). Nothing registers the binary to start on boot.

**Acceptance criteria (draft; the contract refines them):**
- [ ] A one-time install step registers `pos.exe` to start at user logon with Windows Task Scheduler (no third-party service manager), and a matching uninstall step removes it.
- [ ] After a reboot with no human action, the POS answers on its port in under 30 s.
- [ ] Retention is 30 daily backups; the 31st oldest is removed.
- [ ] The backup folder is configurable by the admin in Settings; the default is the current location. A folder that does not exist or is not writable is reported to the admin, and the backup falls back to the default location instead of being skipped.
- [ ] A backup taken while a register is open and sales are being written opens cleanly and passes `PRAGMA integrity_check`.

## US-090 — Certify that every sale flow works with DIAN turned off

**As** the Tech Lead,
**I want** a test that runs every sale entry point with electronic invoicing disabled,
**so that** the first client, who has no DIAN setup, never hits a nil-pointer panic or a DIAN control.

**Estimated effort:** 0.5 day (level M). **Impact:** High.

**Acceptance criteria (draft):**
- [x] With fiscal config disabled, create, read, reprint and void a sale, and check out a table order, with no panic and no error in the log.
- [x] `void-with-credit-note` and the fiscal document endpoints answer a clear 4xx, not a 500.
- [x] No DIAN send or status control is visible in the UI when DIAN is off.

**Verified criteria (closed 2026-11-07):**

Contract discrepancies: the Settings DIAN section called `/fiscal-config`, a route that does not exist (404), instead of `/fiscal/config` (D-1, fixed); the spec claimed a non-invoicing shop never sees DIAN, but the section always rendered (D-2, fixed); the `eledocRepo` "nil if FE disabled" comment was false (D-3, fixed); DIAN-on findings D-4, D-5, D-7 and the void deadlock D-6 went to the roadmap (R-124..R-127). GATE 1 (PO, 2026-11-07): Q-1..Q-4 accepted as recommended. The section renders only when `enabled === true`; the DIAN-off codes stay 400 / 400 / 404. No backend behavior changed: the backend already met AC-1..AC-8 when run in the contract. QA round 1: OK, 0 BLOCKER, 4 MINOR (R-127, R-128, R-129; spec delta applied at close). Gemini diff round 1: clean. Smoke test on the compiled binary: PASS. Spec: `spec/domain/facturacion-dian.md` § Activation and "With DIAN off". Manual check: `gaps.md` US-090 M-1 (logged-in render of `/settings`, not seen in a real browser).

AC-1..AC-9 hold WHILE `fiscal_config.enabled = 0` at startup (`h.factelect == nil`, asserted first in every backend test). All backend tests are in `internal/handler/dian_off_test.go` and run through `h.Routes()`.

- AC-1, AC-2, AC-6: WHEN a sale is posted (with a `buyer_email`), read, or its ticket requested, the system SHALL answer 201 / 200 / 200 with the stored totals and create no `electronic_documents` row, with no panic and nothing written to the standard logger. 1 × Arroz $50,000 cash → total 50000, ticket item `Arroz`, 0 documents. Test: `TestDianOff_T1_CreateReadTicket`.
- AC-3, AC-6: WHEN a sale is reprinted to a network ESC/POS printer, the system SHALL send it and answer 200 `printed: true`; the bytes contain `$50.000`. Test: `TestDianOff_T2_ReprintToNetworkPrinter`.
- AC-4, AC-6: WHEN an admin voids a sale with no electronic document, the system SHALL answer 200 `status voided` and return the stock (19 → 20); a second void → 400 `la venta ya fue anulada`. Test: `TestDianOff_T3_VoidRestoresStock`.
- AC-5, AC-6: WHEN a table order is checked out (with a `buyer_email`), the system SHALL answer 201 with `order_id`, `sale_id` and `total` and create no document. Café × 2 + Torta × 1 → total 17000, sale 1, 0 documents, the sale then voids with 200. Test: `TestDianOff_T4_TableCheckout`.
- AC-7: WHEN an admin calls `POST /sales/{id}/void-with-credit-note` with a motive, the system SHALL answer 400 `facturación electrónica no habilitada` and leave the sale `completed`; IF the motive is missing, THEN 400 `motive requerido`; a cashier gets 403. Test: `TestDianOff_T5_CreditNoteRefused`.
- AC-8: WHEN an admin calls `GET /sales/{id}/factura` / `GET /sales/{id}/factura/pdf`, the system SHALL answer 400 / 404 with `facturación electrónica no habilitada`, never 5xx. Test: `TestDianOff_T6_FacturaEndpointsRefused`.
- AC-9: WHEN an admin opens Settings, the frontend SHALL load the config from `/fiscal/config`, and IF `enabled` is not `true` (false, or the load failed), THEN it SHALL NOT render the "Facturación electrónica DIAN" section. Tests: `TestDianOff_T7_FiscalConfigRoute` (`/fiscal/config` → 200 `enabled false`, `/fiscal-config` → 404); `frontend/lib/fiscalSettings.test.ts` → "points at the backend route /fiscal/config", "hides the section when the config could not be loaded", "hides the section when DIAN is off".
- AC-10: WHEN the saved config has `enabled: true`, the Settings section SHALL render and save through `PUT /fiscal/config`. Test: `frontend/lib/fiscalSettings.test.ts` → "shows the section when DIAN is on". That `page.tsx` uses the helpers is checked by review only (R-129).

## US-091 — End-to-end certification of register close and *fiado* math, and F5 on every route of the binary

**As** Don Carlos (owner),
**I want** the register close and customer balances proven to the peso in one scripted day of trading,
**so that** I can trust the close on day one.

**Estimated effort:** 1 day. **Impact:** High.

**Acceptance criteria (draft):**
- [x] One test runs a full shift: opening $100.000; cash sale $30.000; credit sale $50.000; cash *abono* $20.000; transfer *abono* $10.000; card *abono* $5.000; cash in $15.000; cash out $8.000; void of a $12.000 cash sale made in the same shift. Expected close = 100.000 + 30.000 + 20.000 + 15.000 − 8.000 = **$157.000**; the customer balance = 50.000 − 20.000 − 10.000 − 5.000 = **$15.000**.
- [x] A void of a credit sale lowers the customer balance by its credit amount and returns the stock.
- [x] A cashier cannot register a cash *abono* with no open register.
- [x] In the compiled binary, pressing F5 on every frontend route (including dynamic ones) reloads the same page, never a 404.

**Verified criteria (closed 2026-11-08):**

Contract findings, verified by running: the draft totals are right (close $157.000, balance $15.000; the $12.000 sale takes `expected` to $169.000 before its void). The cashier *abono* rule was already enforced since US-082 (D-1), and every route already served its own page on reload (D-2); four public pages redirect on load by design and an unknown path gets the home page with 200. There is no void action in the UI (D-3), transfer and card *abonos* reach no close figure (D-4), an HTTP close on a `:memory:` test DB fails 174 of 200 runs (D-5), and `scripts/copy-frontend.go` deleted the tracked `web/out/.gitkeep` (D-6). GATE 1 (PO, 2026-11-08): Q-1..Q-4 accepted as recommended, plus the AC-6 fix (admin *abono* amount and resulting balance, a verified Gemini finding). The UI void became US-095 (from R-111); non-cash *abonos* on the close went to R-134; the real-export tests run in CI; the shift tests use a file-backed DB. No production Go code changed; `scripts/copy-frontend.go` now recreates `.gitkeep` and CI runs the export tests after `pnpm run build` (`.github/workflows/test.yml:34-43`). QA round 1: OK, 0 BLOCKER, 3 MINOR (R-135, R-136), 18/18 mutations caught, binary smoke test PASS. Gemini diff round 1: clean. Spec: `spec/domain/caja.md` ("Rule: expected cash at close"), `spec/domain/fiado.md` ("Rule: balance"), `spec/domain/servidor.md` ("Route certification"). Manual check: `gaps.md` US-091 M-1.

Shift data for AC-1..AC-4: opening $100.000; sale #1 cash $30.000; sale #2 credit $50.000 to customer 1; *abonos* $20.000 cash, $10.000 transfer, $5.000 card; `cash_in` $15.000; `cash_out` $8.000; sale #3 cash $12.000; admin voids sale #3; the cashier closes with $157.000. Backend tests are in `internal/handler/shift_certification_test.go`, through `h.Routes()` on a file-backed DB.

- AC-1: WHEN the shift has run up to and including the void of sale #3, `GET /registers/1` SHALL return `cash_sales` 30000, `cash_abonos` 20000, `cash_in` 15000, `cash_out` 8000, `expected` 157000 and `by_payment_method` exactly cash 30000 × 1 and credit 50000 × 1. Test: `TestShiftCert_T1_FullShift`.
- AC-2: WHEN the cashier closes register 1 with `actual_close` 157000, the system SHALL answer 200 and store `closed`, `expected_close` 157000, `actual_close` 157000, `difference` 0; a second close → 409. Test: `TestShiftCert_T2_CloseAndAfterClose`.
- AC-3: WHEN the shift has run, the customer balance SHALL be 15000 in `GET /customers/{id}/balance` and in `GET /customers` (`total_portfolio` 15000), with the three *abonos* stored with methods cash, transfer, card and `register_id` 1. Test: `TestShiftCert_T1_FullShift`.
- AC-4: WHEN an admin voids cash sale #3, the system SHALL answer 200, bring `expected` 169000 → 157000 and `cash_sales` 42000 → 30000, restore Panela's stock 9 → 10 and add no movement (6 before and after); IF a cashier sends the void, THEN 403 and nothing changes. Test: `TestShiftCert_T1_FullShift`.
- AC-5: WHEN an admin voids a credit sale, the system SHALL lower the balance by its credit portion, restore the stock and change `expected` only by the cash portion: 50000 → 0; after *abonos* 15000 → −35000 (`total_portfolio` 0); mixed $30.000 cash + $20.000 credit 20000 → 0 with `expected` 130000 → 100000; on a closed register 14000 → −36000 with neither register changed. Tests: `TestShiftCert_T3_CreditVoidLowersBalance`, `TestShiftCert_T4_CreditVoidOnClosedRegister`.
- AC-6: IF a cashier registers an *abono* after the HTTP close with no register open, THEN 403 `Debe haber una caja abierta para registrar un abono`, nothing stored, balance 15000; WHEN an admin registers $1.000 cash, 201 with `register_id` null, balance 15000 → 14000, register 1 `expected_close` 157000 and `difference` 0 unchanged. Test: `TestShiftCert_T2_CloseAndAfterClose`.
- AC-7: WHEN the binary receives a GET for any of the 21 page routes (dynamic ones with a concrete segment, with or without a trailing `/` or a query string), the system SHALL answer 200 `text/html` with that route's exported file, never a 404. Tests: `TestFileServer_T18_EveryAppRouteResolves` (`web/static_routes_test.go`), `TestFileServer_T19_RealExportEveryRoute` (`web/static_export_test.go`, build tag `export`, run in CI).
- AC-8: IF a `page.tsx` has a dynamic prefix missing from `dynamicRoutes` or a page that does not bake `'_'`, THEN `go test ./web/` SHALL fail naming the route. Test: `TestFileServer_T18_EveryAppRouteResolves`.
- AC-9: WHILE an admin is logged in on `pos.exe`, F5 on the 16 authenticated routes and on `/catalogo/<token>` SHALL show the same page with its data. Manual: `gaps.md` US-091 M-1.
- AC-10: WHEN `go run scripts/copy-frontend.go` replaces `web/out`, it SHALL leave an empty `web/out/.gitkeep`. Test: `TestCopyFrontend_T20_KeepsGitkeep` (`web/static_export_test.go`) plus `git diff --exit-code -- web/out/.gitkeep` in CI.

## US-095 — Admin void button for a sale in the UI

**Status:** added by the PO at US-091 GATE 1 (2026-11-08), from roadmap R-111. Not part of US-091.

**As** Don Carlos (owner, admin),
**I want** to void a wrong sale from the screen,
**so that** a mistaken charge on day one can be undone at the counter without calling the API.

**Estimated effort:** 1 day. **Impact:** High for go-live.

**Today:** voids exist only in the admin API (`DELETE /sales/{id}/void`, `internal/handler/handler.go:169`); no frontend file calls it. The UI only lists voided sales (`Ver anuladas`, badge `Anulada`, `frontend/app/page.tsx:566`, `:621`). The money effect of a void (register `expected`, *fiado* balance, stock) is certified by US-091.

**Acceptance criteria (draft; the contract refines them):**
- [ ] An admin sees an `Anular` action on a completed sale and confirms it before anything is sent; a cashier does not see the action (the API already answers 403).
- [ ] After a void, the sale shows as `Anulada`, and the register summary and the customer balance shown on screen reflect it without a manual reload.
- [ ] A sale with an accepted or pending electronic document, an already voided sale, or a server error shows the server message and changes nothing.
- [ ] The contract decides, with the PO, what the action does for a sale whose register is already closed (roadmap R-112).

## US-092 — Opening *fiado* balance per customer

**As** Don Carlos (owner),
**I want** to enter what each customer already owes from my paper notebook when I create them,
**so that** the POS ledger starts where the notebook ends.

**Estimated effort:** 1 day. **Impact:** High for go-live. **Changes a business rule** (`spec/domain/fiado.md`, balance formula), so the contract asks the PO how it is stored and shown.

**Acceptance criteria (draft):**
- [ ] The admin can record an opening balance with a date for a customer; it adds to the balance and to the debt age, and *abonos* pay it down like any credit.
- [ ] It does not count as a sale in any report, dashboard or register close.
- [ ] It is recorded in the audit log and cannot be entered twice by accident.

## US-093 — Reset tool: clear test transactions, keep master data

**As** the Tech Lead installing the POS,
**I want** a single command that removes sales, registers, *abonos* and movements made during the dry run, keeping products, categories, customers and users,
**so that** the client starts with a clean history and the real catalog.

**Estimated effort:** 0.5 day. **Impact:** Medium (one-off, but done by hand it risks the catalog).

**Acceptance criteria (draft):**
- [ ] A CLI utility next to `cmd/reset-password` takes a backup first, then clears transaction tables and restores stock to a value the PO decides in the contract.
- [ ] It refuses to run while the POS is running, and asks for confirmation.
- [ ] Physical `DELETE` is allowed here only because the rows are test data; the contract states this exception to the logical-delete rule.

## US-094 — Product catalog import from CSV

**Status:** approved by the PO on 2026-11-08 (added on 2026-11-07 after verifying that no import exists). PO direction: the import must also serve a shop that is moving from another POS program, so the input format is decided in the contract. Candidates are a CSV with column mapping (most POS programs and spreadsheets can export one) or a SQL/SQLite dump of the other program. The analyst recommends one format with its trade-off at GATE 1.

**As** Don Carlos (owner),
**I want** to load my product list from a spreadsheet,
**so that** I do not type hundreds of products by hand before opening.

**Estimated effort:** 1.5 days. **Impact:** High for go-live if the catalog has more than about 50 products.

**Acceptance criteria (draft):**
- [ ] The admin uploads a CSV with name, barcode, category, unit (`unit`, `kg`, `lb`, `liter`), cost, price and stock; a downloadable template is provided.
- [ ] Rows are validated first and errors are shown per row before anything is written; the import is all-or-nothing.
- [ ] Missing categories are created; a duplicate barcode is reported, not silently overwritten.
