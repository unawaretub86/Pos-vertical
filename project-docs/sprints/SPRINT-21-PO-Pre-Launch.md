# SPRINT-21 — PO Pre-Launch: Close v1.0 and Go Out to Sell

> **Type:** sprint · **Owner:** Andrey (PO) · **Created:** 2026-11-14
> **This is a PO checklist, not pipeline work.** It has no US for the `/us` pipeline; the only code is block D, which starts after the PO approves it.

**Goal:** everything the PO must see, decide or check before selling: v1.0 accepted on a real PC and released, the offer closed with the `Respaldo` plan, the accountant's two answers, and client #1 live.

**Why now:** code and tests are green on `dev` (2026-11-14: `go build`, `go vet`, `go test ./...`, 333 vitest tests, lint 0), but no screen has been seen by a human on a PC like the client's, v1.0 is not a release, and the offer still has open values. Details of every manual check stay in [`../gaps.md`](../gaps.md) → "Pending PO validations"; this document only orders them and says which ones belong to v1.0. Business rules are in [`../sales-strategy.md`](../sales-strategy.md); upcoming work is in [`../roadmap.md`](../roadmap.md) § Revenue path.

Mark `[x]` when done. When a manual check passes, remove its row from `gaps.md`.

---

## Block A — Decisions (about 30 minutes, at the desk)

| Done | Decision | Recommendation | Source |
|---|---|---|---|
| [ ] | Commercial name | ~~Cuadra POS~~ (PO 2026-11-15) **dropped 2026-11-17:** *Kuadra* (Medellín fintech, Techstars/JP Morgan US$120k 2023, Credicorp, expanding to Ecuador) sells sales, inventory, e-invoicing, CRM and credit to the same micro-businesses; same sound, so SIC confusion risk. **PO 2026-11-17: Apúntelo** (what a customer says to ask for *fiado*, in "usted"). Risks told to the PO: *Apúntalo*, an appointments app with WhatsApp reminders, is one letter away; `apuntelo.com` is taken (`.co`, `.com.co`, `.app` did not resolve); a common phrase is a weak mark, so register it as a mixed mark (word + logo). Still for the PO: buy `apuntelo.co`, SIPI search "Apúntelo" and "Apúntalo" in classes 9, 35, 42, handles. History: the name need not contain "POS": in Colombia many shopkeepers call the card terminal (datáfono) "el POS". Checked and rejected 2026-11-17: Vecino (*Super Vecino*, MIQ's e-invoicing app for Colombian neighbourhood shops, 2025), Cajón (generic cash-drawer hardware term), Tranqui (Cali debt-collection fintech), Contao (German CMS), Tendo (close to Tivendo), Al Día (Alegra's "Siempre al Día"). Shortlist with no conflict found: **Apúnteme** (what a customer says to ask for *fiado*; `apunteme.co` does not resolve; leans on *fiado*) and Cuentica (`cuentica.co` free, but `cuentica.com` is a Spanish accounting SaaS in the same classes). Still for the PO: pick, SIPI search classes 9, 35, 42, handles | R-233, `content-plan.md` § 3.1 |
| [x] | Licence price and payment form | **PO 2026-11-16:** founder $500.000 for the first 5 shops, paid 50/50; from shop 6, $1.000.000 (down to $800.000 only if demos do not close); list $1.000.000 | `sales-strategy.md` § 4, `docs/comercial/oferta-y-contrato.md` |
| [x] | `Respaldo`: name, months included, price after | **PO 2026-11-16:** `Respaldo`, $180.000 a year; 12 months included for founder shops, 6 months from shop 6. It is the bridge to Connected, not the profit (`sales-strategy.md` § 4.4, § 5.7) | R-225, `sales-strategy.md` § 4.1 |
| [x] | Second year of Respaldo at the sale | **PO 2026-11-16:** $150.000 (licence + 2 years = $1.150.000) | same |
| [x] | Price per support case without a plan | **PO 2026-11-16:** $80.000 | same |
| [x] | Support hours | **PO 2026-11-16: a response time, not office hours.** WhatsApp Monday to Saturday, answered within one business day (usually the same day) in two fixed blocks; a shop that cannot sell is answered as soon as seen | `sales-strategy.md` § 7 |
| [x] | Founder offer for the first 5 shops | **PO 2026-11-16:** 30-day trial with refund, Respaldo at $150.000 a year, **3 extra months of Respaldo per referral (no cash)**, and what is asked in return | `sales-strategy.md` § 4.3, `oferta-y-contrato.md` § "Oferta Tienda Fundadora" |
| [x] | Phones on the shop WiFi in v1.0 | **PO 2026-11-14: it stays in the offer; checked in block C** (US-081 M-1..M-3, US-054). If it fails there, the sentence leaves the offer until fixed | `gaps.md` US-081, `oferta-y-contrato.md:17` |
| [x] | Go-live decisions D2..D5 | **PO 2026-11-16:** as recommended in the roadmap (train, do not build) | `roadmap.md` § Decide before go-live |
| [x] | First variable to test after the founder shops | **PO 2026-11-16:** the guarantee (variable 7); the founder offer runs unchanged for all 5 | `sales-strategy.md` § 13 |

## Block B — One accountant consult

| Done | Question | Source |
|---|---|---|
| [ ] | Which title must the printed ticket carry for a shop not obliged to invoice? | R-200, go-live D6 |
| [ ] | Client #1 charges IVA: is it obliged to issue electronic invoices or electronic POS documents? | R-234 |
| [ ] | How does Andrey invoice the licence and the Respaldo (RUT, regime)? | R-204 |
| [ ] | Does selling hardware in a bundle change that regime? (only if "Caja lista" is pursued) | R-228 |

If the second answer is yes, stop and tell the Tech Lead before go-live: DIAN is frozen and off (`roadmap.md` § Freeze).

## Block C — v1.0 acceptance session (3 to 4 hours, on a PC like the client's)

Build the real binary first (`make build`). Each row is a check in `gaps.md` with its full steps. Note every failure with the screen and the step; do not fix on the spot.

**C1. Install and data (the G-2 dry run)**

| Done | Check |
|---|---|
| [ ] | SPRINT-20 G-2: fresh folder, install, reboot, POS answers in under 30 s |
| [ ] | US-089 M-1: start with Windows |
| [ ] | US-089 M-2 and M-3: backup folder on USB or `D:`, section and banner |
| [ ] | US-094 M-1: catalog import from CSV |
| [ ] | US-103 M-1: starter catalog |
| [ ] | US-092 M-1: opening *fiado* balances |
| [ ] | US-090 M-1: Settings with DIAN off |
| [ ] | US-077, US-076, US-078: login, email password not exposed, logical delete of customers |

**C2. Selling, *fiado* and register (one full shift by hand)**

| Done | Check |
|---|---|
| [ ] | US-043 M-1 steps 1–5 and 7: *fiado* at checkout, credit limit, "Lo de siempre" |
| [ ] | US-107 M-1 steps 1–3 and 5: cart discount, `Fiado` label, negative amounts |
| [ ] | US-105 M-1: cashier *abono* from `Punto de Venta`, Customers screens guarded |
| [ ] | US-082 M-1..M-3, US-084 M-1: *abono* modal, close with *abonos* |
| [ ] | US-085 M-1, US-083 M-1, US-086 M-1 step 1: debt-age chips, `Cobrar` without a phone, customer detail |
| [ ] | US-095 M-1: void dialog |
| [ ] | US-104 M-1: close with a *faltante* |
| [ ] | US-091 M-1: F5 on the shop routes (skip `/mesas`, `/barra`, `/catalogo`) |
| [ ] | US-079 M-1..M-3, US-074: theme, sort icons, smoke of the pages, 320 px |
| [ ] | US-054, US-081 M-1..M-3: two phones on the shop WiFi charging into the same register (sold in the offer, PO 2026-11-14) |
| [ ] | US-098 M-1: one sale **after 19:00**, checked again the next morning |

**C3. Printer and scanner**

| Done | Check |
|---|---|
| [ ] | US-080 P-2 (USB printer through Windows) or P-1 (network printer), whichever the client has |
| [ ] | US-080 P-3 and P-4 (`/pos` part): paper width, accents, automatic ticket, *fiado* ticket |
| [ ] | SPRINT-20 G-1: on the client's own printer and scanner (can be done on go-live day, before opening) |

**C4. Reset before go-live**

| Done | Check |
|---|---|
| [ ] | US-093 M-1: reset tool on a **copy** first, then on the real file |

**Only if the shop will use it**

| Done | Check | When |
|---|---|---|
| [ ] | US-108 M-1: close email in a real mailbox | If the owner wants the email |
| [ ] | US-062 M-1..M-9: assistant on a phone | If the assistant is shown in the demo |

**Not part of v1.0 for a shop (do not run now):** US-087 M-1 and US-086 M-1 steps 2–3 (restaurant mode and public catalog), US-043 M-1 step 6 and US-107 M-1 step 4 (tables), US-034 (DIAN sandbox). They stay in `gaps.md` and must not be promised (`features.md` § 8).

## Block D — Release v1.0.0 (Tech Lead, after block C)

| Done | Task | Note |
|---|---|---|
| [ ] | Fix what block C found | Each fix goes through the usual pipeline; the PO re-checks only the failed step |
| [x] | The dead `Exportar` button on `Clientes`: remove it or build the customer export | R-236; **PO 2026-11-17: build the export** (admin-only customer CSV). Done 2026-11-17: [R-236 below](#r-236--customer-export-from-clientes) |
| [ ] | Version shown in `Configuración` and at startup | R-235; needs PO approval before it starts |
| [ ] | `dev` merged to `master`, tag `v1.0.0`, binary built from the tag | `master` is 77 commits behind `dev` (2026-11-14) |
| [ ] | `features.md` re-verified against the tag | It is the list to sell from |

### R-236 — Customer export from `Clientes`

- [x] **R-236** (2026-11-17) `Exportar` on `Clientes · Cartera` downloads a CSV of every active customer (name, phone, credit limit, balance, oldest-debt date), admin only, in a layout that Spanish (Colombia) Excel opens in columns and that R-219 can read back. Built through the `/us` pipeline; GATE 1 PO 2026-11-17 (Q-1..Q-4 as recommended); QA round 1 OK (0 BLOCKER, 4 MINOR). Browser-built, no backend change. Spec: `spec/domain/fiado.md` § Customer export. Manual check: `gaps.md` R-236 M-1.

**Verified criteria** (tests in `frontend/lib/customerExport.test.ts` unless stated)

| AC | Criterion (EARS) | Test |
|---|---|---|
| AC-1 | WHEN the export file is built, the system SHALL write UTF-8 with a leading BOM, `;` as the delimiter, CRLF between lines and no line break after the last line, and the header `Nombre;Teléfono;Tope de crédito;Saldo;Deuda más antigua`. | T-1 `buildCustomersCSV header (AC-1)` › "an empty list gives exactly the BOM and the header, with no trailing CRLF" |
| AC-2 | WHEN the export file is built, the system SHALL write one row per customer received and no total or summary row. | T-2 `buildCustomersCSV rows` › "writes the header and one line per customer…", "has no TOTAL row and no trailing CRLF" |
| AC-3 | WHEN a money column is written, the system SHALL write whole pesos as plain digits (no `$`, no separators, `Math.round`), `-` for a negative balance, and an empty cell for a `null` credit limit. | T-2; T-3 `buildCustomersCSV money rounding (AC-3)` › "rounds balances with Math.round and writes plain digits" (15000.4 and 14999.6 → `15000`) |
| AC-4 | WHEN a customer's balance is greater than 0, the system SHALL write `Deuda más antigua` as `dd/mm/yyyy`, the day of `oldest_debt_at` in `America/Bogota`. | T-2 › "writes the oldest debt as the Bogota day, not the UTC day" (`2026-11-18T02:30:00Z` → `17/11/2026`) |
| AC-5 | IF a customer's balance is 0 or negative, or `oldest_debt_at` is null, THEN the system SHALL leave `Deuda más antigua` empty. | T-2 (`Fer;;0;0;`, `Gus;;;-20000;`) |
| AC-6 | IF `Nombre` or `Teléfono` contains `;`, `"`, CR or LF, THEN the system SHALL quote it RFC 4180 style, doubling inner `"`; other values unquoted. | T-4 `buildCustomersCSV text quoting (AC-6)` › "quotes and doubles inner double quotes", "quotes a name with a line feed", "writes a plain name unquoted"; T-2 (`"Álvaro; ""el Mono"""`) |
| AC-7 | WHEN the export file is built, the system SHALL order rows by name with `localeCompare`. | T-2 (input Gus, Ana, Beto, Fer, Álvaro → Álvaro, Ana, Beto, Fer, Gus) |
| AC-8 | WHEN the admin clicks `Exportar`, the system SHALL request `GET /customers` with no `q` and no `has_balance`. | T-6 `runCustomerExport success (AC-8, AC-9)` › "requests /customers with no query and downloads the file"; manual R-236 M-1 step 1 |
| AC-9 | WHEN that request answers with one or more customers, the system SHALL download `clientes-YYYY-MM-DD.csv` with today's date in `America/Bogota`. | T-5 `customersExportFileName (AC-9)` › "uses the Bogota date at 21:30 Bogota (already 18/11 in UTC)", "uses the Bogota date at 10:00 Bogota"; T-6 |
| AC-10 | IF that request answers with no customers, THEN the system SHALL download nothing and show `No hay clientes para exportar`. | T-7 `runCustomerExport empty (AC-10)` › "null customers downloads nothing and returns empty", "an empty list downloads nothing and returns empty"; manual M-1 step 4 |
| AC-11 | IF that request fails, THEN the system SHALL download nothing and show `No se pudo exportar los clientes`. | T-8 `runCustomerExport failure (AC-11)` › "a rejected request resolves to error, never rejects, and downloads nothing"; manual M-1 step 5 |
| AC-12 | WHILE an export request is in flight, the system SHALL keep `Exportar` disabled, and SHALL enable it again when the request finishes, whether it succeeded, found no customers or failed. | T-10 `runCustomerExport busy state (AC-12)` › "sets busy true before get and false after get settles" (ok, empty, error); manual M-1 step 3 |
| AC-13 | WHILE the user is a cashier, the system SHALL keep the export unreachable (redirect to `/pos`, no request) and the backend is not changed. | T-9 = `frontend/lib/customerAccess.test.ts` › `T-8 canSeeCustomers` (US-105, unchanged); `git diff dev...HEAD` touches no Go file |

Limitations that ship in v1.0 and are said in the training, not hidden: the black console window must stay open (D3), only recent sales can be voided (D2), the starter catalog has $1.000 placeholder prices (R-182), a cashier cannot charge the wholesale price (R-215, client #1 does not sell wholesale).

## Block E — Ready to sell

| Done | Task | Who |
|---|---|---|
| [ ] | Offer and delivery sheet with the block A values, no `[confirmar]` left; two printed copies | Claude edited 2026-11-16 (`oferta-y-contrato.md`, `guion-de-venta.md`); PO reads and prints |
| [ ] | Sales script read once aloud, including the Respaldo objection | PO |
| [ ] | Support log and renewal log opened (one spreadsheet, two sheets) | PO; `sales-strategy.md` § 4.2 and § 7 |
| [ ] | Support WhatsApp line ready (number never written in the repository) | PO |
| [ ] | Stopwatch and notebook for the pilot measurements | PO; R-175 |
| [ ] | The four waitlist questions in the lead log, asked to every founder shop (transfers a day, fake transfers received, would pay $29.900 for the payment notifier, must e-invoice) | PO; R-239 |
| [ ] | A USB stick for each install: the backup folder points to it, never `D:` | PO; `sales-strategy.md` § 7 |

## Block F — Client #1 live

| Done | Task | Source |
|---|---|---|
| [ ] | SPRINT-20 G-3: go-live and 45-minute training | `gaps.md` |
| [ ] | Delivery sheet signed, Respaldo end date written on it | `oferta-y-contrato.md` clause 4 |
| [ ] | First payment collected | Offer |
| [ ] | 7 days in a row with the register closed in the system and matching the count | Stage 1 exit gate |

## Block G — In parallel (does not wait for block F)

| Done | Task | Source |
|---|---|---|
| [ ] | Two visits to distributors or IVA-responsible shops, pitching the distributor as a channel | R-208, R-227; visit script in `docs/comercial/guion-visita-distribuidor.md` (Spanish product content) |
| [ ] | Ask every lead whether there is a PC at the counter | R-231 |
| [ ] | Weekly content running | `content-plan.md` § 8 |

---

## Exit gate

v1.0.0 tagged and installed at client #1; the offer has no open value; the accountant answered the first two questions of block B; the register closed 7 days in a row; the support and renewal logs have their first rows. Then stage 2 (shops 2 to 5) starts, with one variable of `sales-strategy.md` § 13 under test.
