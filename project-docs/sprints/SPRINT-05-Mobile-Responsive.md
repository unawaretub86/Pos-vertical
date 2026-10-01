# SPRINT-05 — Mobile Responsive: Demo-Ready on a Phone

**Goal:** The system can be demoed and used on a phone (320px–430px). The current blocker is the fixed 220px sidebar that crushes the content on small screens. At the end of this sprint, a prospect can see the POS on their own phone.

**Start date:** —

**Mobile usage context:**
- Demo to prospects on the salesperson's or the customer's phone
- Cashier on a tablet/small phone making sales
- Admin checking the dashboard or receivables from the phone

---

## Identified problems (iPhone 14 Pro Max — 430px)

1. **Fixed 220px sidebar** — takes ~50% of the width, the content becomes unusable
2. **Truncated texts** — "TRANSACCI...", "Todos los est..." on the dashboard
3. **No hamburger menu** — there is no way to navigate on mobile
4. **AppLayout has no breakpoints** — `display: flex` always side-by-side, never stacked

(Quoted strings are UI labels, which stay in Spanish.)

---

## Priority 1 — Mobile navigation

### US-022: Collapsible sidebar with a drawer on mobile

- [x] Turn the sidebar into a sliding drawer on screens < 768px
- **Acceptance Criteria:**
  - On screens ≥ 768px: fixed 220px side sidebar (current behavior, unchanged)
  - On screens < 768px: sidebar hidden by default, opened with a hamburger button (☰)
  - The hamburger button appears in the top-left corner on mobile, visible on every page
  - When the drawer opens, a semi-transparent overlay appears behind it; tapping it closes the drawer
  - The drawer has the same content as the current sidebar (nav items, user, logout, theme toggle)
  - On navigating to a page, the drawer closes automatically
  - Smooth enter/exit animation (transform translateX), no external libraries
  - Breakpoint detection is done with `window.matchMedia` or a `useMediaQuery` hook, not pure CSS (to avoid a layout flash)
- **Files to modify:**
  - `frontend/components/Sidebar.tsx` — add open/closed state and overlay logic
  - `frontend/components/AppLayout.tsx` — pass the drawer state if needed, adjust layout on mobile
- **Do not break:** current desktop behavior (fixed 220px, no visible changes at ≥ 768px)

---

## Priority 2 — Mobile dashboard

### US-023: Readable dashboard at 320px–430px

- [x] Fix truncated texts and the dashboard layout on mobile
- **Specific problems (seen in a screenshot):**
  - "TRANSACCI..." card — title cut off, needs ellipsis truncation or abbreviation
  - "Ventas recientes — Todos los est..." — button/text cut off
  - Stat cards stacked vertically on mobile (today they take a fixed row that overflows)
  - Topbar with too much info stacked at 430px
- **Acceptance Criteria:**
  - On mobile (< 768px): the stat cards (`Ventas hoy`, `Transacciones`) stack in a column or a 2×2 grid without overflowing
  - No text is cut with `...` except customer/product names (expected behavior)
  - The "Todos los..." button in recent sales has full text or is intentionally abbreviated ("Ver todos")
  - The Topbar on mobile shows only the essentials (greeting + POS button), without overflowing
  - Recent sales shows at most 3 rows on mobile so it does not require excessive scrolling
- **Files to modify:**
  - `frontend/app/page.tsx` — layout and text adjustments in the dashboard sections
  - `frontend/components/Topbar.tsx` — simplify on mobile if applicable

---

## Priority 3 — Usable POS on mobile

### US-024: Functional point of sale on a phone

- [x] The full sale flow must be doable on mobile (320px–430px)
- **Context:** The cashier may be in a small shop using the phone as a terminal
- **Acceptance Criteria:**
  - The product search takes the full width on mobile
  - Cart items are legible (name, quantity, price) without overflowing
  - The totals summary (subtotal, discount, VAT, total) is visible without horizontal scroll
  - The "Cobrar" (charge) button is prominent and easy to tap (minimum 48px height)
  - The payment modal (method selection, amount received, change) works on mobile without the keyboard covering critical fields
  - The post-sale ticket modal can be closed easily on mobile
- **Files to modify:**
  - `frontend/app/pos/page.tsx` — mobile-specific layout adjustments

---

## Priority 4 — Secondary pages

### US-025: Customers, Receivables and Inventory readable on mobile

- [x] The pages admins use most must be navigable on mobile
- **Scope:** Only readability and basic navigation — editing on mobile is not required
- **Acceptance Criteria:**
  - `/customers`: customer list with name, phone and balance visible without horizontal scroll
  - `/portfolio`: readable receivables table; columns are reduced on mobile (show name + debt, hide date)
  - `/inventory`: product list with name, stock and price visible; actions (edit/add stock) accessible
  - No table requires horizontal scroll at 320px
  - Tables on mobile may show fewer columns than on desktop (secondary columns are hidden)
- **Files to modify:**
  - `frontend/app/customers/page.tsx`
  - `frontend/app/portfolio/page.tsx`
  - `frontend/app/inventory/page.tsx`

---

## Out of scope for this sprint

- Heavy admin pages (Users, Settings, Reports) — used on desktop
- Installation as a PWA / mobile app
- Support for screens < 320px

---

## Sprint success criterion

When done, Andrey can open the system on his phone, create a complete test sale (search product → add to cart → charge in cash → see ticket), and show that to a prospect on his phone without embarrassment.

---

## Progress log

| US | Name | Status | Date |
|----|------|--------|------|
| US-022 | Sidebar drawer on mobile | ✅ Complete | 2026-04-09 |
| US-023 | Mobile dashboard | ✅ Complete | 2026-04-09 |
| US-024 | Functional POS on mobile | ✅ Complete | 2026-04-09 |
| US-025 | Customers/Receivables/Inventory on mobile | ✅ Complete | 2026-04-10 |
