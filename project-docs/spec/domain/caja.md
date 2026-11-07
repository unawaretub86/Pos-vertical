# Domain — Cash register (*caja*: opening, movements, close)

> **Type:** spec · **Current:** yes · **Rules over:** cash count (*arqueo*), reconciliation (*cuadre*), register difference
> **Updated:** 2026-11-07 (US-081: several LAN devices share the one open register; open race noted, R-121) · 2026-11-07 (US-088: a void inserts no register movement, R-110 closed) · 2026-11-05 (US-082: cash *abonos* in expected cash) · Verified against the code on 2026-11-05 for the expected-cash rule (US-082) and on 2026-10-13 for the rest (US-071: transactional opening, US-053: email at close, US-076: App Password never returned by the API).

The register is where the system earns or loses the cashier's trust. A reconciliation that comes out wrong with no explanation sends the shop back to the notebook.

---

## States

A register (`registers`) is `open` or `closed`. Only one can be open at a time.

## Movements

```
valid types (register_movements.type): sale · cash_in · cash_out · opening
```

- `opening` — the amount the shift starts with (float).
- `sale` — a sale recorded against that register.
- `cash_in` — money comes in that is not a sale. E.g.: the owner hands change to the cashier.
- `cash_out` — money goes out that is not a refund. E.g.: paying a delivery, an expense. A void does **not** write a `cash_out` (since US-088).

A **void** (*anulación*, `DELETE /sales/{id}/void` or `POST /sales/{id}/void-with-credit-note`, admin only, `internal/handler/handler.go:167-170`) inserts **no** register movement. The voided sale's own `sale` movement (`Venta #N` / `Crédito #N`) stays in the list with no paired line. Source: `SaleRepository.Void` (`internal/repository/sale.go:275`), which sets `status = 'voided'` (`:338`), restores stock and writes the `void` audit row (`:344`), and nothing else. There is no void action in the UI; voids are reachable only through the admin API.

**Expenses** (`expenses`) are automatically mirrored as `cash_out` when a register is open, so the count picks them up without duplicating cash-flow logic. Source: `internal/service/expense.go`.

## Rule: expected cash at close

```
expected   = opening_float + cash_sales + cash_abonos + cash_in − cash_out
difference = actual_counted − expected
```

Source: `internal/service/register.go:116` (`Summary`) and `:149` (`Close`).

- `cash_abonos` = `SUM(customer_payments.amount)` with `method = 'cash'` and `register_id` = this register (`RegisterRepository.SumCashAbonos`, `internal/repository/register.go:223-232`). An *abono* (a payment toward a fiado balance) is linked to the register that was open when it was registered; see `fiado.md`, "Rule: payments (*abonos*)".
- `difference > 0` → surplus (*sobrante*)
- `difference < 0` → shortage (*faltante*)
- Only **cash** counts. Sales by transfer, card or fiado do not enter the count: that money is not in the drawer. The same holds for transfer and card *abonos*.
- An *abono* registered with no open register (`register_id NULL`), and every *abono* recorded before migration 21 (US-082), never enters any register count.
- The register summary (`GET /registers/{id}`, `GET /registers/open`, `POST /registers/{id}/close`) returns `cash_abonos` (`internal/service/register.go:21-31`). The close modal, the close ticket and the close email always show the line `Abonos en efectivo`, also at `$0` (`frontend/app/register/page.tsx:366`, `frontend/components/CloseTicketPreview.tsx:135`, `internal/mailer/mailer.go:81`).
- If summing cash *abonos* fails, `Close` returns an error and does not close the register (`internal/service/register.go:143-148`), and `Summary` returns an error (`:95-98`). `GET /registers/{id}` answers 404 only when the register does not exist and 500 for any other failure (`internal/handler/register.go:54-62`); `GET /registers/open` answers 500 on a summary failure (`:76-80`).
- **Known inconsistency:** `Close` still ignores errors from `SumCashSales`, `SumCashIn` and `SumCashOut` (`internal/service/register.go:140-142`); only the *abono* sum is strict. Roadmap R-88.
- **Voided sales.** A voided sale leaves `cash_sales` because only `status = 'completed'` sales count (`internal/repository/register.go:166-175`, filter at `:172`), so a void brings `expected` back to its value before the sale, by the sale's cash portion only. Opening $100.000 + cash sale $50.000 → $150.000; void → $100.000. A $50.000 credit, transfer or card sale changes nothing, before or after its void. Cash $30.000 + card $20.000 → $130.000; void → $100.000. No `cash_out` is written (US-088, R-110). Rows `cash_out` `Anulación venta #N` written by voids before US-088 are left as they are and still count in `SumCashOut` (`internal/repository/register.go:201-207`); there is no migration, because voids were only reachable through the API.
- **Voids after the close.** A sale can be voided after its register is closed (200). Nothing is written to that register or to the register open at that moment, and the stored `expected_close`, `actual_close` and `difference` do not change. `GET /registers/{id}` recomputes `expected`, so it then no longer matches the stored `expected_close`: opening $100.000, closed with `expected_close` $150.000, then its $50.000 cash sale voided → `expected` $100.000, `expected_close` still $150.000. The register history (`GET /registers`, `GET /reports/registers`) shows the stored values. A refund flow and a policy for voids on closed registers are on the roadmap.

> **Historical note:** `archive/2026-05-plan-mvp.md:1135` (formerly `PLAN.md`) reports that `cash_in` was missing from this formula. **It is already included** (`internal/service/register.go:116`). It is an example of why that document is not a source of truth about current state.

## Rule: estimated cash of the day (dashboard)

```
estimated cash = cash payments of completed sales in the range
                 − cash_out movements in the range
```

It does not include the opening float, `cash_in` or cash *abonos*: it is a daily flow indicator, not the register count. A void subtracts nothing here: the voided sale just stops counting as a completed cash payment (US-088). The assistant's cash answer uses the same query (`internal/service/assistant.go:608-615`). The register count is "expected cash at close". Counting cash *abonos* here was left out of US-082 by the PO (2026-11-05); roadmap R-86.

Source: `MetricsRepository.CashFlow` (`internal/repository/metrics.go:127`, `cash` payments at `:133`, `cash_out` at `:143`); the subtraction is done in `internal/handler/dashboard.go:67`.

## Rule: expenses

```
valid categories: servicios · arriendo · nomina · proveedor · otro
```

(utilities · rent · payroll · supplier · other — stored as these Spanish keys.)

Source: `internal/service/expense.go:17-19`. Closed list, validated in the service. Adding a category means touching that constant **and** reviewing the reports that group by it.

```
net profit = gross profit − expenses of the same range
```

---

## Rule: email at close (US-053)

- The owner's email address (`owner_email`) and Gmail App Password (`owner_email_app_password`) live in `business_config`.
- `GET /api/setup` and the `PUT /api/setup` response never include `owner_email_app_password`; they expose `has_email_app_password` (bool). The password is write-only from the API; it is used server-side only (close summary, manual summary, email test via `POST /api/settings/email/test`, admin only).
- `PUT /api/setup` with an omitted or empty `owner_email_app_password` keeps the stored one; a non-empty value replaces it (`internal/repository/setup.go`, `Upsert`). There is no way to clear a saved password (roadmap R-70).
- Changing `owner_email` without retyping the password keeps the old password; the email then fails at SMTP and the "Probar" button shows the error.

---

## What does NOT exist yet

- Printable shift close (X/Z report) beyond what SPRINT-11 covers.
- Multiple simultaneous registers — roadmap `R-18` (LAN sync). What does exist (US-081): several devices on the LAN charge into the **one** open register at the same time, and its close adds up all of them (see `servidor.md`, "Concurrent devices on the LAN").
- Opening the register is check-then-insert (`internal/service/register.go:47-55`), so two simultaneous opens could in theory create two open registers. This was not reproduced (5 runs of 5 concurrent `POST /registers`: always 1 × 201 + 4 × 409). Roadmap `R-121`.
- Count by denomination beyond what SPRINT-11 delivered (`frontend/lib/denominations.ts`).
