# Domain — Restaurant (restaurant mode and tables)

> **Type:** spec · **Current:** yes · **Rules over:** restaurant mode, tables and table state
> **Updated:** 2026-10-13 (US-076: `PUT /api/setup` password exception) · Verified against the code on that date (US-055: `internal/repository/dining_table.go`, `internal/handler/tables.go`, `frontend/lib/nav.ts`; US-056: `internal/repository/order.go`, `internal/handler/order.go`, `internal/handler/handler.go:213-219`).

Module for cafeterías and restaurants: tables, orders and a bar screen (SPRINT-13). Today table management, the switch and per-table orders (open, items, void, cancel) exist; checkout and the bar arrive in US-057 and US-058 and will add their rules here.

Enum values below (`libre`, `ocupada`, `por_cobrar`, route names like `/mesas`) are identifiers in the code and API and stay in Spanish; they are glossed once.

---

## Restaurant mode

- `business_config.restaurant_mode` (0/1, default 0) decides whether the business uses the module. It is enabled in Settings with the "Modo restaurante" toggle (admin only, via `PUT /api/setup`).
- Off: the tables section is not shown in Settings, nor "Mesas" (`/mesas`, tables) and "Barra" (`/barra`, bar) in the sidebar.
- On: the tables section appears in Settings and the sidebar shows "Mesas" and "Barra" to both `admin` and `cashier`. In addition the POS hides the parked carts kept in `localStorage` (mobile chip, desktop chip and "Dejar en espera": `showParked = !restaurant_mode`, `frontend/app/pos/page.tsx:622,1030,1141,1155`); when off, the POS does not change.
- `business_config.lan_access_enabled` (0/1, default 0) is created together with this migration but belongs to US-054 (LAN access); see `servidor.md`.
- `PUT /api/setup` replaces the whole row, except `owner_email_app_password`: an omitted or empty value keeps the stored one (US-076; see `caja.md`). The Settings form must always resend `restaurant_mode` and `lan_access_enabled` (read from `GET /setup`) so that saving another field does not turn them off.

## Tables

Table `dining_tables`: unique `name`, `zone` (free text for grouping), `seats` (not yet used in screens), `sort_order`, `active`.

- Only `admin` creates, edits or deactivates tables (`POST/PUT/DELETE /api/tables`); any authenticated user reads them (`GET /api/tables`, active ones only, ordered by `sort_order` then `name`).
- Empty or whitespace-only name: 400.
- Duplicate name: 409 "Ya existe una mesa con ese nombre (puede estar desactivada)" (UI message, Spanish). Uniqueness is exact (case-sensitive: "Mesa 1" and "mesa 1" coexist) and **includes deactivated tables**: "Mesa 1" cannot be recreated if a deactivated one exists. A deactivated table does not appear in the UI and today cannot be reactivated.
- Delete = logical deactivation (`active = 0`), never `DELETE`, because historical orders reference it.
  - With an `status='open'` order on that table: 409 "La mesa tiene un pedido abierto" and the table stays active.
  - `closed` or `cancelled` orders do not block.
  - The check and the deactivation are atomic: a single `UPDATE ... WHERE NOT EXISTS (open order)` (`internal/repository/dining_table.go:139-144`), so an order opened in parallel cannot leave an inactive table with an open order (R-59 resolved). `SQLITE_BUSY` → 409 "retry" (`internal/handler/order.go:14`, `tables.go`).
- Opening an order requires an active table (see Orders).

## Table state (derived, never stored)

`GET /api/tables` returns `state`, `open_order_id`, `open_total` and `opened_minutes`.

| State | Condition |
|---|---|
| `libre` (free) | There is no `open` order with that `table_id`. `open_order_id` null, `open_total` and `opened_minutes` 0/null. |
| `ocupada` (occupied) | There is an `open` order and at least one non-void item in `pending` or `preparing`. An open order **without items**, or with **all its items `void`**, is also `ocupada`. |
| `por_cobrar` (awaiting payment) | There is an `open` order, at least one non-void item, and all non-void items are `ready` or `delivered`. |

- `open_total` = sum of `quantity * unit_price` of items with `status != 'void'`. Example: coffee $4,500 x2 + cake $8,000 x1, both `ready` → `por_cobrar`, `$17,000`; with the cake voided → `$9,000`.
- `opened_minutes` = whole minutes since `orders.created_at` (UTC against UTC).
- A list with no tables serializes as `[]`, not `null`.

## Orders

An order is the open tab of a table. All authenticated users (`cashier` and `admin`) operate orders; without a token, 401. They do not write to `sales` (checkout arrives with US-057). Routes in `internal/handler/handler.go:213-219`.

| Endpoint | Result |
|---|---|
| `POST /api/orders` `{table_id}` | 201 `{id,type,table_id,status,created_at}`. 400 if `table_id` is missing/≤0 or `type` differs from `table`; 404 table does not exist or is deactivated; 409 if the table already has an `open` order or `SQLITE_BUSY`. |
| `GET /api/orders?status=open` | `[{id,table_id,table_name,total,items_pending,opened_minutes}]`, `[]` if none. Valid `status`: `open|closed|cancelled`, anything else → 400. `items_pending` = non-void items in `pending` or `preparing`. |
| `GET /api/orders/{id}` | `{id,type,table_id,status,total,items[]}`; `items` includes the `void` ones and is never `null`. 404 if it does not exist. |
| `POST /api/orders/{id}/items` `{product_id,quantity,note?}` | 201 with the item `{id,product_id,product_name,quantity,unit_price,subtotal,note,status}`. 404/400 product nonexistent or inactive; 400 invalid quantity; 409 order not `open`. |
| `PATCH /api/orders/{id}/items/{itemId}` `{quantity?,note?,status?}` | Updates only what is sent. 400 invalid quantity or `status`; 404 item not part of the order; 409 item `void` or order not `open`. |
| `DELETE /api/orders/{id}/items/{itemId}` | Voids (`status='void'`), 200 with the item. 409 if it was already `void` (no second audit entry) or order not `open`. |
| `POST /api/orders/{id}/cancel` `{reason}` | 200; 400 empty or whitespace-only reason; 404 does not exist; 409 if not `open`. |

Rules:

- Lifecycle: `open → cancelled` (`closed` arrives with US-057). `type` is always `table` in v1.
- **Only one `open` order per table**, guaranteed by an `INSERT ... SELECT ... WHERE EXISTS (active table) AND NOT EXISTS (open order)` in a single statement (`internal/repository/order.go:131-136`), not by a unique index (there is no migration 19). Whoever loses the race gets 409 (`ErrTableHasOpenOrder`) or, if SQLite answers `SQLITE_BUSY` (code 5), `ErrBusy` → 409 "El sistema está ocupado, reintenta en un momento" (`internal/handler/order.go:14,22-23`), never 500.
- `unit_price` is a **snapshot** of `products.sale_price` when the item is added; changing the price afterwards does not alter existing orders.
- `quantity` must be an **integer > 0** (`0.5`, `2.0`, `"2"`, 0 and negatives → 400). It is stored in the REAL column as an integer.
- Order `total` = Σ `quantity * unit_price` of items with `status != 'void'`; line subtotal = `quantity * unit_price`.
- `PATCH` accepts `status` ∈ `pending|preparing|ready|delivered`, with no ordering rule between transitions. `void` or any other value → 400: an item is voided only by `DELETE`.
- `quantity` and `note` of a `ready` or `delivered` item can be edited; that does not change its `status`. Only a `void` item is immutable.
- `order_items.ready_at` is set when moving to `ready` from another state; resending `ready` does not overwrite it; on leaving `ready` it goes back to `NULL` (`internal/repository/order.go:235-237`).
- Adding, editing or voiding items only while the order is `open`; otherwise 409 with no changes.
- Every item change (add, edit, void) touches `orders.updated_at` in the same transaction.
- **Item void:** `audit_log` `action='order_item_void'`, `entity='order_item'`, `entity_id=<itemId>`, `user_id` of whoever voids. `details` is JSON with exactly the keys `order_id, product_id, product_name, quantity, unit_price, note, status, table_id, user_id`, where `status` is the item's status right before voiding. No physical `DELETE`.
- **Cancellation:** only if `open`; `reason` is mandatory (after `TrimSpace`). It moves to `cancelled`, sets `closed_at` and `updated_at`, and inserts `audit_log` `action='order_cancel'`, `entity='order'`, `entity_id=<orderId>`, `details=<reason>`, all in one transaction. Items are kept. The table becomes `libre` with no extra code (derived state).
- No stock deduction when adding an item (the sale will do it, US-057). No automatic refresh of `/mesas`: it refreshes on entering, on navigating and after each own action (roadmap R-63).

### Screens

- `/mesas`: grid of active tables with color by state; occupied ones with total and minutes open; tapping a free one opens an order and navigates to `/mesas/<id>`; tapping an occupied or awaiting-payment one navigates to `open_order_id` without creating another; if the `POST` gives 409 it refreshes and navigates to the existing order. With no tables, a link to Settings → Tables.
- `/mesas/[orderId]`: catalog with search (tapping adds the item to the server immediately, no "send to kitchen"), lines with quantity, note and status (pendiente/preparando/listo/entregado — pending/preparing/ready/delivered), void with confirmation (struck-through line), cancellation with reason. A failure on add shows a message and leaves no phantom line. At 320px catalog and order alternate through tabs. There are no buttons to advance an item's status (arrives with the bar, US-058).
- Dynamic route statically exported with `generateStaticParams [{orderId:'_'}]`; see `servidor.md`.

## Schema (migration 18)

`dining_tables`, `orders` (`type` `table|takeaway`, `status` `open|closed|cancelled`) and `order_items` (`status` `pending|preparing|ready|delivered|void`) are created at once; the `orders`/`order_items` endpoints were introduced by US-056 (see Orders). Foreign keys are enforced at runtime (roadmap R-56, closed), so deleting a referenced row fails; integrity also relies on logical deletion and service validations.
