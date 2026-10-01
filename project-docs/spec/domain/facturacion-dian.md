# Domain — DIAN electronic invoicing

> **Type:** spec · **Current:** yes · **Rules over:** how invoicing relates to the sale
> **Updated:** 2026-10-01 · Verified against `internal/factelect/` on 2026-09-21.

This document covers **the product rule**. Implementation detail and provider research live in `project-docs/reference/facturacion-electronica.md` (formerly `fix-gaps.md`) and `project-docs/research/2026-05-dian-proveedores.md` (formerly `facElect-finds.md`), both from May 2026 and marked as implementation reference, not current state.

---

## The exception to "100% offline"

Electronic invoicing is **the only part of the product that touches the internet**, because the law requires it: DIAN (the Colombian tax authority) validates through an authorized provider. There is no offline way to issue a valid electronic invoice.

**That is why it is designed not to block the sale.** The sale is recorded and charged locally and immediately; the electronic document is issued afterwards, in the background (`internal/handler/sale.go` triggers invoicing in a goroutine after creating the sale, and a poller in `internal/factelect/poller.go` retries).

If the provider is down, Don Carlos keeps selling. That is the rule that is not negotiable.

## Activation

It is **off by default**. `FiscalConfig.Enabled` controls everything: with the flag false, the service is not even constructed (`internal/handler/handler.go`, in `New`).

A shopkeeper who does not invoice electronically never sees any of this. It is deliberate: most of the target market is not yet obliged to.

## Fiscal configuration

It resides in a single row (`FiscalConfig`): NIT (tax ID), legal name, address, CIIU (activity code), DIAN resolution, prefix, authorized range from/to, resolution and expiry dates, provider credentials and environment (test or production).

The **authorized range** and the **resolution expiry date** are real legal limits: when they run out or expire, DIAN rejects. Any future invoicing work should warn before that happens, not after.

## Documents

Issued electronic documents are stored in their own repository (`internal/repository/eledoc.go`), separate from `sales`. A sale can exist without an electronic document; a document always points to a sale.

The **credit note** exists to cancel an already-invoiced sale (`POST /sales/{id}/void-with-credit-note`): an issued invoice is not deleted, it is offset.

---

## Status and debt

- **US-034 — full Factus sandbox validation:** Completed in SPRINT-17, with the structure and mappings validated against the provider's API.
- The code structure is validated and ready to go to production once the customer provides final credentials.
