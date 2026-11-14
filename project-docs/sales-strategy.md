# Sales strategy — Pos-vertical

> **Type:** current · **Current:** yes · **Rules over:** business model, pricing, cost model, segments, channels and support model
> **Updated:** 2026-11-13

This document says **how the product makes money and what that costs**. What the product is and promises stays in [spec/product.md](spec/product.md); the work it implies is tracked only in [roadmap.md](roadmap.md) (R-200..R-209 and the launch stages). Customer-facing texts (offer, pitch, contract) are in [`../docs/comercial/`](../docs/comercial/) in Spanish, and they follow this document, not the other way round.

Every number below is an **order of magnitude with an explicit assumption**, not a quote. Rows marked *verify* must be confirmed against the official source before they reach a customer (R-209, R-203).

---

## 1. Where we are (2026-11-13)

| Fact | Value | Source |
|---|---|---|
| Paying customers | 0. Client #1 (a neighborhood shop) is about to be installed | roadmap, stage 1 |
| Product | Offline POS, *fiado* (credit extended to customers), register close with count, owner/cashier roles, local backups, catalog import, DIAN e-invoicing built but off | `features.md`; DIAN gate `internal/handler/handler.go:79` |
| Install cost today | About one working day per shop | PO estimate; stage 2 target is under 30 minutes (R-175) |
| Commercial offer | Drafted: list $1.000.000, founder $500.000 paid 50/50, 3 months of WhatsApp support | `docs/comercial/oferta-y-contrato.md`, R-174 |

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
| **Licence** (one-time) | List $1.000.000; founder $500.000 for the first 5 shops only, then $800.000 | `pos.exe` for one business, install, catalog and *fiado* load, 45-minute training, 3 months of WhatsApp support, updates for 12 months | Segment A |
| **Connected** (`Plan Conectado`) | $29.900/month or $299.000/year (2 months free) | Off-machine encrypted backup and restore, owner summary on the phone, WhatsApp *fiado* reminders (fair use: 300/month), updates, support | A and B, optional |
| **Connected + DIAN** | Connected + a DIAN volume tier priced at provider cost × ~1.5 | E-invoices / electronic POS documents within the tier | Segment B |
| **Rent-to-own** (to validate) | $39.900/month × 24 months ($957.600) | Licence owned at month 24; Connected included during the term | Shops that cannot pay upfront |
| **Free tier** (later) | $0 | One register, basic features, self-install, no personal support | Only after self-install works (R-207) |

Rules that protect the promise:

1. A lapsed Connected subscription **never** blocks selling, *fiado*, the register close or the export of data; only the connected services stop, after a 15-day grace period.
2. If the VPS is down, the shop does not notice until it opens a connected feature.
3. The cloud holds encrypted backups and summaries, **never the live database**.

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

## 6. Channels

| Channel | How | Cost | When |
|---|---|---|---|
| Founder direct | Visits, referrals from client #1 (founder price in exchange for a referral) | Time | Now |
| Accountants | They have dozens of clients obliged to e-invoice; perfect for segment B | 20–30% of the licence; 10–20% of Connected in year 1 | Stage 3 (R-206) |
| Hardware sellers (thermal printers, scanners, local or Mercado Libre) | Bundle: scanner + printer + licence | Same commission | Stage 3 (R-206) |
| Neighborhood PC technicians | They already enter the shops; they can do the install | Commission + install fee | Stage 3 (R-206) |
| Content and free tools | Short videos ("how to control *fiado*", "how to close the register"), a free *fiado* spreadsheet template, an IVA calculator | Time | After stage 2 |

The first sale a channel brings without the founder in the room is the signal that the business can become passive.

## 7. Support model

- Hours: Monday to Saturday, 8:00–18:00, by WhatsApp (offer draft).
- Order: manual and videos first, then WhatsApp, then a remote session, then a visit (paid outside the included period).
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

## Sources

- Factus endpoints: `internal/factelect/client.go:16-17`.
- Obligation to invoice: [Actualícese, no responsables de IVA and POS tickets](https://actualicese.com/obligacion-de-no-responsables-del-iva-de-convertir-tiquetes-pos-a-factura-electronica/); [La República, tiendas de barrio](https://www.larepublica.co/especiales/facturacion-electronica-2024/todas-las-tiendas-de-barrio-estan-obligadas-a-cumplir-con-la-facturacion-electronica-3941175); [Accounter, Resolución 165 de 2023](https://accounter.co/noticias/actualidad/la-republica-el-documento-equivalente-electronico); [Alegra, SIMPLE regime](https://blog.alegra.com/colombia/como-deben-facturar-las-pymes-en-colombia/); [Decreto 358 de 2020](https://normograma.icfes.gov.co/docs/pdf/decreto_0358_2020.pdf).
- DIAN provider prices: [Matias API](https://matias-api.com/); [Facturo](https://facturo.com.co/planes-y-precios/).
- VPS: [Hetzner price adjustment](https://docs.hetzner.com/general/infrastructure-and-availability/price-adjustment/); [Better Stack, Hetzner review 2026](https://betterstack.com/community/guides/web-servers/hetzner-cloud-review/).
- WhatsApp: [Meta, WhatsApp Business Platform pricing](https://developers.facebook.com/docs/whatsapp/pricing).
- Payments: [Cambio, Wompi QR rate](https://cambiocolombia.com/peso-a-peso-paso-a-paso/articulo/2026/9/wompi-amplia-los-pagos-sin-contacto-y-baja-al-uno-por-ciento-la-tarifa-de-los-pagos-con-qr); [Bancolombia, Wompi datáfono](https://www.bancolombia.com/negocios/productos/recaudos/datafono).
- Market figures (Confecámaras, Fenalco, NielsenIQ, Fenaltiendas) and competitor prices come from the PO's valuation notes of 2026-11-13 and comparison blogs; they are unverified until R-209.
