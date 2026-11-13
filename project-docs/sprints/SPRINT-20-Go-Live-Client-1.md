# SPRINT-20 — Go-Live for Client #1: Install, Back Up, Certify the Money Flows

**Goal:** Package, harden, validate and install the production build in the first client's shop: no data loss in SQLite, a register close that matches to the peso, a working *fiado* (store credit) and *abono* (partial payment toward a fiado balance) flow, and continuous operation with electronic invoicing (DIAN) turned off.

**Start date:** 2026-11-09 (planned)

**Origin:** PO planning draft of 2026-11-07 (`planificaci_n_del_sprint_lanzamiento_mvp_pos.md`, Spanish, untracked; replaced by this document). Every item was verified against the code on 2026-11-07 before being kept; the corrections are listed below. Scope approved by the PO on 2026-11-07 ("use all recommended"). References: `project-docs/spec/domain/caja.md`, `project-docs/spec/domain/fiado.md`, `project-docs/spec/domain/servidor.md`.

**Draft items already covered by the code (not planned again):**
- SP-01 SQLite hardening: WAL, `busy_timeout(5000)` and `foreign_keys(1)` are set on every pooled connection (`internal/database/database.go:23`); migrations run at startup on an empty file (`internal/database/database.go:33`). `synchronous=NORMAL` is deliberately not added: the default `FULL` is the safer choice for a shop PC that loses power.
- SP-03 money formulas: balance = completed credit sales − *abonos* (`internal/repository/customer.go:120-128` at planning time; since US-092 an opening balance is added, `fiado.md` "Rule: customer balance"), so a voided credit sale drops out of the balance; only cash *abonos* reach expected cash (`internal/repository/register.go:226`, US-082); voids no longer double-subtract (US-088). Remaining: one end-to-end certification test (US-091).
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
| US-089 | Start on Windows logon, and daily backups to a configurable folder with 30-day retention | [x] | 2026-11-11 |
| US-090 | Certify that every sale flow works with DIAN turned off | [x] | 2026-11-07 |
| US-091 | End-to-end certification of register close and *fiado* math, and F5 on every route of the binary | [x] | 2026-11-08 |
| US-092 | Opening *fiado* balance per customer (moving debts from the paper notebook) | [x] | 2026-11-09 |
| US-093 | Reset tool: clear test transactions, keep master data | [x] | 2026-11-10 |
| US-094 | Product catalog import from CSV | [x] | 2026-11-09 |
| US-095 | Admin void button for a sale in the UI | [x] | 2026-11-09 |
| US-096 | Money QA layer 1: property-based invariant tests over random sequences of money operations | [x] | 2026-11-08 |
| US-097 | Singular/plural wording in the catalog import UI (roadmap R-157) | [x] | 2026-11-09 |
| US-098 | Day windows and stored timestamps agree on one zone: after-19:00 sales missing from "today" (roadmap R-158, R-05) | [x] | 2026-11-10 |
| US-099 | The pre-migration safety copy gets its own name and survives the daily backup (roadmap R-168, K2) | [x] | 2026-11-12 |
| US-101 | One name for the Settings screen: "Configuración" in `INSTALL.md`, `MANUAL-USUARIO.md` and the Mesas empty state (roadmap R-170, go-live K4) | [x] | 2026-11-12 |
| US-102 | Whole-peso money with one rounding rule: line IVA rounded, totals and payments exact, migration 24 for existing rows, no sub-peso *fiado* residue (roadmap R-133 + R-106, go-live K5) | [x] | 2026-11-13 |

**Total estimated effort: ~8 days**

---

## Recommended implementation order

| Order | US | Reason |
|-------|----|--------|
| **1** | US-090 | Small audit, level M. Confirms the base the other tests run on. |
| **2** | US-091 | Certifies the money flows before real money goes through them. |
| **3** | US-095 | Lets the admin undo a wrong charge at the counter; the G-2 dry run voids a sale through it. Its money effect is certified by US-091. |
| **4** | US-096 | Generated money invariants over random sequences, after the fixed US-091 shift; its validation fixes land before opening balances and the dry run. |
| **5** | US-092 | Needed before loading the client's existing debts. |
| **6** | US-094 | Needed before loading the client's catalog (if approved). |
| **7** | US-093 | Run after the dry run, right before go-live. |
| **8** | US-089 | Last code before install; tested on the real install path. |
| **9** | US-098 | Fixes the 7 p.m.-midnight gap in "today" figures before the on-site dry run reads them; independent of the other US. |
| **10** | US-099 (done) | Small fix to the US-089 backup code: the pre-upgrade copy must not be overwritten by the same-day daily backup. Land before the first real upgrade of the client's `data.db`. |

---

## US-089 — Start on Windows logon, and daily backups to a configurable folder with 30-day retention

**As** Don Carlos (owner),
**I want** the POS to come back by itself after a power cut and to keep a month of daily backups in a folder I choose,
**so that** a reboot or a dead disk does not stop the shop or lose the *fiado* ledger.

**Estimated effort:** 1 day. **Impact:** High.

**Today:** backups run at startup and every 24 h, written as `<db>.backup-YYYYMMDD` next to the database, keeping 7 (`internal/database/backup.go:12`, `:16-48`, started at `cmd/pos/main.go:61`). Nothing registers the binary to start on boot.

**Input from US-093 (2026-11-10):** the go-live reset tool writes `<db>.pre-reset-YYYYMMDD-HHMMSS` next to `data.db` and never prunes it. The US-089 retention must keep matching only `<db>.backup-*` so it never deletes a `.pre-reset-*` file (roadmap R-167).

**Status:** closed 2026-11-11 in code and tests. The real Task Scheduler install and the reboot were **never run** (QA did not create a task); the PO manual checks are `gaps.md` US-089 M-1..M-3.

**Acceptance criteria (draft; the contract refines them):**
- [x] A one-time install step registers `pos.exe` to start at user logon with Windows Task Scheduler (no third-party service manager), and a matching uninstall step removes it.
- [x] After a reboot with no human action, the POS answers on its port in under 30 s (measured from the user's logon; the reboot itself is manual check M-1).
- [x] Retention is 30 daily backups; the 31st oldest is removed.
- [x] The backup folder is configurable by the admin in Settings; the default is the current location. A folder that does not exist or is not writable is reported to the admin, and the backup falls back to the default location instead of being skipped.
- [x] A backup taken while a register is open and sales are being written opens cleanly and passes `PRAGMA integrity_check`.

**Verified criteria (closed 2026-11-11):**

GATE 1 (PO, 2026-11-11): Q-1..Q-10 accepted as recommended: 30 s counts from the user's logon (the shop PC signs in by itself, `netplwiz`); install and uninstall are `pos.exe` flags that build the task XML and call `schtasks.exe`; a bad folder is rejected at save time and also handled by the fallback at backup time; a banner on the admin dashboard besides the Settings section; "30" means the 30 newest files; `reset-data` does not copy its backup to the backup folder (**closes roadmap R-167**); the console window stays (document "minimize, do not close"); the pre-migration copy name is not touched; the copy is written to `.tmp`, verified, then renamed; the new surface (two endpoints, migration 23, new files) approved. Contract discrepancies: `data.db` is a relative path, so the task must set the working directory (D-1); `INSTALL.md` claimed no window opens but the exe is a console app (D-2); the glob prune broke on `[` in the path and could delete a manual `data.db.backup-*` copy (D-3, fixed with a strict name match); a failed same-day re-run destroyed the good copy (D-4a, fixed; D-4b, the pre-migration name, is R-168); backup settings could not live in `PUT /setup` (D-5); `gaps.md` G-2 said `pos.db` (D-6, fixed). QA round 1: OK, 0 BLOCKER, 5 MINOR (spec delta applied at close; R-170, R-171; two accepted deviations below); Gemini diff round 1: clean. `go build`, `go test ./... -count=1`, `pnpm build`, `pnpm test` (254 tests) and `pnpm lint` (0 problems) green. Smoke on a throwaway database with the compiled binary (port 18089): a configured folder `bk[1]` with 35 backups plus `.pre-reset-*`, `.backup-manual`, `.backup-<date>.tmp` and another database's file ended with exactly 30 strict-name backups and all four extra files intact; a removed folder fell back to the default and reported it; the endpoints answered 200 / 403 / 401 as specified. Not run: `--install-autostart` / `--uninstall-autostart` against the real Task Scheduler, and the frontend pages in a browser. Spec: `spec/domain/backup.md` (new), `servidor.md` § Start with Windows, `reset.md`, `tiempo.md`, `project-docs/README.md`.

Accepted deviations (analyst notes; PO to confirm): (1) migration 23 adds a ninth column `backup_effective_dir` (`internal/database/migrate.go:627`), required so `effective_dir` survives a restart (AC-14); the contract listed eight. (2) The Settings section and the banner are two new components, `frontend/components/BackupSection.tsx` and `BackupBanner.tsx`, not inline code in `settings/page.tsx` and `app/page.tsx`. (3) The pre-migration copy is skipped on a brand-new database (`current == 0`, `internal/database/migrate.go:80`): it only removes a useless empty copy; the upgrade path (`current > 0`) still copies, and the same-name overwrite (R-168) is untouched.

Backend tests: `internal/database/backup_test.go` (B), `internal/handler/backup_test.go` (H), `internal/database/migrate_test.go` and `schemaversion_test.go` (M), `cmd/pos/autostart_test.go` (A); frontend tests in `frontend/lib/backupSettings.test.ts` (F).

- AC-1, AC-2: WHEN a user runs `pos.exe --install-autostart` the system SHALL register the per-user task `POS-Vertical` (replacing an existing one) and print `Inicio automático instalado`, exit 0; `--uninstall-autostart` SHALL delete it and print `Inicio automático quitado`, or `El inicio automático no estaba instalado` with exit 0 when absent; neither opens the database, starts the server or opens the browser (the flag branch exits before `database.New`, `cmd/pos/main.go:47-48` vs `:57`, checked by reading). Tests: A `TestAutostart_T16_InstallAndUninstallCalls` (`/Create /TN POS-Vertical /XML <file> /F` once; twice → the same calls; `/Delete /TN POS-Vertical /F`).
- AC-3: IF `schtasks.exe` exits non-zero or the OS is not Windows, THEN the system SHALL exit 1 with a Spanish message including the `schtasks` text and delete its temporary XML. Test: A `TestAutostart_T17_Failures`.
- AC-4: The task SHALL use a `LogonTrigger` with no delay, run the absolute `pos.exe` path with the working directory set to its folder, `PT0S`, `IgnoreNew`, start on battery, restart on failure 3 times at 1 minute, normal rights, and XML-escape `C:\POS & Co\pos.exe`. Test: A `TestAutostart_T15_BuildTaskXML`.
- AC-5: StartBackupScheduler SHALL return without waiting for the backup, and the process SHALL log the absolute database path (`database: <path>`). Tests: B `TestBackup_T8_SchedulerReturnsImmediately` (a source blocking 2 s; returns in under 200 ms), A `TestResolvedDBPath_T18_IsAbsolute`. The reboot and the 30 s are manual: `gaps.md` US-089 M-1.
- AC-6: WHEN a backup completes, the system SHALL keep the 30 newest `data.db.backup-YYYYMMDD` files: 31 → the oldest is removed; 29 + one new → 30, none removed; a same-day re-run → still 30. The date is the server-local day: 02:30 UTC on 01/11/2026 (21:30 Bogota on 31/10) → `...-20261031`. Tests: B `TestBackup_T1_KeepsThirtyNewest`, `TestBackup_T9b_FileDateUsesLocalZone`.
- AC-7, AC-8: The prune SHALL delete only `<base>.backup-` + 8 digits and work in a folder whose path has `[`. 33 backups plus `data.db.pre-reset-20261110-143005`, `data.db.backup-manual`, `data.db.backup-20261001.tmp` and `other.db.backup-20250101` → `20261001..20261003` removed, 30 left, the four extra files untouched; folder `a[1]` with 32 backups → 30. Tests: B `TestBackup_T2_PruneIgnoresForeignFiles`, `TestBackup_T3_PruneWorksInFolderWithBrackets`.
- AC-9: IF a copy, its verification or the rename fails, THEN the system SHALL NOT prune, SHALL leave the earlier same-day file byte-identical, SHALL remove its `.tmp` and record the failure. A directory named `data.db.backup-20261031.tmp` makes the copy fail: marker bytes unchanged, 31 older files still there, `last_ok` false with a non-empty `last_error`. Test: B `TestBackup_T5_FailedCopyKeepsGoodFileAndDoesNotPrune`.
- AC-10: WHEN a backup runs, the system SHALL `VACUUM INTO` a `.tmp`, require `integrity_check` = `ok` and per-table counts within `[min(before, after), max(before, after)]` of the live counts, then rename. Live `sales` 10 before and 12 after: a copy with 11 passes, 9 or 13 fails; a quiet table (3 users) whose copy has 4 fails and is not retried; a 4096-byte file of `0x41` fails; no `*.tmp` is left. Tests: B `TestBackup_T4_CopyVerifiedAndNoTmpLeft`, `TestBackup_T6_VerifyBackup`, `TestBackup_T6b_RowCountTolerance`.
- AC-11, AC-12: WHEN a folder is configured and usable, the file SHALL be written there and that folder pruned; with none, next to `data.db`. IF the folder does not exist, is a file, or is not writable, THEN the backup SHALL go to the default folder with `fell_back` true and the failing folder and reason in `last_error`, and the old folder's 30 files stay untouched. Test: B `TestBackup_T7_DestinationAndFallback` (cases a-e).
- AC-13: WHEN an admin calls `PUT /api/backup/config` with a valid absolute existing writable folder, the system SHALL store it (trimmed) and answer 200; `""` clears it; relative → 400 `Escriba la ruta completa…`; missing → 400 `La carpeta no existe`; a file → 400; not writable → 400 `No se puede escribir en la carpeta`; `{}` or `{"dir":null}` → 400 `Falta la carpeta`; the stored value is unchanged on every 400; cashier 403, no token 401. Test: H `TestBackupConfig_T12_PutValidation`.
- AC-14: WHEN an admin calls `GET /api/backup/status`, the system SHALL return the nine fields (RFC 3339 timestamps with offset or `null`), persisted across a restart; before any run `last_attempt_at` null, `last_ok` false, `effective_dir` `""`; after a good run followed by a failed one, `last_success_at` and `last_file` keep the good run's values while `last_ok` is false and `last_error` is set; cashier 403. Tests: H `TestBackupStatus_T13_ReportsRunsAndSurvivesRestart`, `TestBackupStatus_T13_InitialValuesAndFailedRunKeepsLastSuccess`.
- AC-15: The backup settings SHALL stay out of `/setup`: `PUT /setup {"name":"Tienda Uno"}` leaves the folder unchanged, `PUT /backup/config` leaves `name` and `lan_access_enabled` unchanged, and `GET /setup` as a cashier has no key containing `backup`. Test: H `TestBackupConfig_T14_IndependentFromSetup`.
- AC-16: WHILE an admin is on `/settings` the page SHALL show the "Copias de seguridad" section, and on the dashboard a banner when the last backup failed or fell back; cashiers see neither. Tests: F `isAbsoluteWindowsPath` (`D:\Respaldos`, `C:/POS/copias`, `\\PC\share` accepted; `Respaldos`, `D:Respaldos`, `/tmp` rejected), `describeBackupStatus` (no attempt → `Aún no se ha hecho ninguna copia.`; ok → `Última copia: 31/10/2026 21:05 en D:\Respaldos`; fallback text with the failing and default folders), `shouldShowBackupBanner` (false never run or clean; true on fallback or failure). The rendering, 320 px layout and role visibility: review and `pnpm build` only, `gaps.md` US-089 M-3.
- AC-17: WHILE a register is open and cash sales of $10.000 are being committed, 5 backups SHALL each pass `integrity_check` = `ok` and `foreign_key_check` (empty), contain the open register, hold a `sales` count between the counts read just before and after, with 0 writer errors. Test: B `TestBackup_T9_ConsistentWhileSellingWithOpenRegister`.
- AC-18: Migration 23 SHALL add the `backup_*` columns to `business_config`; a v22 database with a $179.000 customer balance upgrades to 23 keeping the balance, with `backup_dir` `''` and `backup_last_ok` 0; a fresh database reports version 23; the two version pins moved 22 → 23. Tests: M `TestMigration23_T10_BackupColumnsOnV22Database`, `TestMigration23_T10_FreshDatabaseReportsVersion23`, `TestSchemaVersion_T15_MatchesLastMigrationAndFreshDB`.
- AC-19: `INSTALL.md` documents install and uninstall, signing in to Windows by itself, not closing the console window, the folder setting, 30 copies, copying one off the machine and closing `pos.exe` before `reset-data` (`INSTALL.md:19`, `:34`, `:63-78`); `reset.md` says `pos.exe` must be closed before `reset-data`. Review. `INSTALL.md` names the screen "Ajustes" (lines 40, 42, 76) while the sidebar and page title are "Configuración" (`frontend/components/Sidebar.tsx:142`, `frontend/app/settings/page.tsx:378`): roadmap R-170.

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
- [x] An admin sees an `Anular` action on a completed sale and confirms it before anything is sent; a cashier does not see the action (the API already answers 403).
- [x] After a void, the sale shows as `Anulada`, and the register summary and the customer balance shown on screen reflect it without a manual reload.
- [x] A sale with an accepted or pending electronic document, an already voided sale, or a server error shows the server message and changes nothing.
- [x] The contract decides, with the PO, what the action does for a sale whose register is already closed (roadmap R-112).

**Verified criteria (closed 2026-11-09):**

GATE 1 (PO, 2026-11-09): Q-1..Q-5 accepted as recommended: the entry point is only the dashboard `Ventas recientes` default list; voids on a closed register stay allowed with a warning (R-112 stays open); no void reason; the 409 and 403 texts are mapped in the frontend; after a void the dashboard is re-read. No production Go code changed. Frontend tests are in `frontend/lib/saleVoid.test.ts` (F) and backend tests in `internal/handler/sale_void_ui_test.go` (H). QA round 1: OK, 0 BLOCKER, 3 MINOR (spec delta applied at close; R-149). Gemini: clean. Spec: `spec/domain/caja.md` ("Movements", "Voids after the close"), `spec/domain/facturacion-dian.md` ("With DIAN off"). Manual check: `gaps.md` US-095 M-1.

- AC-1: WHILE the user is an admin, the dashboard `Ventas recientes` default view SHALL show `Anular` on every `completed` row, in the `Estado` cell; IF not admin or the row is `voided`, no button; `Ver anuladas` never shows it. Test: F `T-1`.
- AC-2: WHEN the admin taps `Anular` on sale #N, the dialog SHALL open and request `GET /sales/{N}`, showing `Cargando venta...` with `Anular venta` disabled until it answers; no void request is sent until `Anular venta`; `Cancelar`, Escape or the backdrop close it. Manual: `gaps.md` US-095 M-1.
- AC-3: WHEN the sale loads, the dialog SHALL show `¿Anular la venta #N?`, `Total`, one line per payment (`Efectivo`, `Transferencia`, `Tarjeta`, `Fiado`, the raw key if unknown, never `mixed`), `Cliente: <name>` and the stock and undo notice. Mixed $7.000 cash + $5.000 credit, Dora → `Total: $12.000`, `Efectivo $7.000`, `Fiado $5.000`, `Cliente: Dora`. Test: F `T-2` (five cases).
- AC-4: IF the sale's register is not the open one, the dialog SHALL add `Esta venta es de una caja ya cerrada; su cierre no cambia.` and, with cash C > 0, the refund hint (register open or none). Void on closed register 1 (`expected_close` 107000): stored values unchanged, `GET /registers/1` `expected` 100000, register 2 `expected` 50000, Dora 5000 → 0. Tests: F `T-3` (five cases), H `TestSaleVoidUI_T8_MixedVoidOnClosedRegister`.
- AC-5: WHEN the admin taps `Anular venta`, exactly one `DELETE /sales/{N}/void` with no body SHALL be sent; both buttons disabled and `Anulando...` while in flight. Test: F `T-5` (200 case); the disabled state by review.
- AC-6: WHEN the void answers 200, the dialog SHALL close, toast `Venta #N anulada`, re-read `/dashboard` and `/dashboard/today` and drop the cached `Ver anuladas` list. Sale #3 voided: `Caja` $137.000 → $130.000, `Cartera` $55.000 → $50.000, `recent_sales` `[3,2,1]` → `[2,1]`. Test: H `TestSaleVoidUI_T7_DashboardReflectsVoid`.
- AC-7: IF `GET /sales/{N}` fails, the dialog SHALL show the `api` error text (`No se pudo cargar la venta` when blank) and keep `Anular venta` disabled. Test: F `T-9` (text); the dialog by review.
- AC-8: IF the void fails, the dialog SHALL close and an error toast SHALL show the mapped text (`sale_has_accepted_invoice` → DIAN text, `forbidden` → `Solo un administrador puede anular ventas.`, others as is, empty → `No se pudo anular la venta`), and the dashboard is re-read. Tests: F `T-4`, F `T-5` (409, 403, network cases).
- AC-9: WHEN the 409 occurs, the system SHALL NOT call `POST /sales/{id}/void-with-credit-note`. Test: F `T-5` (409 case).
- AC-10: WHILE the viewport is 320 px, the dialog SHALL fit with no horizontal scroll and use only CSS variables. Manual: `gaps.md` US-095 M-1.
- AC-11: the backend SHALL stay as certified: 200 `voided`; 400 `la venta ya fue anulada`; 403 `forbidden`; 404; 409 `sale_has_accepted_invoice` with `requires_credit_note` true for an `accepted` or `pending` document (sale stays `completed`, stock unchanged). Test: H `TestSaleVoidUI_T6_ErrorBodies`.


## US-092 — Opening *fiado* balance per customer

**As** Don Carlos (owner),
**I want** to enter what each customer already owes from my paper notebook when I create them,
**so that** the POS ledger starts where the notebook ends.

**Estimated effort:** 1 day. **Impact:** High for go-live. **Changes a business rule** (`spec/domain/fiado.md`, balance formula), so the contract asks the PO how it is stored and shown.

**Acceptance criteria (draft):**
- [x] The admin can record an opening balance with a date for a customer; it adds to the balance and to the debt age, and *abonos* pay it down like any credit.
- [x] It does not count as a sale in any report, dashboard or register close.
- [x] It is recorded in the audit log and cannot be entered twice by accident.

**Verified criteria (closed 2026-11-09):**

GATE 1 (PO, 2026-11-09): Q-1..Q-5 accepted as recommended: a separate table `customer_opening_balances` (migration 22), one per customer, correction by an audited `PUT` with no delete, entry from the customer detail and from `Nuevo cliente`, master data for the US-093 reset, whole pesos from $1 to $100.000.000 with no future date. Q-6: tuteo (`Regístrala`). Contract discrepancies: US-096's invariant I-3 read credit − *abonos* (D-1, reworded to opening + credit − *abonos* in `fiado.md` and `caja.md`); stale `Balance` line refs (D-2, fixed); `Nuevo cliente` / `Editar` shown to cashiers (D-3, roadmap R-144). QA round 1: OK, 0 BLOCKER, 5 MINOR; 20 of 21 mutations caught; API smoke test passed; the UI was not smoke-tested (`gaps.md` US-092 M-1). Gemini diff round 1: clean. PO after QA: a correction may leave a negative balance (documented; UI warning R-141); malformed JSON for an unknown customer answers 400 before 404 (accepted, documented); the untested note trim (R-142) and the double toast (R-143) go to the roadmap; client vs server clock for "today" is documented only. Spec: `spec/domain/fiado.md` § Rule: customer balance, § Rule: debt age, § Rule: opening balance; `caja.md`, `rentabilidad.md`, `glosario.md`.

Backend tests are in `internal/handler/customer_opening_test.go` (H), `internal/repository/customer_opening_test.go` (R) and `internal/database/migrate_test.go` (M); frontend tests in `frontend/lib/openingBalance.test.ts` (F).

- AC-1, AC-8, AC-13: WHEN an admin posts `{amount, debt_date, note}` to `POST /customers/{id}/opening-balance` for an active customer with none, the system SHALL store it and answer 201; the balance, `/detail` `balance` and `opening_balance` SHALL carry it. $150.000 dated 45 days ago → 201, balance 150000, `credit_sales` `[]`. Test: H `TestCustomerOpening_T1_CreateAddsToBalanceAndDetail`.
- AC-8: WHILE a customer has an opening balance, every balance reader SHALL return opening + completed credit − *abonos*. Dora: opening $150.000 + credit $40.000 + voided credit $25.000 − *abono* $30.000 → 160000 in `Balance`, `List`, `TotalPortfolioBalance`, `Debtors`, `AgingReceivables` and `GetPOSSummary`; Beto: $20.000 − $50.000 → −30000, out of the portfolio. Test: R `TestCustomerOpening_T2_EveryBalanceQueryCarriesOpening`.
- AC-9: WHILE a customer has an opening balance, the debt age SHALL use the older of the oldest completed credit sale and the opening date. Opening only, 45 days → 45; plus a 60-day credit sale → 60; a 10-day sale plus a 45-day opening → 45; fully paid → `days_overdue` null. Test: R `TestCustomerOpening_T3_DebtAge`.
- AC-2, AC-3: IF the amount is not whole, `≤ 0` or above 100,000,000, or the date is missing, malformed, non-existent or in the future, THEN the system SHALL answer 400 with the exact message and store nothing and audit nothing; amount 1, amount 100000000 and today are accepted. Test: H `TestCustomerOpening_T4_Validation`.
- AC-4: IF the customer already has one, THEN `POST` SHALL answer 409 `este cliente ya tiene una deuda anterior registrada` and change nothing; 10 concurrent requests → one 201, nine 409, one row. Test: H `TestCustomerOpening_T5_OnePerCustomer`.
- AC-5, AC-7: WHEN an admin sends `PUT`, the system SHALL replace amount, date and note, set `updated_at`, and audit `opening_balance_update` with from/to values; $150.000 → $120.000 and 45 → 50 days; a customer with none → 404. Test: H `TestCustomerOpening_T6_Correction`.
- AC-6: IF the caller is a cashier → 403; non-numeric id → 400; unknown or inactive customer → 404 `cliente no encontrado`. Test: H `TestCustomerOpening_T7_RoleIDAndCustomerChecks`.
- AC-7: WHEN an opening balance is created, the system SHALL write one `opening_balance_create` audit row in the same transaction; IF the audit insert fails, THEN 500 and no opening row. Test: H `TestCustomerOpening_T8_Audit`.
- AC-10: WHEN *abonos* are posted against an opening balance, the existing rules SHALL apply. Opening $150.000: cash $100.000 → 50000; $60.000 → 400; delete → 409; transfer $50.000 → 0. Test: H `TestCustomerOpening_T9_AbonosPayItDown`.
- AC-12: WHEN an opening balance is recorded or corrected, no sales report, dashboard, register or close figure SHALL change: `total_revenue` 50000, `by_payment_method` cash 10000 / credit 40000, `sold_today` 50000, `credit_new_today` 40000, `expected` 60000 before and after, `expected_close` 60000. A cash *abono* of $20.000 against it → `credit_collected_today` 20000 and the next register's `cash_abonos` 20000. Tests: H `TestCustomerOpening_T10_NotASale`, `TestCustomerOpening_T11_AbonoAgainstOpeningIsRealMoney`.
- AC-11: The balance invariant SHALL read opening + completed credit − *abonos*: $50.000 credit − $35.000 *abonos* → 15000 (the US-091 number); plus a $100.000 opening → 115000. Test: R `TestCustomerOpening_T12_InvariantWithAndWithoutOpening`.
- Migration 22: a v21 database with a $179.000 balance upgrades to version 22 and keeps the balance; a second row for the same customer fails `UNIQUE`. Test: M `TestMigration22_T13_CustomerOpeningBalances`.
- AC-14, AC-15: amount and date helpers and the card label (`150.000` → 150000; tomorrow rejected; `$150.000 desde 25/09/2026`). Tests: F `T-14 parseAmountInput`, `T-14 isValidDebtDate`, `T-14 openingBalanceLabel`. The 320 px layout, role visibility and the `Nuevo cliente` second request were checked by review only (`gaps.md` US-092 M-1).
- AC-16: the audit page describes `Cliente #7: $150.000 desde 25/09/2026` and `Cliente #7: de $150.000 (25/09/2026) a $120.000 (20/09/2026)`. Test: F `T-15 describeOpeningBalanceAudit`.

## US-093 — Reset tool: clear test transactions, keep master data

**As** the Tech Lead installing the POS,
**I want** a single command that removes sales, registers, *abonos* and movements made during the dry run, keeping products, categories, customers and users,
**so that** the client starts with a clean history and the real catalog.

**Estimated effort:** 0.5 day. **Impact:** Medium (one-off, but done by hand it risks the catalog).

**Status:** closed 2026-11-10 in code and tests. The tool has **never been run against real data**; it was smoked only on a throwaway database. The PO manual check is `gaps.md` US-093 M-1.

**Acceptance criteria:**
- [x] A CLI utility next to `cmd/reset-password` takes a backup first, then clears transaction tables and restores stock by rule B (`stock = SUM(stock_entries)`), decided at GATE 1.
- [x] It refuses to run while the POS is running, and asks for a typed confirmation.
- [x] Physical `DELETE` is allowed here only because the rows are test data; this is the stated exception to the logical-delete rule (`spec/domain/reset.md`).

**Input from US-094 (PO Q-5, 2026-11-09):** the stock loaded by the catalog import must survive the reset. The import writes one `stock_entries` row with note `Importación de catálogo` per product with stock > 0, plus one `product_import` audit row (`spec/domain/catalogo.md` § Catalog import). Products, categories and those stock entries are master data: the reset keeps them and rebuilds stock from the entries.

**Verified criteria (closed 2026-11-10):**

GATE 1 (PO, 2026-11-10): Q-1..Q-12 accepted as recommended: stock rule B with an untraced-stock report; keep purchases and every `stock_entries` row; keep all users and settings; restart ids at 1 for the cleared tables; partial audit clear plus one `data_reset` row; dry-run by default and typed `BORRAR <N>`; second-run guard with `--force`; no UI button; Spanish tool output; `cmd/reset-data` + `internal/resetdata`; backup next to `data.db`, never pruned; test products, customers and users survive. Q-13 (three verified gaps found in review): the fiscal guard refuses on any `accepted`, `draft` or `pending` electronic document whatever `fiscal_config.ambiente` says (the table has no `ambiente` column), with `--discard-fiscal-documents` as the only override; refuse before the backup when no active admin exists; rule B covers inactive products. Contract discrepancies D-1..D-6: stock cannot be rebuilt purely from entries (manual form, R-153); the ticket number is the sale id, so the `sqlite_sequence` counters restart; the SQLite exclusive lock is not kept after COMMIT; the daily backup prunes `<db>.backup-*`; `electronic_documents` statuses; an active admin is needed for the audit row. A contract miscount (24 instead of 25 user tables) was fixed after the implementer's first run.

Results: QA round 1 OK (0 BLOCKER, 5 MINOR); Gemini diff round 1 CLEAN (input about 90 KB, possibly truncated); `go build`, `go test ./...`, `pnpm build`, `pnpm test` (240) and `pnpm lint` green. Smoke on a throwaway database with the real binaries (port 18099): dry-run, refusal with the POS up, wrong and empty confirmation, apply, second run with and without `--force`, missing file. The destructive path was never run against real data.

Backend tests are in `internal/resetdata/reset_test.go` (R), `guard_test.go` (G), `backup_test.go` (B), `confirm_test.go` (C), `exitcode_test.go` (E), `cmd/reset-data/main_test.go` (CLI) and `internal/database/schemaversion_test.go` (D). Fixture F: 4 sales, 2 registers, 1 abono, opening balance $150.000, 4 products, 4 stock entries.

- AC-1: WHEN the tool runs without `--apply`, the system SHALL print the per-table counts and stock changes, change nothing and create no backup, exit 0. On F: sales 4, `customer_payments` 1, registers 2, `audit_log` 3 of 6, stock P1 106→110, P2 90→100, P4 20→0 (3 change, 1 untraced: 20 + 5 − 0 = 25). Tests: R `TestPlan_T1_CountsAndNoWrites`; CLI `TestRun_T1_DryRunDefault`.
- AC-2, AC-15: WHEN `--apply` completes, the 11 transaction tables SHALL be empty and `audit_log` SHALL hold 3 kept rows plus one `data_reset` (4); users 2, products 4, customers 2, `stock_entries` 4, purchases 1, opening balances 1 and `business_config` (with `jwt_secret`), `fiscal_config` and password hashes SHALL be unchanged. Tests: R `TestApply_T2_AfterCountsAndKeptData`; CLI `TestRun_T2_ApplyWithConfirmFlag`.
- AC-3: WHEN `--apply` completes, `products.stock` of every `product`-type product (inactive included) SHALL equal the sum of its entries: P1 110, P2 100, P4 0, service P3 stays 0; a product with `active = 0`, stock 12 and one entry of 8 → 8. Tests: R `TestApply_T3_StockRuleB`, `TestApply_T4_StockRuleVariants`.
- AC-4: WHEN `--apply` completes, the `sqlite_sequence` rows of the cleared tables SHALL be deleted and no others: the next sale, register and order get id 1, the next `stock_entries` and `products` id is 5. Test: R `TestApply_T5_SequencesRestart`.
- AC-5: WHEN `--apply` completes, every customer balance SHALL equal its opening balance: Dora $170.000 → $150.000, Beto $0. Test: R `TestApply_T6_BalancesEqualOpening`.
- AC-6: WHEN `--apply` runs, the system SHALL write and verify `<db>.pre-reset-YYYYMMDD-HHMMSS` (integrity check ok, same counts, name not matching `*.backup-*`) before any delete; IF the copy fails, THEN delete nothing, exit 3. Tests: B `TestBackup_T7_CopyIsVerifiedAndNamedAside`, `TestBackup_T7_FailureMapsToExit3AndDeletesNothing`.
- AC-7: IF a TCP connection to `127.0.0.1:<port>` succeeds, THEN refuse with exit 1 before the backup; IF `BEGIN EXCLUSIVE` is busy, THEN refuse and roll back. Tests: G `TestGuard_T8_PortProbe`, `TestGuard_T8_LockBusyRefuses`; CLI `TestRun_T8_PortInUseRefusedNoBackup`. The busy lock is tested in-process with a writer holder; the idle-connection case across processes was checked by hand on Windows during the contract, and a live `pos.exe` on port 18099 was refused in the QA smoke (smoke only).
- AC-8: IF an electronic document is `accepted`, `draft` or `pending`, regardless of `ambiente`, THEN refuse with exit 1 before the backup; `--force` does not override it, `--discard-fiscal-documents` does and records the discarded counts; `rejected` and `cancelled` never block; dry-run prints the reason and exits 0. Tests: G `TestGuard_T9_AcceptedDocumentRefuses`, `TestGuard_T9_DiscardFlagClearsAndRecords`, `TestGuard_T9_RejectedAndCancelledClearWithoutFlag`, `TestGuard_T9b_InFlightDocumentsRefuse`; CLI `TestRun_T9_ApplyWithAcceptedDocRefusedNoBackup`, `TestRun_T9b_DryRunWithAcceptedDocExit0`.
- AC-9: IF `audit_log` already has a `data_reset` row, THEN refuse with exit 1 unless `--force`. Tests: R `TestApply_T10_AuditRowAndSecondRun`; CLI `TestRun_T10_SecondRunNeedsForce`.
- AC-10: WHEN `--apply` passes the guards, the system SHALL ask for exactly `BORRAR <N>` (F: `BORRAR 4`), before the backup; `borrar 4`, `BORRAR 5`, empty input → cancelled, nothing deleted; `--confirm` supplies it. Tests: C `TestConfirm_T11_TypedConfirmation`; CLI `TestRun_T11_WrongConfirmationExit1`, `TestRun_T11_ApplyWithTypedStdin`.
- AC-11: WHILE applying, every delete, the stock rebuild, the sequence deletes and the `data_reset` insert SHALL run in one `BEGIN EXCLUSIVE` transaction with `foreign_key_check` before commit; IF a statement fails, a foreign key is violated or the counts differ from the backup's, THEN roll back, exit 4. Tests: R `TestApply_T12_FailingStepRollsBack`, `TestApply_T12_ForeignKeyViolationRollsBack` (a kept product with `category_id = 99`), `TestApply_T13_FingerprintMismatchAborts` (`Options.BeforeLock` + `Options.Fingerprint`).
- AC-12: WHEN `--apply` completes, exactly one `data_reset` row (user = lowest-id active admin, entity `system`, entity_id 0, compact JSON with `"sales":4` and the backup name) SHALL be written in that transaction. Test: R `TestApply_T10_AuditRowAndSecondRun`.
- AC-13: The tool SHALL exit 0 on success or dry-run, 1 on a refusal, 2 on a usage error (a missing `data.db` creates no file), 3 on a backup failure, 4 on a database error. Tests: E `TestExitCode_T14_Table`, `TestOpen_T14_MissingFileIsUsageErrorAndCreatesNothing`; CLI `TestRun_T14_MissingFileExit2`, `TestRun_T14_BadFlagExit2`.
- AC-14: IF `schema_migrations` differs from `database.SchemaVersion()` (22 today), THEN refuse with exit 1; the tool never migrates. Tests: E `TestCheckSchemaVersion_T14`; CLI `TestRun_T14_SchemaVersionMismatchExit1`; D `TestSchemaVersion_T15_MatchesLastMigrationAndFreshDB`.
- AC-16: IF no user is an active admin, THEN refuse with exit 1 before the backup; the `data_reset` user is the lowest-id **active** admin. Tests: G `TestGuard_T16_NoActiveAdminRefuses`, `TestGuard_T16_LowestActiveAdminIsUsed`; CLI `TestRun_T16_NoActiveAdminRefusedNoBackup`.

Covered by smoke only (no automated test): a live `pos.exe` refused through the port probe (AC-7); the 1-second advance of the backup name on a quick `--force` re-run (the name can run seconds ahead of the clock, `reset.md`); the orphan `*.pre-reset-*` file left by the three refusals that need the backup first (lock busy, in-transaction guard recheck, fingerprint mismatch).

Spec: `spec/domain/reset.md` (new), `catalogo.md`, `fiado.md`, `caja.md`, `facturacion-dian.md`, `project-docs/README.md`.

## US-094 — Product catalog import from CSV

**Status:** approved by the PO on 2026-11-08 (added on 2026-11-07 after verifying that no import exists). PO direction: the import must also serve a shop that is moving from another POS program, so the input format is decided in the contract. Candidates are a CSV with column mapping (most POS programs and spreadsheets can export one) or a SQL/SQLite dump of the other program. The analyst recommends one format with its trade-off at GATE 1.

**As** Don Carlos (owner),
**I want** to load my product list from a spreadsheet,
**so that** I do not type hundreds of products by hand before opening.

**Estimated effort:** 1.5 days. **Impact:** High for go-live if the catalog has more than about 50 products.

**Acceptance criteria (draft):**
- [x] The admin uploads a CSV with name, barcode, category, unit (`unit`, `kg`, `lb`, `liter`), cost, price and stock; a downloadable template is provided.
- [x] Rows are validated first and errors are shown per row before anything is written; the import is all-or-nothing.
- [x] Missing categories are created; a duplicate barcode is reported, not silently overwritten.

**Verified criteria (closed 2026-11-09):**

The catalog import was implemented per contract (CSV with auto-recognition and manual mapping, 2 MB / 5,000-row limit, atomic transaction with `stock_entries` and `audit_log`, cashier gating). GATE 1 (PO, 2026-11-09): Q-1..Q-6 accepted as recommended (CSV with mapping; no update of existing products, roadmap R-154; duplicate names are an error; min stock, IVA 0/5/19 and internal code imported; one stock entry per product plus one audit row; limits 2 MB / 5,000 rows). QA round 1: OK, 0 BLOCKER, 5 MINOR (R-154, R-156 and R-157 on the roadmap with R-150..R-153 and R-155 from the contract discrepancies); Gemini diff clean; smoke test on the compiled binary passed. Manual browser checks: `gaps.md` US-094 M-1. Spec: `spec/domain/catalogo.md`.

Tests: 23 Go test functions covering contract cases T-1..T-22 (`TestProductImport_T12_*` has two functions) and 6 Vitest `describe` groups F-1..F-6 (10 `it` cases). Handler tests (H) are in `internal/handler/product_import_test.go`, service tests (S) in `internal/service/product_import_test.go`, frontend tests (F) in `frontend/lib/productImport.test.ts`. A `SQLITE_BUSY` answer (503) is covered by no test; it reuses `mapBusy` (`internal/repository/order.go:53-59`).

- AC-1: WHEN an admin posts to `POST /products/import`, the system SHALL process it; a cashier gets 403 `forbidden`, no token 401. Route in the admin products group (`internal/handler/handler.go:158`). Test: H `TestProductImport_T1_AdminOnly`.
- AC-2: IF the body exceeds 3 MiB or the decoded file exceeds 2 MiB, THEN 413 `el archivo supera 2 MB`; malformed base64 or an empty file → 400 `archivo inválido`; nothing is written. Test: H `TestProductImport_T2_SizeAndMalformedLimits`.
- AC-3, AC-4: UTF-8 BOM stripped; UTF-8 or Windows-1252 detected and reported in `encoding`; delimiter `;`, `,` or tab reported in `delimiter`; a UTF-16 BOM and an `.xlsx` (`PK`) answer 400 with their messages. Test: S `TestProductImport_T3_EncodingAndDelimiter`.
- AC-5: WHEN rows are read, the system SHALL number them by physical line (blank lines skipped, multi-line cells counted), accept short rows, flag non-blank cells beyond the header, and reject more than 5,000 rows or no rows. Tests: S `TestProductImport_T17_RowWidth`, S `TestProductImport_T18_PhysicalLineNumbers`, H `TestProductImport_T16_RowLimits` (5,000 rows commit; 5,001 → 400; header only → 400).
- AC-6: Without `mapping`, headers are auto-mapped by normalized synonyms, leftmost wins, the rest go to `ignored_headers`; the template header maps all 10 fields. Test: S `TestProductImport_T4_AutoMapping`.
- AC-7, AC-8: An explicit `mapping` replaces the automatic one; unknown field, missing header and a header used twice answer 400; unmapped `name` or `sale_price` → 400 `faltan columnas obligatorias: nombre, precio de venta`. Test: H `TestProductImport_T5_ManualMapping`.
- AC-9, AC-16, AC-18: Name required and at most 200 characters; duplicate normalized name in the file (`producto repetido en la fila <n>`) or equal to an active product (`ya existe un producto con este nombre`), while a deleted product's name does not block; internal code at most 64 characters. Test: H `TestProductImport_T11_NameAndCodeRules`.
- AC-10: Money in whole pesos: `$`, `COP`, `.`/`,` thousands, `,00` decimals accepted; cents, negative (`-2.500` → `el valor no puede ser negativo`), invalid text, above $100.000.000 and a sale price of 0 or blank rejected; blank cost is 0. Test: S `TestProductImport_T6_MoneyParsing`.
- AC-11: Quantities: comma or dot decimals, at most 3 decimals, at most 1.000.000, no negatives, whole for `unit`, blank is 0. Test: S `TestProductImport_T7_QuantityParsing`.
- AC-12: Unit synonyms map to `unit`, `kg`, `lb`, `liter`; anything else rejected. Test: S `TestProductImport_T8_UnitParsing`.
- AC-13: IVA 0, 5, 19 with optional `%`, blank is 0, anything else rejected. Test: S `TestProductImport_T9_TaxParsing`.
- AC-14, AC-15: Barcode trimmed, `="…"` unwrapped, scientific notation and over 64 characters rejected; a barcode repeated in the file (`código de barras repetido en la fila 4`) or already in `products`, active or deleted (`… ya existe en el producto «Avena» (eliminado)`), is reported and nothing existing changes. Test: H `TestProductImport_T10_BarcodeRules`.
- AC-17: Categories matched ignoring case and accents, lowest id wins among several matches, new ones listed in `new_categories` in order of first appearance and created once. Tests: H `TestProductImport_T12_CategoryMatching`, H `TestProductImport_T12_CategoryLowestIDWins`.
- AC-19, AC-20: A dry run answers 200 with the report (errors capped at 200, `error_count` the full count, preview of 5) and writes nothing; a commit re-validates and answers 422 `el archivo tiene <n> errores; no se importó nada`. Tests: H `TestProductImport_T13_DryRunReportAndRejectedCommit`, S `TestProductImport_T19_ErrorListCappedAt200`, H `TestProductImport_T20_CommitRevalidates` (barcode taken between dry run and commit → 422).
- AC-21, AC-23: A valid commit writes categories, products (consecutive ids in file order), one `Importación de catálogo` stock entry per product with stock > 0 and one `product_import` audit row, and answers 201; the products and categories are then returned by `GET /products`, `GET /products/barcode/{barcode}` and `GET /categories`. Test: H `TestProductImport_T15_CommitWritesEverything`.
- AC-22: IF a write fails, THEN everything rolls back (`error importando productos`, 500); a barcode race answers 409 naming the barcode and a category race answers 409 naming the category, each without the other's text. Tests: H `TestProductImport_T14_RollbackOnFailure`, H `TestProductImport_T21_CategoryRace`, H `TestProductImport_T22_BarcodeRace`. The 503 on `SQLITE_BUSY` has no test.
- AC-24: `Importar productos` button and modal are rendered only when `canImportProducts(getUser())` is true (`frontend/app/inventory/page.tsx:504`). Test: F `F-6 canImportProducts`; the rendering itself by review and the QA smoke on the binary.
- AC-25: The file is read as bytes and base64-encoded, and the mapping payload omits `(no importar)`. Tests: F `F-2 bytesToBase64`, F `F-3 buildMappingPayload`; the modal flow by review.
- AC-26: The error table text and the commit gate. Test: F `F-4 canCommit, formatImportError, moreErrorsText`.
- AC-27: The commit sends `dry_run: false`, shows the toast and reloads; an error keeps the modal open. Review and QA smoke on the binary (no unit test); manual check `gaps.md` US-094 M-1.
- AC-28: `Descargar plantilla` builds `plantilla-productos.csv` with BOM, `;`, CRLF, the 10-column header and two example rows. Test: F `F-1 buildTemplateCSV`.
- AC-29: At 320 px the tables scroll horizontally and colors come from CSS variables. Review only; manual check `gaps.md` US-094 M-1.
- AC-30: A `product_import` audit row reads `Importación de productos` and `<n> productos importados de «<file_name>»`. Test: F `F-5 audit description`.

## US-096 — Money QA layer 1: property-based invariant tests over random sequences of money operations

**Status:** prioritized by the PO on 2026-11-08 (roadmap R-130; layers 2 and 3 are R-131 and R-132). Contract approved at GATE 1 (PO, 2026-11-08). First drafted as US-095; renumbered because US-095 is the admin void button on the unmerged branch `us/US-091`.

**As** Don Carlos (owner),
**I want** the money rules checked against thousands of random shifts, not only one scripted day,
**so that** a combination nobody thought of cannot make the register close or a *fiado* balance wrong.

**Estimated effort:** 1.5 days. **Impact:** High. Complements US-091 (one fixed shift), which stays as the readable regression example.

**Today (verified 2026-11-08):** no property or fuzz test exists. Payments may differ from the stored sale total by up to 0.5 through the UI and by up to 1.21 through the API (`internal/service/sale.go:84-88`, `:147`). Stored amounts are not whole pesos (taxed lines, kg quantities, *abonos* such as 0.4; roadmap R-133). The server accepts a negative opening amount, a negative `cash_in`, a zero `cash_out` and a negative sale discount, which the UI blocks (`internal/service/register.go:46-68`, `:217-231`, `internal/service/sale.go:118-168`). It also accepts a sale discount equal to the subtotal paid with $0,30 (total 0).

**Acceptance criteria (draft; the contract refines them):**
- [x] A state-machine property test drives random sequences of money operations through the HTTP routes and, after every step, checks: payments vs sale total; register expected cash against an independent model; customer balance agreeing across every query; exact void reversal of register, balance and stock; and that a rejected request changes nothing.
- [x] Failures shrink to a minimal sequence and reproduce from their seed; the suite runs in CI in under 40 s (budget raised from 30 s by PO decision).
- [x] Fuzz targets on the sale and *abono* amount input never get a 5xx and never store an inconsistent sale.
- [x] The whole-peso invariant and the server-side validation gaps are handled as decided by the PO at GATE 1.
- [x] A sale whose server-recalculated total is 0 or less is rejected (400 `el total de la venta debe ser mayor a cero`) and stores nothing; e.g. a discount equal to the subtotal paid with $0,30.

**Verified criteria (closed 2026-11-08):**

Contract findings and server-side validation gaps resolved: the suite drives random sequences through `h.Routes()` with real JWT tokens on a file-backed database under `t.TempDir()`. Six validation gaps were fixed with clear 400 responses: negative opening float (AC-12), non-positive register movements (AC-13), negative sale-level discounts (AC-14), payments exceeding server-recalculated total by > 0.5 (AC-15), sale discounts exceeding subtotal (AC-16), and non-positive recalculated sale totals (AC-17). The whole-peso invariant is verified in CI on zero-tax unit products (AC-7) and confirmed red on the full domain under build tag `moneyexact` (AC-11, R-133). In QA, T-1 + T-2 wall time ran in ~32.4 s; the PO approved raising AC-8 budget to <= 40 s to keep the 100-sequence coverage frozen. All 3 mutations caught. Seed corpus fuzz targets PASS. Four minor items opened on roadmap (R-137..R-140). Spec updated: `spec/domain/caja.md`, `spec/domain/fiado.md`.

- AC-1 (rapid state machine, 100 sequences × ~30 steps): `TestMoneyProperties_StateMachine` (`internal/handler/money_props_test.go`).
- AC-2 (I-1, payments vs total `|Σ payments − total| ≤ 0.5`): `TestMoneyProperties_StateMachine`, `TestMoneyModel_T0_ScriptedSequence`, `FuzzSaleAmounts`.
- AC-3 (I-2, register expected cash vs oracle model within ±0.01): `TestMoneyProperties_StateMachine`, `TestMoneyModel_T0_ScriptedSequence`.
- AC-4 (I-3, customer balance agreement across `GET /customers/{id}/balance`, `GET /customers`, `pos-summary`, `total_portfolio`): `TestMoneyProperties_StateMachine`, `TestMoneyModel_T0_ScriptedSequence`.
- AC-5 (I-4, exact void reversal of cash in register, credit in balance, and product stock): `TestMoneyProperties_StateMachine`, `TestMoneyModel_T0_ScriptedSequence`.
- AC-6 (I-5, rejected step changes nothing across all tables; asserts 400, 403, 409, 500): `TestMoneyProperties_StateMachine`, `TestMoneyModel_T3_Rejections`.
- AC-7 (I-6, positive amounts, integer amounts on zero-tax unit products): `TestMoneyProperties_StateMachine`, `TestMoneyProperties_WholePesoZeroTax`.
- AC-8 (reproducible seed, shrunk counterexample, wall time <= 40 s): `TestMoneyProperties_StateMachine`, `TestMoneyProperties_WholePesoZeroTax` (~32.4 s).
- AC-9 (mutations caught: register cashAbonos m1, customer completed m2, sale stock restore m3): 3/3 mutations caught by `TestMoneyProperties_StateMachine`.
- AC-10 (fuzz targets seed corpus: no panics, no 5xx, invariants hold on 201, clean rejection on 4xx): `FuzzSaleAmounts`, `FuzzAbonoAmount` (`internal/handler/money_fuzz_test.go`).
- AC-11 (whole-peso full domain fails with counterexample under build tag): `TestMoneyProperties_WholePeso` (`internal/handler/money_wholepeso_test.go`, run with `go test -count=1 -tags moneyexact ./internal/handler/ -run 'TestMoneyProperties_WholePeso$'`).
- AC-12 (reject opening amount < 0 with 400): `TestMoneyValidation_T7_NegativeOpeningAmount` (`internal/handler/money_validation_test.go`).
- AC-13 (reject movement amount <= 0 with 400): `TestMoneyValidation_T8_MovementAmountMustBePositive`.
- AC-14 (reject sale discount < 0 with 400): `TestMoneyValidation_T9_NegativeSaleDiscount`.
- AC-15 (reject payments differing from recalculated total > 0.5 with 400): `TestMoneyValidation_T10_PaymentsVsRecalculatedTotal`.
- AC-16 (reject sale discount > subtotal with 400): `TestMoneyValidation_T11_DiscountAboveSubtotal`.
- AC-17 (reject recalculated total <= 0 with 400): `TestMoneyValidation_T12_TotalMustBePositive`.


## US-097 — Singular/plural wording in the catalog import UI

**Status:** picked up from roadmap R-157 (US-094 QA MINOR 3). Level L. Contract approved at GATE 1 (PO, 2026-11-09): wording as recommended, backend singular message included.

**As** Don Carlos (owner),
**I want** the import screen to read `1 producto`, not `1 productos`,
**so that** the first thing I see while loading my catalog does not look like a bug.

**Acceptance criteria (verified 2026-11-09):**
- [x] AC-1: WHEN a valid report has `valid_rows = 1`, the system SHALL show `1 producto listo para importar`; for any other count `<n> productos listos para importar`.
- [x] AC-2: WHEN a valid report has `valid_rows = 1`, the commit button SHALL read `Importar 1 producto`; for `n >= 2`, `Importar <n> productos`.
- [x] AC-3: WHILE no file is picked, or the report is invalid, or it has 0 valid rows, the commit button SHALL read plain `Importar` and be disabled (`Importando...` while committing).
- [x] AC-4: WHEN a commit creates 1 product, the toast SHALL read `Se importó 1 producto`; for `n >= 2`, `Se importaron <n> productos`.
- [x] AC-5: WHEN exactly 1 error is not shown, `moreErrorsText` SHALL return `y 1 error más`; for `k >= 2`, `y <k> errores más`; for `k <= 0`, `''`.
- [x] AC-6: WHEN an audit row has `products_created = 1`, the description SHALL read `1 producto importado de «<file>»`; otherwise `<n> productos importados de «<file>»`.
- [x] AC-7: WHEN a commit is rejected with exactly 1 row error, the 422 message SHALL read `el archivo tiene 1 error; no se importó nada`; for `n >= 2` unchanged.

**Verified criteria:**
- AC-1..AC-4: `frontend/lib/productImport.test.ts`, block `R-157 T-1 count wording helpers` (`readyText: 1 is singular, 3 is plural`, `commitLabel: 1 is singular, 3 is plural`, `commitLabel: plain Importar with no report, an invalid report or 0 valid rows`, `importedToast: 1 is singular, 3 is plural`). The modal has no component test; it only calls these helpers.
- AC-5: `R-157 T-2 moreErrorsText singular` (`one hidden error reads y 1 error más; 100 and 0 unchanged`).
- AC-6: `R-157 T-3 audit description singular` (`1 product reads singular; 3 and 0 plural`).
- AC-7: `internal/service/product_import_test.go` `TestProductImport_R157_T4_RejectedCommitSingularMessage` (1 bad row → `el archivo tiene 1 error…`) and `TestProductImport_R157_T4_RejectedCommitPluralMessageUnchanged` (2 bad rows → `el archivo tiene 2 errores…`).

**Results:** QA round 1 found no code issue (its BLOCKER was the spec delta, applied at close; 2 MINOR doc items fixed at close); Gemini diff review clean. `go test ./...` fails only the pre-existing `TestCustomerOpening_T10_NotASale` and `_T11_AbonoAgainstOpeningIsRealMoney` after 19:00 America/Bogota (roadmap R-158, root cause R-05).


## US-098 — Day windows and stored timestamps agree on one zone

**Status:** picked up from roadmap R-158 and R-05 (same root cause), 2026-11-10. Level L. Done 2026-11-10 (pipeline ID R-158). Approved at GATE 1 (PO, 2026-11-10): Q-1..Q-6 as recommended.

**As** Don Carlos (owner),
**I want** the dashboard "today", the sales report and the assistant's "hoy/ayer" to include every sale made until midnight,
**so that** the 7 p.m. to midnight rush is not missing from my numbers.

**Problem (verified 2026-11-09 and 2026-11-10):** rows get `created_at` from `DEFAULT CURRENT_TIMESTAMP` (UTC text, no zone suffix) while windows are built as local-midnight `time.Time` values that the driver binds with their offset (`-05:00`) and SQLite compares as text, so from 19:00 America/Bogota the window omits the evening's sales. Even bounds converted with `.UTC()` mis-bucket a row at the exact boundary second (`+00:00` suffix).

**Acceptance criteria (verified 2026-11-10):**
- [x] AC-1: WHEN `GET /dashboard/today` is answered at 2026-11-09 20:30 Bogota, `sold_today` SHALL include a sale made at 20:18:41 (day total $87.000 over 4 sales, with sales at 00:00:00, 18:59, 20:18:41 and 23:59:59) and exclude one made at 00:00:00 on 2026-11-10.
- [x] AC-2: WHEN `GET /dashboard` is answered at the same instant, `today` SHALL be $87.000 / 4 and `yesterday` $5.000 / 1.
- [x] AC-3: WHEN `GET /reports/sales` is asked for `[2026-11-09 00:00, 2026-11-10 00:00)` Bogota, in `-05:00` or `Z` form, the system SHALL return $87.000 / 4, the row exactly at `from` included and the row exactly at `to` excluded.
- [x] AC-4: WHEN two contiguous day windows are queried, no sale SHALL be counted twice or dropped ($87.000 + $3.000 = $90.000).
- [x] AC-5: WHEN an abono, cash-out movement, expense, purchase or audit row is stored at 20:18:41 Bogota, it SHALL belong to that Bogota day in every range query.
- [x] AC-6: WHEN the assistant answers "hoy", "ayer" and "este mes" at 2026-11-09 20:30 Bogota, the figures SHALL include that evening's sales ("este mes" = $72.000 / 4).
- [x] AC-7: WHEN sales are grouped by day, a sale at 20:18:41 Bogota SHALL fall in its Bogota date.
- [x] AC-8: WHILE the database holds rows written by the previous version, results SHALL be identical to new rows with no data migration, and new `created_at` text SHALL keep the `YYYY-MM-DD HH:MM:SS` format.
- [x] AC-9: WHILE the server zone is UTC, "today" SHALL be the UTC day.
- [x] AC-10: aging rows SHALL appear with the right `days_overdue`; a legacy unparseable date stays excluded (closes R-05's aging note).
- [x] AC-11: the low-stock 7-day window SHALL compare UTC against UTC (a sale 7 days 1 hour old is not counted).
- [x] AC-12: a range bound of any offset SHALL be compared as its UTC instant; the boundary row belongs to the window that starts there.
- [x] AC-13: `TestCustomerOpening_T10_NotASale` and `_T11_AbonoAgainstOpeningIsRealMoney` SHALL pass at any wall-clock hour (pinned clock and pinned `created_at`).

**Decisions (GATE 1, PO, 2026-11-10):** design A (range bounds bound as UTC text `YYYY-MM-DD HH:MM:SS` by `dbTime`, no data migration), no production or pilot data to preserve, "today" is the server's local zone, `LowStock` fixed here, injectable `Handler.now` and `AssistantService.now`, new `spec/domain/tiempo.md`. Spec: `spec/domain/tiempo.md` (new), pointers in `fiado.md`, `caja.md`, `rentabilidad.md`, row in `project-docs/README.md`.

**Verified criteria:**
- AC-1: `internal/handler/dashboard_test.go` `TestGetDashboardToday_T1_EveningSaleIsToday`.
- AC-2: `TestGetDashboard_T2_TodayAndYesterdayAtEvening` (same file).
- AC-3, AC-12 (route level): `TestGetSalesReport_T3_WindowBoundsAnyOffset` (same file); AC-12 (repository level): `internal/repository/report_test.go` `TestReportRepo_T13_BoundInstantBelongsToOneWindow`, `TestReportRepo_T22_BuildDateWhereIsHalfOpen`.
- AC-4: `internal/repository/report_test.go` `TestReportRepo_T4_ContiguousDayWindows`.
- AC-5: `internal/repository/metrics_test.go` `TestMetricsRepo_T5_EveningFlowsBelongToTheLocalDay` (abono, cash-out, expense); `internal/repository/list_filters_utc_test.go` `TestListFilters_T6_EveningRowsBelongToTheLocalDay` (purchase, audit, expense lists).
- AC-6: `internal/service/assistant_test.go` `TestAssistant_T7_DayWindowsAtEvening` ("hoy" 87000, "ayer" 5000, "este mes" 72000 / 4).
- AC-7: `internal/repository/report_test.go` `TestReportRepo_T8_GroupByDayLocalPeriod` (strict branch only on a UTC-5 host; roadmap R-159, R-163).
- AC-8: `internal/handler/dashboard_test.go` `TestPostSales_T9_StoredTimestampFormatAndMigrationCount` (stored text format, 22 migrations; no migration added).
- AC-9: `internal/repository/metrics_test.go` `TestMetricsRepo_T10_ServerZoneUTCWindow` (74000 / 4).
- AC-10: `TestMetricsRepo_T11_AgingDaysOverdueAndLegacyText` (same file); existing `TestMetricsRepo_T30_AgingExcludesNullDaysOverdue`.
- AC-11: `TestMetricsRepo_T11_LowStockSevenDayWindow` (same file).
- AC-13: `internal/handler/customer_opening_test.go` `TestCustomerOpening_T12_PinnedClockAnyHour` (pins at 20:18:41, 18:59 and 23:59:59 Bogota), with `TestCustomerOpening_T10_NotASale` and `TestCustomerOpening_T11_AbonoAgainstOpeningIsRealMoney` now clock-independent.
- Existing tests tightened by corrected UTC fixtures: `TestMetricsRepo_T2_SoldHalfOpenMidnight`, `TestAssistant_T9_PeriodLabelNamedInText`.

**Results:** QA round 1 OK (0 BLOCKER, 3 MINOR, 1 NOTE: the spec delta, applied at close; MINOR 1 is roadmap R-159, the other two are style/defensive and not tracked); Gemini diff review clean. `go build ./...`, `go test ./...` and frontend `pnpm test` (240 tests), `pnpm lint` and `pnpm build` green; the real-binary smoke ran at 14:19 Bogota, before the 19:00 gap, so the evening behaviour is proven by the pinned-clock tests.

---

## US-099 — The pre-migration safety copy gets its own name and survives the daily backup

**Status:** picked up from roadmap R-168 (go-live item K2), 2026-11-12. Level M. Done 2026-11-12 (pipeline ID R-168). Approved at GATE 1 (PO, 2026-11-12): Q-1..Q-3 as recommended.

**As** Don Carlos (owner),
**I want** the copy the POS takes right before an update to stay on disk after the daily backup runs,
**so that** a bad update never costs me my *fiado* (store credit) ledger.

**Problem (verified 2026-11-12):** `backupBeforeMigration` writes `<db>.backup-YYYYMMDD` (`internal/database/migrate.go:140`), the same name as the daily backup (`internal/database/backup.go:120`). `database.New` migrates first and the startup backup then overwrites that file the same day (reproduced: the copy held schema v22 before, v23 after).

**Acceptance criteria:**
- [x] AC-1: WHEN pending migrations exist on a database at version N > 0, the system SHALL write `<db>.pre-migration-N-YYYYMMDD` (local day) before any migration runs.
- [x] AC-2: WHEN the daily backup runs the same day, the pre-migration copy SHALL stay byte-identical.
- [x] AC-3: the daily prune SHALL never delete a `<db>.pre-migration-*` file.
- [x] AC-4: WHILE another backup folder is configured, the pre-migration copy SHALL stay beside `data.db`.
- [x] AC-5: two upgrades the same day from different versions SHALL leave two files.
- [x] AC-6: a same-day retry from the same version SHALL replace the earlier file of that name.
- [x] AC-7: a brand-new database, `:memory:` or no pending migration SHALL write no copy (unchanged).
- [x] AC-8: IF the copy fails, THEN startup SHALL stop with `backup before migration` and run no migration (unchanged).
- [x] AC-9: spec, `INSTALL.md` and `.gitignore` SHALL describe the new name.

**Decisions (GATE 1, PO, 2026-11-12):** name `<db>.pre-migration-<from>-YYYYMMDD` with the from-version unpadded; copies are never pruned (delete by hand); `.gitignore` gets `data.db.pre-migration-*` and `data.db.pre-reset-*`. Spec: `spec/domain/backup.md`.

**Verified criteria:**
- AC-1: `internal/database/migrate_test.go` `TestPreMigration_T1_CopySurvivesDailyBackup` (v22 copy named `.pre-migration-22-<day>`), `TestPreMigration_T8_DateUsesLocalZone` (21:30 Bogota gives the local day, not the UTC day).
- AC-2: `TestPreMigration_T1_CopySurvivesDailyBackup` (size and SHA-256 unchanged after `RunBackup`; daily file at v23).
- AC-3: `internal/database/backup_test.go` `TestBackup_T2_PruneIgnoresForeignFiles` (pre-migration names among the kept foreign files; `isBackupName` false).
- AC-4: `TestBackup_PreMigrationCopyStaysBesideDB` (same file).
- AC-5: `TestPreMigration_T4_DifferentVersionsSameDayKeepBoth`.
- AC-6: `TestPreMigration_T5_SameVersionSameDayReplaces`.
- AC-7: `TestPreMigration_T6_NoCopyWhenNewOrUpToDate`.
- AC-8: `TestPreMigration_T7_FailedCopyAbortsBeforeMigrating` (error contains `backup before migration`, version stays 22).
- AC-9: `spec/domain/backup.md`, `INSTALL.md` (one Spanish line), `.gitignore` (two lines).

**Results:** QA round 1 OK (0 BLOCKER, 3 MINOR: spec and roadmap applied at close; T-5 is correct but would fail on the old code only by compile error, informational); Gemini diff clean. `go build ./...` and `go test ./...` green; a real-binary smoke left `data.db.pre-migration-22-<day>` at schema v22 next to a same-day daily backup and live `data.db` at v23.

---

## US-100 — Startup warning when the PC time zone is not UTC-5

**Status:** picked up from roadmap R-162 (go-live item K3), 2026-11-12. Level M (pipeline ID R-162). Done 2026-11-12 (pipeline ID R-162). Approved at GATE 1 (PO): log only, fixed -05:00 offset.

**As** Don Carlos (owner) or whoever installs the POS,
**I want** the POS to say at startup when the Windows time zone is not Bogota time,
**so that** a wrong zone does not silently shift "today", the reports and the register close.

**Problem (verified 2026-11-12):** day windows follow the PC zone (`time.Local`, `spec/domain/tiempo.md`) and nothing checks it at startup (`cmd/pos/main.go:40-60`).

**Acceptance criteria:**
- [x] AC-1: WHEN `pos.exe` starts the server, the system SHALL log the local UTC offset (`time zone: UTC-05:00`).
- [x] AC-2: IF the offset is not -05:00, THEN the system SHALL log a warning with the offset found and how to fix the Windows zone.
- [x] AC-3: WHILE the offset is -05:00, the system SHALL log no warning.
- [x] AC-4: the check SHALL compare the offset in seconds, never the zone name.
- [x] AC-5: a wrong offset SHALL NOT stop startup.
- [x] AC-6: the offset SHALL come from an injected `time.Time`.
- [x] AC-7: `spec/domain/tiempo.md` SHALL describe the check.

**Verified criteria:**
- AC-1: `cmd/pos/main_test.go` `TestTimeZoneNotice_T1_Bogota` (`COT` -18000 gives info `time zone: UTC-05:00`), `TestTimeZoneNotice_T2_UTC` (UTC gives `UTC+00:00`); logged from `cmd/pos/main.go:76-77`, after `resolvePort` and before the `database:` log; a real-binary run printed `time zone: UTC-05:00` first.
- AC-2: `TestTimeZoneNotice_T2_UTC` (offset 0, warning contains `+00:00`), `TestTimeZoneNotice_T3_Minutes` (UTC-4 gives `-04:00`, UTC+5:30 gives `+05:30`).
- AC-3: `TestTimeZoneNotice_T1_Bogota` (warning empty at -18000).
- AC-4: `TestTimeZoneNotice_T4_OffsetNotName` (name `UTC` at -18000 gives no warning; name `COT` at 0 warns); `secs != -5*3600` at `cmd/pos/main.go:54`.
- AC-5: read in the diff: `main.go:76-80` only calls `log.Print`, no `log.Fatal` or `os.Exit`.
- AC-6: `TestTimeZoneNotice_T5_Injected` (same helper, -18000 vs 0, different results whatever the host zone).
- AC-7: `spec/domain/tiempo.md` § Server zone caveat.

**Results:** QA round 1 OK (0 BLOCKER, 1 MINOR: the spec line, applied at close); Gemini diff clean. `go build ./...` and `go test ./...` green. The warning path is covered by unit tests only: Go on Windows ignores `TZ`, so it cannot be forced from the environment. Follow-up: roadmap R-176 (UI banner).

---

## US-101 — One name for the Settings screen

**Status:** picked up from roadmap R-170, 2026-11-11. Level M, docs and one UI string. Done 2026-11-12. Built in a parallel session as "US-099" (pipeline ID R-170, implementation `d69099c`, QA round 1 OK); renumbered US-101 when merged because US-099 had meanwhile been taken by R-168. Approved at GATE 1 (PO, 2026-11-11): Q-1..Q-3 as recommended.

**Problem (verified 2026-11-11):** `INSTALL.md` named the screen "Ajustes" while the sidebar and page title say "Configuración" (`frontend/components/Sidebar.tsx:142`, `frontend/app/settings/page.tsx:378`). The same mismatch existed in `MANUAL-USUARIO.md:20` and the Mesas empty-state link (`frontend/app/mesas/page.tsx:80`).

**Verified criteria:**
- AC-1: every reference to the screen in `INSTALL.md` SHALL read "Configuración" (`:40`, `:42`, `:76` twice) and "Ajustes" SHALL NOT appear. Check: `grep -n "Ajustes" INSTALL.md` returns nothing; `grep -o "Configuración" INSTALL.md | wc -l` → 4.
- AC-2: `MANUAL-USUARIO.md:20` SHALL say "Configuración".
- AC-3: WHEN an admin opens Mesas with no tables, the link SHALL read "Crea mesas en Configuración". No test asserts the string.
- AC-4: No other behavior changes; the `nav.test.ts:11` fixture label is renamed, assertions untouched.

**Results:** QA round 1 OK (0 BLOCKER, 1 MINOR: the contract check used `grep -c`, which counts lines, not occurrences; corrected in AC-1). No spec change (the spec never names the screen). Gemini skipped (docs-only).

---

## US-102 — Whole-peso money with one rounding rule

**Status:** registered 2026-11-12 (pipeline ID R-133, merged with R-106; go-live K5, `project-docs/roadmap.md:38`). PO confirmed on 2026-11-11 that client #1 charges IVA. Level L. Closed 2026-11-13 (RED `b4c59bc`, implementation `ca70c15`); it changes a business rule and the data (migration 24).

**As** Don Carlos (owner),
**I want** every amount in the POS to be a whole peso, with IVA rounded the same way on the screen, the ticket and the books,
**so that** a customer who pays what the screen shows owes exactly $0 and no `$0` in red remains on the *fiado* list.

**Problem (verified 2026-11-12):** line IVA is never rounded (`internal/service/sale.go:71`): $10.001 at 19% is stored as $11.901,19, and a customer paying the displayed $11.901 keeps a $0,19 balance. Table checkout with an IVA product cannot be paid at all (contract V-7). Details and numbers: contract `changes/R-133/contract.md` (disappears with the worktree; the verified criteria are written here at close).

**Acceptance criteria (draft; the contract refines them):**
- [x] Line subtotal and line IVA are rounded to whole pesos by one rule on the server, mirrored in the frontend; $10.001 at 19% → $11.901.
- [x] Client total and payments must equal the server total exactly.
- [x] Migration 24 converts existing rows and leaves no sub-peso *fiado* residue (one rare case remains, R-178).
- [x] A table order with IVA products can be checked out.
- [x] The whole-peso property test passes over the full domain (taxed and kg products).

**Verified criteria (closed 2026-11-13):**

GATE 1 (PO, 2026-11-12): Q-1..Q-9 accepted as recommended: round **per line**; keep `float64`/`REAL` and enforce whole pesos in the service and by migration (no `int64`); migration 24 rounds and recomputes; exact tolerance (0); whole-peso validation on every money input; table-checkout IVA fix in scope (Q-6: whether client #1 uses restaurant mode is unanswered, the manual check can wait); residues settled to 0 with one audit row for drift; spec in the new `spec/domain/dinero.md`; the global discount still does not reduce the tax base. Contract discrepancies: the sprint's "$1 gap" was true only for the client-total check, payments used 0,5 (D-1); table checkout with an IVA product could not be paid at all (D-2, fixed); fractional kg quantities needed rounding too, not only IVA (D-3); `moneyTol` and the `0.5 kg Queso → credit 4999.5` case in the money QA layer had to change (D-4); stale spec lines in `caja.md`, `catalogo.md`, `fiado.md` (D-5, fixed at close). Verified by running: reading a `REAL` column into `int64` fails from $1.000.000 up. Six Gemini contract points resolved before code (client tax ignored, zero-total sale, order tax at checkout, audit user and inactive customers, other tax rates, discount bounds). QA round 1: OK, 0 BLOCKER, 5 MINOR (R-177..R-180 opened; the fifth was this spec update); the implementer changed no test after RED. Gemini diff round 1 timed out with no findings. `go build`, `go test ./... -count=1`, `pnpm build`, `pnpm test` (273 tests) and `pnpm lint` (0 problems) green on the rerun; the very first `go test` run hung once in `internal/service` (R-180). Smoke on the compiled binary (port 8099): IVA sale with a bogus client `tax` 1000 stored tax 1.900 and total 11.901; client totals 11.902 and 11.901,19 → 400; abono 100,5 → 400, abono 11.901 → balance 0; table checkout with cash 11.901 → 201; item discount 10.001 on a $10.001 line with empty payments → 201, total 0. **Not run:** migration 24 against a real v23 database file (covered by tests only), and the frontend in a browser. R-181 (UI edge) was found while writing the rules.

Spec: `spec/domain/dinero.md` (new), `caja.md` (I-1, I-2, I-6, known limitation closed), `fiado.md`, `catalogo.md`, `facturacion-dian.md`, `restaurante.md`, `project-docs/README.md`. `rentabilidad.md` needs no change (it states no rounding rule; cost × quantity stays as is).

- AC-1: WHEN a sale is created, the system SHALL price every line with `lineBase = round(quantity × price)`, `lineTax = round(lineSubtotal × rate)` and store whole pesos. $10.001 at 19% → subtotal 10.001, tax 1.900, total 11.901; 3 × $1.001 → tax 570; $50 at 5% → tax 3; $1.050 at 19% → 200; 0,5 kg × $9.999 → 5.000; 8% on $1.001 → 80. Tests: `internal/service/sale_money_test.go` `TestSaleMoney_T1_TaxRoundedAndClientTotalExact`, `T2_TaxIsRoundedPerLine`, `T3_HalfRoundsUp`, `T4_QuantityFractions`, `T5_Discounts`, `T22_OtherRates`.
- AC-2: The server SHALL ignore the client `tax` and `subtotal` and compare only `total`, exactly: 11.902 → 400 `total inconsistente: esperado 11901, recibido 11902`; 11.901,19 → 400 non-whole; no total (table shape) → priced by the server. Tests: `TestSaleMoney_T1_ErrorNamesBothTotals`, `TestSaleMoneyHTTP_T7_ServerPricesTheSale`.
- AC-3, AC-4: IF an input is not whole, or a discount is negative or above its line base (global: above Σ line subtotals), THEN 400 and nothing stored; payments must equal the total exactly (cash 5.000 + credit 6.900 on 11.901 → 400; 5.000 + 6.901 → 201). Tests: `TestSaleMoneyHTTP_T6_WholePesoInputsAndExactPayments`, `TestSaleMoneyHTTP_T23_DiscountBounds`.
- AC-4a: WHEN the server total is 0, an empty payment list SHALL be accepted (no `sale_payments` row), a supplied line rejected. Test: `TestSaleMoneyHTTP_T21_ZeroTotalSale`; `TestMoneyValidation_T12_TotalMustBePositive` rewritten to the new rule.
- AC-5: WHEN a table order is checked out, the server SHALL price it with the same rule from the non-void items and the products' current `tax_rate`; `GET /orders/{id}` total includes IVA (11.901) and nothing is stored on the order. Tests: `internal/handler/order_money_test.go` `TestOrderMoney_T8_TableCheckoutWithIVA`, `TestOrderMoney_T25_TaxIsNotPersistedOnOrders`.
- AC-6: WHEN a customer pays a credit sale with an *abono* equal to its credit line, the balance SHALL be exactly 0 ($11.901 credit, *abono* $11.901 → 0; mixed cash 5.000 + credit 6.901, *abono* 6.901 → 0, 6.902 → 400). Tests: `TestSaleMoneyHTTP_T9_AbonoOfDisplayedCreditLeavesZero`, `TestSaleMoneyHTTP_T10_MixedSaleAbono`.
- AC-7: IF an *abono*, movement, expense, opening or closing amount, product price, cost, wholesale price or purchase unit cost is not whole, THEN 400 and nothing stored. Test: `TestMoneyValidation_US102_T11_NonWholeInputsRejected`.
- AC-8, AC-9: Migration 24 SHALL round and recompute every stored amount, never fail startup, and write one `money_rounding_migration` audit row for drift (lowest-id active admin; skipped with a log when none; inactive customers included). $11.901,19 credit with abonos 5.000 + 6.901,19 → 0 before and after; credit 11.901,19 with abono 11.901 → 0,19 → 0; two credits of 100,4 and an abono of 200,8 → −1 and one audit row. Tests: `internal/database/migrate_money_test.go` `TestMigration24_T12_RoundsAndRecomputesSales`, `T13_ResiduesSettleToZero`, `T14_DriftWritesOneAuditRow`, `T15_RegisterAndOtherColumns`, `T16_WholeValuesUntouchedAndEmptyDB`, `T24_AuditRowAdminAndInactiveCustomers`; version pins 23 → 24 in `internal/database/schemaversion_test.go`.
- AC-10: The frontend SHALL compute the same numbers and require `paymentsBalanced` exactly. Tests: `frontend/lib/saleCalc.test.ts` (T-19), `frontend/lib/abono.test.ts` "whole-peso fiado balance (US-102 T-20)".
- AC-11: The DIAN adapter SHALL compute each line's `TaxAmount` with the per-line rule so `TotalIVA` = Σ lines = `sales.tax` (3 × $1.001 → 190 × 3 = 570, total 3.573). Tests: `internal/factelect/pos_adapter_test.go` `TestPosAdapter_T17_TaxAmountRoundedPerLine`, `TestPosAdapter_T17_HalfRoundsUpOnTheLine`.
- AC-12: The money property tests SHALL assert whole values over the full domain with exact equality. Tests: `TestMoneyProperties_WholePeso` (build tag removed), `moneyTol` 0 in `internal/handler/money_props_test.go`.

Follow-ups opened: R-177 (open-orders list total without IVA), R-178 (residue 0 < x < 1 may not end at 0), R-179 (clamp can add $1), R-180 (one unexplained `go test` hang), R-181 (cart discount above the base after lowering the quantity).
