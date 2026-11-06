# Domain — Cash register (*caja*: opening, movements, close)

> **Type:** spec · **Current:** yes · **Rules over:** cash count (*arqueo*), reconciliation (*cuadre*), register difference
> **Updated:** 2026-11-05 (US-082: cash *abonos* in expected cash) · Verified against the code on 2026-11-05 for the expected-cash rule (US-082) and on 2026-10-13 for the rest (US-071: transactional opening, US-053: email at close, US-076: App Password never returned by the API).

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
- `cash_out` — money goes out that is not a refund. E.g.: paying a delivery, an expense.

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

> **Historical note:** `archive/2026-05-plan-mvp.md:1135` (formerly `PLAN.md`) reports that `cash_in` was missing from this formula. **It is already included** (`internal/service/register.go:116`). It is an example of why that document is not a source of truth about current state.

## Rule: estimated cash of the day (dashboard)

```
estimated cash = cash payments of completed sales in the range
                 − cash_out movements in the range
```

It does not include the opening float, `cash_in` or cash *abonos*: it is a daily flow indicator, not the register count. The register count is "expected cash at close". Counting cash *abonos* here was left out of US-082 by the PO (2026-11-05); roadmap R-86.

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
- Multiple simultaneous registers — roadmap `R-18` (LAN sync).
- Count by denomination beyond what SPRINT-11 delivered (`frontend/lib/denominations.ts`).
