# Domain — Cash register (*caja*: opening, movements, close)

> **Type:** spec · **Current:** yes · **Rules over:** cash count (*arqueo*), reconciliation (*cuadre*), register difference
> **Updated:** 2026-10-13 · Verified against the code on 2026-10-13 (US-071: transactional opening, US-053: email at close, US-076: App Password never returned by the API).

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
expected   = opening_float + cash_sales + cash_in − cash_out
difference = actual_counted − expected
```

Source: `internal/service/register.go:113` and `:138`.

- `difference > 0` → surplus (*sobrante*)
- `difference < 0` → shortage (*faltante*)
- Only **cash** counts. Sales by transfer, card or fiado do not enter the count: that money is not in the drawer.

> **Historical note:** `archive/2026-05-plan-mvp.md:1135` (formerly `PLAN.md`) reports that `cash_in` was missing from this formula. **It is already included** (`register.go:113`). It is an example of why that document is not a source of truth about current state.

## Rule: estimated cash of the day (dashboard)

```
estimated cash = cash payments of completed sales in the range
                 − cash_out movements in the range
```

It does not include the opening float or `cash_in`: it is a daily flow indicator, not the register count. The register count is "expected cash at close".

Source: `MetricsRepository.CashFlow` (`internal/repository/metrics.go:112`, `cash` payments at `:118`, `cash_out` at `:128`); the subtraction is done in `internal/handler/dashboard.go`.

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
