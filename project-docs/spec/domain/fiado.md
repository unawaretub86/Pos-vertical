# Domain — Fiado (credit to customers)

> **Type:** spec · **Current:** yes · **Rules over:** customer balance, payments (*abonos*), receivables (*cartera*)
> **Updated:** 2026-11-17 (R-236: new "Rule: customer export", the `Exportar` CSV on `Clientes`; customers-page refs of "Rule: what the cashier sees" re-verified at `f728729`) · 2026-11-14 (US-107: the credit payment method reads `Fiado` in the payment panel) · 2026-11-13 (US-102: whole-peso balances, exact payments, no cents residue; R-106 closed, see `dinero.md`) · 2026-11-10 (US-093: pointer to the go-live reset) · 2026-11-10 (US-098: pointer to `tiempo.md` for day windows and stored timestamps) · 2026-11-09 (US-092: opening *fiado* balance (*deuda anterior*) added to the balance formula and the debt age, new "Rule: opening balance"; balance and metrics line refs re-verified at `8540648`) · 2026-11-08 (US-096: sale validation rules and property-tested balance) · 2026-11-08 (US-091: certified balance after the shift and after credit-sale voids, under "Rule: balance") · 2026-11-07 (US-085: aging chips on the Customers list under "Rule: debt age"; Customers-list line refs re-verified at `df481de`) · 2026-11-07 (US-084: shared *abono* modal with quick amounts on the Customers list, detail and dashboard; `Saldo a favor` display; verified against the code at `9a28d52`) · 2026-11-07 (US-043: credit confirmation at checkout, customer context from `pos-summary`; verified against the code at `992d1ad`) · 2026-11-06 (US-086: customer detail id from the URL path; US-083: collection WhatsApp message with the business name) · 2026-11-05 (US-082: *abono* method, register link, who may register; line refs re-verified) · Verified against the code on 2026-11-06 for the collection message (US-083), on 2026-11-05 for *abonos* (US-082) and on 2026-09-23 (US-059: metrics in `internal/repository/metrics.go`; US-063: aging receivables use the same balance formula; US-064: `total_portfolio` of `GET /customers` uses the same formula and excludes balances `≤ 0`).

*Fiado* (selling on credit) is **the product's differentiator**. Competitors treat it as an accounting field; here it is a relationship with history. Any change to these rules changes the value proposition, not just the code.

---

## Model

A sale is paid with one or more lines in `sale_payments`, each with a method. When the method is `credit`, that line **is** the debt. The only debts table is `customer_opening_balances`, at most one row per customer, for debt carried over from the paper notebook at go-live ("Rule: opening balance", US-092). Otherwise the debt is the sum of what was put on credit minus the sum of what was paid back.

```
valid methods (sale_payments.method):  cash · transfer · card · credit
              (sales.payment_method):  cash · transfer · card · credit · mixed
```

`sales.payment_method = 'mixed'` is a sale-level label; the real breakdown always lives in `sale_payments`.

In the payment panel (`/pos` and table checkout) the methods read `Efectivo`, `Transferencia`, `Tarjeta` and `Fiado` (`credit`), from one map used by the buttons, the mixed-mode first line and the extra-line selector (`PAYMENT_METHOD_LABELS`, `frontend/components/SalePaymentPanel.tsx:16-21`, used at `:310`, `:324`, `:353`; US-107, R-185; before it the button read `Credito`). The dashboard and the void dialog also say `Fiado`; the customer ticket, the close ticket and Reports keep `Crédito` (PO, 2026-11-14). Test: `frontend/lib/paymentMethodLabels.test.ts`.

- The payment lines of a sale must sum **exactly** to the server-recalculated total (and the client total, when sent, must equal it), all in whole pesos (`internal/service/sale.go`, US-102; the tolerance was ±0.5). A sale-level discount below 0 is rejected with 400 `el descuento no puede ser negativo`, and one above the sale subtotal with 400 `el descuento no puede superar el subtotal de la venta`. A sale whose server total is 0 (every line discounted 100%) is valid and needs no payment line; a total above 0 with no payment → 400 `la venta debe tener al menos un método de pago` (the US-096 rule "total `≤ 0` → 400" was replaced by US-102). Full rules and numbers: `dinero.md`.

## Rule: customer balance

The balance is **never stored**. It is computed on the fly in every query.

```
balance(customer) = customer_opening_balances.amount        (0 when absent)
                  + SUM(sale_payments.amount where method='credit'
                        and the sale is 'completed')
                  − SUM(customer_payments.amount)
```

Source: `CustomerRepository.Balance` (`internal/repository/customer.go:123-139`); lists and metrics use `customerBalanceExpr` (`internal/repository/metrics.go:338`) over `customerBalanceFrom` (`metrics.go:349-361`, which joins `customer_opening_balances ob`). Both carry the opening amount, so they never disagree (US-092, `TestCustomerOpening_T2_EveryBalanceQueryCarriesOpening` asserts that `Balance`, `List`, `TotalPortfolioBalance`, `Debtors`, `AgingReceivables` and `GetPOSSummary` give the same number). The *abono* method plays no part: every *abono* subtracts, whatever its method.

Consequences that must be respected:

- **A voided sale stops counting automatically**, because the filter is `status='completed'`. There is no need to "reverse" the debt by hand. A void lowers the balance by the sale's credit portion only and restores the stock of its product items; the register's `expected` changes only by the cash portion (`caja.md`). Certified cases (US-091): $50.000 on credit, no *abonos* → 50.000 → 0; the same sale after *abonos* of $20.000 cash + $10.000 transfer + $5.000 card → 15.000 → −35.000 (`total_portfolio` 0); mixed $30.000 cash + $20.000 credit → 20.000 → 0, `expected` 130.000 → 100.000; a void of a credit sale whose register is already closed changes the balance the same way and writes nothing to either register. Tests: `TestShiftCert_T3_CreditVoidLowersBalance`, `TestShiftCert_T4_CreditVoidOnClosedRegister` (`internal/handler/shift_certification_test.go`).
- **Certified shift balance (US-091).** $50.000 on credit, then *abonos* of $20.000 cash, $10.000 transfer and $5.000 card → balance **$15.000** in `GET /customers/{id}/balance` and in the `GET /customers` row (`total_portfolio` 15000); each *abono* is stored with its method and the open register. After the register is closed, a cashier's *abono* answers 403 and stores nothing; an admin's $1.000 cash *abono* answers 201 with `register_id` null and the balance goes 15.000 → 14.000. Tests: `TestShiftCert_T1_FullShift`, `TestShiftCert_T2_CloseAndAfterClose`.
- **There is no such thing as a stale balance.** Any screen that shows a balance reads it from the same query. If two screens ever show different balances, it is a query bug, not a synchronization bug. Property-tested: `Balance`, `GET /customers`, `pos-summary` and `total_portfolio` agree after every step (US-096). The invariant is `balance = opening + completed credit − abonos`; the US-096 generator never records an opening balance, so opening is 0 for every customer it creates and its check reduces to credit − *abonos* (`TestCustomerOpening_T12_InvariantWithAndWithoutOpening`: $50.000 credit, *abonos* $20.000 + $10.000 + $5.000 → 15.000; plus an opening of $100.000 → 115.000).
- A customer with balance `0` or negative is **not in receivables**. Receivables filter `balance > 0`.
- **Display** (`customerBalanceLabel`, `frontend/lib/abono.ts:80-84`), the same on the Customers list (`frontend/app/customers/page.tsx:593`, `:650-662`) and the customer detail `Saldo pendiente` card (`frontend/app/customers/[id]/CustomerDetailClient.tsx:169`, `:217-220`): balance `> 0` → the amount (`$450.000`) in `var(--danger)`; `0` → `Al día` in `var(--success)`; `< 0` (overpaid, a *saldo a favor*) → `Saldo a favor: $50.000` in `var(--success)`. A negative balance happens, for example, when a credit sale is voided after the customer paid it: $50.000 on credit, an *abono* of $50.000, void the sale → `-50000` (US-084); or when an opening balance is corrected below the *abonos* already received ("Rule: opening balance").
- **Balances are whole pesos** (US-102, roadmap R-106 and R-133 closed). IVA is rounded per line (`dinero.md`): one item at $10.001 with 19% IVA puts exactly $11.901 on credit, and an *abono* of $11.901 leaves $0 (`Al día`); no sub-peso residue shown as `$0` in red can come from a new sale. Existing residues were settled by migration 24 (`dinero.md` "Rule: existing data"), which can leave a rare ±$1 drift reported in the audit log.
- Receivables order: balance descending and, on a tie, name ascending. Source: `MetricsRepository.Debtors` (`internal/repository/metrics.go:366`, ordering at `:377`), same balance formula as `CustomerRepository.Balance`.
- The dashboard's aging receivables (`MetricsRepository.AgingReceivables`, `internal/repository/metrics.go:409`) use the same balance formula as `Debtors`: both build the balance with `customerBalanceExpr` (`metrics.go:338`) over `customerBalanceFrom` (`metrics.go:349`), which aggregates credit sales and payments per customer before joining, so each payment counts exactly once; the opening balance joins one row at most (`UNIQUE(customer_id)`). Filters `balance > 0`.
- Aging receivables order: `days_overdue` descending first and, on a tie of days, the same tie-break as `Debtors` — balance descending and, if that also ties, name ascending (`metrics.go:419`).
- `total_portfolio` of `GET /customers` (`CustomerRepository.TotalPortfolioBalance`, `internal/repository/customer.go:147`) uses the same per-customer aggregated formula (`customerBalanceExpr`/`customerBalanceFrom`) and excludes balances `≤ 0`, summing only the positive ones — not a global subtraction of all credit sales minus all payments (that was the R-44 bug, fixed in US-064).

## Rule: debt age

```
oldest_debt_at = the older of
                   MIN(created_at) of the customer's 'completed' sales
                   that have a credit line, and
                   customer_opening_balances.debt_date
days_overdue   = today − oldest_debt_at
```

Source: `customerOldestDebtExpr` (`internal/repository/metrics.go:344`) = `COALESCE(min(sp_sum.oldest_at, ob.debt_date), sp_sum.oldest_at, ob.debt_date)`. The `COALESCE` is required: SQLite's two-argument `min(a, b)` returns NULL when either side is NULL, which would drop the opening date of a customer with no credit sale. Used by `oldest_debt_at` in `GET /customers`, `days_overdue` in `pos-summary` and dashboard `aging_receivables`, so the aging chips, the row badge, the debt badge and the checkout `> 30` days confirmation all see the opening debt (US-092). Example: opening $150.000 dated 45 days ago and no sales → `days_overdue` 45; with a credit sale from 60 days ago → 60; with only a credit sale from 10 days ago plus the opening → 45. Test: `TestCustomerOpening_T3_DebtAge`.

It is the date of the **oldest debt without distinguishing partial payments**: a customer who pays without settling in full keeps their original date. This is deliberate — it measures how long that customer has owed something, not how long they have owed that particular peso.

**Aging chips on the Customers list** (US-085, helpers in `frontend/lib/aging.ts`, page `frontend/app/customers/page.tsx:427-461`):

- **Day count.** Whole elapsed 24-hour periods, `floor((now − oldest_debt_at) / 24 h)`; a `null` date counts as 0 (`daysOverdue`, `aging.ts:22-26`). The row badge (`hoy` / `Nd`, desktop only) and the chips use the same function and the same `now`, so a row showing `14d` is never in `15–30 días`. This is not a calendar-day count: a *fiado* from yesterday at 20:00 Bogotá reads 0 days (`hoy`) at 08:00 today (roadmap R-117). `now` is taken when `GET /customers` answers (`page.tsx:304`, `:312`), so ages refresh on load, on every search and after an *abono*; a tab left open across midnight without one of those keeps the previous ages (roadmap R-118).
- **Buckets** (`agingBucket`, `aging.ts:28-32`): `Hasta 14 días` = 0–14 days, `15–30 días` = 15–30, `Más de 30 días` = `> 30` (the same threshold as the checkout confirmation, "Rule: credit limit"). Boundaries: 0 and 14 → `Hasta 14 días`; 15 and 30 → `15–30 días`; 31 → `Más de 30 días`.
- **Amounts** (`bucketTotals`, `aging.ts:34-51`; label `bucketLabel`, `:77-79`): each chip shows `<label>: <balance sum with formatCOP> (<count>)`. Only balances `> 0` count; `0` and negative (*saldo a favor*) do not, a residue of $0,19 does. Example: $75.000 at 5 days, $300.000 at 20, $330.000 at 28 → `Hasta 14 días: $75.000 (1)`, `15–30 días: $630.000 (2)`, `Más de 30 días: $0 (0)`.
- **Scope.** Chips are computed over the list the page loaded (`page.tsx:348`). `GET /customers` has no pagination, so with an empty search they cover every active customer and add up to the `por cobrar en total` subtitle. While a search is typed they cover its results only; the subtitle stays global (`total_portfolio` ignores `q`).
- **Filter** (`filterByBucket`, `aging.ts:53-63`; `toggleBucket`, `:65-75`). One chip at a time. Tapping a chip shows only its customers with debt, AND with the search, keeping the current sort (`page.tsx:349`). Tapping the active chip clears the filter, even when its count has dropped to 0; tapping another chip switches to it. An inactive chip with count 0 is shown but disabled (`page.tsx:441`). Each chip exposes `aria-pressed`. The chips are hidden while the list loads and wrap at 320 px.
- **Empty bucket.** While a chip is active and no row is left (with or without a search, on selection or after an *abono* refetch), the table reads `Ningún cliente con deuda en este rango`, with no `Agregar primer cliente` button (`page.tsx:467-471`). Example: `15–30 días` active with Beto $300.000 as its only customer; an *abono* of $300.000 → `15–30 días: $0 (0)` stays active and tappable, and the table shows that text.
- **Colors.** The chips use theme variables only (`var(--accent)` and `var(--accent-dim)` when active, `var(--surface-2)` / `var(--border)` / `var(--text-secondary)` otherwise). Row colors keep their own bands (`< 7`, `7–14`, `> 14`), which do not match the chip buckets (roadmap R-116).

## Rule: credit limit

`customers.credit_limit` is **optional** (`NULL` = no limit defined; a stored `0` also means no limit). It does **not block** the sale: the sale backend has no limit check, and a credit sale only requires an active customer (`internal/service/sale.go`, `internal/repository/sale.go`). At checkout it warns and, in the cases below, asks for confirmation (US-043).

Source: `internal/service/customer.go:54,68` (`Create`, `Update`; the value is not validated). Scenario covered in `internal/service/customer_test.go:230` (`TestBalance_CreditLimitWarningScenario`).

```
credit amount = SUM(amount of the sale's payment lines with method 'credito')
                (IVA and discounts already applied; not the cart subtotal,
                 and in a mixed payment only the credit part)
reasons       = 'limit'    when credit_limit > 0 and balance + credit amount > credit_limit
                'overdue'  when balance > 0 and days_overdue > 30
                none at all when credit amount ≤ 0
```

Source: `creditAmount` and `needsCreditConfirmation` (`frontend/lib/creditCheck.ts:22-36`); the limit test is `exceedsCreditLimit` (`frontend/lib/saleCalc.ts:63-66`). Equal to the limit is not exceeding; exactly 30 days does not trigger. `balance` and `days_overdue` come from `GET /customers/{id}/pos-summary` ("Rule: customer context at checkout"); `days_overdue` follows "Rule: debt age".

- **Notice.** In the payment panel's credit block, when `balance + credit amount > credit_limit`, a red notice shows `⚠️ Tope de crédito: $500.000. Deuda actual: $450.000.` (`frontend/components/SalePaymentPanel.tsx:478-493`). It does not block.
- **Confirmation.** When `Cobrar` is pressed, the existing validations pass, a payment line is credit and at least one reason applies, a `ConfirmDialog` opens before anything is sent (`SalePaymentPanel.tsx:248-269`, `:540-548`):
  - title `¿Fiar de todas formas?`, confirm `Fiar`, cancel `Cancelar`;
  - the message has one sentence per reason, limit first, joined by one space (`creditConfirmationMessage`, `frontend/lib/creditCheck.ts:39-47`): `Con esta venta la deuda queda en $510.000 y el tope es $500.000.` and `Tiene deuda de hace 45 días.`;
  - `Cancelar` closes the dialog and changes nothing (cart, payment lines, cash received and customer stay); `Fiar` completes the sale with the same data as a sale with no dialog.
- **Fail-open.** If the summary request failed there is no confirmation and `Cobrar` works: a failed lookup must not stop the counter, and the backend does not block either.
- Applies to `/pos` and to table checkout: both render the shared `SalePaymentPanel`.

Example: limit $500.000, balance $450.000, 26 days, a $60.000 sale on credit → `limit` → dialog. The same customer with a $40.000 sale → no dialog.

If it is ever decided that it should block, that is a product change Andrey must approve: a shopkeeper who cannot extend credit because of a cap they set wrong stops using the system.

## Rule: customer context at checkout (POS summary)

`GET /customers/{id}/pos-summary` (`internal/handler/handler.go:293`, `internal/handler/customer.go:267-282`) is open to **any authenticated role**, outside `RequireAdmin`, because the cashier needs it at the counter. Repository: `GetPOSSummary` (`internal/repository/customer.go:233-295`).

```
{ "data": { "last_visit_at": "2026-11-04 05:15:55" | null,
            "top_products":  [ { "product_id", "name", "times_bought" } ],   // never null
            "balance":       450000,
            "days_overdue":  26 | null,
            "credit_limit":  500000 | null } }
```

- `balance`: the shared formula (`customerBalanceExpr`/`customerBalanceFrom`, "Rule: customer balance").
- `days_overdue`: set only when `balance > 0`; whole days since the oldest debt (completed credit sale or opening balance date), as in "Rule: debt age" (`internal/repository/customer.go:256-261`). A customer who paid off an old *fiado* and owes a new one keeps the old date, so the `> 30` confirmation can fire for a fresh debt (roadmap R-102).
- `last_visit_at`: `MAX(created_at)` of the customer's **completed** sales of any method; voided sales are ignored. It is a **zone-less UTC string** `YYYY-MM-DD HH:MM:SS` (`CURRENT_TIMESTAMP`), not RFC 3339 like other endpoints (`internal/repository/customer.go:260-268`). A failure of that query is returned as an error (500), not swallowed.
- `top_products`: at most 3 **active** products (`p.active = 1`) by number of distinct completed sales, descending, ties by name ascending (`internal/repository/customer.go:270-279`). An inactive product is never returned.
- A customer with no sales answers 200 with `last_visit_at: null`, `top_products: []`, `balance: 0`, `days_overdue: null`. An inactive customer still answers 200.
- HTTP: non-numeric id → 400 `id inválido`; unknown id → 404 `cliente no encontrado`; any other failure → 500 `error obteniendo cliente`.

**Payment panel** (`frontend/components/SalePaymentPanel.tsx`). The customer picker, and therefore this panel, exists only while some payment line is credit. The panel no longer calls `/customers/{id}/balance`.

- **Loading.** One `pos-summary` request per customer selection, none while typing in the picker search; a response for a customer no longer selected is discarded (`SalePaymentPanel.tsx:192-205`). While it loads, `Cargando historial...` shows and, when a line is credit, `Cobrar` is disabled (`:293-294`, `:417-421`). Known gap: re-selecting the same customer shows the previous summary until the new one arrives (roadmap R-104).
- **Failure.** `No se pudo cargar el historial del cliente`, no badge, no confirmation (`:422-426`).
- **Header** (the collapse toggle, `:437-452`): the debt badge when `balance > 0`, otherwise `Historial del cliente`; on the right `Ocultar ▴` when expanded or `Ver más ▾` when collapsed. One click toggles; a new selection starts expanded. Collapsing hides the details below, never the confirmation dialog.
- **Debt badge** (`debtBadge`, `frontend/lib/posSummary.ts:71-76`): `Debe: $450.000 (26 días)`, `(1 día)` for 1, no parenthesis when `days_overdue` is null; none when `balance ≤ 0` (no "Sin deuda" text). Color by age: `< 7` `var(--text-muted)`, `7–14` `var(--warning)`, `> 14` `var(--danger)` (`agingColor`, `:63-68`). The Customers list uses the same thresholds, but its `7–14` band is a hard-coded `#ca8a04` with `rgba(...)` backgrounds (`agingColor` / `agingBg`, `frontend/app/customers/page.tsx:44-54`), not `var(--warning)` (roadmap R-103).
- **Details** (expanded, `SalePaymentPanel.tsx:453-473`):
  - `Superó tope de crédito ($500.000)` in `var(--danger)` when `credit_limit > 0` and `balance > credit_limit` (`limitExceededLabel`, `posSummary.ts:125-128`);
  - `Cliente nuevo — sin historial` when `last_visit_at` is null and `top_products` is empty (`newCustomerLabel`, `:99-102`);
  - `Última visita: hoy` / `Última visita: hace 1 día` / `Última visita: hace N días`, counting calendar days in `America/Bogota` after reading `last_visit_at` as UTC (`lastVisitLabel`, `posSummary.ts:84-96`). Example: now 07/11/2026 08:00 Bogotá, `last_visit_at` `2026-11-07 01:00:00` (06/11 20:00 Bogotá) → `hace 1 día`, not `hoy`;
  - `Lo de siempre: Arroz · Aceite · Leche`, names in server order (`usualProductsLabel`, `:105-108`).
- **`Agregar lo de siempre`** (`/pos` only: table checkout passes no `onAddUsual` callback, so the button is hidden there; also hidden without `top_products`). It fetches each top product with `GET /products/{id}` and adds one unit of each (`addUsualProducts`, `frontend/app/pos/page.tsx:598-612`). A `product`-type item with no stock for one more unit is skipped with toast `Arroz no tiene stock — no se agregó`; an inactive product or a failed request is skipped with `Arroz no está disponible — no se agregó`; a `service`-type item ignores stock (`decideUsualAdd`, `frontend/lib/posSummary.ts:140-148`). While the fetches are in flight the button reads `Agregando...` and is disabled, and a second press adds nothing (`SalePaymentPanel.tsx:281-291`, `:459-471`). Known gap: the result is written over the cart as a value, not a functional update (roadmap R-105).

## Rule: what the cashier sees (US-105)

Receivables are the owner's information (`spec/personas.md`). The cashier needs customer names to sell on credit and one customer's balance to take an *abono*, never the list of who owes how much (PO GATE 1, 2026-11-14).

- **`GET /customers` is role-shaped** (`listCustomers`, `internal/handler/customer.go:15-55`; role branch `:22-33`). An admin gets the response as before: rows with `balance` and `oldest_debt_at`, `total_portfolio`, and `has_balance=true` filtering to balances `> 0`. A cashier, **or a request with no claims** (fail closed, as `catalog_token` in `internal/handler/setup.go:33-35`), gets `{"customers": [...]}` only: each row has exactly `id`, `name`, `phone`, `address`, `notes`, `credit_limit`, `created_at`, active customers ordered by name; `total_portfolio` is not computed; `has_balance` is ignored, so the list never tells her who owes; `q` still filters. Example (Dora $30.000, Eli $15.000, Fer $0, Gus −$20.000): cashier → 4 rows with no balance, also with `?has_balance=true`; `?q=Do` → Dora; admin → `total_portfolio` 45000, `has_balance=true` → Dora and Eli. Tests: `TestCustomerRole_T1`..`T4` (`internal/handler/customer_role_test.go`).
- **`GET /customers/{id}/detail` is admin-only** (inside `RequireAdmin`, `internal/handler/handler.go:299-301`): a cashier gets 403, an admin 200 as before (`TestCustomerRole_T6_DetailIsAdminOnly`). It carries the customer's full credit and *abono* history and opening balance.
- **Open to any role** (one customer at a time, `handler.go:288-298`): `GET /customers/{id}`, `/pos-summary` (credit checkout, "Rule: customer context at checkout"), `/balance` (the POS *abono*), `/payments` and `POST /customers/{id}/payments`. A cashier still gets `pos-summary` and `/balance` with Dora's `balance` 30000 (`TestCustomerRole_T5_CashierKeepsPerCustomerBalance`), so the checkout picker, debt badge, limit notice and `¿Fiar de todas formas?` confirmation work as before: the picker reads only `id`, `name`, `phone`, `notes`, `credit_limit` from the list (`frontend/components/SalePaymentPanel.tsx:15-21`, `:53`).
- **Customers screens are admin-only.** `/customers` and `/customers/{id}` check `canSeeCustomers` (`frontend/lib/customerAccess.ts`; admin only, `frontend/lib/customerAccess.test.ts`). For a cashier they call `router.replace('/pos')`, render nothing and send no `GET /customers` or `/customers/{id}/detail` (`frontend/app/customers/page.tsx:368`, `:410`, `:480`; `frontend/app/customers/[id]/CustomerDetailClient.tsx:133`, `:157`, `:165`). `Clientes` stays `adminOnly` in the menu (`frontend/components/Sidebar.tsx:68-76`). Known gap: with no stored user, `/customers` renders blank instead of reaching `AppLayout`'s `/login` redirect, because its guard returns before `<AppLayout>` (`page.tsx:480`, the same pattern as `frontend/app/audit/page.tsx:135`; roadmap R-214). The detail page's guard sits inside `AppLayout`, so it is not affected.
- **Where the cashier takes an *abono*:** `Punto de Venta › Abonar` ("Rule: payments" → "The *abono* modal" → "From the POS").
- **The customer export has no endpoint** (R-236, "Rule: customer export"). It is built in the browser from `GET /customers` on a page a cashier never reaches, and if she called the endpoint herself she would get rows with no balance or date, so no file with receivables can be built for her.

## Rule: customer export

`Clientes · Cartera` has an `Exportar` button (`frontend/app/customers/page.tsx:490`) that downloads the customer list as a CSV, so the owner can take the receivables with him at any time (`spec/product.md:41`: the export of data is never blocked). PO GATE 1, 2026-11-17. Helpers: `frontend/lib/customerExport.ts`; page wiring `handleExport` (`page.tsx:448-457`) and `downloadCSV` (`page.tsx:61-69`, a Blob link as in `Reportes`).

- **Who.** Admin only: the button renders after the page guard (`page.tsx:480`), and a cashier is sent to `/pos` (`page.tsx:410`, "Rule: what the cashier sees"). There is no backend change and no export endpoint.
- **Which rows.** On click the page requests `GET /customers` with no `q` and no `has_balance` (`runCustomerExport`, `customerExport.ts:95-113`, request at `:99`), so the file holds **every active customer** whatever search or debt-age chip is on screen, including those at $0 and with *saldo a favor*. Logically deleted customers (`active = 0`) are not exported. There is no total row: a future import (roadmap R-219) would read it as a customer.
- **Format** (`buildCustomersCSV`, `customerExport.ts:70-82`). UTF-8 with a BOM, `;` as the delimiter (Excel's default in Colombia, as in the catalog template, `catalogo.md`), CRLF between lines and no line break after the last one. Header (`CUSTOMERS_CSV_HEADER`, `:28`), frozen by the PO as the layout of the free *fiado* spreadsheet's `Clientes` sheet and of R-219: `Nombre;Teléfono;Tope de crédito;Saldo;Deuda más antigua`.
  - `Tope de crédito` and `Saldo`: whole pesos as plain digits, no `$`, no thousands separator, rounded with `Math.round` as `formatCOP` does; `-` for *saldo a favor*; an empty cell when there is no credit limit (`moneyCell`, `:55-58`). Plain digits read as a number in Excel under any regional setting.
  - `Deuda más antigua`: `dd/mm/yyyy`, the calendar day of `oldest_debt_at` ("Rule: debt age") in `America/Bogota`, written **only when the balance is `> 0`**; empty for $0, *saldo a favor* or no date (`oldestDebtCell`, `:61-67`). `GET /customers` still sends a date for a customer who paid off, because `oldest_debt_at` is the oldest credit sale ever.
  - `Nombre` and `Teléfono` are quoted RFC 4180 style (inner `"` doubled) only when they hold `;`, `"`, CR or LF (`textCell`, `:49-52`).
  - Rows are sorted by name with `localeCompare`, as the screen's name sort (`:71`), not in the server's binary order (which puts `Álvaro` after `Gus`).
- **File name.** `clientes-YYYY-MM-DD.csv` with today's date in `America/Bogota` (`customersExportFileName`, `:85-88`): at 21:30 Bogotá on 17/11/2026 (already 18/11 in UTC) → `clientes-2026-11-17.csv`.
- **Outcomes.** `runCustomerExport` never rejects; the page maps its result (`page.tsx:455-456`). Customers found → the file downloads, no toast. No active customers (`customers` empty or `null`) → no file, toast `No hay clientes para exportar`. Request failed (or the download threw, `customerExport.ts:107-111`) → no file, toast `No se pudo exportar los clientes`. The button is disabled from the click until the request settles, in all three outcomes, so a double click gives one file (`setBusy`, `:96`, `:102`, `:105`; `disabled={exporting}`, `page.tsx:490`).
- **Worked example** (T-2). `Gus` (no limit, −$20.000), `Ana Pérez` (`3001234567`, limit $200.000, $150.000 since `2026-09-25T05:00:00Z`), `Beto` (no limit, $1.234.567 since `2026-11-18T02:30:00Z`), `Fer` (limit $0, $0), `Álvaro; "el Mono"` (`+57 300 123 4567`, limit $50.000, $30.000 since `2026-11-10T15:00:00Z`) → after the header, in this order:

  ```
  "Álvaro; ""el Mono""";+57 300 123 4567;50000;30000;10/11/2026
  Ana Pérez;3001234567;200000;150000;25/09/2026
  Beto;;;1234567;17/11/2026
  Fer;;0;0;
  Gus;;;-20000;
  ```

  `Beto` reads 17/11 (Bogotá), not the UTC 18/11.
- **Known limits.** A phone with a leading `0` may lose it when Excel reads the cell as a number; names starting with `=`, `+`, `-` or `@` are not escaped against spreadsheet formulas (only the admin creates customers). `Dirección` and `Notas` are not exported (PO GATE 1). Whether Excel opens the file in five columns is checked by hand (`gaps.md` R-236 M-1).
- **Tests:** `frontend/lib/customerExport.test.ts` T-1..T-8 and T-10; access by role: `frontend/lib/customerAccess.test.ts`.

## Rule: payments (*abonos*)

An *abono* is a row in `customer_payments`. It is not tied to a specific sale — it is applied to the customer's total balance, not to an invoice. That is how the notebook the system replaces works, and changing it to per-invoice allocation would be a redesign, not an adjustment.

```
valid methods (customer_payments.method):  cash · transfer · card     (default cash)
customer_payments.register_id:             the register open at the time, or NULL
```

Schema: migration 21 (`internal/database/migrate.go:588-595`, US-082). Every *abono* recorded before it is `cash` with `register_id NULL`: it changes no balance and no past register close.

- **Method.** `POST /customers/{id}/payments` takes `{amount, method, note}`. A missing or empty `method` is stored as `cash`; any value other than `cash`, `transfer`, `card` → 400 (`internal/service/customer.go:113-119`). Every *abono* form in the UI is the shared `AbonoModal` (`frontend/components/AbonoModal.tsx`), which offers `Efectivo` / `Transferencia` / `Tarjeta`, default `Efectivo`, with an optional note, and always sends `method` (`:86`). Since US-084 that includes the dashboard: its modal is now titled `Registrar abono` and no longer stores every *abono* as cash.
- **Register link.** The server links the *abono* to the open register, if any, whatever the method and the role (`internal/service/customer.go:121-133`). Only cash *abonos* linked to a register enter its expected cash (`caja.md`, "Rule: expected cash at close").
- **Who may register one** (PO, 2026-11-05). Any authenticated role may call the endpoint; it is outside the admin group (`internal/handler/handler.go:296-298`) and the check lives in the service. While a register is open, a cashier or an admin may register an *abono*. With no open register, only an admin may; a cashier gets 403 `Debe haber una caja abierta para registrar un abono` (`internal/service/customer.go:127-136`, `:37`). The cashier's screen for it is `Punto de Venta › Abonar` (US-105, below); the Customers screens and the dashboard are admin-only.
- **Unattributed *abonos*.** An admin's *abono* with no open register is stored with `register_id NULL` and appears in no register close. When it is cash, the UI shows the toast `Este abono no quedará en ningún cierre de caja` next to the save confirmation (`frontend/components/AbonoModal.tsx:87-90`).
- **The *abono* modal** (`AbonoModal`, US-084). It opens from the Customers list `Abonar` button (`frontend/app/customers/page.tsx:759`, modal `:626-632`), the customer detail `Registrar abono` / `Registrar primer abono` (buttons `CustomerDetailClient.tsx:222`, `:426`; modal `:466-472`), the dashboard `Pagar` (`frontend/app/page.tsx:539`, modal `:707-713`) — all three admin-only — and, for any role, `Punto de Venta › Abonar` (`frontend/app/pos/page.tsx:826`, flow `frontend/components/PosAbonoFlow.tsx`, US-105). On the list, a click on the row itself still opens the customer detail (`page.tsx:605`, `:669`).
  - **From the POS** (US-105). `Abonar` sits in the `Punto de Venta` top bar for both roles. It opens a customer search (`Buscar cliente...`, `GET /customers?q=`, debounced like the checkout picker) that lists names and phones only, never a balance (`PosAbonoFlow.tsx:64`). Picking a customer requests that customer's balance once (`GET /customers/{id}/balance`, `:85`) and decides the next step with `posAbonoStep` (`frontend/lib/abono.ts:98-102`): balance `> 0` → this modal with the customer's name and balance (Dora $30.000 → `Saldo pendiente $30.000`, quick amounts `Total`, `Mitad` $15.000, `$20.000`); `0` → `Al día` and `< 0` → `Saldo a favor: $20.000`, with no amount form and a `Buscar otro cliente` button back to the search; a failed lookup → `No se pudo cargar el saldo del cliente`, no form, nothing sent. After a 201 the modal closes and nothing on `/pos` is re-read; the cart, held sales and payment panel are untouched by the whole flow. A cashier with no open register sees the server 403 inside the modal, as below. Tests: `posAbonoStep` block in `frontend/lib/abono.test.ts`; `TestCustomerRole_T7_CashierAbonoFromPOSLowersBalance` (Dora $30.000, cashier cash *abono* $10.000 with an open register → balance 20.000, `register_id` set). Page wiring: manual check `gaps.md` US-105 M-1.
  - Header `Registrar abono`, the customer name and `Saldo pendiente` in `formatCOP`.
  - **Quick amounts** (`quickAbonoAmounts`, `frontend/lib/abono.ts:34-46`), in this order: `Total` = the exact balance from the server, shown whenever the balance is `> 0`, even below $1; `Mitad` = `floor(balance / 2)`; `$50.000`; `$20.000`. `Mitad` and the fixed amounts show only when `1 ≤ amount ≤ balance`. Examples: $450.000 → all four (`Mitad` $225.000); $30.000 → `Total`, `Mitad`, `$20.000`; $0,19 → `Total` only.
  - A quick amount fills the amount field and does not submit. The field is digits only, so it gets the amount rounded to whole pesos, at least 1 when positive (`abonoFieldValue`, `abono.ts:49-51`): `Total` on $11.901 → `11.901`. Since US-102 balances are whole, so the rounding changes nothing (the old $11.901,19 and $0,19 examples are history).
  - **Exact balance.** When the field equals the balance's field value (what `Total` fills), the modal sends the exact balance (`abonoAmountToSend`, `abono.ts:57-59`), the same whole number. Any other value is sent as typed; an empty field is never mapped to the balance. The server rejects an *abono* that is not whole: 400 `el monto del abono debe ser un número entero de pesos`, nothing stored.
  - **Preview** (`abonoPreview`, `abono.ts:62-77`): while the amount is `> 0` and not above the balance, `Saldo anterior: $450.000 → Nuevo saldo: $400.000` ($50.000 typed), `Nuevo saldo: $0` when it covers the balance. Above the balance: the confirm button is disabled and `El monto no puede superar $450.000` shows, with no preview. Empty: disabled, no message. The confirm button reads `Registrar <amount>`, `Guardando...` while the request is in flight; a second press sends nothing (`AbonoModal.tsx:59-60`, `:79-99`, `:252-259`).
  - **Closing.** `Cancelar`, the `×` button, a click on the overlay or `Escape` close it without sending anything (`AbonoModal.tsx:66-73`, `:133`, `:145`, `:249`). Known gap: they also close it while the request is in flight, and the *abono* is still saved (roadmap R-109).
  - **Errors** (400, 403, 500) stay inside the open modal with the server message, e.g. `Debe haber una caja abierta para registrar un abono`; nothing else changes.
  - **After a 201**: the success toast (plus the no-register notice when it applies), the modal closes, and the screen re-reads its data. The Customers list re-requests `GET /customers` with the current search **without** the loading spinner (`fetchCustomers`, `frontend/app/customers/page.tsx:374-383`, `:631`), so the row balance and the `por cobrar en total` subtitle update in place: Dora $450.000 and Beto $80.000, *abono* of $50.000 to Dora → Dora `$400.000`, `$480.000 por cobrar en total`. The detail reloads `GET /customers/{id}/detail`; the dashboard subtracts the amount from its aging list as before (`handlePaymentSuccess`, `frontend/app/page.tsx:306`).
- An amount `≤ 0` or above the current balance is rejected (`internal/service/customer.go:110-111`, `:148-149`).
- `GET /customers/{id}/payments`, `GET /customers/{id}/detail` (`payments`) and the 201 response carry `method` and `register_id`.
- **HTTP codes of `POST /customers/{id}/payments`** (`internal/handler/customer.go:190-207`): validation errors → 400 (amount, method, amount above the balance, unknown customer); cashier with no open register → 403; internal failures (open-register lookup, a customer lookup failing for a reason other than "not found", the balance query) → 500 `error registrando abono`, and nothing is stored. An unknown customer answers 400 `cliente no encontrado`, not 404 (`internal/service/customer.go:136-139`); roadmap R-83.
- Order of checks: amount and method first, then the open register and role, then the customer and balance. A cashier with no open register therefore gets 403 even for an unknown customer.

## Rule: opening balance (*deuda anterior*)

The *deuda anterior* (opening balance) is what a customer already owed in the paper notebook (*cuaderno de fiados*) when the shop started using the POS (US-092). It is one amount with the date the debt started, stored in `customer_opening_balances` (migration 22, `internal/database/migrate.go:598-613`): `customer_id` `UNIQUE`, `amount > 0`, `debt_date`, `note`, `created_by`, `created_at`, `updated_at`.

- **Effect.** It adds to the balance ("Rule: customer balance"), to receivables, `total_portfolio`, the dashboard portfolio and aging, `/reports/portfolio` and the assistant's debtors answer. It ages from `debt_date` ("Rule: debt age"). The credit-limit notice and confirmation at checkout see the larger balance. *Abonos* pay it down with the usual rules ("Rule: payments"). Example: with only an opening balance of $150.000 and an open register, a cash *abono* of $100.000 → 50.000; one of $60.000 → 400 `el pago ($60000) supera la deuda actual ($50000)`; a transfer *abono* of $50.000 → 0 (`TestCustomerOpening_T9_AbonosPayItDown`). A customer whose balance is `> 0` only because of it cannot be deleted (409).
- **Not a sale.** It creates no `sales`, `sale_payments`, `customer_payments` or `register_movements` row. No sales, report, dashboard sales or profit, `by_payment_method`, `credit_new_today` or register query reads the table. Recording or correcting one changes none of these: `GET /reports/sales`; `GET /dashboard/today` (`sold_today`, `credit_new_today`, `credit_collected_today`, `cash_estimated`, profit); `GET /dashboard` (`today`, `yesterday`, `recent_sales`); the open register's `cash_sales` / `expected`; a close's `expected_close`. It is not in `credit_sales` of the customer detail and does not change `last_visit_at` / `top_products` of `pos-summary`. An *abono* that pays it down is real money and counts like any *abono* (`credit_collected_today`, and a cash one in the register's `cash_abonos`). Tests: `TestCustomerOpening_T10_NotASale`, `TestCustomerOpening_T11_AbonoAgainstOpeningIsRealMoney`.
- **Endpoints** (admin only, inside `RequireAdmin`, `internal/handler/handler.go:305-308`; handler `saveOpeningBalance`, `internal/handler/customer.go:287`; service `internal/service/customer.go:170-282`):
  - `POST /customers/{id}/opening-balance` `{amount, debt_date, note}` → 201 `{id, customer_id, amount, debt_date, note, created_by, created_at, updated_at}`.
  - `PUT /customers/{id}/opening-balance` (same body) replaces amount, date and note and sets `updated_at` → 200. A customer with none → 404 `el cliente no tiene deuda anterior`. There is no DELETE.
  - `GET /customers/{id}/detail` (admin only since US-105) carries `opening_balance`: that object, or `null`.
- **Validation.** `amount`: whole pesos from 1 to 100,000,000, else 400 `el monto debe ser un valor entero entre $1 y $100.000.000`. `debt_date`: `YYYY-MM-DD`, else 400 `fecha de la deuda inválida`; a date after today (server-local) → 400 `la fecha de la deuda no puede ser futura`; today is accepted. `note` is optional and trimmed (no test guards the trim yet, roadmap R-142).
- **One per customer.** A second `POST` → 409 `este cliente ya tiene una deuda anterior registrada`. The `UNIQUE(customer_id)` constraint enforces it, so it also holds when two requests race: 10 concurrent requests → one 201, nine 409 (`TestCustomerOpening_T5_OnePerCustomer`). A transcription mistake is fixed with `PUT`, not by entering it again.
- **A correction may leave a negative balance** (PO, 2026-11-09). `PUT` does not check the *abonos* already received. Example: an opening of $150.000 with an *abono* of $100.000, corrected to $50.000, leaves −$50.000, shown as `Saldo a favor: $50.000`. This is the same situation as a credit sale voided after it was paid. The UI gives no warning before saving such a correction (roadmap R-141).
- **Checks and codes, in this order:** cashier → 403 (`RequireAdmin`); non-numeric id → 400 `id inválido`; malformed JSON → 400 `request inválido`; unknown or inactive (`active = 0`) customer → 404 `cliente no encontrado`; then the validation above. Malformed JSON is checked **before** the customer, so malformed JSON for an unknown customer answers 400, not 404 (accepted by the PO, 2026-11-09). Any other failure → 500 `error registrando deuda anterior`, and nothing is stored.
- **Audit.** Each `POST` / `PUT` writes one `audit_log` row **in the same transaction** as the change (`CreateOpeningBalance` / `UpdateOpeningBalance`, `internal/repository/customer.go:360`, `:398`). The row has `entity` `customer`, `entity_id` the customer id and `user_id` the admin. For a create, `action` is `opening_balance_create` with details `{"amount":150000,"debt_date":"2026-09-25"}`. For a correction, `action` is `opening_balance_update` with `{"from_amount":150000,"to_amount":120000,"from_date":"2026-09-25","to_date":"2026-09-20"}`. If the audit insert fails, the change is rolled back (500). Rejected requests write nothing. The audit page labels the rows `Deuda anterior registrada` / `Deuda anterior corregida` and describes them as `Cliente #7: $150.000 desde 25/09/2026` / `Cliente #7: de $150.000 (25/09/2026) a $120.000 (20/09/2026)`. Its action filter offers both (`describeOpeningBalanceAudit`, `frontend/lib/openingBalance.ts:76`; `frontend/app/audit/page.tsx:166-167`). Tests: `TestCustomerOpening_T6_Correction`, `TestCustomerOpening_T8_Audit`.
- **Stored date.** The admin picks a calendar date. It is stored as UTC text `YYYY-MM-DD HH:MM:SS` for 00:00 server-local time of that date, the layout of `MIN(sales.created_at)`, and is bound as a string (`openingDateToStored`, `internal/repository/customer.go:317`). On a Bogotá machine `2026-09-25` → `2026-09-25 05:00:00`. The API returns `debt_date` as `YYYY-MM-DD` server-local.
- **UI** (tuteo, PO 2026-11-09).
  - Customer detail (`frontend/app/customers/[id]/CustomerDetailClient.tsx:270-310`): a `Deuda anterior` card with the amount in `formatCOP`, the date as `dd/mm/yyyy` and the note. Since US-105 the detail page is admin-only ("Rule: what the cashier sees"), so only the admin sees the card (it was visible to every role; the code comment at `:270` still says so). The card has a `Corregir` button, or `Registrar deuda anterior` when there is none. Both open one modal with `Monto`, `Fecha de la deuda` (default and `max` today), `Nota` and `Guardar`; server 400/409 errors stay inside the open modal.
  - `Nuevo cliente` (admin only, create only; `frontend/app/customers/page.tsx:130`, `:290-313`) offers the optional fields `Deuda anterior (cuaderno)` and `Fecha de la deuda`. If an amount is entered, the page sends the `POST` after the customer is created. If that call fails, the customer stays created and the toast reads `Cliente creado, pero la deuda anterior no se guardó. Regístrala desde el detalle del cliente.` (`page.tsx:163`). Today that error toast follows a `Cliente creado` success toast (roadmap R-143).
  - Helpers: `frontend/lib/openingBalance.ts`.
- **Client and server "today".** The date input's `max` and the client pre-check use the browser clock (`todayISODate`, `frontend/lib/openingBalance.ts:32`), while the server enforces the rule with its own clock. On a LAN device whose clock or time zone is wrong, the two may disagree around midnight. The server's 400 then shows in the modal, and nothing wrong is stored. Stored timestamps are UTC text and "today" is the server's local day: see [tiempo.md](tiempo.md).
- **Reset (US-093).** An opening balance is **master data** (PO, 2026-11-09). The reset tool keeps `customer_opening_balances`, like customers, and clears the dry-run *abonos* that paid it down. An opening balance typed as a test during the dry run survives the reset and must be fixed with `Corregir`. Full rule: [reset.md](reset.md).

## Rule: deleting a customer

`DELETE /customers/{id}` (admin) is a **logical delete**: it sets `customers.active = 0` (migration 19) and never removes the row, because sales and payments reference it.

- A customer with balance `> 0` (an opening balance included) cannot be deleted → 409. Balance `0` or negative (overpaid) → deactivated, 200. Unknown or already inactive id → 404.
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
- The business name must be read inside `AppLayout`, which mounts `BusinessConfigProvider`: `CustomerDetailContent` (`frontend/app/customers/[id]/CustomerDetailClient.tsx:111`) and `CustomerRow` (`frontend/app/customers/page.tsx:589-590`). A hook in a component that itself returns `<AppLayout>` gets the empty default (US-083, formerly R-61).
- The customer detail takes its id from the URL path, not the route param (`frontend/app/customers/[id]/CustomerDetailClient.tsx:116-117`, see `servidor.md` → Dynamic pages). The id must be a positive integer. Otherwise (`/customers/abc`, `/customers/_`) no `/customers/<x>/detail` request is sent and the page shows `Cliente no encontrado` (US-086).

---

## What does NOT exist yet

- **Customer risk scoring** — roadmap `R-17`. It is the strongest differentiator not yet built: with the payment history already stored, a score, suggested limit and early late-payment alert can be computed. No local competitor stores or analyzes this.
- **Blocking a sale for exceeding the limit** — pending product decision. A confirmation exists since US-043 ("Rule: credit limit"); a hard block waits for the admin PIN (roadmap R-11).
- **Interest or late fees** — never considered; the Colombian shopkeeper does not charge them.
