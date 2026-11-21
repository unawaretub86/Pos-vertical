# Feature inventory — Pos-vertical

> **Type:** current · **Rules over:** what the product does today, who sees it, and what not to promise
> **Updated:** 2026-11-17 (R-236: customer CSV export row) · 2026-11-14 (US-105: cashier *abono* from `Punto de Venta › Abonar`, receivables API and Customers screens admin-only; roles note and customer route ref) · 2026-11-13 (US-104: register close row; the shortage limitation row removed) · 2026-11-13 · Verified against `dev` at `ba4ae01` (routes in `internal/handler/handler.go:124-306`, sidebar in `frontend/components/Sidebar.tsx`, rules in `spec/`).

This is the list to sell from. Each row says what exists, who can use it and where the rule lives. The rules themselves stay in `spec/`; this file only points at them. When a US adds or removes a feature, update the row in the same commit (CLAUDE.md Rule 2).

**Status legend.** **Live** = on by default, in every install. **Toggle** = built, off until the admin turns it on in `Configuración`. **Frozen** = built, but do not sell or switch on until a customer asks (roadmap § Freeze). **Tool** = a separate `.exe` the installer runs, not the shop.

**Roles.** `admin` = the owner (Don Carlos). `cashier` = the counter (Laura). The "Who" column is what each role can reach **in the UI**: the cashier's menu has only `Punto de Venta` and `Caja` (plus `Mesas` and `Barra` in restaurant mode; every other entry is `adminOnly`, `frontend/components/Sidebar.tsx`). A few read APIs are still open to both roles where the menu is not (for example `GET /purchases`, `internal/handler/handler.go:250`). Receivables are not among them: since US-105 a cashier's customer list carries no balances, the customer detail API is admin-only, and the Customers screens send her to `/pos` (`fiado.md` § What the cashier sees).

---

## 1. Selling at the counter (`/pos`, *Punto de Venta*)

| Feature | Status | Who | Rule / source |
|---|---|---|---|
| Search products by name or code; scan a barcode (the scanner's `Enter` adds the item) | Live | both | `catalogo.md`; `GET /products/barcode/{barcode}` |
| Cart with quantities, retail or wholesale price per line (when the product has a wholesale price) | Live | both | `frontend/app/pos/page.tsx:230` |
| Per-line `Descuento`, up to 100% of the line, audited (`discount`, entity `sale_item`, one row per line, US-106) | Live (no toggle) | both | `frontend/app/pos/page.tsx:410-453`; `internal/service/sale.go:183-191`; `internal/repository/sale.go:130` |
| Global discount on the sale, audited (`discount`) | Toggle (`Descuento global en ventas`) | both | Settings → Caja registradora; `internal/repository/sale.go:182` |
| Price override on a cart line, audited (`price_override`) | Live | admin | `frontend/app/pos/page.tsx:374-389`; `internal/service/sale.go:73-77`; `internal/repository/sale.go:112` |
| Selling requires an open register (`no hay caja abierta`) | Live | — | `internal/service/sale.go:46-50` |
| Payment methods: `Efectivo`, `Transferencia` (Nequi, Daviplata, bank), `Tarjeta`, `Fiado` (*fiado*, store credit), and mixed payments that split one sale across several | Live | both | `fiado.md` § Model; `SalePaymentPanel.tsx` (`PAYMENT_METHOD_LABELS`) |
| Cash received and change shown (`Recibido` / `Cambio`) | Live | both | `frontend/components/SalePaymentPanel.tsx:500-516` |
| IVA (Colombian VAT) per product at 0%, 5% or 19%, rounded per line to whole pesos | Live | — | `dinero.md` |
| Park a cart and serve the next customer (`Dejar en espera`) | Live (hidden in restaurant mode) | both | `restaurante.md` § Restaurant mode |
| The open cart survives a session expiry and is restored after login (up to 12 h) | Live | both | `servidor.md` § Session expiry |
| Ticket printing: browser dialog, or direct to a thermal printer (network ESC/POS or a Windows-installed printer), 58 or 80 mm, accents and ñ correct | Live (browser by default) | both print; admin configures | `impresion.md` |
| Void a sale (stock comes back, *fiado* is lowered; expected cash drops only while the sale's register is still open) | Live | admin only | `caja.md`; only from the dashboard's `Ventas recientes` (roadmap D2) |

## 2. Fiado — credit to customers (the differentiator)

| Feature | Status | Who | Rule / source |
|---|---|---|---|
| Customer list and detail with history, balance always computed, never stored: no screen can disagree with another | Live | admin (menu); the cashier sees one customer's debt at checkout | `fiado.md` § Customer balance |
| Create and edit customers | Live | admin | `internal/handler/handler.go:302-303` |
| At checkout on *fiado*: debt badge with days owed, last visit, "Lo de siempre" (top 3 products) and `Agregar lo de siempre` | Live | both | `fiado.md` § Customer context at checkout |
| Optional credit limit per customer; warning, plus a `¿Fiar de todas formas?` confirmation over the limit or with debt older than 30 days. Never blocks the sale | Live | both | `fiado.md` § Credit limit |
| *Abonos* (payments toward the balance) with method `Efectivo` / `Transferencia` / `Tarjeta`, quick amounts (`Total`, `Mitad`, $50.000, $20.000), preview of the new balance | Live | both: the cashier from `Punto de Venta › Abonar` while a register is open (one customer's balance only, US-105); the admin also from `Clientes`, the detail and the dashboard | `fiado.md` § Payments |
| *Deuda anterior*: the debt carried over from the paper notebook, with its date, so day one starts with real balances | Live | admin | `fiado.md` § Opening balance |
| Debt age chips on the customer list: `Hasta 14 días`, `15–30 días`, `Más de 30 días`, with totals | Live | admin | `fiado.md` § Debt age |
| `Exportar` on `Clientes`: a CSV of every active customer (name, phone, credit limit, balance, oldest-debt date) that Spanish Excel opens in columns, so the owner's data always leaves with him | Live (Excel check pending, `gaps.md` R-236 M-1) | admin | `fiado.md` § Customer export |
| `Cartera` (receivables) page: who owes, how much, ordered by amount | Live | admin | `/portfolio` |
| WhatsApp collection reminder: opens the shop's own WhatsApp with the message ready; nothing is sent automatically and there is no cost | Live | admin | `fiado.md` § Collection WhatsApp message |
| *Saldo a favor* (credit balance, after voiding a paid credit sale or correcting a *deuda anterior* down; an *abono* can never exceed the debt) shown in green | Live | admin | `fiado.md` § Customer balance |
| A customer with debt cannot be deleted; deleting is logical, history stays | Live | admin | `fiado.md` § Deleting a customer |

## 3. Register (`/register`, *Caja*)

| Feature | Status | Who | Rule / source |
|---|---|---|---|
| Open with a starting float (default amount configurable) | Live | both | `caja.md` |
| Cash in / cash out movements with a note | Live | both | `caja.md` § Movements |
| Close with count by bill and coin, expected vs counted. A *sobrante* (surplus) closes directly; a *faltante* (shortage) closes after the confirmation `¿Cerrar la caja con faltante?` and is shown as `Faltante $X` on the close ticket, `Ver histórico` and `Reportes` (US-104, R-186). The close modal breakdown includes `Entradas`; a failed close shows an error toast | Live | both | `caja.md` § Expected cash; `frontend/lib/registerClose.ts`, `frontend/app/register/page.tsx` (`CloseModal`) |
| Register history with movements, every past register (`Ver histórico`) | Live | both (`Caja`); admin also in `Reportes` | `frontend/app/register/page.tsx:541-567`, `:685`; `GET /registers` not admin-only |
| Only cash counts: transfer, card and *fiado* never distort the drawer | Live | — | `caja.md` |
| Close ticket printed | Live | both | `impresion.md` |
| Close summary e-mailed to the owner (Gmail app password; needs internet at that moment, never blocks the close) | Toggle (fill `Correo del dueño`) | admin configures | `caja.md` § Email at close |
| Several devices on the shop WiFi charge into the same register at the same time | Toggle (LAN access) | both | `servidor.md` § Concurrent devices |
| Business expenses by category (`Servicios`, `Arriendo`, `Nómina`, `Proveedor`, `Otro`), mirrored as cash out when a register is open | Live | admin (`Reportes › Registrar gasto`); the cashier uses `Salida` | `caja.md` § Expenses |

## 4. Products and stock (`/inventory`, *Inventario*)

| Feature | Status | Who | Rule / source |
|---|---|---|---|
| Products or services; units `Unidad`, `Kilogramo`, `Libra`, `Litro` (`frontend/app/inventory/page.tsx:185`); cost, sale and wholesale price; IVA **added on top of** the sale price (`frontend/lib/saleCalc.ts:61-63`); stock and minimum stock | Live | admin; the cashier only searches them in `/pos` | `catalogo.md` § Product |
| Categories | Live | admin | `catalogo.md` § Category |
| Stock entries (`Entrada de inventario`) | Live | admin | `catalogo.md` § Stock traceability |
| Low-stock flag (`stock <= min_stock`) on the inventory list, and the assistant's "qué se está acabando" | Live | admin | `frontend/app/inventory/page.tsx:558`, `:606` |
| CSV import (save the Excel file as CSV; `.xlsx` is refused) with column mapping, a dry run and all-or-nothing commit | Live | admin | `catalogo.md` § Catalog import |
| Starter catalog for neighborhood shops: 158 products, 12 categories, every price a $1.000 placeholder | Live | admin | `catalogo.md` § Starter catalog; roadmap R-182 |
| Purchases from suppliers (`Compras`) that add stock, and supplier create, edit and delete | Live | admin | `/purchases`; `internal/handler/handler.go:220-224` |

## 5. Owner's view (admin only)

| Feature | Status | Rule / source |
|---|---|---|
| Dashboard: `Vendido hoy`, `Ventas hoy`, `Efectivo en caja`, `Fiado nuevo hoy`, `Cobrado de fiado`, `Más vendido hoy`, gross and net profit, top products by margin %, overdue debtors with `Pagar`, recent sales with `Anular` and `Ver anuladas` | Live | `frontend/app/page.tsx`; `rentabilidad.md` |
| Real profit: products without a cost are left out and counted, so the margin is never a fake 100% | Live | `rentabilidad.md` § Gross profit |
| Best seller vs best margin ("the rice sells, the oil earns"); the margin list ranks by margin %, not by pesos earned | Live | `rentabilidad.md` § The contrast that sells; `frontend/app/page.tsx:485-492` |
| Reports: sales by day, week, month or custom range, with CSV export; top products; receivables; expenses; register history with differences | Live | `/reports` |
| Local assistant that answers in Spanish without internet: sales, profit, margin, best seller, best margin, low stock, who owes me, expenses, transfer income (asked as "nequi", answered as all transfers together), cash, *fiado* given | Live (sell it; do not extend, roadmap § Freeze) | `internal/service/assistant.go:327-338`, `:586` |
| Audit log: voids, global and per-line discounts, price overrides, cash out (best effort), imports, *deuda anterior* changes | Live | `/audit` |
| Users: create, edit, deactivate; roles `admin` and `cashier` | Live | `/users` |

## 6. Safety and operation

| Feature | Status | Rule / source |
|---|---|---|
| Works with no internet. One `pos.exe`, all data in one `data.db` file next to it | Live | `product.md` |
| First-run wizard: owner's name, user and password, business name and NIT | Live | `internal/handler/onboarding.go:31` |
| Change own password | Live, both roles | `PUT /users/me/password`, `internal/handler/handler.go:279` |
| The default `admin` password must be changed before use (server-enforced) | Live | `servidor.md` § Default password |
| Login lock: 10 failed attempts in 10 min lock that device for 5 min | Live | `servidor.md` § Login attempt limit |
| Automatic daily backup, 30 copies kept, verified before replacing; folder configurable (USB, `D:`), with fallback and a dashboard warning | Live | `backup.md` |
| Safety copy before an update that migrates the database (`data.db.pre-migration-*`), never deleted automatically | Live | `backup.md` § Pre-migration copy |
| Manual copy and restore: copy a `data.db.backup-*` file, never `data.db` alone (WAL mode); move `data.db-wal` and `data.db-shm` aside when restoring | Procedure | `backup.md` § Restore; `internal/database/database.go:27` |
| Starts with Windows (`pos.exe --install-autostart`) | Live (installer runs it) | `servidor.md` § Start with Windows |
| Warning in the console when the PC time zone is not Colombia's | Live | `tiempo.md` |
| Light and dark theme; works on a phone from 320 px | Live | `CLAUDE.md` |
| Phone or tablet on the shop WiFi | Toggle (`Permitir acceso desde otros dispositivos de mi red WiFi`, then restart) | `servidor.md` § LAN access |

## 7. Installer tools (not for the shop)

| Tool | What it does | Rule / source |
|---|---|---|
| `reset-password.exe` | Resets a forgotten password on the shop's PC | `cmd/reset-password` |
| `reset-data` | Clears the dry-run sales, registers, *abonos*, expenses, orders and assistant queries before go-live; keeps products, customers and *deudas anteriores*; dry run by default, `--apply` plus a typed `BORRAR N`; takes a `.pre-reset-*` copy first | `reset.md`; `cmd/reset-data/main.go:50-131` |

## 8. Built but frozen (do not sell yet)

| Feature | Why frozen | Source |
|---|---|---|
| Restaurant mode: tables, orders, bar screen (`Mesas`, `Barra`) | Client #1 is not a restaurant | `restaurante.md`; Toggle `Modo restaurante` |
| DIAN electronic invoicing (Factus provider), credit notes | Off by default; needs the provider secret, a restart and sandbox access | `facturacion-dian.md`; roadmap R-124..R-129 |
| Public catalog link (`Catálogo en línea`) | Served by the shop's PC: reachable only from phones on the shop WiFi with LAN access on, never from the internet | `restaurante.md` (catalog token); `servidor.md:9` |

---

## Do not promise (it does not exist)

| Gap | Say instead | Source |
|---|---|---|
| Cloud, remote access from home, several shops | "Works on this PC and on phones on your WiFi" | `product.md` § What it is NOT |
| Several registers open at once | "Several devices, one register" | `caja.md` § What does NOT exist yet |
| Accounting, payroll, DIAN reports | Out of scope | `product.md` |
| Automatic WhatsApp sending, card terminal or Nequi integration | Methods are recorded, not integrated | `product.md` |
| Blocking a *fiado* over the limit; interest or late fees | Warning and confirmation only | `fiado.md` § What does NOT exist yet |
| Customer risk score | Roadmap R-17 | `fiado.md` |
| Voiding any old sale from a history page | Only from the recent-sales list | roadmap D2, R-145 |
| "The cashier sees no totals or debts" | The cashier does not see the dashboard, profit, reports or receivables, but does see each customer's debt at checkout, every register's movements (`Ver histórico`), the expected cash and the close ticket by payment method, and one customer's balance when taking an *abono*. The Customers screens and the receivables stay with the owner (US-105) | `fiado.md` § What the cashier sees (R-188 closed by US-105) |
| Closing the black console window without stopping the POS | The window must stay open (minimized) | roadmap D3, R-169 |
| Real prices in the starter catalog | Every price must be set by the shop | roadmap R-182 |
