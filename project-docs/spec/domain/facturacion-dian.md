# Domain — DIAN electronic invoicing

> **Type:** spec · **Current:** yes · **Rules over:** how invoicing relates to the sale
> **Updated:** 2026-11-13 (US-102: per-line whole-peso tax in the document) · 2026-11-10 (US-093: pointer to the go-live reset) · 2026-11-09 (US-095: void dialog text for the 409) · 2026-11-07 (US-090: behavior with DIAN off) · Verified against `internal/factelect/` on 2026-09-21 and against `internal/handler/` on 2026-11-07.

This document covers **the product rule**. Implementation detail and provider research live in `project-docs/reference/facturacion-electronica.md` (formerly `fix-gaps.md`) and `project-docs/research/2026-05-dian-proveedores.md` (formerly `facElect-finds.md`), both from May 2026 and marked as implementation reference, not current state.

---

## The exception to "100% offline"

Electronic invoicing is **the only part of the product that touches the internet**, because the law requires it: DIAN (the Colombian tax authority) validates through an authorized provider. There is no offline way to issue a valid electronic invoice.

**That is why it is designed not to block the sale.** The sale is recorded and charged locally and immediately; the electronic document is issued afterwards, in the background (`internal/handler/sale.go` triggers invoicing in a goroutine after creating the sale, and a poller in `internal/factelect/poller.go` retries).

If the provider is down, Don Carlos keeps selling. That is the rule that is not negotiable.

## Activation

It is **off by default**. `FiscalConfig.Enabled` controls everything: with the flag false, the service is not even constructed (`internal/handler/handler.go`, in `New`). The flag is read once at startup, so a change takes effect after a restart.

A shopkeeper who does not invoice electronically never sees any of this: the Settings section "Facturación electrónica DIAN" renders only when the saved config has `enabled` true (`frontend/lib/fiscalSettings.ts`). It is deliberate: most of the target market is not yet obliged to.

### With DIAN off

Every sale flow works unchanged: create, read, ticket, reprint, void and table checkout, and no electronic document is created even if a buyer email is sent. The DIAN-only endpoints answer a 4xx with `facturación electrónica no habilitada`: `POST /sales/{id}/void-with-credit-note` → 400 (a missing motive is checked first → 400 `motive requerido`), `GET /sales/{id}/factura` → 400, `GET /sales/{id}/factura/pdf` → 404. Certified by `internal/handler/dian_off_test.go` (US-090).

A sale that already has an `accepted` or `pending` document (DIAN was on before) cannot be voided while DIAN is off: `DELETE /sales/{id}/void` answers 409 with `error` `sale_has_accepted_invoice` and `requires_credit_note: true` (for an `accepted` or `pending` document), and the credit-note endpoint answers 400. The void dialog shows `Esta venta tiene un documento electrónico ante la DIAN; para anularla se necesita una nota crédito.` and offers no credit-note action (test: `TestSaleVoidUI_T6_ErrorBodies`). This is a known limitation, tracked in the roadmap (R-126).

### Amounts in the document (US-102)

The adapter builds the document from the stored sale (`internal/factelect/pos_adapter.go`, `buildItems`): each line's `TaxAmount` is the same whole-peso per-line tax as the sale (`service.LineTax`, [dinero.md](dinero.md)), so `TotalIVA` = Σ line `TaxAmount` = `sales.tax`, and `Total` = `sales.total`. Example: 3 lines of $1.001 at 19% → each `TaxAmount` 190, `TotalIVA` 570, `Total` 3.573. Documents already issued before US-102 are not rebuilt. Whether DIAN's validation would also accept a per-rate-group rounding was not verified; the per-line choice does not depend on it.

## Fiscal configuration

It resides in a single row (`FiscalConfig`): NIT (tax ID), legal name, address, CIIU (activity code), DIAN resolution, prefix, authorized range from/to, resolution and expiry dates, provider credentials and environment (test or production).

The **authorized range** and the **resolution expiry date** are real legal limits: when they run out or expire, DIAN rejects. Any future invoicing work should warn before that happens, not after.

## Documents

Issued electronic documents are stored in their own repository (`internal/repository/eledoc.go`), separate from `sales`. A sale can exist without an electronic document; a document always points to a sale.

The **credit note** exists to cancel an already-invoiced sale (`POST /sales/{id}/void-with-credit-note`): an issued invoice is not deleted, it is offset.

---

## Go-live reset (US-093)

The go-live reset (`reset.md`) never touches `fiscal_config`, the DIAN resolution or the provider numbering. It refuses to run while any electronic document is `accepted`, `draft` or `pending`, whatever `fiscal_config.ambiente` says (`electronic_documents` has no `ambiente` column); the only override is `--discard-fiscal-documents`, and `--force` does not override it. `rejected` and `cancelled` documents are cleared with the dry-run sales.

## Status and debt

- **US-034 — full Factus sandbox validation:** Completed in SPRINT-17, with the structure and mappings validated against the provider's API.
- The code structure is validated and ready to go to production once the customer provides final credentials.
