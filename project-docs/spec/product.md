# Product — Pos-vertical

> **Type:** spec · **Current:** yes · **Rules over:** what the product is and is not
> **Updated:** 2026-11-13 (hybrid model: optional Connected plan; *fiado* differentiator restated) · 2026-10-01

## What it is

A local POS that works without internet, handles customer *fiado* (credit extended to customers), and charges no monthly fee for the base product. Services that by nature need the internet are sold apart as an optional monthly **Connected plan** (see promise 3 and [../sales-strategy.md](../sales-strategy.md)).

It is distributed as **a single executable** that the shopkeeper (*tendero*) copies to their computer and opens. No complicated installer, no cloud account, no server setup.

## Who it is for

Small and medium businesses in Colombia: neighborhood shops (*tiendas de barrio*), food distributors, hardware stores, cafeterías. See [personas.md](personas.md).

The hardware fleet is **4GB of RAM as the common case**, 8GB is "good", many with spinning disks and CPUs without a GPU. Every technical decision is measured against that machine, not the developer's.

## The three promises

They are the product. Breaking any of the three is changing product, and that is Andrey's call.

### 1. Works without internet

There is no call to an external service on the critical path of selling, charging, extending credit or closing the register. Pull the cable and everything stays the same.

The accepted exceptions are **DIAN electronic invoicing**, which by law requires an authorized provider, and the optional **Connected plan** services (off-machine backup, owner summary, WhatsApp reminders). Both are designed not to block the sale when they fail.

### 2. Handles fiado for real

*Fiado* is core and stays deep (see [domain/fiado.md](domain/fiado.md)), but it is **not unique on its own**: Treinta, a free phone app, records *fiado* and sends payment reminders (decided 2026-11-13; competitor facts to verify, R-209). The differentiator is the combination a phone app does not offer:

- runs on the counter PC with barcode scanner and thermal printer, without internet;
- register close with a cash count and the difference shown;
- owner and cashier separated: the cashier does not see profit or receivables;
- the database is a file the owner keeps, not a vendor account.

### 3. No monthly fee for the base product

The base product (selling, *fiado*, inventory, register close, reports, local backup) has no recurring cost for the customer. Paid APIs, automatic WhatsApp sending and cloud hosting are allowed **only** inside the optional Connected plan, sold separately, under three rules:

1. A lapsed Connected subscription never blocks selling, *fiado*, the register close or the export of data; only the connected services stop.
2. If the Connected backend is down, the shop does not notice outside the connected features.
3. The cloud holds encrypted backups and summaries, never the live database.

Prices, costs and plans live in [../sales-strategy.md](../sales-strategy.md).

## What it is NOT

- **Not an ERP.** No accounting, payroll or fixed assets.
- **Not multi-site in the cloud.** Multi-register over a local LAN is on the roadmap (`R-18`); replication between shops is not. The Connected plan uses the cloud for backups and read-only summaries only.
- **Not a payment gateway.** Nequi and bank transfer are recorded as a payment method; no one's API is integrated.
- **Does not lend money.** Embedded capital (`R-41`) requires a regulated fintech partner and is not pursued alone.

## How we decide what to build

Priority order when a choice has to be made:

1. **What the first customer needs to drop the notebook.** If it does not block that, it is not urgent.
2. **What deepens fiado and the register close on the counter PC.** It is where a phone app does not reach.
3. **What closes an obvious gap with local players** (Alegra, Siigo, Tiendatek, Vendty). Do not lack what everyone takes for granted.
4. **What looks good in a demo** — it matters, but never before 1 and 2.

What comes next is in [../roadmap.md](../roadmap.md), the single source.
