# Personas

> **Type:** spec · **Current:** yes · **Rules over:** who each US is written for
> **Updated:** 2026-11-14 (US-105: what the cashier sees of customers and receivables) · 2026-10-01 · Origin: `PRD.md:29-38`, verified against the system roles.

Every US is written as *"As Don Carlos (owner)…"* or *"As Laura (cashier)…"*. If a US cannot be attributed to either, it is probably technical work and goes as *"As a developer…"*.

---

## Don Carlos — owner

Small food distributor in Bogotá. 45–60 years old. Basic computer skills: uses WhatsApp, Excel now and then.

Sells on fiado (credit) to restaurant owners and keeps the debts in a notebook.

**What he needs:** sell simply, know who owes him, print receipts.

**What makes him distrust the system:** a number that does not add up. A register reconciliation with an unexplained difference, or two screens showing different figures for the same data, and he goes back to the notebook.

**Technical role:** `admin`. Sees dashboard, reports, profit and receivables.

## Laura — cashier

Staffs the counter, 50–150 sales per day. 20–35 years old, comfortable with a phone and basic apps.

**What she needs:** fast product search, fast checkout, no complicated menus.

**What frustrates her:** anything that takes her hands off the keyboard at rush hour, and any extra step between scanning and charging.

**Technical role:** `cashier`. **Does not see** dashboard, reports, profit or receivables (the list of balances, `total_portfolio`, debt aging, a customer's credit and *abono* history) — they are under `RequireAdmin`, and the Customers screens (`/customers`, `/customers/{id}`) send her to `/pos`. She sees customer names and phones to pick one, and the balance of **one selected customer** only where she needs it: the credit checkout (`GET /customers/{id}/pos-summary`) and taking an *abono* from `Punto de Venta › Abonar` (`GET /customers/{id}/balance`). Since US-105 `GET /customers` answers her without balances (`internal/handler/customer.go:22-33`); detail in `domain/fiado.md` → "Rule: what the cashier sees".

---

## Permissions rule

```
system roles (users.role): admin · cashier
```

Source: `internal/database/migrate.go:156`.

**Anything that exposes total sales, profit, margin or receivables goes under `appmw.RequireAdmin`.** Not out of bureaucracy: Laura is an employee, and margins and customer debts are the owner's information.

When a new screen or endpoint is added, the question is always: *should Laura be able to see this?* If the answer is no, it goes in an admin group.
