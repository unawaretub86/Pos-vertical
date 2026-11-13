// Pure sale-total math shared by the POS checkout UI. Kept side-effect-free
// and framework-free so it can be unit tested without React/DOM, and so the
// formulas here can be pinned 1:1 against internal/service/sale.go and money.go — any
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

/** num / den rounded half away from zero, in integer arithmetic (den > 0). */
function divRound(num: number, den: number): number {
  const n = Math.abs(num) + Math.floor(den / 2);
  const q = (n - (n % den)) / den;
  return num < 0 ? -q : q;
}

/**
 * round(quantity x unitPrice) in whole pesos. The quantity is taken in
 * thousandths (the catalog allows at most 3 decimals) so that, e.g., 1,005 kg
 * at $1.000 is 1005 and not 1004,9999... Mirrors service.LineBase.
 */
export function lineBase(unitPrice: number, quantity: number): number {
  return divRound(Math.round(quantity * 1000) * Math.round(unitPrice), 1000);
}

/** lineBase minus the whole-peso item discount. Mirrors the server's line subtotal. */
export function lineSubtotal(item: CalcItem): number {
  return lineBase(item.unitPrice, item.quantity) - item.discount;
}

/**
 * round(lineSubtotal x taxRate) using basis points of the rate (19% -> 1900).
 * Mirrors service.LineTax.
 */
export function lineTax(item: CalcItem): number {
  return divRound(lineSubtotal(item) * Math.round(item.taxRate * 10000), 10000);
}

/** Sum of the whole-peso line subtotals. */
export function calcSubtotal(items: CalcItem[]): number {
  return items.reduce((sum, i) => sum + lineSubtotal(i), 0);
}

/**
 * IVA is exclusive (added on top of price) and rounded per line, matching
 * internal/service/sale.go. The global/cart-level discount does NOT reduce the
 * tax base.
 */
export function calcTax(items: CalcItem[]): number {
  return items.reduce((sum, i) => sum + lineTax(i), 0);
}

/** subtotal - globalDiscount + tax. Must equal the backend's recalculated total exactly. */
export function calcTotal(items: CalcItem[], globalDiscount: number): number {
  return calcSubtotal(items) - globalDiscount + calcTax(items);
}

export function sumPayments(lines: PaymentLine[]): number {
  return lines.reduce((sum, l) => sum + l.amount, 0);
}

/**
 * Whether payment lines cover the total exactly. Money is whole pesos and the
 * server requires sum(payments) = total with tolerance 0 (internal/service/sale.go).
 */
export function paymentsBalanced(lines: PaymentLine[], total: number): boolean {
  return sumPayments(lines) === total;
}

/** Change owed on a cash payment. Mirrors frontend/app/pos/page.tsx's `change`. */
export function calcChange(cashReceived: number, total: number): number {
  return Math.max(0, cashReceived - total);
}

/**
 * Whether a credit sale would push the customer over their credit limit.
 * `amount` is what the sale adds to the debt: the sum of its credit payment
 * lines (see creditAmount in ./creditCheck). Customers with no `creditLimit`
 * set (0/undefined) are never over it.
 */
export function exceedsCreditLimit(customerBalance: number, amount: number, creditLimit: number | undefined | null): boolean {
  if (!creditLimit) return false;
  return customerBalance + amount > creditLimit;
}
