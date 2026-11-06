# SPRINT-19 — Fiado Collection: Cash-Correct Payments and a Collection-First Customers Tab

**Goal:** Turn the *fiado* (store credit) module from a ledger into a collection tool, starting with the one money bug it has today: cash *abonos* (partial payments toward a fiado balance) never reach the register close. Then make debt visible at the point of sale and make collecting faster from the Customers tab.

**Start date:** 2026-11-05 (planned)

**Origin:** PO request on 2026-11-05 (fiado value proposal reviewed against the code; scope cut by the PO to five items). References: `project-docs/roadmap.md`, `project-docs/spec/domain/fiado.md`, `project-docs/spec/domain/caja.md`.

**Incoming technical debt:**
- New finding (2026-11-05): a cash *abono* is not counted in the register's expected cash. -> **US-082**.
- R-61: the collection WhatsApp message goes out without the business name. -> **US-083**.
- US-043 (SPRINT-09, pending): customer history in the POS at checkout. Backend already built; frontend missing. -> **moved here**.

**Out of scope (PO, 2026-11-05):** customer risk scoring (R-17), "quincena" collection mode (R-03), admin PIN to override the credit limit (R-11), tone-based WhatsApp templates, printed *abono* receipt over ESC/POS. Each is a full US and none is a quick win.

---

## Tasks

| US | Name | Status | Date |
|----|------|--------|------|
| US-082 | Payment method on *abonos*; cash *abonos* count in the register close | [x] Complete in code (manual PO checks pending, see `gaps.md`) | 2026-11-05 |
| US-083 | Business name in the collection WhatsApp message (R-61) | ⬜ Pending | |
| US-043 | Customer debt and history in the POS at checkout, with credit-limit confirmation | ⬜ Pending | |
| US-084 | Quick *abono* modal from the Customers list | ⬜ Pending | |
| US-085 | Aging filter chips with amounts on the Customers tab | ⬜ Pending | |

**Total estimated effort: ~4.5 days**

---

## Recommended implementation order

| Order | US | Reason |
|-------|----|--------|
| **1** | US-082 | It is a money bug: the register close shows an unexplained surplus for every cash *abono*. US-084 builds on its `method` field. |
| **2** | US-083 | Half a day, fixes a pre-existing defect that undermines every collection message. |
| **3** | US-043 | Backend endpoint already exists; frontend only. Prevents extending credit to someone already overdue. |
| **4** | US-084 | Depends on US-082 (method selector) to be cash-correct. |
| **5** | US-085 | Pure UI over data the list already returns. |

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
- [ ] With business name `Tienda Don Carlos`, the WhatsApp link from the customer detail and from the list includes `Tienda Don Carlos`.
- [ ] With an empty business name, the message is still valid and does not show an empty gap.

**DoD:**
- Vitest test for the message builder covering both cases.
- R-61 closed in `roadmap.md`.

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
- [ ] The panel uses `pos-summary`; the separate `/balance` call is removed.
- [ ] Customer with balance $450,000 and `days_overdue` 26 → badge `Debe: $450.000 (26 días)` shown as soon as the customer is selected.
- [ ] Limit $500,000, balance $450,000, credit sale of $60,000 → confirmation shown; cancel leaves the cart intact; confirm completes the sale.
- [ ] Credit sale of $40,000 to the same customer with `days_overdue` 10 → no confirmation.

**DoD:**
- Vitest tests for the confirmation rule (limit exceeded, >30 days, neither).
- `spec/domain/fiado.md` "Rule: credit limit" updated (warns and asks for confirmation; still does not block).
- SPRINT-09 row for US-043 points here.

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
- [ ] Balance $450,000: `Mitad` fills $225,000; typing $50,000 shows `Nuevo saldo: $400.000`.
- [ ] Balance $30,000: the `$50.000` button is not shown; `Total` fills $30,000.
- [ ] An amount above the balance cannot be confirmed (button disabled, message shown).
- [ ] Works at 320 px.

**DoD:**
- Vitest tests for the quick-amount and new-balance helpers.
- Uses `formatCOP`, `api`, CSS variables only.

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
