# SPRINT-19 — Fiado Collection: Cash-Correct Payments and a Collection-First Customers Tab

**Goal:** Turn the *fiado* (store credit) module from a ledger into a collection tool, starting with the one money bug it has today: cash *abonos* (partial payments toward a fiado balance) never reach the register close. Then make debt visible at the point of sale and make collecting faster from the Customers tab.

**Start date:** 2026-11-05 (planned)

**Origin:** PO request on 2026-11-05 (fiado value proposal reviewed against the code; scope cut by the PO to five items). References: `project-docs/roadmap.md`, `project-docs/spec/domain/fiado.md`, `project-docs/spec/domain/caja.md`.

**Incoming technical debt:**
- New finding (2026-11-05): a cash *abono* is not counted in the register's expected cash. -> **US-082**.
- R-61: the collection WhatsApp message goes out without the business name. -> **US-083**.
- US-043 (SPRINT-09, pending): customer history in the POS at checkout. Backend already built; frontend missing. -> **moved here**.
- R-92 (US-083 QA, confirmed 2026-11-06 on a binary built from `dev`): the customer detail page asks for `/api/customers/NaN/detail` in the compiled binary. -> **US-086**, prioritized by the PO ahead of US-043.
- R-93 (US-083 QA): `/pos` and table checkout read the business config outside its provider, so ESC/POS auto-print never fires and `/pos` ignores restaurant mode. -> **US-087**, prioritized by the PO on 2026-11-06 right after US-086, ahead of US-043.
- R-110 (US-084 QA, verified by running): voiding a sale inserts a `cash_out` for its full total, so the register's expected cash comes out too low. -> **US-088**, prioritized by the PO on 2026-11-07 ahead of US-085.

**Out of scope (PO, 2026-11-05):** customer risk scoring (R-17), "quincena" collection mode (R-03), admin PIN to override the credit limit (R-11), tone-based WhatsApp templates, printed *abono* receipt over ESC/POS. Each is a full US and none is a quick win.

---

## Tasks

| US | Name | Status | Date |
|----|------|--------|------|
| US-082 | Payment method on *abonos*; cash *abonos* count in the register close | [x] Complete in code (manual PO checks pending, see `gaps.md`) | 2026-11-05 |
| US-083 | Business name in the collection WhatsApp message (R-61) | [x] Complete (one manual PO check pending, see `gaps.md`) | 2026-11-06 |
| US-086 | Dynamic pages read their id from the URL in the compiled binary (R-92) | [x] Complete (one manual PO check pending, see `gaps.md`) | 2026-11-06 |
| US-087 | Business config read inside its provider in `/pos`, table checkout and `/barra` (R-93) | [x] Complete (one manual PO check pending, see `gaps.md`) | 2026-11-06 |
| US-043 | Customer debt and history in the POS at checkout, with credit-limit confirmation | [x] Complete (one manual PO check pending, see `gaps.md`) | 2026-11-07 |
| US-084 | Quick *abono* modal from the Customers list | [x] Complete (one manual PO check pending, see `gaps.md`) | 2026-11-07 |
| US-088 | Voiding a sale no longer subtracts cash from the register (R-110) | [x] Complete | 2026-11-07 |
| US-085 | Aging filter chips with amounts on the Customers tab | ⬜ Pending | |

**Total estimated effort: ~5 days**

---

## Recommended implementation order

| Order | US | Reason |
|-------|----|--------|
| **1** | US-082 | It is a money bug: the register close shows an unexplained surplus for every cash *abono*. US-084 builds on its `method` field. |
| **2** | US-083 | Half a day, fixes a pre-existing defect that undermines every collection message. |
| **3** | US-086 | PO priority (2026-11-06): in the shipped `.exe` the customer detail (*abonos*, `Cobrar`, history) does not load, and US-084 builds on that flow. |
| **4** | US-087 | PO priority (2026-11-06), right after US-086: with ESC/POS configured, no ticket prints by itself after a sale, and `/pos` ignores restaurant mode. |
| **5** | US-043 | Backend endpoint already exists; frontend only. Prevents extending credit to someone already overdue. |
| **6** | US-084 | Depends on US-082 (method selector) to be cash-correct. |
| **7** | US-088 | PO priority (2026-11-07), ahead of US-085: a money bug in the register close, the same family as US-082. |
| **8** | US-085 | Pure UI over data the list already returns. |

---

## US-082 — Payment method on *abonos*; cash *abonos* count in the register close

**As** Don Carlos (owner),
**I want** each *abono* to record how it was paid, and cash *abonos* to be part of the register's expected cash,
**so that** the register close matches the cash drawer and transfers (Nequi, Daviplata) are not counted as cash.

**Estimated effort:** 1.5 days. **Impact:** High (register close is wrong today).

**Technical context (verified 2026-11-05):**
- `customer_payments` has only `customer_id`, `amount REAL`, `note`, `created_at` (`internal/database/migrate.go:245-251`): no method and no `register_id`.
- The *abono* form asks for the method as free text in the note (`frontend/app/customers/[id]/CustomerDetailClient.tsx:233`, placeholder `Ej: efectivo, transferencia Nequi...`).
- Expected cash is `opening + cashSales + cashIn − cashOut` (`internal/service/register.go:111` and `:137`); no term reads `customer_payments`. A cash *abono* therefore appears as an unexplained surplus at close.
- `CreatePayment` (`internal/repository/customer.go:180`) inserts the row; `POST /customers/{id}/payments` is admin-only (`internal/handler/handler.go:290`).
- The dashboard's credit flow sums all *abonos* regardless of method (`internal/repository/metrics.go:167-171`).

**Business rule:**
- New migration (the next number after 20; never edit an applied one) adds `method TEXT NOT NULL DEFAULT 'cash' CHECK (method IN ('cash','transfer','card'))` and `register_id INTEGER NULL REFERENCES registers(id)` to `customer_payments`. Existing rows stay `cash` with `register_id NULL` (not attributed to any register).
- On creating an *abono*, the server links it to the currently open register (`GetOpen`, `internal/repository/register.go:106`), if any.
- Expected cash at close = `opening + cashSales + cashAbonos + cashIn − cashOut`, where `cashAbonos` = sum of `method = 'cash'` *abonos* with that `register_id`.
- The close summary lists cash *abonos* as their own line, so the owner can see where the extra cash came from.

**PO decisions (2026-11-05):**
- Q-1: a cashier may register an *abono* while a register is open, because the cashier is the one who takes the money. With no open register, only an admin may. `POST /customers/{id}/payments` moves out of the admin-only group (`internal/handler/handler.go:290`) and the role check goes into the service.
- Q-2: a cash *abono* with no open register is accepted unattributed (`register_id NULL`), and the UI shows a notice that it will not appear in any register close.

**Acceptance Criteria:**
- [x] The *abono* form has a method selector `Efectivo` / `Transferencia` / `Tarjeta` (default `Efectivo`); the note stays optional.
- [x] With an open register of opening $100,000 and one cash sale of $30,000, a cash *abono* of $20,000 → expected cash $150,000, and the close summary shows a line `Abonos en efectivo: $20.000`.
- [x] A transfer *abono* of $50,000 in the same register does not change expected cash.
- [x] Existing *abonos* keep their amounts; the customer balance does not change after the migration.
- [x] A cashier with an open register registers an *abono* → 201, linked to that register. A cashier with no open register → 403. An admin with no open register → 201 with `register_id` null.
- [x] A cash *abono* saved with no open register shows the notice `Este abono no quedará en ningún cierre de caja` (UI text).

**DoD:**
- Migration + repository, service and handler changes, with tests carrying concrete numbers (pattern `setupTestDB`, `internal/handler/dashboard_test.go:14`).
- `spec/domain/caja.md` (expected-cash formula) and `spec/domain/fiado.md` (*abonos* rule) updated in the same commit.

**Verified criteria (closed 2026-11-05):**

Contract discrepancies: the over-balance check is at `internal/service/customer.go:99` on `dev`, not `:94-95` (D-1); an unknown customer on `POST /customers/{id}/payments` answers 400, not 404 (D-2, R-83); every register close sends the owner email twice (D-3, R-84); the close modal omits `cash_in` (D-4, R-85). GATE 1: close email carries `Abonos en efectivo` (Q-A yes); dashboard estimated cash unchanged (Q-B, R-86); duplicate email not fixed here (Q-C, R-84). QA round 1: OK, 0 BLOCKER, 6 MINOR (R-87, R-88; internal failures on the payment endpoint now answer 500, recorded in `spec/domain/fiado.md`). Spec: `spec/domain/caja.md` (expected cash), `spec/domain/fiado.md` (*abonos*).

- AC-1: WHEN migration 21 runs on a database with existing *abonos*, the system SHALL add `customer_payments.method` (`cash`/`transfer`/`card`, default `cash`) and `register_id` (nullable FK), leaving existing rows `cash`, `NULL`, same amount and same balances. Test: `TestMigration21_T1_AbonoMethodAndRegisterBackfill` (`internal/database/migrate_test.go`).
- AC-2, AC-3: WHEN `method` is omitted or empty, the system SHALL store `cash`; IF it is not `cash`/`transfer`/`card`, THEN 400 and nothing stored. Test: `TestCustomerPayment_T6_MethodDefaultAndValidation` (`internal/handler/customer_payment_test.go`).
- AC-4, AC-10, AC-11, AC-8: WHEN an *abono* is created while a register is open, the system SHALL link it to that register; expected cash SHALL be `opening + cash_sales + cash_abonos + cash_in − cash_out` in `Summary` and `Close`, returned as `cash_abonos`; payments list and detail carry `method` and `register_id`. Opening $100,000 + cash sale $30,000 + cash *abono* $20,000 → expected $150,000. Tests: `TestCustomerPayment_T2_CashAbonoLinkedAndCountedInExpected`; `TestRegisterService_T7_SummaryExpectedIncludesCashAbonos`, `TestRegisterService_T7_CloseStoresExpectedWithCashAbonos` (`internal/service/register_test.go`); `TestRegisterRepo_T10_SumCashAbonosPerRegister` (`internal/repository/register_test.go`, includes a register with no *abonos* → 0); `TestCustomerRepo_T11_CreatePaymentStoresMethodAndRegister` (`internal/repository/customer_test.go`).
- AC-13: Transfer and card *abonos*, and *abonos* unattributed or on another register, SHALL NOT change expected cash (transfer $50,000 + card $10,000 → expected stays $150,000). Tests: `TestCustomerPayment_T3_TransferAndCardAbonosDoNotChangeExpected`, `TestRegisterRepo_T10_SumCashAbonosPerRegister`.
- AC-5, AC-6: WHEN an admin creates an *abono* with no open register, the system SHALL answer 201 with `register_id: null`; IF a cashier does, THEN 403 `Debe haber una caja abierta para registrar un abono` and nothing stored; the route is outside `RequireAdmin` and the check is in `CustomerService`. Tests: `TestCustomerPayment_T4_CashierNeedsOpenRegister`, `TestCustomerPayment_T5_AdminWithoutRegisterStoresUnattributed`; `TestCreatePayment_T9_CashierWithoutOpenRegisterRejected`, `TestCreatePayment_T9_CashierWithOpenRegisterLinksRegister` (`internal/service/customer_test.go`).
- AC-7: IF the open-register lookup fails for a reason other than "none open", THEN the system SHALL return an error (500) and store nothing. Test: `TestCreatePayment_T8_OpenRegisterLookupFailureStoresNothing`.
- AC-9: Amount ≤ 0 or above the balance SHALL still be rejected (400) for every role. Tests: `TestCreatePayment_T9_CashierOverBalanceRejected`, `TestCreatePayment_T9_CashierZeroAmountRejected`.
- AC-12: IF summing cash *abonos* fails, THEN `Close` SHALL fail without closing the register and `Summary` SHALL fail, so `GET /registers/{id}` and `GET /registers/open` answer 500. Tests: `TestRegisterService_T7_CloseFailsWhenAbonoSumFails`, `TestRegisterService_T7_SummaryFailsWhenAbonoSumFails`, `TestRegisterSummary_T7_FailingAbonoSumReturns500` (`internal/handler/customer_payment_test.go`).
- AC-14: WHEN the close modal, close ticket and close email are shown, the system SHALL always show `Abonos en efectivo: <amount>`, also at $0. Tests: `TestBuildRegisterSummaryEmail_T12_CashAbonosLine`, `TestBuildRegisterSummaryEmail_T12_CashAbonosLineShownAtZero` (`internal/mailer/mailer_test.go`); modal and ticket by code reading (`frontend/app/register/page.tsx:366`, `frontend/components/CloseTicketPreview.tsx:135`), manual PO check in `gaps.md`.
- AC-15, AC-16: The form SHALL offer `Efectivo`/`Transferencia`/`Tarjeta` (default `Efectivo`) and send `method`; WHEN a cash *abono* is saved with `register_id: null`, the UI SHALL show the toast `Este abono no quedará en ningún cierre de caja` next to the save confirmation. Tests: `frontend/lib/abono.test.ts` (`ABONO_METHOD_OPTIONS`, `showsNoRegisterNotice`).
- AC-17, AC-18: a 403 shows the server message and keeps the modal open; the form works at 320 px with CSS variables only. Manual PO checks in `gaps.md`.

---

## US-083 — Business name in the collection WhatsApp message (R-61)

**As** Don Carlos,
**I want** the collection WhatsApp message to name my business,
**so that** the customer knows who is writing and the reminder carries authority.

**Estimated effort:** 0.5 day. **Impact:** Medium.

**Technical context (verified 2026-11-05):**
- `CustomerDetailClient` calls `useBusinessConfig()` (`frontend/app/customers/[id]/CustomerDetailClient.tsx:278`) in the same component that renders `<AppLayout>` (`:304`, `:314`, `:330`); the provider lives inside `AppLayout`, so the hook gets `defaultConfig` with an empty name.
- The list's WhatsApp link (`frontend/app/customers/page.tsx:56-60`, used at `:608`) does not include the business name at all.

**Business rule:**
- Move the body of `CustomerDetailClient` into an inner component rendered inside `<AppLayout>` (same pattern as `SettingsContent`, `frontend/app/settings/page.tsx`).
- Both messages (detail and list) start with the business name when it is set; without a name, the current text is kept.

**Acceptance Criteria:**
- [x] With business name `Tienda Don Carlos`, the WhatsApp link from the customer detail and from the list includes `Tienda Don Carlos`.
- [x] With an empty business name, the message is still valid and does not show an empty gap.

**DoD:**
- Vitest test for the message builder covering both cases.
- R-61 closed in `roadmap.md`.

**Verified criteria (closed 2026-11-06):**

Contract discrepancies: line refs in this US and in R-61 were stale after US-082 (D-1); the two existing texts differed and the detail one ended with the filler `en el negocio` (D-2); the provider failure was confirmed by running a mirror of the component tree (D-3); an empty name happens on a fresh database, while `GET /setup` loads or fails, and with a whitespace-only name, which `PUT /setup` accepts (D-4, R-91). GATE 1 (PO, 2026-11-06): Q-1..Q-4 accepted as recommended. The name goes in the first sentence (`le escribimos de <negocio>`), both buttons share one builder and one text, the no-name text drops `en el negocio`, and the new `frontend/lib/whatsapp.ts` comes with no DOM test dependency. QA round 1: OK, 0 BLOCKER, 3 MINOR, all pre-existing (R-92, R-93, R-94). Gemini diff round 1: clean. Spec: `spec/domain/fiado.md` "Rule: collection WhatsApp message".

- AC-1: WHEN the message is built with business `Tienda Don Carlos`, customer `Ana` and balance 45,000, the system SHALL produce `Hola Ana, le escribimos de Tienda Don Carlos. Le recordamos que tiene un saldo pendiente de $45.000. Gracias.` Test: `T-1 names the business after the greeting` (`frontend/lib/whatsapp.test.ts`).
- AC-2: IF the business name is empty or whitespace-only, THEN the system SHALL produce `Hola Ana, le recordamos que tiene un saldo pendiente de $45.000. Gracias.`, with no `el negocio` and no double space. Tests: `T-2 empty or whitespace-only business name drops the business sentence`, `T-6 empty business name yields the no-name message in the URL`.
- AC-3: WHEN the name has surrounding spaces, the system SHALL use it trimmed. Test: `T-3 trims leading and trailing spaces of the business name`.
- AC-4: The URL SHALL be `https://wa.me/<57 + digits>?text=<encodeURIComponent(text)>` with no double `57`, and `&` in the name SHALL be encoded as `%26` (phone `300 123 4567`, balance 1,234,567 → `wa.me/573001234567`, text contains `$1.234.567`). Tests: `T-4 prefixes 57 to the phone digits and encodes the message with name and balance`, `T-5 does not double the 57 prefix and percent-encodes & in the business name`.
- AC-5: The detail `Cobrar` button and the list link SHALL use the one shared builder `buildCollectionWhatsAppUrl` (`frontend/lib/whatsapp.ts`); the local builders are removed. Covered by QA grep review (T-7): no `wa.me`, `buildWhatsAppUrl` or `whatsappUrl` left in `frontend/app/customers/`.
- AC-6, AC-7: The business name SHALL be read inside `<AppLayout>`, in `CustomerDetailContent` (`frontend/app/customers/[id]/CustomerDetailClient.tsx:295`) and `CustomerRow` (`frontend/app/customers/page.tsx:523`). QA ran this in a browser (T-M1: playwright-core + Edge over the compiled binary, admin and cashier, 1280 px and 320 px). The detail URL and the list `href` contain `Tienda Don Carlos`.
- AC-8: WHILE `GET /setup` has not answered or has failed, the system SHALL use the no-name text and keep the action enabled. T-M1 with `/api/setup` aborted showed `Cobrar` enabled and the AC-2 text.
- AC-9: The visibility rules SHALL stay unchanged (detail button disabled without a phone, list link only with debt and a phone), with lint at 0 and working at 320 px. Covered by review, `pnpm lint` (0 problems) and T-M1 at 320 px (`Cobrar` visible, no horizontal overflow). Not exercised: the disabled state without a phone, a manual PO check in `gaps.md`.

---

## US-086 — Dynamic pages read their id from the URL in the compiled binary (R-92)

**As** Don Carlos (owner) and whoever works the counter,
**I want** the customer detail, the table order and the public catalog to open in the installed `pos.exe`,
**so that** I can register *abonos* (partial payments toward a fiado balance), send `Cobrar`, work table orders and share the catalog link.

**Estimated effort:** 0.5 day. **Impact:** High (three features do not work in the shipped binary). **Origin:** R-92 (US-083 QA), PO priority on 2026-11-06 ahead of US-043.

**Technical context (verified 2026-11-06 by running a binary built from `dev` at `de27ebd` in headless Edge):**
- The static export pre-renders each dynamic route only as `_` (`frontend/app/customers/[id]/page.tsx:3-5`, `frontend/app/mesas/[orderId]/page.tsx:3-5`, `frontend/app/catalogo/[token]/page.tsx:3-5`). The server serves that placeholder for every `/<prefix>/*` (`web/static.go:19`, `:47-52`).
- `/customers/1`, opened by direct URL or from the list, requests `/api/customers/NaN/detail` → 400 and shows `Cliente no encontrado` (`frontend/app/customers/[id]/CustomerDetailClient.tsx:292`, `:296`).
- `/mesas/1` loads no order at all because of the `'_'` guards (`frontend/app/mesas/[orderId]/OrderClient.tsx:44`, `:53`). Every action goes to `/api/orders/_/...` → 400.
- `/catalogo/<token>` requests `/api/public/catalog/undefined` → 404 (`Catálogo no encontrado o no disponible.`). The page reads `params.token` synchronously (`page.tsx:7-8`), Next 16 passes `params` as a Promise, and the export bakes `"token":"$undefined"`.

**Business rule:**
- No rule changes. The three pages take their segment from the browser path through one shared pure helper in `frontend/lib/`, and use the route param only when it is a real value (`pnpm dev`). The Go file server and the URLs stay as they are.

**Acceptance Criteria:**
- [x] In the compiled binary, `/customers/1` opened by direct URL and from the Customers list loads customer 1 (`GET /api/customers/1/detail`); no request carries `NaN`.
- [x] An invalid customer id (`/customers/abc`) shows `Cliente no encontrado` and sends no detail request.
- [x] In the compiled binary, `/mesas/1` loads order 1, and adding an item posts to `/api/orders/1/items`.
- [x] In the compiled binary and with no session, `/catalogo/<token>` loads the catalog of that token.
- [x] Behavior under `pnpm dev` is unchanged (reworded by the PO on 2026-11-06, see below); no hydration error on hard load; `pnpm lint` stays at 0 problems.

**DoD:**
- Vitest tests for the helper (placeholder `_`, real param, trailing slash, missing segment, catalog token, table id, invalid customer id).
- Browser check of the compiled binary (headless) by QA.
- `spec/domain/servidor.md` (how dynamic pages get their id) updated at close.
- R-92 closed in `roadmap.md`.

**Verified criteria (closed 2026-11-06):**

Contract discrepancies:
- D-1: the order page did not request `NaN`. Its `'_'` guards made it load nothing, and every action went to `/api/orders/_/...` → 400.
- D-2: `/catalogo/<token>` was broken too. It read `params.token` synchronously, Next 16 passes `params` as a Promise, and the export baked `"token":"$undefined"`.
- D-3: the spec and the Go tests only covered serving the placeholder HTML.
- D-4: navigating into these pages in the binary is a full page load (R-98).

GATE 1 (PO, 2026-11-06): Q-1..Q-4 accepted as recommended. Mesas and catalog are in scope, the fix is a client-side helper with URLs unchanged, D-4 goes to the roadmap, and verification is QA T-M1 plus a PO manual check in `gaps.md`. A contract update approved by the PO (Gemini G-2, G-3, G-5) added malformed percent-encoding to AC-2 and added AC-10 and AC-11.

QA round 1: OK, 0 BLOCKER, 5 MINOR:
- M-1: AC-7 was reworded by the PO. Under `next dev` the three pages already answered 500 before this change, because of `output: "export"` (`frontend/next.config.ts:4`) with placeholder-only `generateStaticParams` (R-95).
- M-2, M-3: duplicated `subscribePath` hook, which listens to `popstate` only (R-96).
- M-4: order id not validated (R-97).
- M-5: R-92 row contradicted itself; fixed at close.

Gemini diff round 1: clean. No Go change (`git diff dev...HEAD -- web/ internal/` is empty). Spec: `spec/domain/servidor.md` → Dynamic pages, `spec/domain/restaurante.md` (order and public catalog), `spec/domain/fiado.md` (customer detail id).

- AC-1: WHEN a dynamic page needs its segment, the system SHALL use `resolveRouteParam(param, pathname, prefix)` (`frontend/lib/routeParam.ts`). It SHALL return the route param when it is a non-empty string other than `'_'`. Otherwise it SHALL return the URI-decoded first segment after `/<prefix>/`, ignoring a trailing `/`. Tests (`frontend/lib/routeParam.test.ts`): `T-1 placeholder param reads the customer id from the path` (`'_'`, `/customers/1` → `'1'`), `T-2 placeholder param ignores a trailing slash` (`/customers/42/` → `'42'`), `T-3 a real param wins on the dev server`, `T-4 undefined param reads the catalog token from the path, URI-decoded` (`/catalogo/a%20b` → `'a b'`), `T-5 placeholder param reads the order id from the mesas path` (`/mesas/12` → `'12'`).
- AC-2: IF the segment is missing, empty, `'_'` or badly percent-encoded, THEN the helper SHALL return `null` and never throw. Tests: `T-6 missing, empty or placeholder segment, or a different prefix, gives null`, `T-8 malformed percent-encoding gives null without throwing` (`/catalogo/%E0` → `null`).
- AC-3: WHEN `/customers/1` is opened by direct URL or from the Customers list in the binary, the system SHALL request `GET /api/customers/1/detail` and render the customer, never `/NaN/`. Test: QA T-M1 (a), headless Edge over the compiled binary.
- AC-4: IF the customer segment is not a positive integer, THEN no detail request SHALL be sent and the page SHALL show `Cliente no encontrado`. Tests: `T-7 accepts only a positive integer segment` (`'1'` → 1; `'abc'`, `'0'`, `'-3'`, `'1.5'`, `null` → `null`); T-M1 (b) for `/customers/abc` and `/customers/_`.
- AC-5: WHEN `/mesas/1` is opened by direct URL or from the `/mesas` grid in the binary, the system SHALL request `GET /api/orders/1`, and add, update, void, cancel and checkout SHALL target `/api/orders/<id>/...`. Test: T-M1 (c), covering `POST /api/orders/1/items` → 201, `PATCH` and `DELETE` → 200, cancel on order 2 → 200, and checkout → 201 with ticket `TOTAL $2.500`.
- AC-6: WHEN `/catalogo/tok086abc` is opened with no session in the binary, the system SHALL request `GET /api/public/catalog/tok086abc` and render the catalog, never `/undefined` or `/_`. Test: T-M1 (d).
- AC-7 (reworded by the PO, 2026-11-06): behavior under `pnpm dev` SHALL be unchanged. Originally "SHALL keep working"; QA found that the three pages already answered 500 under `next dev` before this change (R-95). Covered by QA review: `next.config.ts` and the three `page.tsx` `generateStaticParams` are untouched, and the real-param branch is covered by T-3.
- AC-8: The first client render SHALL match the exported HTML: the path is read with `useSyncExternalStore` and a `null` server snapshot. Test: T-M1 (e), with no hydration error or warning in the console.
- AC-9: `web/static.go` and its tests SHALL be unchanged, and lint, tests and builds SHALL pass. Test: T-R1 (`git diff dev...HEAD -- web/` empty, `pnpm lint` 0 problems, `pnpm test` 110 passed, `go test ./...` green).
- AC-10: IF the helper returns `null` on `/mesas/<x>`, THEN no `/api/orders/...` request SHALL be sent and the empty order SHALL show. Test: T-M1 (b2) `/mesas/_`.
- AC-11: IF the helper returns `null` on `/catalogo/<x>`, THEN no `/api/public/catalog/...` request SHALL be sent and `Catálogo no encontrado o no disponible.` SHALL show. Test: T-M1 (b3), covering `/catalogo/_` and `/catalogo/%E0` with no exception.
- Manual PO check: `gaps.md`, US-086 M-1.

---

## US-087 — Business config read inside its provider in `/pos`, table checkout and `/barra` (R-93)

**As** whoever works the counter or the tables and Don Carlos (owner),
**I want** the ticket to print by itself after a sale when direct printing is configured, and restaurant mode to apply in the POS and on the bar screen,
**so that** I do not print each ticket by hand, parked carts do not mix with table orders, and the bar sees its orders.

**Estimated effort:** 0.5 day. **Impact:** High (US-080 auto-print never fires in the shipped binary, and the bar screen is blank). **Origin:** R-93 (US-083 QA), PO priority on 2026-11-06 right after US-086, ahead of US-043.

**Technical context (verified 2026-11-06 at `555ed60` by running a binary built from `dev` in headless Edge, with restaurant mode on and `print_method = escpos_network`):**
- `useBusinessConfig()` returns `defaultConfig` (`print_method: 'browser'`, `restaurant_mode: false`, `loading: true`) when it is called outside `BusinessConfigProvider` (`frontend/lib/business-config-context.tsx:19`, `:21-25`). Only `AppLayout` mounts the provider (`frontend/components/AppLayout.tsx:47`).
- `POSPage` calls the hook at `frontend/app/pos/page.tsx:451` and returns `<AppLayout>` itself (`:788`). A `/pos` sale sends no `POST /api/printer/print-ticket/{id}`, and `Dejar en espera` shows in restaurant mode (`showParked`, `:452`).
- `OrderClient` calls the hook at `frontend/app/mesas/[orderId]/OrderClient.tsx:35` and returns `<AppLayout>` at `:293`. A table checkout sends no print-ticket request either. Its ticket view (`:172-179`) renders outside `AppLayout`, so even `Imprimir` opens the browser dialog instead of printing over ESC/POS.
- `/barra` mounts no provider at all. `loading` stays `true` and the page returns `null` (`frontend/app/barra/page.tsx:35`, `:135`), so it has been blank in every mode since US-058.

**Business rule:**
- No rule changes. The spec already says that `/pos` hides parked carts in restaurant mode (`spec/domain/restaurante.md`) and that ESC/POS auto-prints after a sale (`spec/domain/impresion.md`); after the fix the code does what the spec says. The fix is the inner-component pattern already used by `SettingsContent` and `CustomerDetailContent`.

**Acceptance Criteria:**
- [x] With ESC/POS configured, a `/pos` sale and a table checkout each send the ticket to the printer once, by themselves, and the preview stays on screen.
- [x] On the table-checkout ticket, `Imprimir` prints a second copy over ESC/POS, not through the browser dialog.
- [x] With restaurant mode on, `/pos` shows no parked carts and no `Dejar en espera`.
- [x] With restaurant mode on, `/barra` shows the bar screen with its orders; with it off, `/barra` redirects to `/`.
- [x] In `Navegador` (browser) mode no ticket prints by itself and `Imprimir` opens the browser dialog, whatever the restaurant mode; with restaurant mode off, `/pos` shows parked carts as before, whatever the print method (split at GATE 1, see below). `pnpm lint` stays at 0 problems, and there is no Go change.

**DoD:**
- Tests that fail today because a page reads the config outside its provider (see the contract).
- Browser check of the compiled binary by QA.
- `spec/domain/restaurante.md` and `spec/domain/impresion.md` refreshed at close.
- R-93 closed in `roadmap.md`.

**Verified criteria (closed 2026-11-06):**

Contract discrepancies:
- D-1: `/barra` mounted no provider, so `loading` stayed `true` and the page was blank in every mode, as it had been since US-058 (`52b7f2b`). R-93 did not mention it.
- D-2: the table-checkout ticket view rendered outside `AppLayout`, so its `Imprimir` called `window.print()` even in ESC/POS mode. In `/pos`, `Imprimir` already worked, because that modal sat inside the layout.
- D-3: `spec/domain/restaurante.md` and `spec/domain/impresion.md` described behavior that did not happen, with stale line refs (also in R-80).
- D-4: `spec/domain/restaurante.md` had no rule for `/barra`.

GATE 1 (PO, 2026-11-06): Q-1..Q-4 accepted as recommended. `/barra` is in scope and gets the provider only, with no `AppLayout`. The table-checkout ticket renders inside `AppLayout`. A new `frontend/vitest.config.mts` holds only the `@` alias. R-80 stays out of scope. Gemini contract fixes approved by the PO: AC-8 split into AC-8a (by print method) and AC-8b (by restaurant mode), the `Imprimir` toasts stated in AC-5, test coverage made explicit, and AC-10 narrowed. A later PO decision dropped the `CustomersPage` case from AC-10 and T-4, because it could not fail (no `CustomerRow` renders under SSR).

QA round 1: OK, 0 BLOCKER, 0 MINOR, 3 INFO:
- the 320 px toast covers the ticket buttons (R-101);
- `/barra` sends one KDS poll before its redirect (pre-existing, harmless);
- prefetch 404s (R-98).

RED `3c9232b`: T-1..T-3 fail and T-4 passes. HEAD `f2d5358`: `pnpm test` 115 passed, `pnpm lint` 0 problems, no Go diff. Gemini diff round 1: clean. Spec: `spec/domain/restaurante.md` (Restaurant mode, Screens, bar screen) and `spec/domain/impresion.md` (Frontend flow).

Tests T-1..T-4 are in `frontend/lib/providerPlacement.test.tsx`; T-M1 is the QA check in headless Edge over the compiled binary.
- AC-1: WHEN `/pos`, `/mesas/<orderId>` or `/barra` renders, every `useBusinessConfig()` call SHALL run inside a `BusinessConfigProvider`. Tests: `T-1 POSPage calls useBusinessConfig only inside the provider`, `T-2 OrderClient calls useBusinessConfig only inside the provider`, `T-3 BarraPage calls useBusinessConfig only inside the provider`. These check where the hook is called, not behavior.
- AC-2: WHILE restaurant mode is on, `/pos` SHALL NOT show `Dejar en espera` or the parked-carts chips, even with items in the cart. Test: T-M1 (a), at 1280 px and 320 px, by direct URL and by in-app navigation.
- AC-3: WHEN a `/pos` sale completes with an ESC/POS method, the frontend SHALL send `POST /api/printer/print-ticket/{saleId}` exactly once, after `GET /api/sales/{saleId}/ticket`, keep the preview, and show `Ticket enviado a la impresora` or an error toast with the server message. Test: T-M1 (b) (closed port → 502 `No se pudo imprimir: ...`), plus a success run against a fake printer (200, toast shown).
- AC-4: WHEN a table checkout completes with an ESC/POS method, the same SHALL happen. Test: T-M1 (c), checkout $2.500 → print-ticket once, plus the success run.
- AC-5: WHEN `Imprimir` is pressed on the table-checkout ticket with an ESC/POS method, the frontend SHALL send print-ticket (second copy), SHALL NOT call `window.print()`, SHALL show the same toasts as AC-3, and the view SHALL render inside the app layout. Test: T-M1 (c) (`window.print` 0 → 0, sidebar visible).
- AC-6: WHEN `/barra` opens with restaurant mode on, it SHALL render `Pantalla de Barra (KDS)` full screen with no sidebar, and poll `GET /api/kds/orders` every 3 s. Test: T-M1 (d), polls at 3.57 s, 6.57 s and 9.57 s, `1 x Gaseosa` under `Mesa 1`.
- AC-7: IF `/barra` opens with restaurant mode off, THEN it SHALL redirect to `/`. Test: T-M1 (e2).
- AC-8a: WHILE `print_method` is `browser`, whatever the mode, no print-ticket request SHALL be sent after a sale or checkout, and `Imprimir` SHALL call `window.print()`. Test: T-M1 (e), restaurant on + browser.
- AC-8b: WHILE restaurant mode is off, whatever the method, `/pos` SHALL show the parked carts. Test: T-M1 (e2), restaurant off + ESC/POS.
- AC-9: IF `GET /api/setup` fails, THEN `/pos` and table checkout SHALL behave as browser with restaurant mode off, and `/barra` SHALL redirect to `/`. Test: T-M1 (f), with `/api/setup` blocked.
- AC-10: `SettingsPage` and the customer detail page SHALL keep reading inside the provider. Tests: `T-4 CustomerDetailClient default export`, `T-4 SettingsPage`.
- AC-11: No Go change; lint, tests and builds SHALL pass. Test: T-R1 (`git diff dev...HEAD -- internal/ cmd/ web/ go.mod go.sum` empty, `pnpm lint` exit 0, `pnpm test` 115 passed, `pnpm build`, `go build ./...`, `go test ./...` green).
- Manual PO check: `gaps.md`, US-087 M-1 (it also unblocks US-080 P-4).

---

## US-043 — Customer debt and history in the POS at checkout, with credit-limit confirmation

Moved from SPRINT-09 on 2026-11-05. The original definition and criteria stay in `SPRINT-09-Historial-Pedidos-Catalogo.md` → US-043; this section records what changed.

**Estimated effort:** 1 day (down from 2: the backend is built). **Impact:** High.

**Technical context (verified 2026-11-05):**
- `GET /customers/{id}/pos-summary` exists (`internal/handler/handler.go:281`, `internal/handler/customer.go:238`) and returns `last_visit_at`, `top_products`, `balance`, `days_overdue`, `credit_limit` (`internal/model/model.go:120-126`). No frontend file calls it.
- The payment panel fetches only `/customers/{id}/balance` (`frontend/components/SalePaymentPanel.tsx:176`) and shows a red notice only when the sale exceeds the limit (`:353-367`, `exceedsCreditLimit`). It does not block or ask for confirmation.
- The credit limit is informational by spec (`spec/domain/fiado.md`, "Rule: credit limit").

**Change from SPRINT-09 (PO, 2026-11-05):** when a sale with a credit line would exceed the customer's limit, or the customer has debt older than 30 days, `Cobrar` asks for a confirmation (`¿Fiar de todas formas?`) before completing. The backend does not reject the sale: the shopkeeper decides, not the system. A hard block waits for an admin PIN (R-11).

**Acceptance Criteria (in addition to SPRINT-09's):**
- [x] The panel uses `pos-summary`; the separate `/balance` call is removed.
- [x] Customer with balance $450,000 and `days_overdue` 26 → badge `Debe: $450.000 (26 días)` shown as soon as the customer is selected.
- [x] Limit $500,000, balance $450,000, credit sale of $60,000 → confirmation shown; cancel leaves the cart intact; confirm completes the sale.
- [x] Credit sale of $40,000 to the same customer with `days_overdue` 10 → no confirmation.

**DoD:**
- Vitest tests for the confirmation rule (limit exceeded, >30 days, neither).
- `spec/domain/fiado.md` "Rule: credit limit" updated (warns and asks for confirmation; still does not block).
- SPRINT-09 row for US-043 points here.

**Verified criteria (closed 2026-11-07):**

Contract discrepancies:
- D-1: stale line refs in this section (handler at `internal/handler/customer.go:247`, model at `internal/model/model.go:122-128`).
- D-2: the endpoint had no tests (SPRINT-09's DoD tests were never written), and it swallowed the last-visit query error (`internal/repository/customer.go:263`). Fixed here.
- D-3: `last_visit_at` is a zone-less UTC string; a naive browser parse is 5 h off in Bogotá. The frontend parses it as UTC.
- D-4: `days_overdue` counts from the oldest credit sale ever, even if paid off (run: 90 days for a 5-day-old debt). Kept by the PO; FIFO age is R-102.
- D-5: the old limit notice added `cartTotal`, the subtotal before IVA and the global discount. It now uses the sum of the credit lines.
- D-6: a customer can only be picked inside the credit block, not "in the cart" as SPRINT-09 says. Kept.
- D-7: the contract said the spec's `internal/service/customer.go:54,68` refs were off by one; at close they were re-verified as correct (`Create` at `:54`, `Update` at `:68`), so they were not changed.
- D-8 (Gemini, PO-approved): `top_products` returned inactive products. Fixed (AC-21).

GATE 1 (PO, 2026-11-07): Q-1..Q-8 accepted as recommended. The panel and confirmation stay in the credit block; `> 30` days uses the existing debt-age rule; the limit is checked against the sum of the credit lines; `hace X días` counts Bogotá calendar days; `Cobrar` waits for the summary and is fail-open on error; the last-visit error is fixed; `Agregar lo de siempre` is `/pos` only; the dialog labels are `¿Fiar de todas formas?` / `Fiar` / `Cancelar`. Approved the same day: the active-product filter (D-8), the both-reasons message, and the in-flight guard on `Agregar lo de siempre`.

QA round 1: OK, 0 BLOCKER, 4 MINOR:
- M-1: re-selecting the same customer shows the old summary and enables `Cobrar` until the new one arrives (R-104).
- M-2: `addUsualProducts` writes the cart as a value from `cartRef`, not a functional update (R-105).
- M-3: the `cartTotal` prop was removed from `OrderClient` (`frontend/app/mesas/[orderId]/OrderClient.tsx`), which the contract did not predict; it is the minimum change after AC-9.
- M-4: four UI strings the contract did not name (`Cargando historial...`, `Historial del cliente`, `Ver más ▾` / `Ocultar ▴`, `Agregando...`); recorded in the spec at close. The browser part of T-M1 was not run by QA (no browser automation); it is a PO check in `gaps.md`.

RED `26d1dd7`, implementation `992d1ad`. QA ran `go build ./...`, `go test ./...` green, `pnpm build`, `pnpm test` 144 passed, `pnpm lint` exit 0; `posSummary.test.ts` also passes with `TZ=UTC`. Gemini diff round 1: clean. Spec: `spec/domain/fiado.md` → "Rule: credit limit", new "Rule: customer context at checkout (POS summary)", "What does NOT exist yet".

Frontend tests are in `frontend/lib/creditCheck.test.ts` (T-1..T-8) and `frontend/lib/posSummary.test.ts` (T-9..T-13); Go tests in `internal/handler/customer_pos_summary_test.go` (T-14, T-15) and `internal/repository/customer_test.go` (T-16..T-18). T-M1 is manual.
- AC-1: WHEN a customer is selected in the payment panel, the system SHALL request `GET /customers/{id}/pos-summary` once for that selection and SHALL NOT request `/customers/{id}/balance`; typing in the picker search sends no request. Test: T-M1 (Network tab, `gaps.md` US-043 M-1); QA code review (`grep "/balance"` → 0 hits in `SalePaymentPanel.tsx`).
- AC-2: IF the selected customer changes or is cleared before a pending summary arrives, THEN the stale response SHALL be discarded; WHEN the customer is cleared, the summary panel SHALL disappear. Test: QA code review (`stale` flag in the effect cleanup); T-M1.
- AC-3: WHILE the summary is loading and a payment line is credit, `Cobrar` SHALL be disabled. Test: QA code review; T-M1. Gap on re-selection: R-104.
- AC-4: WHEN `last_visit_at` is not null, the panel SHALL show `Última visita: hoy` / `hace 1 día` / `hace N días`, counting Bogotá calendar days and reading `last_visit_at` as UTC. Tests: `T-10: last_visit_at is read as UTC and counted in Bogotá calendar days`, `T-10: Bogotá midnight edge — 05:00:00 UTC is today, 04:59:59 UTC is yesterday`, `T-10: uses Bogotá days, not UTC days (now 07/11 22:00 Bogotá = 08/11 03:00 UTC)`, `T-10: null -> null`.
- AC-5: IF `last_visit_at` is null and `top_products` is empty, THEN the panel SHALL show `Cliente nuevo — sin historial` and no `Lo de siempre`. Test: `T-11: no history -> "Cliente nuevo — sin historial" and no "Lo de siempre"`.
- AC-6: WHEN `top_products` is not empty, the panel SHALL show `Lo de siempre: ` and the names in server order joined with ` · `. Test: `T-11: two products in server order joined with " · "`.
- AC-7: WHEN `balance > 0`, the panel SHALL show `Debe: $450.000 (26 días)` (`(1 día)` for 1, no parenthesis when null), colored `var(--text-muted)` `< 7`, `var(--warning)` `7–14`, `var(--danger)` `> 14`; WHEN `balance ≤ 0`, no badge. Tests: `T-9: badge text`, `T-9: no badge when balance <= 0`, `T-9: color band 6 muted, 7 and 14 warning, 15 danger`.
- AC-8: IF `credit_limit > 0` and `balance > credit_limit`, THEN the panel SHALL show `Superó tope de crédito ($500.000)`. Tests: `T-12: limit 150,000, balance 180,000 -> exceeded text`, `T-12: balance equal to the limit -> none`, `T-12: no limit (null) -> none`.
- AC-9: The pre-sale limit notice SHALL use the summary's `balance` and `credit_limit` and the sum of the credit payment lines, formatted with `formatCOP`. Tests: `T-7: /pos sale 50,000 + 19% IVA = 59,500 all on credit -> credit amount 59,500 -> limit`, `T-7: mixed 60,000 = 40,000 cash + 20,000 credit -> credit amount 20,000 -> none`.
- AC-10: The panel SHALL collapse and expand with one click, never hiding the confirmation; a new selection starts expanded. Test: QA code review (`collapsedFor` reset on customer change); T-M1.
- AC-11: The rule SHALL live in a pure helper (`needsCreditConfirmation`, `frontend/lib/creditCheck.ts`): `limit` WHEN `credit_limit > 0` and `balance + creditAmount > credit_limit` (equal is not exceeding); `overdue` WHEN `balance > 0` and `days_overdue > 30`; nothing WHEN `creditAmount ≤ 0`. Tests: `T-1: limit 500,000, balance 450,000, credit 60,000, 26 days -> limit`, `T-2: limit 500,000, balance 450,000, credit 40,000, 10 days -> none`, `T-3: balance + credit equal to the limit (500,000) is not exceeding -> none`, `T-4: no limit (null and 0), balance 80,000, 31 days -> overdue; 30 days -> none`, `T-5: limit 500,000, balance 480,000, 45 days, credit 30,000 -> limit and overdue, in that order`, `T-6: credit amount 0 (cash sale), balance 480,000, 45 days -> none`.
- AC-12: WHEN `Cobrar` is pressed, the validations pass and a reason applies, the system SHALL open `ConfirmDialog` titled `¿Fiar de todas formas?`, labels `Fiar` / `Cancelar`, with one sentence per reason, limit first, separated by one space. Tests: `T-8: limit reason names the resulting debt and the limit`, `T-8: overdue reason names the debt age`, `T-8: both reasons -> limit sentence first, one space, overdue sentence`; T-M1 for the dialog.
- AC-13: IF the user cancels, THEN nothing is sent and cart, payment lines, cash received and customer stay; WHEN the user confirms, `onCheckout` SHALL receive the same data as today. Test: QA code review; T-M1.
- AC-14: WHEN no reason applies, `Cobrar` SHALL complete with no dialog. Test: T-2 (helper); QA code review; T-M1 ($40.000 cart).
- AC-15: IF the summary request fails, THEN the panel SHALL show `No se pudo cargar el historial del cliente`, no badge, and `Cobrar` SHALL work without confirmation. Test: QA code review.
- AC-16: The backend SHALL keep accepting credit sales over the limit; no change under `internal/{service,repository,handler}/sale.go`. Test: QA (`git diff dev...HEAD --stat`).
- AC-17: WHEN `top_products` is not empty in `/pos`, `Agregar lo de siempre` SHALL show; never in table checkout or without history. Test: QA code review (`OrderClient` passes no `onAddUsual`); T-M1.
- AC-18: WHEN `Agregar lo de siempre` is pressed, each top product SHALL be fetched (`GET /products/{id}`) and one unit added; no stock → `Arroz no tiene stock — no se agregó`; inactive or failed → `Arroz no está disponible — no se agregó`; WHILE in flight the button SHALL be disabled and a second press SHALL add nothing. Tests: `T-13: product with stock 0 -> skipped with stock toast`, `T-13: stock 5 with 4 already in cart -> added`, `T-13: stock 5 with 5 already in cart -> skipped with stock toast`, `T-13: inactive product -> skipped with unavailable toast`, `T-13: failed product request (null) -> skipped with unavailable toast`, `T-13: service-type product with stock 0 -> added`; in-flight guard: T-M1.
- AC-19: The panel, badge and confirmation SHALL work in `/pos` and table checkout, at 320 px, with CSS variables only and `pnpm lint` at 0. Test: `pnpm lint` exit 0, QA review (no new hex); 320 px and dark theme: T-M1.
- AC-20: `GET /customers/{id}/pos-summary` SHALL answer 200 to a cashier with the documented shape; 404 for an unknown id; 400 for a non-numeric id; 200 with `last_visit_at: null`, `top_products: []`, `balance: 0`, `days_overdue: null` with no sales. Tests: `TestPOSSummary_T14_CashierGetsSummaryAnd404And400` (subtests `known customer as cashier`, `unknown id is 404`, `non-numeric id is 400`), `TestPOSSummary_T15_CustomerWithNoSales`.
- AC-21: `top_products` SHALL return at most 3 active products by distinct completed sales, descending, ties by name ascending, ignoring voided sales. Test: `TestPOSSummaryRepo_T16_TopProducts` (subtests `top 3 by distinct completed sales, voided sale ignored`, `tie broken by name ascending`, `inactive product never returned`). AC-20 also covers the last visit: `TestPOSSummaryRepo_T17_LastVisitIgnoresVoidedSale`.
- AC-22: IF the last-visit query fails, THEN the endpoint SHALL return an error (500) instead of `last_visit_at: null`. Test: `TestPOSSummaryRepo_T18_LastVisitQueryErrorIsReturned` (subtest `only the last-visit query fails`).
- Manual PO check: `gaps.md`, US-043 M-1.

---

## US-084 — Quick *abono* modal from the Customers list

**As** whoever takes the money at the counter,
**I want** to register an *abono* from the Customers list in one step,
**so that** "le abono veinte mil" takes seconds and the customer sees their new balance.

**Estimated effort:** 1 day. **Impact:** Medium-high. **Depends on:** US-082.

**Technical context (verified 2026-11-05):**
- `Abonar` in the list navigates to the detail page (`frontend/app/customers/page.tsx:485`).
- `CreatePayment` rejects an amount above the balance (`internal/service/customer.go:94-95`), so an *abono* cannot create a credit balance.

**Business rule:**
- `Abonar` opens a modal over the list: quick-amount buttons `Total`, `Mitad`, `$50.000`, `$20.000` (only those ≤ balance), a free numeric field, the method selector from US-082, and a live line `Saldo anterior: $450.000 → Nuevo saldo: $400.000`.
- After confirming, the row's balance refreshes without a page reload.
- A customer with a negative balance (overpaid) shows `Saldo a favor: $X` in the success color instead of a debt.

**Acceptance Criteria:**
- [x] Balance $450,000: `Mitad` fills $225,000; typing $50,000 shows `Nuevo saldo: $400.000`.
- [x] Balance $30,000: the `$50.000` button is not shown; `Total` fills $30,000.
- [x] An amount above the balance cannot be confirmed (button disabled, message shown).
- [x] Works at 320 px.

**DoD:**
- Vitest tests for the quick-amount and new-balance helpers.
- Uses `formatCOP`, `api`, CSS variables only.

**Verified criteria (closed 2026-11-07):**

Contract discrepancies:
- D-1: the sprint's `frontend/app/customers/page.tsx:485` was stale; `Abonar` and the whole row shared one handler that navigated. The row click now opens the detail and `Abonar` opens the modal.
- D-2: the over-balance check is at `internal/service/customer.go:148-149`, not `:94-95`.
- D-3: the old detail modal never showed its over-balance message (button disabled, message only set on submit). The shared modal shows it.
- D-4: negative balances are reachable (run: $50.000 on credit, *abono* $50.000, void → −$50.000); they read `Al día` / `$-50.000` before. Now `Saldo a favor`.
- D-5: balances can carry cents from IVA (run: $10.001 + 19% → $11.901,19, a $0,19 residue after paying $11.901). Root cause is R-106.
- D-6: the dashboard modal sent no method, so every *abono* there was stored as cash. Fixed here (PO Q-2).
- D-7: the server's over-balance message is not in peso format (R-107).
- D-8: the `Clientes` menu is admin-only while `GET /customers` returns `total_portfolio` to a cashier (R-108).

GATE 1 (PO, 2026-11-07): Q-1..Q-8 accepted as recommended. One shared `AbonoModal` (`frontend/components/AbonoModal.tsx`) for the Customers list, the detail and the dashboard (whose title is now `Registrar abono`); `Saldo a favor` on the list and the detail card; `Total` sends the exact balance; `Mitad` = `floor(balance/2)`; the menu stays admin-only. Q-7 and Q-8 came from the Gemini contract review (11 raised, 2 verified, 9 discarded): `Total` shows below $1, and the digits-only field gets the amount rounded to whole pesos, at least 1, mapped back to the exact balance on send.

QA round 1: OK, 0 BLOCKER, 3 MINOR. M-1: the modal can be closed while the request is in flight (R-109). M-2, M-3: a sub-$1 residue reads `$0` / `Registrar $0` in the modal and `$0` in red with `Abonar` in the list (folded into R-106). Out-of-scope observation, pre-existing and verified by running: voiding a sale inserts a `cash_out` for its full total, so the register's expected cash comes out too low (R-110, high priority). RED `c53c566`, implementation `9a28d52`. QA ran `go build ./...`, `go test ./...` green, `pnpm build`, `pnpm test` 170 passed, `pnpm lint` exit 0, and T-M1 in headless Edge over the compiled binary (all steps passed). Gemini diff round 1: clean. Spec: `spec/domain/fiado.md` → "Rule: customer balance" (display, cents), "Rule: payments (*abonos*)" (*abono* modal); `spec/domain/glosario.md` (`Saldo a favor`); `spec/domain/caja.md` (known defect R-110).

Unit tests are in `frontend/lib/abono.test.ts` (T-1..T-8, T-10; the US-082 tests, T-9, are unchanged). T-M1 was run by QA; the real-device part is `gaps.md` US-084 M-1.
- AC-1, AC-2, AC-3: WHEN `Abonar` is pressed on a row with balance `> 0`, the system SHALL open the modal over the list with no navigation (`Registrar abono`, name, `Saldo pendiente`); a click on the row itself SHALL still open `/customers/{id}`; `Cancelar`, `×`, overlay or `Escape` SHALL close it and send nothing. Test: T-M1 step (1) and the AC-3 close paths.
- AC-4: The modal SHALL show `Total` (exact balance, whenever balance `> 0`), `Mitad` (`floor(balance/2)`), `$50.000`, `$20.000`, the last three only when `1 ≤ amount ≤ balance`. Tests: `T-1`..`T-4` (450,000 → four buttons, `Mitad` 225,000; 30,000 → no `$50.000`; 15,000; 50,000; 45,001 → `Mitad` 22,500; 11,901.19 → `Total` 11901.19, `Mitad` 5,950; 1 → `Total`; 0.19 → `[{Total,0.19}]`; 0 and −50,000 → none).
- AC-5, AC-6: A quick amount SHALL fill the digits-only field with `abonoFieldValue` (whole pesos, at least 1) and SHALL NOT submit; typing SHALL keep digits only. Tests: `T-10` (225,000; 11,901.19 → 11,901; 11,901.6 → 11,902; 0.19 → 1; 0.5 → 1; 0 → 0); T-M1 step (2) (`Mitad` → `225.000`, `12a3` → `123`).
- AC-7, AC-9: WHILE `0 < amount ≤ balance`, the system SHALL show `Saldo anterior: $450.000 → Nuevo saldo: $400.000`; IF the amount is above the balance, THEN confirm SHALL be disabled with `El monto no puede superar $30.000`; empty → disabled, no message. Tests: `T-5`, `T-6`.
- AC-8: WHEN the field equals the balance's field value, the system SHALL send the exact balance (11,901.19 / 11,901.6 / 0.19), so the balance reaches 0. Tests: `T-7`; T-M1 residue step (POST `0.19000000000050932`, row → `Al día`).
- AC-10, AC-11, AC-12: The modal SHALL send `{amount, method, note}` through `api` with `Efectivo` / `Transferencia` / `Tarjeta` (default `Efectivo`); while in flight confirm SHALL read `Guardando...` and a second press SHALL send nothing; on 201 the success toast and, for cash with `register_id: null`, `Este abono no quedará en ningún cierre de caja`. Tests: `T-9` (existing helpers); T-M1 steps (3), (6) and the AC-12 notice.
- AC-13: WHEN a 201 arrives on the list, the system SHALL re-read `GET /customers` without the loading spinner and update the row and subtitle (Dora $450.000 → `$400.000`, `$480.000 por cobrar en total`). Test: T-M1 step (3) (MutationObserver: rows never removed).
- AC-14: IF the server rejects, THEN the modal SHALL stay open with the server message and the list SHALL not change. Test: T-M1 step (5) (cashier, no open register → 403 message).
- AC-15, AC-18: WHEN balance `< 0`, the list row and the detail card SHALL show `Saldo a favor: $50.000` in `var(--success)`; `0` → `Al día`. Tests: `T-8`; T-M1 steps (4) and (8).
- AC-16, AC-17: The detail and the dashboard SHALL open the same modal; the dashboard sends `method`. Test: T-M1 step (8) (dashboard `Transferencia` → stored `method: "transfer"`).
- AC-19: The modal SHALL fit 320 px with no horizontal scroll, CSS variables only, `formatCOP`, `pnpm lint` at 0. Test: T-M1 step (7) (`scrollWidth` 320, quick buttons wrap); QA diff scan (no new hex or `rgba`); `pnpm lint` exit 0. Real-device check: `gaps.md` US-084 M-1.
- AC-20: No file under `internal/` SHALL change. Test: T-R1 (`git diff dev...HEAD --stat -- internal/` empty).

---

## US-088 — Voiding a sale no longer subtracts cash from the register (R-110)

**As** Don Carlos (owner) and whoever closes the register,
**I want** a voided sale to leave the register's expected cash as it was before that sale,
**so that** the close does not show a shortage (*faltante*) that does not exist.

**Estimated effort:** 0.5 day. **Impact:** High (money; every void makes the close and the dashboard cash wrong). **Origin:** R-110 (US-084 QA), PO priority on 2026-11-07 ahead of US-085.

**Technical context (verified 2026-11-07 at `3be043f`, by running through the API):**
- `SaleRepository.Void` inserts a `cash_out` movement `Anulación venta #N` for the full sale total whenever the sale has a `register_id`, whatever the payment method (`internal/repository/sale.go:342-349`).
- Expected cash already drops the voided sale, because `cash_sales` counts only `status = 'completed'` (`internal/repository/register.go:166-175`). A cash void is subtracted twice; a credit, transfer or card void subtracts cash that never entered the drawer.
- Run: opening $100,000 + cash sale $50,000 → expected $150,000; void → $50,000 (should be $100,000); then void a $50,000 credit sale → $0. Cash $30,000 + card $20,000, voided → −$80,000 instead of −$30,000.
- The dashboard's estimated cash subtracts every `cash_out`, so it subtracts voids too (`internal/repository/metrics.go:142-143`).
- A void after the register is closed is accepted and writes the `cash_out` into that closed register.
- There is no void action in the UI: voids happen only through the admin API (`DELETE /api/sales/{id}/void`).

**Business rule:**
- A void writes no register movement. The voided sale stops counting through the existing `status = 'completed'` filter, so expected cash drops by the sale's cash portion only.
- `cash_out` rows already written by voids are left as they are (no migration).

**Acceptance Criteria:**
- [x] Opening $100,000 + cash sale $50,000 → expected $150,000; void it → $100,000.
- [x] A $50,000 credit, transfer or card sale leaves expected at $100,000 before and after its void.
- [x] Cash $30,000 + card $20,000 → expected $130,000; void it → $100,000.
- [x] Closing that register counting $100,000 after the cash void → difference $0.
- [x] The dashboard's estimated cash does not subtract voided sales.
- [x] A void after the close writes nothing into the closed register and does not change its stored close.

**DoD:**
- Regression tests with the numbers above (pattern `setupTestDB`, `internal/handler/dashboard_test.go:14`), failing on `dev` first.
- `spec/domain/caja.md` updated: the R-110 "known defect" note becomes the voided-sales rule.

**Verified criteria (closed 2026-11-07):**

Contract discrepancies:
- D-1: there is no void action in the UI, now or in git history; voids are reachable only through the admin API (`DELETE /api/sales/{id}/void`). So the field impact was lower than R-110 stated, and no data migration was needed (roadmap R-111).
- D-2: the dashboard's estimated cash and the assistant's cash answer subtracted every void too (run: −$250.000 after 5 voids of $50.000). Fixed by the same change.
- D-3: a void after the close wrote the `cash_out` into the sale's own, already closed register, bypassing the closed-register guard of manual movements.
- D-4: `spec/domain/caja.md` said `cash_out` "is not a refund" while voids used it; now true again.
- D-5: transfer, card and mixed sales show as green `+` cash in the movement list (roadmap R-114).

GATE 1 (PO, 2026-11-07): Q-1..Q-4 accepted as recommended. A void writes no register movement. The fix is forward only, with no migration, and old `Anulación venta #N` rows are left as they are. Voids after the close stay allowed and write nothing (refund and closed-register policy: R-112). The US goes in SPRINT-19. Gemini contract review: 10 raised, 1 verified and accepted (AC-7 now states what `GET /registers/{id}` returns for the closed register), 9 discarded.

QA round 1: OK, 0 BLOCKER, 2 MINOR. M-1: the R-110 roadmap row cited the removed `sale.go:342-347`, fixed at close. M-2: after a void the open register still lists `Venta #N +$50.000` with no paired line; this is the accepted Q-1 trade-off (R-113). RED `9d8a6dd`, implementation `f3164b1`, which changes only `internal/repository/sale.go`. QA ran `go build ./...`, `go test -count=1 ./...` green, the new tests 5 times (10/10 ok) and on RED (T-1..T-7 fail with the `dev` numbers), `pnpm build`, `pnpm test` 170 passed and `pnpm lint` 0. It also ran a smoke test on the compiled binary that reproduced the R-110 scenario fixed, including `/register` showing `TOTAL ESPERADO $100.000`. Gemini diff round 1: clean. Spec: `spec/domain/caja.md` → "Movements" (voids write no movement), "Rule: expected cash at close" (voided sales, voids after the close), "Rule: estimated cash of the day".

- AC-1: WHEN a completed sale with a `register_id` is voided, the system SHALL insert no `register_movements` row, whatever the payment method; the rest of the void is unchanged (status `voided`, `voided_by`, stock 9 → 10, one `void` audit row `{"total":50000.00}`). Tests: `TestSaleRepo_T1_VoidWritesNoRegisterMovement` (`internal/repository/sale_void_test.go`); movement counts in `TestRegisterVoid_T3_NonCashSaleVoidKeepsExpected` (`internal/handler/register_void_test.go`).
- AC-2: WHEN a cash sale is voided while its register is open, opening $100.000 + cash sale $50.000 → `expected` 150,000; void → 100,000, `cash_sales` 0, `cash_out` 0, no `Anulación` movement. Test: `TestRegisterVoid_T2_CashSaleVoidRestoresExpected`.
- AC-3: WHEN a $50.000 credit, transfer or card sale is voided, `expected` SHALL stay 100,000 before and after, `cash_out` 0. Test: `TestRegisterVoid_T3_NonCashSaleVoidKeepsExpected` (table-driven over the three methods).
- AC-4: WHEN a mixed sale is voided, `expected` SHALL drop by its cash portion only: cash $30.000 + card $20.000 → 130,000 → 100,000; cash $30.000 + credit $20.000 → 130,000 → 100,000. Test: `TestRegisterVoid_T4_MixedSaleVoidRemovesCashPortionOnly`.
- AC-5: WHEN a register with a voided $50.000 cash sale is closed counting $100.000, the system SHALL store `expected_close` 100,000 and `difference` 0. Test: `TestRegisterVoid_T5_CloseAfterVoidHasNoShortage` (closes through `RegisterService.Close`, not the HTTP route, see R-115).
- AC-6: WHEN cash flow is computed over a range with a voided cash sale, the void SHALL not be subtracted: completed cash $30.000 + voided cash $50.000 + manual `cash_out` $5.000 → `cashSales` 30,000, `cashOut` 5,000. Test: `TestMetricsRepo_US088_T7_CashFlowIgnoresVoid` (`internal/repository/metrics_test.go`).
- AC-7: WHEN a sale of an already closed register is voided, the system SHALL answer 200, write no movement in any register, and keep the stored `expected_close` 150,000, `actual_close` 150,000 and `difference` 0. `GET /registers/1` SHALL return the recomputed `expected` 100,000 (`cash_sales` 0, `cash_out` 0) and `register.expected_close` 150,000. The open register 2 SHALL stay at 80,000. Test: `TestRegisterVoid_T6_VoidAfterCloseWritesNothing`.
- AC-8: A second void of the same sale SHALL answer 400 `la venta ya fue anulada` with no new row anywhere; an unknown id SHALL answer 404 `venta no encontrada`. Tests: `TestRegisterVoid_T8_SecondVoidAndUnknownID`; the existing `TestVoidSale_WritesAuditLog` (`internal/handler/audit_test.go`) is unchanged and green.
- AC-9, AC-10: No migration, no change to existing rows, no frontend change. Test: T-R1 (`git diff dev...HEAD -- internal/database/migrate.go frontend/` empty, checked by QA).

---

## US-085 — Aging filter chips with amounts on the Customers tab

**As** Don Carlos,
**I want** to see at the top of Customers how much is owed by age bucket and filter by it,
**so that** I know whom to collect from today.

**Estimated effort:** 0.5 day. **Impact:** Medium.

**Technical context (verified 2026-11-05):**
- The list already colors each row by age (`agingColor`, `frontend/app/customers/page.tsx:44`) from `oldest_debt_at`, returned by `GET /customers` (`internal/repository/customer.go:27`).
- Debt age is measured from the oldest unpaid credit sale (`spec/domain/fiado.md`, "Rule: debt age").
- R-05 (UTC vs local time) may shift `days_overdue` by one day near midnight; not fixed here.

**Business rule:**
- Three chips above the table, each with amount and count: `1–14 días`, `15–30 días`, `Más de 30 días`. Clicking a chip filters the table; clicking it again clears the filter.

**Acceptance Criteria:**
- [ ] Customers owing $75,000 (5 days), $300,000 (20 days), $330,000 (28 days) → chips `1–14 días: $75.000 (1)`, `15–30 días: $630.000 (2)`, `Más de 30 días: $0 (0)`.
- [ ] Customers with balance 0 are not counted.
- [ ] Works at 320 px (chips wrap).

**DoD:**
- Vitest test for the bucketing helper with the numbers above.

---

## Sprint success criterion
- The register close matches the drawer when customers pay their fiado in cash.
- Every collection WhatsApp message names the business.
- The cashier sees a customer's debt before extending more credit, and has to confirm when the customer is over the limit or long overdue.
- Registering an *abono* from the list takes one modal.

## Technical notes for the dev
- Money columns are `REAL` across the schema (`customer_payments.amount`, `sales.total`, `registers.opening_amount`; `internal/database/migrate.go:206-248`). Do not change types in this sprint; follow the existing pattern.
