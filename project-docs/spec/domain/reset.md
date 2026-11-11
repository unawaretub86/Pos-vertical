# Domain — Go-live reset (`reset-data`)

> **Type:** spec · **Current:** yes · **Rules over:** which data the go-live reset deletes and keeps, how stock is rebuilt, the safety order of the tool
> **Updated:** 2026-11-10 (US-093: created) · Verified against the code at `171f836` (`internal/resetdata/reset.go`, `guard.go`, `backup.go`, `confirm.go`, `errors.go`, `cmd/reset-data/main.go`, `internal/database/database.go:46`).

`reset-data` is a command-line tool (`cmd/reset-data`, next to `cmd/reset-password`) that the Tech Lead runs **once**, after the dry run and right before go-live, on a PC where `pos.exe` is closed. It removes the dry-run transactions and keeps master data, so the client starts with a clean history, the real catalog and sale numbers from `#1`. It is not part of the POS server and has no HTTP endpoint or UI button (PO, 2026-11-10).

## Exception to logical delete

The rest of the product never deletes historically referenced rows (`active = 0`; `fiado.md` § deleting a customer, `catalogo.md`). This tool **physically `DELETE`s** transaction rows (`internal/resetdata/reset.go:22-34`) because they are dry-run test data with no historical value, only once, only behind a verified backup and a typed confirmation. The exception applies to this tool only.

## What is kept, cleared or reset

25 user tables (`sqlite_sequence` is SQLite's own table and is not counted). The tool lists 11 tables to clear (`clearSteps`, `reset.go:22-34`); every other table is never named in a `DELETE`.

| Table | Decision | Reason |
|---|---|---|
| `users` | KEEP all | FK target of `audit_log`, registers, sales; test accounts are disabled by hand in the UI. |
| `business_config`, `fiscal_config` | KEEP | Shop, printer, SMTP, `jwt_secret`, DIAN resolution and range. Never touched. |
| `categories`, `products`, `customers`, `suppliers`, `dining_tables` | KEEP rows | Master data. Test products and customers survive too (the tool cannot tell them apart); the Tech Lead removes them in the UI (logical delete). |
| `customer_opening_balances` | KEEP | Master data (`fiado.md`). Customer balances are derived, so after the reset every balance equals its opening balance. |
| `purchases`, `purchase_items`, `stock_entries` | KEEP all rows | They rebuild the stock; dry-run purchases and manual entries survive because they cannot be told apart from real ones. |
| `schema_migrations` | KEEP | Schema state. |
| `products.stock` | RESET | Rule B below. |
| `sales`, `sale_items`, `sale_payments`, `customer_payments`, `registers`, `register_movements`, `expenses`, `orders`, `order_items`, `assistant_queries` | CLEAR | Dry-run transactions. After the reset no register is open. |
| `electronic_documents` | CLEAR, but see the fiscal guard | Refused while a document is `accepted`, `draft` or `pending`. |
| `audit_log` | PARTIAL CLEAR + 1 row | Deleted: actions `void`, `discount`, `price_override`, `cash_out`, `order_item_void`, `order_cancel` (they point to ids of deleted rows that restart at 1, `reset.go:39-41`). Kept: `product_import`, `product_price_change`, `opening_balance_create/update`, earlier `data_reset`. One new `data_reset` row (below). |
| `sqlite_sequence` | rows of the cleared tables deleted | Ids of sales, registers, orders, expenses, abonos, movements, `sale_items`, `sale_payments`, `order_items`, `assistant_queries` restart at 1 (`reset.go:204-211`), so the first real sale prints `Venta #1` (`internal/service/printer.go:259`: the ticket number is the sale id). Counters of kept tables continue. `electronic_documents.id` is TEXT and has no counter. |

Fiscal numbering is not local: `fiscal_config` and the provider's `number` stay as they are; the tool never touches `fiscal_config`.

## Rule: stock after the reset (rule B)

For every product with `type = 'product'`, **including inactive ones (`active = 0`)**, `stock = COALESCE(SUM(stock_entries.quantity), 0)` (`stockRebuildSQL`, `reset.go:48`). Services are untouched. Entries are the catalog import (`Importación de catálogo`), purchases (`Compra #N`) and manual entries (`Entrada de inventario`) (`catalogo.md` § Stock traceability).

Consequence: stock that no entry explains is lost. That is stock created or edited through the manual product form (`POST/PUT /products` write `products.stock` with no entry; roadmap R-153) and, since entries are positive only, any downward correction made through the form.

The dry-run prints, per product whose stock would change (`StockChange`, `reset.go:60`): id, name, `[inactivo]`, stock now → after, entries, sold (completed sales only) and `untraced = now + sold − entries`; a product with `untraced ≠ 0` is marked `!` (`cmd/reset-data/main.go:153`). The Tech Lead re-enters such stock as `Entrada de inventario` after the reset. Example: stock 20, 5 sold, no entries → `20 -> 0`, untraced 25.

## Safety order (`cmd/reset-data/main.go:run`)

Dry-run is the default; `--apply` deletes. In order:

1. Open the file raw (no migrations, `busy_timeout`, `foreign_keys`); a missing file is exit 2 and nothing is created (`guard.go:28`).
2. Schema version must equal `database.SchemaVersion()` (the last migration the binary knows, `internal/database/database.go:46`); otherwise refuse and tell the operator to start `pos.exe` once (`guard.go:46-55`).
3. Guards (`guard.go:115-146`, also repeated inside the transaction): an **active admin** must exist (`audit_log.user_id` is `NOT NULL` + FK); no `accepted`, `draft` or `pending` electronic document (**fiscal guard**, whatever `fiscal_config.ambiente` says, because `electronic_documents` has no `ambiente` column; `rejected` and `cancelled` never block); no earlier `data_reset` row (**second-run guard**). `--force` skips only the second-run guard; `--discard-fiscal-documents` skips only the fiscal guard and the discarded counts per status go in the audit details. In dry-run a refused guard is printed as `AVISO` and the exit is 0.
4. (`--apply` only) **POS running** probe: a TCP connection to `127.0.0.1:<port>` (`--port`, then `POS_PORT`, then 8080) means refuse (`guard.go:59-73`).
5. **Typed confirmation:** the tool prints the per-table counts and asks for exactly `BORRAR <N>`, `N` = the number of sales to delete; anything else or closed stdin cancels. `--confirm "BORRAR <N>"` supplies it for scripts (`confirm.go`).
6. **Backup** `<db>.pre-reset-YYYYMMDD-HHMMSS` next to the database: WAL checkpoint, `VACUUM INTO`, then the copy is opened and must pass `PRAGMA integrity_check` with the same per-table counts (`backup.go:53-82`). The name does not match the daily `<db>.backup-*` glob, so the POS never prunes it (`internal/database/backup.go:57-77`); the tool never deletes it. If the name already exists (a quick `--force` re-run) the CLI advances the timestamp by one second until it is free (`main.go:124-131`), so the name can run a few seconds ahead of the clock.
7. **One `BEGIN EXCLUSIVE` transaction** (`reset.go:154`): `SQLITE_BUSY` means another process has the file open → `ErrLocked`. Inside: guards again, table-count fingerprint equal to the backup's (catches a write between backup and lock), the deletes, the stock rebuild, the `sqlite_sequence` deletes, the `data_reset` audit row, `PRAGMA foreign_key_check` (any violation rolls back), `COMMIT`. After the commit: WAL checkpoint and `VACUUM`; a failure there is a printed warning, not an error.

The exclusive lock only exists while the transaction is open (verified on Windows, 2026-11-10), which is why the port probe, the fingerprint and the order above all exist.

**Audit row.** Exactly one per run: `action` `data_reset`, `entity` `system`, `entity_id` 0, `user_id` the lowest-id active admin; details are compact JSON `{"backup":"<file name>","tool":"reset-data","cleared":{<table>:<count>},"stock_rule":"entries"}` plus `discarded_fiscal` when the flag was used (`reset.go:213-224`). It is also the second-run marker. The audit page shows the raw action name (no label yet, roadmap R-165).

## Exit codes and output

0 success or dry-run · 1 refusal (POS running, lock busy, fiscal guard, no active admin, second run, wrong confirmation, schema version) · 2 usage (bad flag, missing file) · 3 backup failed or failed verification · 4 database error, rolled back (`errors.go`, `ExitCode`). Messages and prompts are Spanish (product text for the operator); a refusal that can be evaluated earlier always happens before any backup file exists. Three refusals need the backup first (lock busy, in-transaction guard recheck, fingerprint mismatch): they leave an unused `*.pre-reset-*` file and say `No se borró nada`.

## Flags

`--db` (default `data.db`), `--port`, `--apply`, `--confirm`, `--force`, `--discard-fiscal-documents`.

## Not covered

No UI button or endpoint; no selective reset (by date or table); no restore command (restore by copying the `.pre-reset-*` file over `data.db` with the POS closed); no automatic removal of test products, customers or users.
