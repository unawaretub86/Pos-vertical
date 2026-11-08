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
| US-090 | Certify that every sale flow works with DIAN turned off | [ ] | |
| US-091 | End-to-end certification of register close and *fiado* math, and F5 on every route of the binary | [ ] | |
| US-092 | Opening *fiado* balance per customer (moving debts from the paper notebook) | [ ] | |
| US-093 | Reset tool: clear test transactions, keep master data | [ ] | |
| US-094 | Product catalog import from CSV | [ ] Pending PO yes | |

**Total estimated effort: ~5.5 days**

---

## Recommended implementation order

| Order | US | Reason |
|-------|----|--------|
| **1** | US-090 | Small audit, level M. Confirms the base the other tests run on. |
| **2** | US-091 | Certifies the money flows before real money goes through them. |
| **3** | US-092 | Needed before loading the client's existing debts. |
| **4** | US-094 | Needed before loading the client's catalog (if approved). |
| **5** | US-093 | Run after the dry run, right before go-live. |
| **6** | US-089 | Last code before install; tested on the real install path. |

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
- [ ] With fiscal config disabled, create, read, reprint and void a sale, and check out a table order, with no panic and no error in the log.
- [ ] `void-with-credit-note` and the fiscal document endpoints answer a clear 4xx, not a 500.
- [ ] No DIAN send or status control is visible in the UI when DIAN is off.

## US-091 — End-to-end certification of register close and *fiado* math, and F5 on every route of the binary

**As** Don Carlos (owner),
**I want** the register close and customer balances proven to the peso in one scripted day of trading,
**so that** I can trust the close on day one.

**Estimated effort:** 1 day. **Impact:** High.

**Acceptance criteria (draft):**
- [ ] One test runs a full shift: opening $100.000; cash sale $30.000; credit sale $50.000; cash *abono* $20.000; transfer *abono* $10.000; card *abono* $5.000; cash in $15.000; cash out $8.000; void of a $12.000 cash sale made in the same shift. Expected close = 100.000 + 30.000 + 20.000 + 15.000 − 8.000 = **$157.000**; the customer balance = 50.000 − 20.000 − 10.000 − 5.000 = **$15.000**.
- [ ] A void of a credit sale lowers the customer balance by its credit amount and returns the stock.
- [ ] A cashier cannot register a cash *abono* with no open register.
- [ ] In the compiled binary, pressing F5 on every frontend route (including dynamic ones) reloads the same page, never a 404.

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

**Status:** pending PO yes (added on 2026-11-07 after verifying that no import exists).

**As** Don Carlos (owner),
**I want** to load my product list from a spreadsheet,
**so that** I do not type hundreds of products by hand before opening.

**Estimated effort:** 1.5 days. **Impact:** High for go-live if the catalog has more than about 50 products.

**Acceptance criteria (draft):**
- [ ] The admin uploads a CSV with name, barcode, category, unit (`unit`, `kg`, `lb`, `liter`), cost, price and stock; a downloadable template is provided.
- [ ] Rows are validated first and errors are shown per row before anything is written; the import is all-or-nothing.
- [ ] Missing categories are created; a duplicate barcode is reported, not silently overwritten.
