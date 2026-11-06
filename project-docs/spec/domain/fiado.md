# Domain — Fiado (credit to customers)

> **Type:** spec · **Current:** yes · **Rules over:** customer balance, payments (*abonos*), receivables (*cartera*)
> **Updated:** 2026-11-05 · Verified against the code on 2026-09-23 (US-059: metrics in `internal/repository/metrics.go`; US-063: aging receivables use the same balance formula; US-064: `total_portfolio` of `GET /customers` uses the same formula and excludes balances `≤ 0`).

*Fiado* (selling on credit) is **the product's differentiator**. Competitors treat it as an accounting field; here it is a relationship with history. Any change to these rules changes the value proposition, not just the code.

---

## Model

A sale is paid with one or more lines in `sale_payments`, each with a method. When the method is `credit`, that line **is** the debt. There is no debts table: the debt is the sum of what was put on credit minus the sum of what was paid back.

```
valid methods (sale_payments.method):  cash · transfer · card · credit
              (sales.payment_method):  cash · transfer · card · credit · mixed
```

`sales.payment_method = 'mixed'` is a sale-level label; the real breakdown always lives in `sale_payments`.

## Rule: customer balance

The balance is **never stored**. It is computed on the fly in every query.

```
balance(customer) = SUM(sale_payments.amount where method='credit'
                        and the sale is 'completed')
                  − SUM(customer_payments.amount)
```

Source: `internal/repository/customer.go:24-40`.

Consequences that must be respected:

- **A voided sale stops counting automatically**, because the filter is `status='completed'`. There is no need to "reverse" the debt by hand.
- **There is no such thing as a stale balance.** Any screen that shows a balance reads it from the same query. If two screens ever show different balances, it is a query bug, not a synchronization bug.
- A customer with balance `0` or negative is **not in receivables**. Receivables filter `balance > 0`.
- Receivables order: balance descending and, on a tie, name ascending. Source: `MetricsRepository.Debtors` (`internal/repository/metrics.go:286`, ordering at `:297`), same balance formula as `customer.go:24-40`.
- The dashboard's aging receivables (`MetricsRepository.AgingReceivables`, `internal/repository/metrics.go:326`) use the same balance formula as `Debtors`: both build the balance with `customerBalanceExpr` (`metrics.go:265`) over `customerBalanceFrom` (`metrics.go:270`), which aggregates credit sales and payments per customer before joining, so each payment counts exactly once. Filters `balance > 0`.
- Aging receivables order: `days_overdue` descending first and, on a tie of days, the same tie-break as `Debtors` — balance descending and, if that also ties, name ascending (`metrics.go:336`).
- `total_portfolio` of `GET /customers` (`CustomerRepository.TotalPortfolioBalance`, `internal/repository/customer.go:145`) uses the same per-customer aggregated formula (`customerBalanceExpr`/`customerBalanceFrom`) and excludes balances `≤ 0`, summing only the positive ones — not a global subtraction of all credit sales minus all payments (that was the R-44 bug, fixed in US-064).

## Rule: debt age

```
oldest_debt_at = MIN(created_at) of the customer's sales that have a
                 credit line, among the 'completed' ones
days_overdue   = today − oldest_debt_at
```

It is the date of the **oldest debt without distinguishing partial payments**: a customer who pays without settling in full keeps their original date. This is deliberate — it measures how long that customer has owed something, not how long they have owed that particular peso.

## Rule: credit limit

`customers.credit_limit` is **optional** (`NULL` = no limit defined) and is currently **informational**: it warns, it does not block the sale.

Source: `internal/service/customer.go:39,54`. Scenario covered in `internal/service/customer_test.go:213`.

If it is ever decided that it should block, that is a product change Andrey must approve: a shopkeeper who cannot extend credit because of a cap they set wrong stops using the system.

## Rule: payments (*abonos*)

An *abono* is a row in `customer_payments`. It is not tied to a specific sale — it is applied to the customer's total balance, not to an invoice. That is how the notebook the system replaces works, and changing it to per-invoice allocation would be a redesign, not an adjustment.

- Only an admin can register an *abono* today (`POST /customers/{id}/payments`, admin group, `internal/handler/handler.go:290`).
- The payment method is not stored: the form asks for it as free text in the note (`frontend/app/customers/[id]/CustomerDetailClient.tsx:233`).
- An amount above the current balance is rejected (`internal/service/customer.go:94`).

Planned change (SPRINT-19 US-082, PO decisions 2026-11-05): a stored method (`cash`/`transfer`/`card`), cash *abonos* counted in the open register's close, and cashiers allowed to register *abonos* while a register is open.

## Rule: deleting a customer

`DELETE /customers/{id}` (admin) is a **logical delete**: it sets `customers.active = 0` (migration 19) and never removes the row, because sales and payments reference it.

- A customer with balance `> 0` cannot be deleted → 409. Balance `0` or negative (overpaid) → deactivated, 200. Unknown or already inactive id → 404.
- Inactive customers are hidden from `GET /customers` and its search (`q`). Everything else keeps resolving them by id: customer detail, balance, payments, sale detail, the dashboard's recent sales, and the receivables metrics (`Debtors`, `AgingReceivables`, `total_portfolio`), which do not filter by `active`. Totals therefore do not change when a customer is deleted.
- A credit sale (`fiado`, payment method `credit`) to an inactive or missing customer is rejected (400, `internal/repository/sale.go`, `ErrCustomerInactive`). A cash, card or transfer sale that merely names an inactive customer is accepted. A payment (`abono`) cannot target an inactive customer either: its balance is never above zero, and `CreatePayment` rejects an amount above the balance.

Source: `internal/service/customer.go:61`, `internal/repository/customer.go:30,106`, `internal/handler/customer.go:110`.

---

## What does NOT exist yet

- **Customer risk scoring** — roadmap `R-17`. It is the strongest differentiator not yet built: with the payment history already stored, a score, suggested limit and early late-payment alert can be computed. No local competitor stores or analyzes this.
- **Blocking a sale for exceeding the limit** — pending product decision.
- **Interest or late fees** — never considered; the Colombian shopkeeper does not charge them.
