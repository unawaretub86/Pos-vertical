# Product — Pos-vertical

> **Type:** spec · **Current:** yes · **Rules over:** what the product is and is not
> **Updated:** 2026-10-01

## What it is

A local POS that works without internet, handles customer *fiado* (credit extended to customers), and charges no monthly fee.

It is distributed as **a single executable** that the shopkeeper (*tendero*) copies to their computer and opens. No complicated installer, no cloud account, no server setup.

## Who it is for

Small and medium businesses in Colombia: neighborhood shops (*tiendas de barrio*), food distributors, hardware stores, cafeterías. See [personas.md](personas.md).

The hardware fleet is **4GB of RAM as the common case**, 8GB is "good", many with spinning disks and CPUs without a GPU. Every technical decision is measured against that machine, not the developer's.

## The three promises

They are the product. Breaking any of the three is changing product, and that is Andrey's call.

### 1. Works without internet

There is no call to an external service on the critical path of selling, charging, extending credit or closing the register. Pull the cable and everything stays the same.

The only accepted exception is **DIAN electronic invoicing**, which by law requires an authorized provider — and is designed not to block the sale when it fails.

### 2. Handles fiado for real

It is the differentiator. No competitor — local or worldwide — does it well. See [domain/fiado.md](domain/fiado.md).

### 3. No monthly fee

No feature may introduce a recurring cost for the customer. That rules out paid APIs, automatic WhatsApp Business sending and cloud hosting as part of the base product.

## What it is NOT

- **Not an ERP.** No accounting, payroll or fixed assets.
- **Not multi-site in the cloud.** Multi-register over a local LAN is on the roadmap (`R-18`); replication between shops is not.
- **Not a payment gateway.** Nequi and bank transfer are recorded as a payment method; no one's API is integrated.
- **Does not lend money.** Embedded capital (`R-41`) requires a regulated fintech partner and is not pursued alone.

## How we decide what to build

Priority order when a choice has to be made:

1. **What the first customer needs to drop the notebook.** If it does not block that, it is not urgent.
2. **What deepens fiado.** It is where the competition does not reach.
3. **What closes an obvious gap with local players** (Alegra, Siigo, Tiendatek, Vendty). Do not lack what everyone takes for granted.
4. **What looks good in a demo** — it matters, but never before 1 and 2.

What comes next is in [../roadmap.md](../roadmap.md), the single source.
