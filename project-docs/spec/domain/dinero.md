# Domain — Money: whole pesos and the rounding rule

> **Type:** spec · **Current:** yes · **Rules over:** how a peso amount is stored, validated, rounded and compared
> **Updated:** 2026-11-13 (US-102, roadmap R-133 and R-106) · Verified against the code on that date (`internal/service/money.go`, `internal/service/sale.go`, `internal/database/migrate_money.go`, `frontend/lib/saleCalc.ts`).

Every peso amount in the POS is a **whole peso**: sale lines, tax, totals, payments, *abonos*, cash movements, expenses, register counts, prices and costs. Fractions can only come from `quantity × price` (a 0,5 kg line) and from a tax rate, and they are rounded by the one rule below, on the server, and mirrored 1:1 in the frontend.

Storage: money columns stay `REAL` in SQLite and `float64` in Go, but they only ever hold integers (exact in a `float64` up to 2^53, and every sum of whole pesos stays exact). `int64` was rejected (US-102 Q-2): it needs 13 tables and 27 columns rebuilt as `INTEGER`, because the pure-Go driver fails to scan a `REAL` that prints in exponent form (`1e+06`) into an `int64`. Quantities, stock and tax rates are not money and stay fractional.

## Rule: rounding

Half away from zero (2,5 → 3), with integer arithmetic so that Go and TypeScript agree on every half (in JS `1.005 * 1000` is `1004.9999999999999`). Implemented in `internal/service/money.go` (`LineBase`, `LineTax`, `RoundHalfUp`, `IsWholePeso`) and mirrored in `frontend/lib/saleCalc.ts`.

Per line:

1. `lineBase = round(quantity × unitPrice)`, the quantity taken in thousandths (the catalog allows at most 3 decimals).
2. `lineSubtotal = lineBase − itemDiscount`.
3. `lineTax = round(lineSubtotal × rate)`, with the rate in basis points (19% = 1900, 5% = 500). Any product rate goes through the same rule, not only 0, 5 and 19: an 8% *impoconsumo* on $1.001 gives 80.

Per sale: `subtotal = Σ lineSubtotal`, `tax = Σ lineTax`, `total = subtotal − discount + tax`. The global discount does not reduce the tax base (unchanged rule).

**Per line, not per sale** (US-102 Q-1): the stored line, the ticket and the DIAN document add up with no allocation step. The cost is that the total can differ by a peso from a calculator on the ticket's subtotal: 3 lines of $1.001 at 19% give tax 190 × 3 = 570 and total $3.573 (per sale it would be 571 and $3.574).

Certified numbers:

| Input | Result |
|---|---|
| $10.001 × 1 at 19% | subtotal 10.001, tax 1.900, total **$11.901** (was $11.901,19) |
| $50 × 1 at 5% | tax 3 (2,5 rounds up), total 53 |
| $1.050 × 1 at 19% | tax 200 (199,5 rounds up), total 1.250 |
| 0,5 kg × $9.999/kg, no IVA | lineBase 5.000 (4.999,5), total 5.000 |
| 0,125 kg × $9.999/kg | 1.250 (1.249,875) |
| $10.001 at 19%, item discount $1 | subtotal 10.000, tax 1.900, total 11.900 |
| $10.001 at 19%, global discount $500 | total 10.001 − 500 + 1.900 = 11.401 |

## Rule: inputs and comparisons

- **The server prices the sale.** It ignores the client's `tax` and `subtotal` (and each item's `subtotal`) and prices from the product's current `tax_rate`, the quantities, the unit prices and the discounts (`internal/service/sale.go`).
- **Client total, exact.** When the request carries a `total`, it must equal the server total: otherwise 400 `total inconsistente: esperado <server>, recibido <client>`. The tolerance used to be $1 (total) and $0,5 (payments); it is now 0. A `total` that is not whole → 400 `el total debe ser un número entero de pesos`. When no `total` is sent (table checkout) nothing is compared and the payments are checked against the server total.
- **Payments, exact.** `Σ payments = total`, every supplied line whole and `> 0` (`la suma de los pagos ($X) no coincide con el total de la venta ($Y)`). Example, total 11.901: cash 5.000 + credit 6.901 → 201; cash 5.000 + credit 6.900 → 400.
- **Zero-total sale.** When the server total is 0 (every line discounted 100%), `payments` may be empty: the sale is stored with total 0 and no `sale_payments` row (`payment_method` `cash`); a supplied payment line is rejected. It adds nothing to any register figure or balance. Before US-102 a total `≤ 0` was rejected (`el total de la venta debe ser mayor a cero`, US-096). A sale with a total above 0 and no payments → 400 `la venta debe tener al menos un método de pago`.
- **Discount bounds.** An item discount that is negative or larger than that line's rounded `lineBase`, or a global discount that is negative or larger than `Σ lineSubtotal`, → 400. A discount equal to the base is allowed (a free line; a global discount of 10.001 on one $10.001 line at 19% leaves tax 1.900 and total 1.900).
- **Whole-peso inputs.** Not whole → 400 with a Spanish message naming the field, nothing stored: unit price, item and global discount, payment line, `cash_tendered`, `change_amount`, *abono*, cash movement, expense, opening and closing register amounts, product `sale_price`, `cost` and `wholesale_price`, purchase `unit_cost` (`service.ErrNotWholePeso` and field-specific messages). Price × quantity fractions are not an input error: the rule rounds them. The CSV catalog import already required whole pesos (`catalogo.md`).
- **Table checkout.** The order total is priced with the same function from the non-void `order_items` and each product's current `tax_rate`, at read time and again at checkout; tax is not stored on `orders` or `order_items` (`restaurante.md`).

## Rule: existing data (migration 24)

Migration 24 (`internal/database/migrate_money.go`) rounds every money column to a whole peso (half away from zero), recomputes `sale_items.subtotal` and `sales.subtotal`, `tax` and `total` with the rule above, and sets a sale's total to `Σ sale_payments` when they differ by at most $1. It never fails startup because of a data inconsistency, leaves `quantity`, `stock`, `tax_rate` and `customers.credit_limit` alone, and runs after the `<db>.pre-migration-23-YYYYMMDD` safety copy.

- A *fiado* residue left by IVA (credit 11.901,19 paid with 11.901 → 0,19) ends at exactly 0 in the case where both sides round the same way.
- `sale_payments`, `expenses` and `customer_opening_balances` have `CHECK (amount > 0)`: a value under 0,5 is clamped to $1 instead of rounding to 0 (the clamp can add up to $1 to a customer's credit, and is reported only if it crosses the drift threshold; roadmap R-179).
- **Drift.** When a customer's balance moves by a whole peso or more beyond the rounding of the old balance (two credits of 100,4 and one abono of 200,8 → 100 + 100 − 201 = −1, shown as `Saldo a favor: $1`), one `audit_log` row `money_rounding_migration` (entity `system`, entity id 0, details with the sales visited and the customer ids) is written for the lowest-id active admin; with no active admin the row is skipped and the counts are logged. The scan includes inactive customers. No adjusting entry is invented.
- Known limits: a residue below $1 may not end at 0 when its credit and abono fractions round in opposite directions (credit 100,6 → 101, abono 100 → balance 1 where it was 0,6; not flagged as drift); roadmap R-178.

## Display

`formatCOP` shows whole pesos. The cart, the ticket, the register, the *fiado* list and the *abono* modal all read whole values now; the old red `$0` with `Abonar` (a $0,19 residue) cannot appear for new sales. The *abono* modal's `Total` still sends the exact balance (`frontend/lib/abono.ts`), which is now the same whole number.

## Tests

Backend: `internal/service/sale_money_test.go`, `internal/handler/sale_money_test.go`, `internal/handler/order_money_test.go`, `internal/handler/money_validation_test.go`, `internal/factelect/pos_adapter_test.go`, `internal/database/migrate_money_test.go`; the property test `TestMoneyProperties_WholePeso` (no build tag since US-102) asserts whole values over the full domain. Frontend: `frontend/lib/saleCalc.test.ts`, `frontend/lib/abono.test.ts`.
