// Pure sale-total math shared by the POS checkout UI. Kept side-effect-free
// and framework-free so it can be unit tested without React/DOM, and so the
// formulas here can be pinned 1:1 against internal/service/sale.go — any
// drift between this file and the backend's recalculation causes every sale
// to be rejected with "total inconsistente" (see SPRINT-10 US-046 QA notes).

export interface CalcItem {
  unitPrice: number;
  quantity: number;
  discount: number; // item-level discount, in COP
  taxRate: number; // fraction, e.g. 0.19 for 19% IVA
}

export interface PaymentLine {
  amount: number;
}

/** Sum of (unitPrice * quantity - itemDiscount) across items. */
export function calcSubtotal(items: CalcItem[]): number {
  return items.reduce((sum, i) => sum + i.unitPrice * i.quantity - i.discount, 0);
}

/**
 * IVA is exclusive (added on top of price), computed per item from the same
 * base the backend uses: (unitPrice * quantity - itemDiscount) * taxRate.
 * The global/cart-level discount does NOT reduce the tax base — matching
 * internal/service/sale.go's `tax += itemSub * product.TaxRate` before
 * `sale.Discount` is subtracted.
 */
export function calcTax(items: CalcItem[]): number {
  return items.reduce((sum, i) => sum + (i.unitPrice * i.quantity - i.discount) * i.taxRate, 0);
}

/** subtotal - globalDiscount + tax. Must equal the backend's recalculated total. */
export function calcTotal(items: CalcItem[], globalDiscount: number): number {
  return calcSubtotal(items) - globalDiscount + calcTax(items);
}

export function sumPayments(lines: PaymentLine[]): number {
  return lines.reduce((sum, l) => sum + l.amount, 0);
}

/**
 * Whether payment lines cover the total within rounding tolerance. Mirrors
 * the backend's validateSale tolerance (internal/service/sale.go): a $1 COP
 * gap is rejected, so the tolerance here must stay under 1, not <=1.
 */
export function paymentsBalanced(lines: PaymentLine[], total: number): boolean {
  return Math.abs(sumPayments(lines) - total) < 0.5;
}
