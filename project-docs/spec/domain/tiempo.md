# Domain — Time (stored timestamps and day windows)

> **Type:** spec · **Current:** yes · **Rules over:** how timestamps are stored, what "today / yesterday / this month" mean, how a date range is compared
> **Updated:** 2026-11-11 (US-089: backup file date) · 2026-11-10 (US-098, roadmap R-158 and R-05: created) · Verified against the code at `d8fce78`.

The POS runs on the shop's PC. Rows are stamped by SQLite in UTC, people think in Bogotá time. Before US-098 the two were compared as text and from 19:00 to 24:00 America/Bogota the dashboard "today", the sales report and the assistant's "hoy" omitted that evening's sales.

## Rule: stored timestamps are zone-less UTC text

Every `created_at`, `opened_at`, `closed_at` of the business tables is `DATETIME DEFAULT CURRENT_TIMESTAMP` (`internal/database/migrate.go`), which SQLite writes as **UTC text `YYYY-MM-DD HH:MM:SS`** with no zone suffix. Go never sets `created_at` on a range-filtered table (`internal/repository/sale.go:59`). Two columns written by Go with a local offset (`sales.voided_at`, `electronic_documents.*`) are never range-filtered. `customer_opening_balances.debt_date` is the UTC text of 00:00 server-local of the picked date (`fiado.md`, "Rule: opening balance").

## Rule: a range is bound as UTC text, half-open

A window is always half-open `[from, to)` (never `BETWEEN`). The caller builds `from` and `to` as `time.Time` in any zone; the repository converts each bound with `dbTime(t) = t.UTC().Format("2006-01-02 15:04:05")` (`internal/repository/dbtime.go`) and compares it **as text** with the column. The bound is a string, never a `time.Time`: the driver would write a local time with its `-05:00` offset, and even a `.UTC()` value with a `+00:00` suffix, and a row stored at exactly the bound instant would fall in the wrong window. The same helper serves every range predicate: the nine of `MetricsRepository`, `buildDateWhere` (sales summary, top products), `ExpenseRepository.SumRange` and the `from`/`to` filters of sales, expenses, purchases and audit (these four keep an inclusive `<=` for `to`, roadmap R-164).

## Rule: "today" is the server's local day

"Today", "yesterday", "this week/month/year" are computed in the **server's local zone** (`time.Local`; on the shop PC, America/Bogotá, UTC−5, no daylight saving) from an injectable clock: `Handler.now` for the dashboard (`internal/handler/dashboard.go:51`, `:139`) and `AssistantService.now` for the assistant (`internal/service/assistant.go:162`), both defaulting to `time.Now`. The reports page sends browser-local instants as RFC 3339; the server compares them as instants.

Worked example (America/Bogotá, clock 2026-11-09 20:30; stored text is UTC):

| Sale | Local instant | Stored `created_at` | Total |
|---|---|---|---|
| S0 | 2026-11-08 23:59:59 | `2026-11-09 04:59:59` | $5.000 |
| S5 | 2026-11-09 00:00:00 | `2026-11-09 05:00:00` | $7.000 |
| S1 | 2026-11-09 18:59:00 | `2026-11-09 23:59:00` | $10.000 |
| S2 | 2026-11-09 20:18:41 | `2026-11-10 01:18:41` | $50.000 |
| S3 | 2026-11-09 23:59:59 | `2026-11-10 04:59:59` | $20.000 |
| S4 | 2026-11-10 00:00:00 | `2026-11-10 05:00:00` | $3.000 |

Today (Nov 9) = S5 + S1 + S2 + S3 = **$87.000, 4 sales**; yesterday = S0 = **$5.000, 1 sale**; Nov 10 = S4 = $3.000. A sale exactly at a window's `from` belongs to it; exactly at `to` belongs to the next. "Este mes" at 20:30 on Nov 9 is `[Nov 1, now)` = S0 + S5 + S1 + S2 = **$72.000, 4 sales** (S3 and S4 are later than the clock).

**Server zone caveat.** The window follows the PC's zone. A Windows PC with a wrong time zone silently shifts every day boundary (with the zone at UTC, 19:00-24:00 Bogotá sales land in tomorrow again). There is no startup check yet (roadmap R-162).

## Rule: what does not use a day window

- **Register close** sums by `register_id`, not by date (`internal/repository/register.go:200-232`); `closed_at = CURRENT_TIMESTAMP`.
- **Elapsed time** (aging `days_overdue` at `internal/repository/metrics.go:415`, order and table minutes at `order.go:375`, `dining_table.go:43`) is `julianday('now') - julianday(created_at)`: UTC against UTC. The low-stock 7-day sales window uses the same form (`metrics.go:303`; before US-098 it compared a local "now" with UTC text and was 5 h too long). Aging rows whose oldest date cannot be parsed by SQLite stay excluded (`metrics.go:418`); a date in the legacy pre-`_time_format=sqlite` Go format would be one, and no stored data has it.

- **Backup file name** `<db>.backup-YYYYMMDD` uses the server's local day (`d.now().In(time.Local)`, `internal/database/backup.go:119`), not the UTC date; the status timestamps of `GET /api/backup/status` are RFC 3339 in that zone (`backup.md`).

## Known limitations

- **Day grouping depends on the OS zone.** The `group_by=day|week|month` labels come from `strftime(..., 'localtime')` (`internal/repository/report.go:121-125`), which follows the OS or `TZ` setting, not Go's `time.Local`. They agree on the shop PC; they can differ when `TZ` is set (roadmap R-163).
- The frontend still mixes three zone sources: browser-local (reports ranges, `todayISODate`), fixed `America/Bogota` (`frontend/lib/posSummary.ts:20`) and the server's zone (roadmap R-161).
- The reports page ends "today" at 23:59:59 against a half-open `<`, so a sale at exactly 23:59:59 is dropped (roadmap R-160).
- Bounds are truncated to whole seconds (`created_at` has 1 s resolution).
