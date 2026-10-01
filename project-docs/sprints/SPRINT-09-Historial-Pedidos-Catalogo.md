# SPRINT-09 — Customer History, Smart Orders and Catalog

**Goal:** Give the shopkeeper customer context at the moment of charging, automate the decision of what to order from the supplier, and let them share their catalog over WhatsApp without leaving the system.

**Start date:** (to be defined)

**References:** `project-docs/find-pos.md` (now `research/2026-05-verticales-y-rentabilidad.md`), `project-docs/SPRINT-08-Rentabilidad-Cartera.md`

**Incoming technical debt:**
- US-034 — Factus sandbox validation (still open from SPRINT-07, requires external access)
- US-042 — Sales velocity / days remaining (postponed from SPRINT-08 for lack of enough history — **enters this sprint** as US-042c)

---

## Tasks

| US | Name | Status | Date |
|----|------|--------|------|
| US-043 | Customer history in the POS at checkout | [ ] Pending | — |
| US-044 | Supplier order suggestion | [ ] Pending | — |
| US-045 | Catalog shareable over WhatsApp | [ ] Pending | — |
| US-042c | Dashboard: stock with days remaining (sales velocity) | [x] Completa (2026-10-01) | 2026-10-01 |

---

## Recommended implementation order

| Order | US | Reason |
|-------|----|--------|
| **1** | US-042c | Query over `sale_items` that already exists. No schema change. Closes SPRINT-08 debt. |
| **2** | US-043 | Same base query as US-042c — reuses the aggregation pattern over sales. No schema change. |
| **3** | US-044 | Extends US-042c's sales-velocity logic to the inventory module. New endpoint. |
| **4** | US-045 | Requires a new column in settings + a public endpoint without JWT. Goes last so as not to block the sprint. |

---

## US-042c — Dashboard: stock with days remaining (sales velocity)

**As** Don Carlos (owner),
**I want** to see not only which products are below the minimum, but how many days I have left of each at the current sales pace,
**so that** I can prioritize what to order before running out of stock.

**Estimated effort:** 0.5 day. **Impact:** Medium-high. A direct improvement to the "Reponer pronto" (restock soon) panel of US-042b.

**Technical context:** US-042b already shows products with `stock < min_stock`. This US adds `days_remaining = stock / avg_daily_sales_7d` to each item. If `avg_daily_sales_7d = 0` (product with no sales in 7 days), the field is not shown (no division by zero).

**Days-remaining rule:**
- `avg_daily = SUM(quantity) / 7` over `sale_items JOIN sales` where `sales.created_at >= now - 7 days` and `sales.status = 'completed'`
- `days_remaining = ROUND(stock / avg_daily)` — only if `avg_daily > 0`
- If `avg_daily = 0` → do not show days (product with no recent movement)

**Acceptance Criteria:**
- [x] The "Reponer pronto" panel on the dashboard shows `days_remaining` next to the stock deficit
- [x] Format: `Arroz Diana · 3 unid / mín 10 · **2 días restantes**` (UI text, Spanish)
- [x] If `avg_daily = 0`, show only `3 unid / mín 10` without days (do not show "∞ días" nor "—")
- [x] Ordered by `days_remaining ASC` (most critical first). Products without velocity data go last
- [x] If the product is below the minimum but has velocity = 0 → it appears in the panel without days remaining
- [x] Limited to top 5. This limit is unchanged
- [x] Responsive: works at 320px

**DoD:**
- [ ] `GET /api/dashboard/today` extends `low_stock []{ product_id, name, stock, min_stock, avg_daily, days_remaining }` — `avg_daily` and `days_remaining` are `null` if there are no sales in 7 days
- [ ] Query does not fail if there are no sales in the last 7 days (LEFT JOIN)
- [ ] Test: product with 5 units, average 2.5/day → `days_remaining = 2`; product with 0 sales in 7 days → `days_remaining = null`
- [ ] Clean build (`go build`, `npm run build`)

---

## US-043 — Customer history in the POS at checkout

**As** Laura (cashier) or Don Carlos,
**I want** to see, on selecting a customer in the cart: their last visit, what they usually buy, and whether they have debt,
**so that** I know who I am dealing with before closing the sale and avoid extending credit to someone who already owes too much.

**Estimated effort:** 2 days. **Impact:** High. No low-cost Colombian POS shows this in the checkout flow.

**Technical context:** All the necessary data exists in `sales JOIN sale_items JOIN products WHERE customer_id = ?`. The balance is already computed in `customerRepo.Balance()`. No schema change. New endpoint `GET /api/customers/:id/pos-summary`.

**What the panel shows in the POS (UI text, Spanish):**
```
┌─ Pedro Ramírez ──────────────────────────────┐
│ Última visita: hace 3 días                    │
│ Lo de siempre: Arroz · Aceite · Leche         │
│ ⚠ Debe: $80.000 (22 días)                     │
└──────────────────────────────────────────────┘
```

**Acceptance Criteria:**

**Cart panel (appears on selecting a customer):**
- [ ] On selecting a customer in the POS cart, a compact panel appears below the selector
- [ ] Shows "Última visita: hace X días" (based on the customer's most recent sale, any payment method)
- [ ] If they have never purchased: "Cliente nuevo — sin historial"
- [ ] Shows "Lo de siempre:" ("the usual") with the top 3 most purchased products by frequency (by number of transactions, not by quantity)
- [ ] If they have fewer than 3 distinct products in history → show the ones there are (1 or 2)
- [ ] If they have no purchase history at all → do not show the "Lo de siempre" section
- [ ] If `balance > 0`: show a debt badge: `⚠ Debe: $80.000 (22 días)` in yellow or red depending on age (SPRINT-08 rule: < 7d gray, 7–14d yellow, > 14d red)
- [ ] If `balance = 0`: show no debt section (do not show "Sin deuda" — it simply does not appear)
- [ ] If `balance > credit_limit` (and `credit_limit != null`): show an additional notice: `"Superó tope de crédito ($150.000)"`
- [ ] The panel does not block the sale — it is informational only
- [ ] The panel can be closed/minimized with one click (so as not to take space during a quick checkout)
- [ ] On deselecting the customer: the panel disappears

**Quick actions from the panel:**
- [ ] A "Agregar lo de siempre" button adds the top 3 products to the cart at once (if they have stock)
- [ ] If any of the top 3 has no stock → it is skipped and a toast is shown: `"Arroz no tiene stock — no se agregó"`
- [ ] The button only appears if the customer has history

**DoD:**
- [ ] `GET /api/customers/:id/pos-summary` returns: `{ last_visit_at, top_products []{product_id, name, times_bought}, balance, days_overdue, credit_limit }`
- [ ] `last_visit_at` = date of the customer's most recent sale (any method)
- [ ] `top_products` = top 3 by `COUNT(DISTINCT sale_id)` in `sale_items JOIN sales WHERE customer_id = ? AND status = 'completed'`
- [ ] Endpoint returns 200 with null fields if the customer has no history (not 404)
- [ ] Test: customer with 5 sales and 8 distinct products → returns exactly 3 products ordered by descending frequency
- [ ] Test: new customer with no sales → `last_visit_at = null`, `top_products = []`, `balance = 0`
- [ ] Panel visible at 320px (vertical stack, reduced text if necessary)
- [ ] The endpoint call is made only when a customer is selected (not on every keystroke)

---

## US-044 — Supplier order suggestion

**As** Don Carlos (owner),
**I want** to see a list of what I should order from the supplier this week, with the suggested quantity based on my recent sales,
**so that** I do not run out of stock of the products that turn over most without having to check inventory product by product.

**Estimated effort:** 2 days. **Impact:** High. Completely differentiating — no low-cost Colombian POS does it.

**Technical context:** Extends US-042c's sales-velocity logic. Query over `sale_items JOIN sales`. No schema change. New endpoint `GET /api/inventory/reorder-suggestions`. New view in the inventory module.

**Suggestion logic:**
```
avg_daily_7d = SUM(quantity_sold) / 7   -- last 7 days
days_remaining = stock / avg_daily_7d
suggested_order = CEIL(avg_daily_7d * 14) - stock   -- to have 2 weeks of stock
```

**Criterion to appear in the list:**
- Product with `active = true`
- `avg_daily_7d > 0` (has recent sales — if it does not sell there is no suggestion)
- `days_remaining < 14` (less than 2 weeks left at the current pace)

**Acceptance Criteria:**

**Suggestions view (section in inventory or a separate tab):**
- [ ] New "Pedido sugerido" section reachable from the inventory module (button or tab)
- [ ] Shows a list of products meeting the criterion: `avg_daily > 0` AND `days_remaining < 14`
- [ ] Per product: name, current stock, daily average, days remaining, suggested quantity to order
- [ ] Format: `Arroz Diana · 5 unid · 2.5/día · **2 días** → Pedir 30 unid`
- [ ] Ordered by `days_remaining ASC` (most urgent first)
- [ ] Urgency badge by days: `≤ 2 days` red, `3–7 days` yellow, `8–13 days` gray
- [ ] If no products meet the criterion: empty panel with the message "Todo en orden — ningún producto requiere pedido esta semana"
- [ ] A "Copiar lista" button generates plain text with the list to paste in WhatsApp to the supplier:
  ```
  Pedido sugerido — [fecha]
  - Arroz Diana: 30 unid
  - Aceite Girasol: 12 unid
  - Leche Alpina: 24 unid
  ```
- [ ] No item limit (show all that meet the criterion — it is a work list, not a widget)

**DoD:**
- [ ] `GET /api/inventory/reorder-suggestions` returns `[]{product_id, name, stock, avg_daily, days_remaining, suggested_order}` ordered by `days_remaining ASC`
- [ ] Query uses an exact 7-day window from `now() - 7 days`
- [ ] Products with `active = false` excluded
- [ ] Products with `avg_daily = 0` excluded (no sales in the last 7 days)
- [ ] `suggested_order` minimum 1 (if the calculation gives 0.something → round up)
- [ ] Test: product with 35 units in stock, average 5/day → `days_remaining = 7`, `suggested_order = 35` (to reach 70 = 14 days)
- [ ] Test: product with 20 units, average 1/day, `days_remaining = 20` → does not appear in the list (> 14 days)
- [ ] Test: inactive product with sales → does not appear
- [ ] Responsive: table with horizontal scroll at 320px if columns do not fit

---

## US-045 — Catalog shareable over WhatsApp

**As** Don Carlos (owner),
**I want** to generate a link with my products and prices to send to customers over WhatsApp,
**so that** they can see what I have and at what price without having to call me.

**Estimated effort:** 2.5 days. **Impact:** Medium-high. A natural extension of US-035's WhatsApp button.

**Technical context:** Requires a new field `catalog_token TEXT` and `catalog_enabled BOOLEAN DEFAULT 0` in the `settings` table (or a new `catalog_config` table). The public endpoint `/public/catalog/:token` carries no JWT middleware. The catalog page is static in Next.js (`/catalog/[token]/page.tsx`) which calls the backend. No photos in the MVP.

**MVP scope — what is NOT included:**
- Product photos (storage + upload complexity)
- Shopping cart in the catalog (it is read-only)
- Color or logo customization
- Granular selection of which products appear (MVP: all active ones with price > 0)

**Acceptance Criteria:**

**Configuration (in Settings):**
- [ ] New "Catálogo público" section on the settings page
- [ ] Toggle to enable/disable the catalog (default: disabled)
- [ ] On first activation: a random UUID token is generated and saved in the DB
- [ ] Shows the generated link: `http://localhost:PORT/catalogo/[token]` (or the server IP if configured)
- [ ] "Copiar link" button copies to the clipboard
- [ ] "Compartir por WhatsApp" button opens `https://wa.me/?text=...` with the link
- [ ] If the catalog is disabled: the public endpoint returns 404 (the link stops working)
- [ ] "Regenerar link" button generates a new token (invalidates the previous one) — with prior confirmation

**Public catalog (`/catalogo/:token`):**
- [ ] Page reachable without login — rendered from the frontend
- [ ] Shows: business name (from `settings.business_name`), list of active products with price > 0
- [ ] Per product: name, COP-formatted price (`$12.000`), unit of measure
- [ ] If `wholesale_price` exists → it is not shown (wholesale price is private)
- [ ] Ordered by category (if it exists), then alphabetically
- [ ] If the token does not exist or the catalog is disabled: a "catálogo no disponible" page
- [ ] No pagination in the MVP (full list)
- [ ] Responsive: works on mobile (the main usage channel is the customer's phone)
- [ ] No app navbar (it is a public page, not the shopkeeper's app)

**DoD:**
- [ ] New SQL migration: `ALTER TABLE settings ADD COLUMN catalog_token TEXT`, `ADD COLUMN catalog_enabled INTEGER DEFAULT 0`
- [ ] `GET /public/catalog/:token` — no JWT middleware — returns `{ business_name, products []{name, sale_price, unit, category} }` if `catalog_enabled = 1` and the token matches
- [ ] `PUT /api/settings/catalog` — with JWT — enables/disables and regenerates the token
- [ ] Token generated with `crypto/rand` (UUID v4 or 32 random hex chars)
- [ ] Test: catalog enabled with a valid token → 200 with active products
- [ ] Test: invalid token → 404
- [ ] Test: catalog disabled with a valid token → 404
- [ ] Test: inactive products or with price = 0 → do not appear in the catalog
- [ ] Clean build

---

## Sprint success criterion

- Laura selects "Pedro Ramírez" in the POS and sees within 1 second that he owes $80k for 22 days and usually takes rice, oil and milk
- Don Carlos opens "Pedido sugerido" and in 10 seconds has a list ready to copy and send to the supplier over WhatsApp
- Don Carlos generates a catalog link and sends it to a new customer without needing to show them the system
- The dashboard panels (US-042b) now show days remaining for products with recent sales

---

## Technical notes for the dev

- **US-043 and US-044** share the same `avg_daily_sales_7d` sub-query — extract it as a CTE or helper function so SQL is not duplicated
- **US-045** public endpoint must be registered in the router **before** the JWT middleware, not inside the authenticated group
- **Catalog token:** use `crypto/rand` (not `math/rand`) to generate the token
- **US-042c** is an extension of the existing `low_stock` field in `dashboard/today` — add `avg_daily` and `days_remaining` as nullable fields, it does not break clients that already read the field
- The sales-velocity query must use `julianday('now') - julianday(sales.created_at) <= 7` for SQLite compatibility
