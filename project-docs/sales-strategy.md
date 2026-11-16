# Sales strategy — Pos-vertical

> **Type:** current · **Current:** yes · **Rules over:** business model, pricing, cost model, segments, channels and support model
> **Updated:** 2026-11-14 (§ 4.3 founder offer) · 2026-11-14 (Respaldo: § 4 row, § 4.1 recurring ladder, § 4.2 how it is sold; § 5.6 at 50% renewal; § 13 variables to test) · 2026-11-14 (§ 5.6 income target, § 12 renewal and guarantee rows) · 2026-11-14 (§ 6 content channel starts now, see `content-plan.md`) · 2026-11-13

This document says **how the product makes money and what that costs**. What the product is and promises stays in [spec/product.md](spec/product.md); the work it implies is tracked only in [roadmap.md](roadmap.md) (R-200..R-209 and the launch stages). Customer-facing texts (offer, pitch, contract) are in [`../docs/comercial/`](../docs/comercial/) in Spanish, and they follow this document, not the other way round.

Every number below is an **order of magnitude with an explicit assumption**, not a quote. Rows marked *verify* must be confirmed against the official source before they reach a customer (R-209, R-203).

---

## 1. Where we are (2026-11-13)

| Fact | Value | Source |
|---|---|---|
| Paying customers | 0. Client #1 (a neighborhood shop) is about to be installed | roadmap, stage 1 |
| Product | Offline POS, *fiado* (credit extended to customers), register close with count, owner/cashier roles, local backups, catalog import, DIAN e-invoicing built but off | `features.md`; DIAN gate `internal/handler/handler.go:79` |
| Install cost today | About one working day per shop | PO estimate; stage 2 target is under 30 minutes (R-175) |
| Commercial offer | Drafted: list $1.000.000, founder $500.000 paid 50/50, 12 months of Respaldo included (was 3 months of support until 2026-11-14), all pending PO confirmation | `docs/comercial/oferta-y-contrato.md`, R-174 |

The code is ahead of the stage. What raises the value of the business now is **evidence of use and a cheaper install**, not more features.

## 2. Positioning

The base product keeps its promise: **runs on the shop's PC, without internet and without a monthly fee, and the data belongs to the owner.** On top of it there is an optional **Connected plan** (UI name `Plan Conectado`) for the services that by nature need the internet and have a recurring cost. This is the hybrid model the PO approved on 2026-11-13 (`spec/product.md`, promise 3).

*Fiado* alone is no longer a sufficient argument: Treinta, a free phone app, records *fiado* and sends payment reminders. What a phone app does not do, and we do:

- runs on the counter PC with a barcode scanner and a thermal printer, with no internet;
- register close with a cash count and the difference shown;
- owner and cashier separated: the cashier does not see profit or receivables;
- the database is a file on the owner's machine, not a vendor account.

### Competitors (to verify, R-209)

| Competitor | Model | Price | Note |
|---|---|---|---|
| Treinta | Phone app, freemium | Free, optional paid features | *Fiado* and reminders; reported ~1.4 million users in Colombia (comparison blogs, unverified) |
| Alegra | Cloud SaaS | From ~$25.900/month | E-invoicing included; needs internet |
| Vendty | Cloud SaaS | ~$85.000–130.000/month | |
| Unnamed desktop POS offered to client #1 | One-time licence | ~$1.000.000 | Reported orally by client #1. Old-looking desktop UI; Andrey believes it has no e-invoicing and no reports (unverified: ask the client for its name) |

The last row matters: a one-time **$1.000.000** for a weaker product was put in front of our first customer. That supports keeping the list price at $1.000.000 instead of undercutting it (section 4).

## 3. Segments

| Segment | Why they buy | What they pay for | Status |
|---|---|---|---|
| A. Neighborhood shops and minimarkets (*tiendas*) | Drop the notebook, control *fiado* and the cash | Licence; Connected optional | Anchor vertical, client #1 |
| B. Distributors and IVA-responsible shops (minimarkets, hardware stores, pharmacies) | Business-to-business *fiado* (receivables from dozens of shops) and **mandatory** e-invoicing | Licence + Connected + DIAN | To validate with 2 visits (R-208) |
| C. Other verticals on the same core (hardware store, pharmacy, bakery, restaurant) | Same core, vertical configuration | Same plans | After stage 2; restaurant module already exists (`internal/service/dining_table.go`, `kds.go`) |

E-invoicing is **not** a selling point for segment A. A shop run by a natural person who is not responsible for IVA, in the ordinary regime, is in general not obliged to invoice (Estatuto Tributario art. 616-2). Shops set up as companies, in the SIMPLE regime, or responsible for IVA are. That is segment B. Sources at the end; confirm with an accountant.

**Legal check before go-live (R-200):** according to Actualícese, a non-obliged shop that hands out "tiquetes POS" voluntarily may become obliged to turn them into electronic documents. The printed ticket title must be reviewed with an accountant before client #1 goes live.

## 4. Plans and prices (proposal, PO decides)

| Plan | Price | Includes | For |
|---|---|---|---|
| **Licence** (one-time) | List $1.000.000; founder $500.000 for the first 5 shops only, then $800.000 | `pos.exe` for one business, install, catalog and *fiado* load, 45-minute training, and the first 12 months of Respaldo | Segment A |
| **Respaldo** (annual, optional, from year 2; UI and sales name `Respaldo`) | ~$180.000/year; at the sale, a second year prepaid for $150.000 (licence + 2 years = $1.150.000) | WhatsApp support, updates, and reinstall with data restore when the PC is replaced or breaks | Every licence that does not take Connected |
| **Connected** (`Plan Conectado`) | $29.900/month or $299.000/year (2 months free) | Off-machine encrypted backup and restore, owner summary on the phone, WhatsApp *fiado* reminders (fair use: 300/month), updates, support | A and B, optional |
| **Connected + DIAN** | Connected + a DIAN volume tier priced at provider cost × ~1.5 | E-invoices / electronic POS documents within the tier | Segment B |
| **Rent-to-own** (to validate) | $39.900/month × 24 months ($957.600) | Licence owned at month 24; Connected included during the term | Shops that cannot pay upfront |
| **Free tier** (later) | $0 | One register, basic features, self-install, no personal support | Only after self-install works (R-207) |

Rules that protect the promise:

1. A lapsed Connected subscription **never** blocks selling, *fiado*, the register close or the export of data; only the connected services stop, after a 15-day grace period.
2. If the VPS is down, the shop does not notice until it opens a connected feature.
3. The cloud holds encrypted backups and summaries, **never the live database**.

### 4.1 The recurring ladder (PO 2026-11-14)

One recurring line with two levels, instead of four loose plans. From year 2 the shop picks one, or none:

| Level | Price | Needs a backend | Sellable |
|---|---|---|---|
| None | $0; each support case is charged apart (~$80.000, to validate) | No | Now |
| **Respaldo** | ~$180.000/year | No | Now |
| **Connected** (includes everything in Respaldo) | $299.000/year or $29.900/month | Yes (R-201) | Stage 4 |

Why a renewal and not only the alternatives:

| Model | For | Against |
|---|---|---|
| One-time licence only | Easy to sell | No recurring income |
| Monthly fee only | Recurring from day 1 | Breaks promise 3 (`spec/product.md`); competes head-on with Alegra and with Treinta's free app |
| Annual Respaldo | No server, sold from client #1, one collection a year by Nequi or transfer at $0 fee | Optional and never blocking, so many shops will not renew if nothing breaks |
| Connected | Best margin, value seen every day | Needs R-201 and R-204 first |

Respaldo is the floor, not the engine: renewal of an optional maintenance plan is planned at **40–60%** (Tech Lead estimate, no data yet; § 5.6 uses 50%). Connected is the upgrade: the gap between the two levels is about $10.000 a month, which is the upsell line once Connected exists.

Twelve months included instead of three raises the support cost of a licence from 2 hours to about 3 (12 × 15 min, § 5.2), about $50.000 less margin per sale than § 5.4 shows. It is accepted because the renewal date then arrives after a full year of use.

### 4.2 How Respaldo is sold

Customer wording is product content, in Spanish with "usted". Final texts live in `../docs/comercial/`.

1. **Never "renovación de licencia".** It sounds like a hidden monthly fee. The program is the shop's forever; what is renewed is the backing: *"El programa es suyo para siempre. Lo que se renueva es el respaldo."*
2. **Sell it as insurance, not as updates:** *"Es como el SOAT: lo paga una vez al año, y si se le daña el computador yo se lo dejo funcionando con todo su fiado."* No delivery time is promised until the first real reinstall has been timed.
3. **Price the alternative.** Without Respaldo each case is charged apart (~$80.000). Two cases a year already cost more than the plan.
4. **Offer the second year at the sale** ($1.000.000 with one year, $1.150.000 with two): willingness to pay is highest that day, and it removes the renewal risk for 24 months.
5. **Collect in month 11 by WhatsApp**, with the shop's own year in numbers (*fiado* collected, days closed), read from its dashboard and reports during a support contact. Nothing automatic is promised.
6. **A lapsed Respaldo never blocks anything** (same as rule 1 for Connected above): the shop keeps the version it has.

Renewal log: one more sheet in the support log of § 7 (shop, install date, Respaldo until, reminder sent, renewed yes/no, reason if no). The reason column is the cheapest market research we will get.

### 4.3 Founder offer for the first 5 shops (proposal 2026-11-14, PO confirms)

Customer text: `../docs/comercial/oferta-y-contrato.md` § "Oferta Tienda Fundadora". The price is not cut further; the offer removes the shop's risk and makes the limit real.

| Element | Value | Why |
|---|---|---|
| Price | $500.000, 5 numbered slots, 50% at install and 50% at day 30 | Already drafted; the list price stays visible as the anchor |
| 30-day trial | The shop may walk away in the first 30 days: no second payment, first payment returned, program removed, the receivables list handed over on paper (there is no customer export yet, R-236) | The shop's fear is paying for something it will not use. Simpler to explain and to verify than the 60-day *fiado* guarantee of R-229, which stays as the alternative (§ 13, variable 7) |
| Respaldo at founder price | $150.000 a year instead of ~$180.000 while renewed without a gap | A reason to renew that costs little |
| Referral | $50.000 per shop that buys on the founder's recommendation | Acquisition cost far below a founder visit day |
| Asked in return | Daily register close in the system for the first month, a short weekly talk, permission to tell the case and film the shop, two introductions if satisfied | The evidence § 1 says the business lacks; filming permission unblocks the founder-diary series of `content-plan.md` § 4 |

Cost, stated plainly: at today's one-day install the founder price already loses about $60.000 per shop (§ 5.4), about $110.000 with 12 months of Respaldo; a shop that walks away costs the install day and the visit. With 5 slots the worst case is bounded, and it is the price of the first proof. The offer is **not** extended past 5 shops.

Open: what happens to a rent-to-own shop that stops paying before month 24. The licence is not owned yet, but rule 1 says data is never locked. Recommendation: selling keeps working for 30 days with a banner, then the app opens read-only with export always available.

## 5. Cost model

Assumptions: 1 USD ≈ $4.000, 1 EUR ≈ $4.400 (verify on the day). Founder time is valued at **$50.000/hour** (a senior full-stack developer in Colombia at $7–12 million/month over 160 hours is $44.000–75.000/hour). Taxes on revenue are a 6% placeholder; the real rate depends on the tax regime (R-204).

### 5.1 Fixed monthly cash costs (once Connected exists)

| Item | Low | High | Assumption |
|---|---|---|---|
| VPS for the Connected backend (Hetzner CX23: 2 vCPU, 4 GB; €5.99 with IPv4 + 20% backups) | $32.000 | $64.000 | One server; high = a second one for staging or failover. Prices rose in June 2026, verify. Same stack as the product: Go + SQLite |
| Object storage for customer backups (Cloudflare R2: 10 GB free, then US$0.015/GB) | $0 | $10.000 | ~1 GB per customer; 150 customers ≈ $8.400 |
| Domain | $5.000 | $12.000 | US$15–35/year |
| Business email | $0 | $30.000 | Free tier vs. paid workspace |
| Code-signing certificate for `pos.exe` | $40.000 | $135.000 | Cloud signing ~US$10/month if available to a Colombian individual (verify), or an OV certificate US$250–400/year. Without it, Windows SmartScreen scares a self-installing customer |
| WhatsApp business line (prepaid SIM) | $10.000 | $20.000 | |
| Monitoring and alerts | $0 | $0 | Free tiers |
| **Infra and tools subtotal** | **$87.000** | **$271.000** | |
| Accountant and formalization | $150.000 | $300.000 | Needed to invoice the shops that ask for an invoice and to file taxes (R-204). Assumption, verify |
| **Total fixed cash** | **$237.000** | **$571.000** | Mid-point used below: **$400.000** |
| AI development tooling (already paid personally) | $400.000 | $800.000 | US$100–200/month. Shown apart: it is paid with or without the business |

### 5.2 Variable cost per Connected customer per month ($29.900)

| Item | Monthly payment | Annual payment | Assumption |
|---|---|---|---|
| Collection fee (Wompi general plan: 2.65% + $700 + IVA on the fee) | $1.776 | $855 | Card QR is 1% + IVA (Sept 2026 news); bank transfer or Nequi is $0 but manual. Verify |
| Storage and bandwidth | $100 | $100 | |
| WhatsApp reminders (300 utility messages × US$0.0008) | $1.000 | $1.000 | Meta rate card for Colombia, April 2026. A change from 1 Oct 2026 is reported but unverified |
| Taxes (6% placeholder) | $1.800 | $1.800 | R-204 |
| **Cash variable** | **$4.700** | **$3.800** | |
| **Cash contribution** | **$25.200** | **$26.100** | |
| Support time, 15 min at $50.000/hour | $12.500 | $12.500 | The biggest cost, and the one under our control |
| **Contribution after support time** | **$12.700 (42%)** | **$13.600 (45%)** | |

**Support minutes decide the margin.** At $29.900:

| Support minutes per customer per month | 5 | 10 | 15 | 30 |
|---|---|---|---|---|
| Support cost | $4.167 | $8.333 | $12.500 | $25.000 |
| Contribution (monthly payment) | $21.033 | $16.867 | $12.700 | $200 |

At 30 minutes a month, Connected earns nothing. The manual, short videos and in-app help are what make the plan profitable.

### 5.3 Break-even and scenarios (Connected only, monthly payment, 15 min support)

Break-even on fixed cash: $400.000 / $25.200 ≈ **16 customers**. Including the development tooling ($600.000 mid-point): $1.000.000 / $25.200 ≈ **40 customers**.

| Connected customers | Revenue/month | Cash variable | Fixed cash | Net cash/month | Support hours/month |
|---|---|---|---|---|---|
| 10 | $299.000 | $47.000 | $400.000 | −$148.000 | 2.5 |
| 30 | $897.000 | $141.000 | $400.000 | $356.000 | 7.5 |
| 50 | $1.495.000 | $235.000 | $400.000 | $860.000 | 12.5 |
| 100 | $2.990.000 | $470.000 | $400.000 | $2.120.000 | 25 |
| 150 | $4.485.000 | $705.000 | $400.000 | $3.380.000 | 37.5 |
| 150, with a part-time support hire (~$1.500.000, assumption) | $4.485.000 | $705.000 | $1.900.000 | $1.880.000 | ~5 (escalations only) |

The last two rows are the price of "passive": past ~100 customers, buying back about 30 hours a month costs about $1.500.000.

### 5.4 Licence economics (one sale)

Included support is assumed at 2 hours over the 3 months ($100.000); transport $30.000 per visit.

| | Founder $500.000 | List $1.000.000 |
|---|---|---|
| Install as today: 8 hours on site | **−$60.000** | $410.000 |
| Target: 30 min install + 45 min training + 30 min data load (1.75 h) | $252.500 | $722.500 |
| Sold and installed by a partner (30% commission), 0.5 h of ours, no transport | $195.000 | $515.000 |

At one day per install, the founder price **loses money**. Getting the install to under 30 minutes (R-175, R-202) is worth more than any feature.

### 5.5 DIAN cost (segment B only)

The provider integrated today is Factus (`internal/factelect/client.go:16`); its price was not found publicly (R-203). For reference, Matias API publishes yearly plans that include the certificate and DIAN enablement: 500 documents for $196.000 (~$392 each), 3.000 for $365.000 (~$122), 8.000 for $560.000 (~$70). Whether a plan covers one issuer (NIT, tax ID) or several is unverified.

| Customer | Documents/year | Provider cost at those list prices |
|---|---|---|
| Distributor, 20 invoices a day × 26 days | ~6.240 | ~$560.000/year ≈ $46.700/month |
| Shop issuing an electronic POS document on every sale, 60 a day | ~21.600 | Above the largest listed plan; >$1.500.000/year at ~$70 each |

DIAN is only viable for high-ticket, low-volume issuers unless a reseller rate is negotiated. It is priced as provider cost × ~1.5 in volume tiers, never bundled flat.

### 5.6 Income target (PO 2026-11-14)

Target: **$3.000.000–5.000.000 a month** to the founder. One scenario that reaches it, built only from the numbers above plus the proposed annual renewal (roadmap R-225, ~$180.000 a year, to validate):

| Line | Assumption | Per month |
|---|---|---|
| Connected | 200 installed shops, 40% attach (§ 9) = 80 × $25.200 | $2.016.000 |
| Respaldo | Half of the other 120 shops renew (§ 4.1) = 60 × $15.000, less ~9% taxes and collection | ~$820.000 |
| Fixed cash | § 5.1 mid-point | −$400.000 |
| **Recurring, before support** | ~35 support hours a month: 140 covered shops at 15 min | **~$2.435.000** |
| Part-time support hire | § 5.3 assumption | −$1.500.000 |
| New licences | 4 a month sold by partners, list price (§ 5.4: $515.000 each) | $2.060.000 |
| **Total with the hire** | ~10 founder hours a month | **~$3.000.000** |

Without the hire the same base gives ~$4.500.000 for about 35 founder hours a month, which is a part-time job, not passive income. The low end of the target needs about **200 installed shops and a channel that sells about 4 licences a month**; it does not need a second product. The levers toward the high end are the Connected attach rate (each 10 points on 200 shops ≈ +$370.000 a month net of the Respaldo it replaces), the Respaldo renewal rate (each 10 points ≈ +$165.000) and second years prepaid at the sale. Per-case support income from shops with no plan is not counted.

## 6. Channels

| Channel | How | Cost | When |
|---|---|---|---|
| Founder direct | Visits, referrals from client #1 (founder price in exchange for a referral) | Time | Now |
| Accountants | They have dozens of clients obliged to e-invoice; perfect for segment B | 20–30% of the licence; 10–20% of Connected in year 1 | Stage 3 (R-206) |
| Hardware sellers (thermal printers, scanners, local or Mercado Libre) | Bundle: scanner + printer + licence | Same commission | Stage 3 (R-206) |
| Neighborhood PC technicians | They already enter the shops; they can do the install | Commission + install fee | Stage 3 (R-206) |
| Content and free tools | Short faceless PAS videos and carousels on TikTok, Instagram and Facebook, a free *fiado* spreadsheet, a WhatsApp tips channel; playbook in [content-plan.md](content-plan.md) | Time; $0 ads for 6–8 weeks | Organic now (PO 2026-11-14); paid promotion after install is under 30 min |

The first sale a channel brings without the founder in the room is the signal that the business can become passive.

## 7. Support model

- Hours: Monday to Saturday, 8:00–18:00, by WhatsApp (offer draft).
- Order: manual and videos first, then WhatsApp, then a remote session, then a visit (paid outside the included period).
- After the 12 months included, support belongs to Respaldo or Connected (§ 4.1); a shop with no plan pays per case.
- **Log every contact**: date, customer, minutes, cause. It is the input to section 5.2 and the cheapest data we can collect. A paper notebook or a spreadsheet is enough.
- Target: ≤ 15 minutes per customer per month after month 3; hire part-time help when support passes ~25 hours a month.

## 8. Connected architecture principles

Used when R-201 is planned. They are constraints, not a design:

1. `pos.exe` talks to the VPS only for Connected features, outside the critical path of selling.
2. Licence and subscription check with a long offline grace period; a failed check never stops a sale.
3. Backups are encrypted on the shop's PC before upload; the owner can restore them without the founder.
4. The owner summary is read-only, pushed from the shop's PC (sales, close, receivables); no live database in the cloud.
5. Holding customers' debtor data (names, phones, debts) in the cloud makes us a data processor under Colombian data-protection law (Ley 1581 de 2012). A privacy policy and a processing agreement are needed before launch (R-204).

## 9. KPIs

| KPI | Target | Where it comes from |
|---|---|---|
| Install to first ticket | < 30 min (median) | R-175 |
| Days with the register closed in the system | 7 in a row per shop | Stage 1 and 2 gates |
| Support minutes per customer per month | ≤ 15 | Section 7 log |
| Connected attach rate | ≥ 40% of licences by month 6 | Billing |
| Monthly Connected churn | < 3% | Billing |
| Customer acquisition cost | Founder hours + commission per sale | Section 7 log, partner payouts |

## 10. Risks (largest first)

1. Shops under pressure: fewer sales and less *fiado* (Fenalco reports a drop in 2025; the figure needs its primary source, R-209).
2. 42.5% of shopkeepers do not see a POS as a priority (Fenaltiendas): the upfront price is the barrier, which is why rent-to-own and the free tier exist.
3. E-invoicing pulls obliged businesses toward Alegra or Siigo; DIAN provider price can make our DIAN tier uncompetitive (R-203).
4. Key person: one founder does development, sales, installs and support.
5. Windows-only and local: no Mac, no multi-site.
6. Legal: ticket naming (R-200), invoicing our own sales and data protection (R-204).

## 11. Spin-offs (not POS work)

The same effort can produce other income without touching this roadmap:

- other vertical POS products on the same core (segment C);
- the agent pipeline used to build this product (`.claude/commands/us.md`, `project-docs/tosdd-plan.md`, `pipeline-bench.md`) as a course, template or consulting offer for other developers.

## 12. Decisions pending the PO

| Decision | Recommendation |
|---|---|
| Connected price | $29.900/month, $299.000/year |
| Licence price | Keep list $1.000.000; founder $500.000 for the first 5 shops only, then $800.000 |
| Rent-to-own | Offer it to 2–3 pilot shops before publishing it; default policy as in section 4 |
| Partner commission | 30% of the licence, 15% of Connected in year 1 |
| When to formalize (RUT, regime, invoicing) | Before the first Connected charge or the first company customer, whichever comes first |
| Free tier | Only after a stranger installs the product alone (R-202, R-207) |
| Respaldo (R-225, § 4.1) | Name `Respaldo`; ~$180.000 a year after the 12 months included; second year at the sale for $150.000; ~$80.000 per case without a plan; never blocks the product |
| Result guarantee (R-229) | 60 days, tied to *fiado* collected and to the register being closed in the system |

## 13. Strategies and variables to test (PO 2026-11-14)

Nothing in sections 3, 4 and 6 is proven: there are 0 paying customers. The PO accepts trying several strategies. Each row is a variable we can move, the option we start with, the alternatives, and what tells us to change.

| # | Variable | Start with | Alternatives to try | Signal that decides |
|---|---|---|---|---|
| 1 | Licence price | Founder $500.000 (first 5), then $800.000, list $1.000.000 | Hold $1.000.000 from shop 6; a lower entry ($300.000–400.000) with Respaldo charged from month 4 | Price objections in the lead log; closes per demo |
| 2 | Payment form | 50% at install, 50% at 30 days | Rent-to-own $39.900 × 24; three instalments; cash discount | Share of demos lost to "no tengo la plata ahora" |
| 3 | Support included in the licence | 12 months of Respaldo | 3 or 6 months, so the renewal comes sooner | Support minutes per shop after month 3 (§ 7 log) |
| 4 | Respaldo price | ~$180.000/year | $150.000 or $240.000; monthly $15.000–19.900 by Nequi | Renewal rate and the "reason if no" column (§ 4.2) |
| 5 | Second year at the sale | Offered at $150.000 | Not offered; offered at $120.000 | Share of buyers who take it |
| 6 | Price per case without a plan | ~$80.000 | $50.000 remote and $120.000 on site | Whether it pushes renewals or produces complaints |
| 7 | Guarantee | 30-day trial with refund for the 5 founder shops (§ 4.3) | The 60-day *fiado* result guarantee (R-229); no guarantee from shop 6 | Closes per demo with and without it; refunds asked |
| 8 | Channel | Founder direct + organic content | Distributor (R-227), accountant, PC technician, hardware seller (R-206) | First sale closed without the founder in the room (§ 6) |
| 9 | Segment | Neighborhood shops (A) | Distributors and IVA-responsible shops (B); hardware stores; farm-supply shops and towns (R-231, R-232) | Demos per 10 contacts and ticket size, per segment |
| 10 | Bundle | Software only | "Caja lista" preinstalled PC, scanner and printer in instalments (R-228) | Share of leads with no PC at the counter (R-231) |
| 11 | Install mode | On site in Bogotá | Remote with the signed installer and videos (R-202) | Install-to-first-ticket time (§ 9) |
| 12 | Lead magnet | Free *fiado* spreadsheet | Web calculator "¿Cuánta plata tiene en la calle?"; printed *fiado* notebook insert | "FIADO" messages per 1.000 views (`content-plan.md` § 7.4) |
| 13 | Demo | 15-minute video call or visit | A one-week trial on the shop's PC with the reset tool before go-live | Demos that become a sale |
| 14 | Connected price and content | $29.900/month, $299.000/year | Backup-only tier at a lower price; DIAN tiers for segment B | Attach rate (§ 9), once R-201 exists |
| 15 | Free tier | Not offered | One register, self-install, no support (R-207) | Only after a stranger installs alone |

Rules for testing with very few customers:

1. **One variable at a time per batch of about 5 shops.** Changing price, guarantee and channel together teaches nothing.
2. **Write the hypothesis and the stop rule before the batch**, in the lead log: what we expect, and what result makes us change.
3. With 5 shops the result is a signal, not a statistic. The words of the shop that said no are worth more than the count.
4. **A promise already made to a customer is kept** when the test moves on: founder prices, guarantees and included months are honoured as signed.
5. What is never a variable: the three promises of `spec/product.md`, the honesty rules of `content-plan.md` § 5, and "a lapsed plan never blocks the product".

## Sources

- Factus endpoints: `internal/factelect/client.go:16-17`.
- Obligation to invoice: [Actualícese, no responsables de IVA and POS tickets](https://actualicese.com/obligacion-de-no-responsables-del-iva-de-convertir-tiquetes-pos-a-factura-electronica/); [La República, tiendas de barrio](https://www.larepublica.co/especiales/facturacion-electronica-2024/todas-las-tiendas-de-barrio-estan-obligadas-a-cumplir-con-la-facturacion-electronica-3941175); [Accounter, Resolución 165 de 2023](https://accounter.co/noticias/actualidad/la-republica-el-documento-equivalente-electronico); [Alegra, SIMPLE regime](https://blog.alegra.com/colombia/como-deben-facturar-las-pymes-en-colombia/); [Decreto 358 de 2020](https://normograma.icfes.gov.co/docs/pdf/decreto_0358_2020.pdf).
- DIAN provider prices: [Matias API](https://matias-api.com/); [Facturo](https://facturo.com.co/planes-y-precios/).
- VPS: [Hetzner price adjustment](https://docs.hetzner.com/general/infrastructure-and-availability/price-adjustment/); [Better Stack, Hetzner review 2026](https://betterstack.com/community/guides/web-servers/hetzner-cloud-review/).
- WhatsApp: [Meta, WhatsApp Business Platform pricing](https://developers.facebook.com/docs/whatsapp/pricing).
- Payments: [Cambio, Wompi QR rate](https://cambiocolombia.com/peso-a-peso-paso-a-paso/articulo/2026/9/wompi-amplia-los-pagos-sin-contacto-y-baja-al-uno-por-ciento-la-tarifa-de-los-pagos-con-qr); [Bancolombia, Wompi datáfono](https://www.bancolombia.com/negocios/productos/recaudos/datafono).
- Market figures (Confecámaras, Fenalco, NielsenIQ, Fenaltiendas) and competitor prices come from the PO's valuation notes of 2026-11-13 and comparison blogs; they are unverified until R-209.
