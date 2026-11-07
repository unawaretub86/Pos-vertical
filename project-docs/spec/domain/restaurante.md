# Domain — Restaurant (restaurant mode and tables)

> **Type:** spec · **Current:** yes · **Rules over:** restaurant mode, tables and table state
> **Updated:** 2026-11-07 (US-081: `SQLITE_BUSY` on orders, tables and table checkout → 503) · 2026-11-06 (US-087: `/pos`, table checkout and `/barra` read the business config inside its provider; bar screen rule) · 2026-11-06 (US-086: order and catalog pages read their segment from the URL path) · 2026-10-13 (US-076: `PUT /api/setup` password exception) · Verified against the code on that date (US-055: `internal/repository/dining_table.go`, `internal/handler/tables.go`, `frontend/lib/nav.ts`; US-056: `internal/repository/order.go`, `internal/handler/order.go`, `internal/handler/handler.go:213-219`).

Module for cafeterías and restaurants: tables, orders and a bar screen (SPRINT-13). Table management, the switch, per-table orders (open, items, void, cancel), table checkout (US-057) and the bar screen (US-058) all exist. This file covers the mode, tables, orders, their screens and the bar screen; the checkout and KDS endpoints are not specified here yet.

Enum values below (`libre`, `ocupada`, `por_cobrar`, route names like `/mesas`) are identifiers in the code and API and stay in Spanish; they are glossed once.

---

## Restaurant mode

- `business_config.restaurant_mode` (0/1, default 0) decides whether the business uses the module. It is enabled in Settings with the "Modo restaurante" toggle (admin only, via `PUT /api/setup`).
- Off: the tables section is not shown in Settings, nor "Mesas" (`/mesas`, tables) and "Barra" (`/barra`, bar) in the sidebar.
- On: the tables section appears in Settings and the sidebar shows "Mesas" and "Barra" to both `admin` and `cashier`. In addition the POS hides the parked carts kept in `localStorage` (mobile chip, desktop chip and "Dejar en espera": `showParked = !restaurant_mode`, `frontend/app/pos/page.tsx:453`, `:829`, `:940`, `:954`); when off, the POS does not change. The rule depends only on the mode, not on the print method.
- Pages read the mode from the business-config context. It holds a real value only inside `BusinessConfigProvider`, which `AppLayout` mounts (`frontend/components/AppLayout.tsx:47`). Outside the provider, or while `GET /setup` is loading or after it fails, the mode reads as off. So `/pos` and the table order page keep their body in an inner component rendered inside `<AppLayout>` (`POSContent`, `frontend/app/pos/page.tsx:449`; `OrderContent`, `frontend/app/mesas/[orderId]/OrderClient.tsx:33`). Until US-087 (2026-11-06) both read it outside the provider and always saw the mode off. Test: `frontend/lib/providerPlacement.test.tsx`.
- `business_config.lan_access_enabled` (0/1, default 0) is created together with this migration but belongs to US-054 (LAN access); see `servidor.md`.
- `PUT /api/setup` replaces the whole row, except `owner_email_app_password`: an omitted or empty value keeps the stored one (US-076; see `caja.md`). The Settings form must always resend `restaurant_mode` and `lan_access_enabled` (read from `GET /setup`) so that saving another field does not turn them off.
- `GET /api/setup` returns `catalog_token` only to `admin`; any other role gets an empty string (R-69, `internal/handler/setup.go`). Name, logo and `restaurant_mode` stay visible to every authenticated user. Test: `TestSetup_R69_CatalogTokenOnlyForAdmin`.
- The public catalog page `/catalogo/<token>` (link shown in Settings as `${origin}/catalogo/<catalog_token>`, `frontend/app/settings/page.tsx:728`) needs no session. It reads the token from the URL path (`frontend/app/catalogo/[token]/CatalogClient.tsx:34-35`) and calls `GET /api/public/catalog/<token>`. A wrong token → 404 and `Catálogo no encontrado o no disponible.`; a missing or malformed token (`/catalogo/_`, `/catalogo/%E0`) shows the same text with no request (US-086).

## Tables

Table `dining_tables`: unique `name`, `zone` (free text for grouping), `seats` (not yet used in screens), `sort_order`, `active`.

- Only `admin` creates, edits or deactivates tables (`POST/PUT/DELETE /api/tables`); any authenticated user reads them (`GET /api/tables`, active ones only, ordered by `sort_order` then `name`).
- Empty or whitespace-only name: 400.
- Duplicate name: 409 "Ya existe una mesa con ese nombre (puede estar desactivada)" (UI message, Spanish). Uniqueness is exact (case-sensitive: "Mesa 1" and "mesa 1" coexist) and **includes deactivated tables**: "Mesa 1" cannot be recreated if a deactivated one exists. A deactivated table does not appear in the UI and today cannot be reactivated.
- Delete = logical deactivation (`active = 0`), never `DELETE`, because historical orders reference it.
  - With an `status='open'` order on that table: 409 "La mesa tiene un pedido abierto" and the table stays active.
  - `closed` or `cancelled` orders do not block.
  - The check and the deactivation are atomic: a single `UPDATE ... WHERE NOT EXISTS (open order)` (`internal/repository/dining_table.go:139-144`), so an order opened in parallel cannot leave an inactive table with an open order (R-59 resolved). `SQLITE_BUSY` → **503** "El sistema está ocupado, reintenta en un momento" and the table stays active (`internal/handler/tables.go:91-92`; it was 409 before US-081). Test: `TestConcurrency_T9_DeleteTableBusyAnswers503` (`internal/handler/concurrency_test.go:492`).
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
| `POST /api/orders` `{table_id}` | 201 `{id,type,table_id,status,created_at}`. 400 if `table_id` is missing/≤0 or `type` differs from `table`; 404 table does not exist or is deactivated; 409 if the table already has an `open` order; 503 on `SQLITE_BUSY`. |
| `GET /api/orders?status=open` | `[{id,table_id,table_name,total,items_pending,opened_minutes}]`, `[]` if none. Valid `status`: `open|closed|cancelled`, anything else → 400. `items_pending` = non-void items in `pending` or `preparing`. |
| `GET /api/orders/{id}` | `{id,type,table_id,status,total,items[]}`; `items` includes the `void` ones and is never `null`. 404 if it does not exist. |
| `POST /api/orders/{id}/items` `{product_id,quantity,note?}` | 201 with the item `{id,product_id,product_name,quantity,unit_price,subtotal,note,status}`. 404/400 product nonexistent or inactive; 400 invalid quantity; 409 order not `open`. |
| `PATCH /api/orders/{id}/items/{itemId}` `{quantity?,note?,status?}` | Updates only what is sent. 400 invalid quantity or `status`; 404 item not part of the order; 409 item `void` or order not `open`. |
| `DELETE /api/orders/{id}/items/{itemId}` | Voids (`status='void'`), 200 with the item. 409 if it was already `void` (no second audit entry) or order not `open`. |
| `POST /api/orders/{id}/cancel` `{reason}` | 200; 400 empty or whitespace-only reason; 404 does not exist; 409 if not `open`. |

Rules:

- Lifecycle: `open → cancelled` (`closed` arrives with US-057). `type` is always `table` in v1.
- **Only one `open` order per table**, guaranteed by an `INSERT ... SELECT ... WHERE EXISTS (active table) AND NOT EXISTS (open order)` in a single statement (`internal/repository/order.go:131-136`), not by a unique index (there is no migration 19). Whoever loses the race gets 409 (`ErrTableHasOpenOrder`). If SQLite answers `SQLITE_BUSY` (primary code 5, including 517), they get `ErrBusy` → **503** "El sistema está ocupado, reintenta en un momento" (`internal/handler/order.go:17,25-26`; 409 before US-081), never 500.
- **Busy writes (US-081):** every orders endpoint that goes through `respondOrderError` answers 503 on `ErrBusy`, and so does table checkout (`internal/handler/order.go:233-236`). A 503 means nothing was written, so the client retries it on its own (`servidor.md`, "Concurrent devices on the LAN"). 409 stays for a table that already has an open order, an order that is not open, and an item that is already void. Tests: `TestConcurrency_T7_OrderItemBusyAnswers503` (`internal/handler/concurrency_test.go:473`), `TestOrder_T22_BusyMapsTo503` (`internal/handler/order_test.go:673`).
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
- `/mesas/[orderId]`: catalog with search (tapping adds the item to the server immediately, no "send to kitchen"), lines with quantity, note and status (pendiente/preparando/listo/entregado — pending/preparing/ready/delivered), void with confirmation (struck-through line), cancellation with reason. A failure on add shows a message and leaves no phantom line. At 320px catalog and order alternate through tabs. This page has no buttons to advance an item's status; the bar screen does that. WHEN the order is checked out, the ticket preview renders inside the app layout, with the sidebar (`OrderClient.tsx:174-181`, US-087). Its printing follows `impresion.md` → Frontend flow.
- Dynamic route statically exported with `generateStaticParams [{orderId:'_'}]`. The order id comes from the URL path, not the route param (`resolveRouteParam`, `frontend/app/mesas/[orderId]/OrderClient.tsx:40`). With no id (`/mesas/_`) the page sends no `/api/orders/...` request and shows the empty order. See `servidor.md` → Dynamic pages.
- `/barra` (bar screen, KDS for kitchen display system): full screen, with no sidebar. `BarraPage` wraps `BarraContent` in `BusinessConfigProvider` only, not `AppLayout` (`frontend/app/barra/page.tsx:36`, `:234-240`). It renders nothing until the config has loaded (`:138`). WHILE restaurant mode is on, it shows three columns, `Pendiente`, `Preparando` and `Listo` (`:223-225`), and polls `GET /api/kds/orders` every 3 s (`:108`). Tapping an item moves it `pending → preparing → ready` (`PATCH /orders/{id}/items/{itemId}`, `:111-123`). IF restaurant mode is off, or `GET /setup` fails, THEN it redirects to `/` (`:48-50`); one `GET /api/kds/orders` may still go out before the redirect. Until US-087 (2026-11-06) the page mounted no provider and stayed blank in every mode.

## Schema (migration 18)

`dining_tables`, `orders` (`type` `table|takeaway`, `status` `open|closed|cancelled`) and `order_items` (`status` `pending|preparing|ready|delivered|void`) are created at once; the `orders`/`order_items` endpoints were introduced by US-056 (see Orders). Foreign keys are enforced at runtime (roadmap R-56, closed), so deleting a referenced row fails; integrity also relies on logical deletion and service validations.
