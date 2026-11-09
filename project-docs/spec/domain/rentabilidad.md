# Domain — Profitability (cost, margin, profit)

> **Type:** spec · **Current:** yes · **Rules over:** gross profit, net profit, margin
> **Updated:** 2026-11-09 (US-092: an opening *fiado* balance is not revenue or profit) · 2026-10-01 · Verified against the code on 2026-09-23 (US-059: metrics in `internal/repository/metrics.go`).

"How much I sold" is not "how much I earned", and the difference between the two is what stops a shopkeeper from selling at a loss without knowing it. `find-pos.md:41` identified it as a gap local competitors do not cover.

An opening *fiado* balance (*deuda anterior*, `fiado.md` "Rule: opening balance", US-092) is not a sale: it never counts as revenue, cost, gross or net profit, or in any sales report or dashboard sales figure.

---

## Rule: gross profit

```
gross profit = SUM( (sale_items.unit_price − products.cost) * sale_items.quantity )
               only for products with cost > 0
               only sales with status = 'completed'
```

Source: `internal/repository/metrics.go:86` (`MetricsRepository.Profit`; `cost > 0` filter at `:91`). Dashboard and assistant read this same function.

**The `cost > 0` filter is the critical rule.** A product with no configured cost adds no profit — it is excluded from the calculation instead of assuming zero cost, because assuming zero would report 100% margin, which would be a dangerous lie.

That is why every profit or margin answer must be accompanied by the **count of products with no configured cost**, so the owner knows the real figure is higher than the one shown.

## Rule: net profit

```
net profit = gross profit − expenses of the same range
```

See [caja.md](caja.md) for the expense categories.

## Rule: margin

```
margin % = (gross profit / revenue) * 100
           0 if revenue = 0
```

Per product, in the top by margin:

```
margin_pct(product) = ((AVG(unit_price) − cost) / AVG(unit_price)) * 100
```

Source: `internal/repository/metrics.go:166` (`MetricsRepository.TopByMargin`). The dashboard shows the top by margin only if there are at least 2 products with a configured cost in the range; with fewer, the list is empty (`internal/handler/dashboard.go:87-90`).

`AVG(unit_price)` is used because the same product may have been sold at different prices (discount, wholesale) within the range.

## The contrast that sells

**The best-selling product is almost never the one that earns the most.** `research/2026-05-verticales-y-rentabilidad.md:93-95` (formerly `find-pos.md`) documents it with the real case: rice at 11% margin and 89 units versus cooking oil at 38% and 24 units.

Any screen or answer that shows "best seller" should be able to show that contrast. It is the difference between delivering a report and delivering advice.

## Rule: price snapshot

`sale_items.unit_price` is a **copy** of the price at the time of sale. Changing a product's price or cost **does not alter already-recorded sales**. Any historical margin calculation uses the price of that time and the product's **current** cost — a known approximation, not a bug.

---

## What does NOT exist yet

- Cost history per product. Cost is a single, current value, so historical margin uses today's cost against the price of then.
- Products with no turnover / dead capital — US-048, pending in SPRINT-09.
- Margin by category or by supplier.
