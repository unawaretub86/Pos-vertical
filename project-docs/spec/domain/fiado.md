# Domain — Fiado (credit to customers)

> **Type:** spec · **Current:** yes · **Rules over:** customer balance, payments (*abonos*), receivables (*cartera*)
> **Updated:** 2026-11-06 (US-086: customer detail id from the URL path; US-083: collection WhatsApp message with the business name) · 2026-11-05 (US-082: *abono* method, register link, who may register; line refs re-verified) · Verified against the code on 2026-11-06 for the collection message (US-083), on 2026-11-05 for *abonos* (US-082) and on 2026-09-23 (US-059: metrics in `internal/repository/metrics.go`; US-063: aging receivables use the same balance formula; US-064: `total_portfolio` of `GET /customers` uses the same formula and excludes balances `≤ 0`).

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

Source: `CustomerRepository.Balance` (`internal/repository/customer.go:121-134`); lists and metrics use `customerBalanceExpr` (`internal/repository/metrics.go:338`) over `customerBalanceFrom` (`metrics.go:343`). The *abono* method plays no part: every *abono* subtracts, whatever its method.

Consequences that must be respected:

- **A voided sale stops counting automatically**, because the filter is `status='completed'`. There is no need to "reverse" the debt by hand.
- **There is no such thing as a stale balance.** Any screen that shows a balance reads it from the same query. If two screens ever show different balances, it is a query bug, not a synchronization bug.
- A customer with balance `0` or negative is **not in receivables**. Receivables filter `balance > 0`.
- Receivables order: balance descending and, on a tie, name ascending. Source: `MetricsRepository.Debtors` (`internal/repository/metrics.go:359`, ordering at `:370`), same balance formula as `CustomerRepository.Balance`.
- The dashboard's aging receivables (`MetricsRepository.AgingReceivables`, `internal/repository/metrics.go:402`) use the same balance formula as `Debtors`: both build the balance with `customerBalanceExpr` (`metrics.go:338`) over `customerBalanceFrom` (`metrics.go:343`), which aggregates credit sales and payments per customer before joining, so each payment counts exactly once. Filters `balance > 0`.
- Aging receivables order: `days_overdue` descending first and, on a tie of days, the same tie-break as `Debtors` — balance descending and, if that also ties, name ascending (`metrics.go:412`).
- `total_portfolio` of `GET /customers` (`CustomerRepository.TotalPortfolioBalance`, `internal/repository/customer.go:144`) uses the same per-customer aggregated formula (`customerBalanceExpr`/`customerBalanceFrom`) and excludes balances `≤ 0`, summing only the positive ones — not a global subtraction of all credit sales minus all payments (that was the R-44 bug, fixed in US-064).

## Rule: debt age

```
oldest_debt_at = MIN(created_at) of the customer's sales that have a
                 credit line, among the 'completed' ones
days_overdue   = today − oldest_debt_at
```

It is the date of the **oldest debt without distinguishing partial payments**: a customer who pays without settling in full keeps their original date. This is deliberate — it measures how long that customer has owed something, not how long they have owed that particular peso.

## Rule: credit limit

`customers.credit_limit` is **optional** (`NULL` = no limit defined) and is currently **informational**: it warns, it does not block the sale.

Source: `internal/service/customer.go:54,68` (`Create`, `Update`). Scenario covered in `internal/service/customer_test.go:230` (`TestBalance_CreditLimitWarningScenario`).

If it is ever decided that it should block, that is a product change Andrey must approve: a shopkeeper who cannot extend credit because of a cap they set wrong stops using the system.

## Rule: payments (*abonos*)

An *abono* is a row in `customer_payments`. It is not tied to a specific sale — it is applied to the customer's total balance, not to an invoice. That is how the notebook the system replaces works, and changing it to per-invoice allocation would be a redesign, not an adjustment.

```
valid methods (customer_payments.method):  cash · transfer · card     (default cash)
customer_payments.register_id:             the register open at the time, or NULL
```

Schema: migration 21 (`internal/database/migrate.go:588-595`, US-082). Every *abono* recorded before it is `cash` with `register_id NULL`: it changes no balance and no past register close.

- **Method.** `POST /customers/{id}/payments` takes `{amount, method, note}`. A missing or empty `method` is stored as `cash`; any value other than `cash`, `transfer`, `card` → 400 (`internal/service/customer.go:113-119`). The form offers `Efectivo` / `Transferencia` / `Tarjeta`, default `Efectivo`; the note stays optional (`frontend/lib/abono.ts`, `frontend/app/customers/[id]/CustomerDetailClient.tsx`).
- **Register link.** The server links the *abono* to the open register, if any, whatever the method and the role (`internal/service/customer.go:121-133`). Only cash *abonos* linked to a register enter its expected cash (`caja.md`, "Rule: expected cash at close").
- **Who may register one** (PO, 2026-11-05). Any authenticated role may call the endpoint; it is outside the admin group (`internal/handler/handler.go:285-287`) and the check lives in the service. While a register is open, a cashier or an admin may register an *abono*. With no open register, only an admin may; a cashier gets 403 `Debe haber una caja abierta para registrar un abono` (`internal/service/customer.go:126-130`, `:35`).
- **Unattributed *abonos*.** An admin's *abono* with no open register is stored with `register_id NULL` and appears in no register close. When it is cash, the UI shows the toast `Este abono no quedará en ningún cierre de caja` next to the save confirmation (`CustomerDetailClient.tsx:140-142`).
- An amount `≤ 0` or above the current balance is rejected (`internal/service/customer.go:110-111`, `:148-149`).
- `GET /customers/{id}/payments`, `GET /customers/{id}/detail` (`payments`) and the 201 response carry `method` and `register_id`.
- **HTTP codes of `POST /customers/{id}/payments`** (`internal/handler/customer.go:190-207`): validation errors → 400 (amount, method, amount above the balance, unknown customer); cashier with no open register → 403; internal failures (open-register lookup, a customer lookup failing for a reason other than "not found", the balance query) → 500 `error registrando abono`, and nothing is stored. An unknown customer answers 400 `cliente no encontrado`, not 404 (`internal/service/customer.go:136-139`); roadmap R-83.
- Order of checks: amount and method first, then the open register and role, then the customer and balance. A cashier with no open register therefore gets 403 even for an unknown customer.

## Rule: deleting a customer

`DELETE /customers/{id}` (admin) is a **logical delete**: it sets `customers.active = 0` (migration 19) and never removes the row, because sales and payments reference it.

- A customer with balance `> 0` cannot be deleted → 409. Balance `0` or negative (overpaid) → deactivated, 200. Unknown or already inactive id → 404.
- Inactive customers are hidden from `GET /customers` and its search (`q`). Everything else keeps resolving them by id: customer detail, balance, payments, sale detail, the dashboard's recent sales, and the receivables metrics (`Debtors`, `AgingReceivables`, `total_portfolio`), which do not filter by `active`. Totals therefore do not change when a customer is deleted.
- A credit sale (`fiado`, payment method `credit`) to an inactive or missing customer is rejected (400, `internal/repository/sale.go`, `ErrCustomerInactive`). A cash, card or transfer sale that merely names an inactive customer is accepted. A payment (`abono`) cannot target an inactive customer either: its balance is never above zero, and `CreatePayment` rejects an amount above the balance.

Source: `internal/service/customer.go:83`, `internal/repository/customer.go:31,107`, `internal/handler/customer.go:110`.

## Rule: collection WhatsApp message

The customer detail `Cobrar` button and the WhatsApp link on each Customers list row open the user's own WhatsApp with a pre-filled reminder; nothing is sent automatically and no WhatsApp API is used (`product.md:34`). Both build the URL with the same helper, `buildCollectionWhatsAppUrl` (`frontend/lib/whatsapp.ts:22-27`), and read the business name from the business config (`GET /setup`, available to every role).

```
URL:            https://wa.me/<phone>?text=<URL-encoded message>
<phone>:        the customer phone's digits, prefixed with 57 unless they already start with 57
with a name:    Hola <cliente>, le escribimos de <negocio>. Le recordamos que tiene un saldo pendiente de <$saldo>. Gracias.
without a name: Hola <cliente>, le recordamos que tiene un saldo pendiente de <$saldo>. Gracias.
```

Example: business `Tienda Don Carlos`, customer `Ana`, balance 45,000 → `Hola Ana, le escribimos de Tienda Don Carlos. Le recordamos que tiene un saldo pendiente de $45.000. Gracias.`

- The business name is trimmed. An empty or whitespace-only name (not configured, config still loading, or `GET /setup` failed) uses the text without a name; the action stays available (`frontend/lib/whatsapp.ts:9-16`).
- Visibility is unchanged: on the detail, `Cobrar` appears when the balance is `> 0` and is disabled without a phone; on the list, the link appears only with debt and a phone.
- The business name must be read inside `AppLayout`, which mounts `BusinessConfigProvider`: `CustomerDetailContent` (`frontend/app/customers/[id]/CustomerDetailClient.tsx:299`) and `CustomerRow` (`frontend/app/customers/page.tsx:523`). A hook in a component that itself returns `<AppLayout>` gets the empty default (US-083, formerly R-61).
- The customer detail takes its id from the URL path, not the route param (`frontend/app/customers/[id]/CustomerDetailClient.tsx:304-305`, see `servidor.md` → Dynamic pages). The id must be a positive integer. Otherwise (`/customers/abc`, `/customers/_`) no `/customers/<x>/detail` request is sent and the page shows `Cliente no encontrado` (US-086).

---

## What does NOT exist yet

- **Customer risk scoring** — roadmap `R-17`. It is the strongest differentiator not yet built: with the payment history already stored, a score, suggested limit and early late-payment alert can be computed. No local competitor stores or analyzes this.
- **Blocking a sale for exceeding the limit** — pending product decision.
- **Interest or late fees** — never considered; the Colombian shopkeeper does not charge them.
