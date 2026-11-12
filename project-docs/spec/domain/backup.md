# Domain — Daily backups

> **Type:** spec · **Current:** yes · **Rules over:** when and where the POS copies its database, how the copy is verified and pruned, the folder setting and its report
> **Updated:** 2026-11-12 (US-099: pre-migration copy has its own name, `migrate.go` `backupBeforeMigration`) · 2026-11-11 (US-089: created) · Verified against the code at the US-099 branch (`internal/database/backup.go`, `internal/handler/backup.go`, `internal/repository/setup.go`, `internal/database/migrate.go` migration 23, `cmd/pos/main.go`).

The POS copies `data.db` once a day so a dead disk or a bad update does not cost the shop its *fiado* (store credit) ledger. The copy is a SQLite `VACUUM INTO` taken while the POS keeps serving (WAL mode), written and checked before it replaces anything.

## Schedule

`StartBackupScheduler` (`cmd/pos/main.go:73`) runs one backup right after startup and then every 24 hours, in a goroutine. It returns at once, so a slow or unreachable folder never delays the listener (`internal/database/backup.go:40-52`, test `TestBackup_T8_SchedulerReturnsImmediately`). A PC that is switched off simply skips those days; there is no catch-up.

## File name and retention

- Name: `<db base name>.backup-YYYYMMDD`, e.g. `data.db.backup-20261031`. The date is the **server's local day** (`d.now().In(time.Local)`, `backup.go:119`): a backup at 21:05 Bogota on 31/10 is `...-20261031`, not the UTC date (`tiempo.md`). One file per day: a second run the same day overwrites it.
- Retention: the **30 newest** files in the folder written (`maxBackups = 30`, `backup.go:16`). A PC that is off at weekends therefore keeps about six weeks. The prune runs only after a successful copy.
- Strict match: only names equal to `<base>.backup-` followed by exactly 8 digits are ever deleted (`isBackupName`, `backup.go:298-310`). The prune lists the directory (no `filepath.Glob`), so `[`, `*` or `?` in the folder path are harmless (`pruneBackups`, `backup.go:312`). Never touched, whatever their age: `<db>.pre-reset-*` (the go-live reset backup, `reset.md`), `<db>.pre-migration-*` (below), manual copies such as `data.db.backup-manual` or `data.db.backup-20261001.tmp`, and files of other databases. The daily job never copies a `.pre-reset-*` file anywhere.
- Tests: `TestBackup_T1_KeepsThirtyNewest`, `TestBackup_T2_PruneIgnoresForeignFiles`, `TestBackup_T3_PruneWorksInFolderWithBrackets`, `TestBackup_T9b_FileDateUsesLocalZone` (`internal/database/backup_test.go`).

## Pre-migration copy

Before an upgrade (pending migrations on a database at schema version N > 0) the POS writes `<db>.pre-migration-N-YYYYMMDD` next to `data.db` (`backupBeforeMigration(from)`, `internal/database/migrate.go`), e.g. `data.db.pre-migration-22-20261112`. N is unpadded and the date is the server's local day. It is a `VACUUM INTO` copy taken after a WAL checkpoint and before any migration runs; if it fails, startup stops with `backup before migration` and no migration runs.

- A separate file from the daily backup: `isBackupName` does not match it, so the daily backup never overwrites it and the prune never deletes it. The daily job never copies it to a configured folder; it stays beside `data.db`.
- Two upgrades the same day from different versions leave two files. A retry the same day from the same version (the database is unchanged because the failed migration rolled back) replaces the earlier file of that name.
- Skipped on a brand-new database (`current == 0`), on `:memory:` and when no migration is pending.
- Never pruned: one file per upgrade; delete by hand (PO, 2026-11-12).
- Tests: `TestPreMigration_T1_CopySurvivesDailyBackup`, `_T4_DifferentVersionsSameDayKeepBoth`, `_T5_SameVersionSameDayReplaces`, `_T6_NoCopyWhenNewOrUpToDate`, `_T7_FailedCopyAbortsBeforeMigrating`, `_T8_DateUsesLocalZone` (`internal/database/migrate_test.go`), `TestBackup_T2_PruneIgnoresForeignFiles`, `TestBackup_PreMigrationCopyStaysBesideDB` (`backup_test.go`).

## Atomic write and verification

1. Live row counts of every user table are read (`before`), then `VACUUM INTO <name>.tmp` runs (`writeVerifiedCopy`, `backup.go:161-187`).
2. The copy is opened and must pass `PRAGMA integrity_check` = `ok` and a per-table row-count check (`verifyCopyCounts`, `backup.go:250`).
3. Only then is `<name>.tmp` renamed over `<name>` (`backup.go:126`).

**Row-count rule.** The live database keeps committing while the copy runs, so for every table (all of `sqlite_master` except `sqlite_%`) the copy's count must lie between the live count read just before and just after: `min(before, after) <= copy <= max(before, after)`. On a quiet table the counts must be equal. A table that fails only while its live count moved is retried once (the whole copy); a quiet-table mismatch is never retried.

A failure at any step removes the `.tmp` file, **does not prune**, leaves the earlier same-day file byte-identical and records the failure (below). Tests: `TestBackup_T4_CopyVerifiedAndNoTmpLeft`, `TestBackup_T5_FailedCopyKeepsGoodFileAndDoesNotPrune`, `TestBackup_T6_VerifyBackup`, `TestBackup_T6b_RowCountTolerance`, `TestBackup_T9_ConsistentWhileSellingWithOpenRegister` (open register, a writer committing $10.000 cash sales, 5 backups, each copy `integrity_check` ok and `foreign_key_check` empty).

## Folder setting and fallback

- Default folder: the folder of `data.db` (`DefaultBackupDir`, `backup.go:67`). `data.db` itself is relative to the process working directory (`cmd/pos/main.go`, `resolvedDBPath`; the absolute path is logged at startup as `database: ...`), so see `servidor.md` § Start with Windows for why the working directory matters.
- The admin may configure another **absolute** folder (a drive letter `D:\Respaldos` or a UNC path). It lives in `business_config.backup_dir` (`''` = default), outside `GET/PUT /setup`: a `PUT /setup` replaces the whole row from its body and `GET /setup` is readable by cashiers, so a backup path there would be wiped or exposed (`internal/handler/setup.go`).
- At backup time, IF the configured folder does not exist, is not a directory or a test file cannot be created in it, THEN the backup is written to the default folder instead of being skipped, `fell_back` is recorded with the reason and the failing folder, and only the folder written to is pruned (`runBackup`, `backup.go:88-139`). The old folder's files are left alone.

### Endpoints (admin only; cashier 403, no token 401; `internal/handler/handler.go:143-144`)

- `PUT /api/backup/config` `{"dir": "<path>"}`: trims the path, then validates (`internal/handler/backup.go:53-89`). `""` clears to the default. A missing key, `null` or a non-string → 400 `Falta la carpeta` (a missing key never clears). Not absolute → 400 `Escriba la ruta completa de la carpeta, por ejemplo D:\Respaldos`; missing → 400 `La carpeta no existe`; a file → 400 `La ruta no es una carpeta`; not writable → 400 `No se puede escribir en la carpeta`. On any 400 the stored value is unchanged. Success answers 200 with the same body as the status.
- `GET /api/backup/status`: `configured_dir`, `default_dir`, `effective_dir`, `last_attempt_at`, `last_success_at`, `last_ok`, `fell_back`, `last_error`, `last_file`. The two timestamps are RFC 3339 strings with the server's offset (`2026-10-31T21:05:00-05:00`) or `null`.
- Tests: `TestBackupConfig_T12_PutValidation`, `TestBackupStatus_T13_ReportsRunsAndSurvivesRestart`, `TestBackupStatus_T13_InitialValuesAndFailedRunKeepsLastSuccess`, `TestBackupConfig_T14_IndependentFromSetup` (`internal/handler/backup_test.go`).

## Report fields (migration 23)

Persisted in `business_config`, so they survive a restart: `backup_dir`, `backup_last_attempt_at`, `backup_last_success_at` (UTC text like the rest of the schema), `backup_last_ok`, `backup_fell_back`, `backup_last_error`, `backup_last_file`, `backup_effective_dir` (`internal/database/migrate.go:617-627`; the ninth column is needed so `effective_dir` survives a restart). The reset tool keeps `business_config` (`reset.md`).

- Before any attempt: timestamps `null`, `last_ok` false, `fell_back` false, `last_error` `""`, `last_file` `""`, `effective_dir` `""`; `default_dir` is never empty.
- Every attempt sets `last_attempt_at`, `last_ok`, `fell_back`, `last_error`, `effective_dir`. `last_error` is empty on a clean success and holds the fallback reason on a fallback success (`backup.go:134`).
- Only a successful attempt (file renamed into place, a fallback success included) sets `last_success_at` and `last_file`. A failed attempt keeps both at their previous good values.

## UI

- Settings has a "Copias de seguridad" section (`frontend/components/BackupSection.tsx`, rendered at `frontend/app/settings/page.tsx:606`): folder field, `Guardar`, the last result as a Spanish sentence with a warning when `fell_back` or `last_ok` is false, and the folder in use.
- The admin dashboard shows a banner linking to Settings only when the last backup failed or fell back (`frontend/components/BackupBanner.tsx`, `frontend/app/page.tsx:356`). Cashiers see neither.
- Helpers and tests: `frontend/lib/backupSettings.ts`, `frontend/lib/backupSettings.test.ts` (`isAbsoluteWindowsPath`, `describeBackupStatus`, `shouldShowBackupBanner`).

## Restore

Manual: close `pos.exe`, copy a `data.db.backup-YYYYMMDD` file over `data.db`, start the POS. After a bad upgrade, copy the `data.db.pre-migration-N-YYYYMMDD` file over `data.db` the same way. There is no restore command or screen.

## Not covered

- **Off-machine copy.** A backup on the same disk does not survive a dead disk; `INSTALL.md` tells the operator to copy one backup to a USB stick or cloud drive each week. Cloud backup is out of scope.
- **Hung network share.** A backup to an unreachable UNC path blocks only the backup goroutine; the server keeps serving. There is no timeout.
- **Old folder.** After the admin changes the folder, the old folder's files are neither pruned nor moved.
- **Pre-migration copies are not pruned.** One file per upgrade accumulates beside `data.db`; delete by hand (see "Pre-migration copy").
- No "back up now" button; no copy of `.pre-reset-*` files to the backup folder (PO, 2026-11-11: it stays next to `data.db`).
