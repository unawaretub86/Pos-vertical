import { describe, it, expect } from 'vitest';
import { calcSubtotal, calcTax, calcTotal, sumPayments, paymentsBalanced, calcChange, exceedsCreditLimit, type CalcItem } from './saleCalc';

describe('calcSubtotal', () => {
  it('sums unitPrice * quantity minus item discount', () => {
    const items: CalcItem[] = [
      { unitPrice: 10000, quantity: 2, discount: 0, taxRate: 0 },
      { unitPrice: 5000, quantity: 1, discount: 1000, taxRate: 0 },
    ];
    expect(calcSubtotal(items)).toBe(24000);
  });
});

describe('calcTax', () => {
  it('is zero when no item has a tax rate', () => {
    const items: CalcItem[] = [{ unitPrice: 10000, quantity: 1, discount: 0, taxRate: 0 }];
    expect(calcTax(items)).toBe(0);
  });

  // Regression: real bug hit in production testing — a 19%-tax product ("nata",
  // 5 x $10.000) was rejected by the backend with "total inconsistente:
  // esperado 59500, recibido 50000" because the POS UI never added tax to the
  // total it sent. This pins the exact numbers from that incident.
  it('matches the nata incident: 5 x $10.000 @ 19% IVA = $9.500 tax', () => {
    const items: CalcItem[] = [{ unitPrice: 10000, quantity: 5, discount: 0, taxRate: 0.19 }];
    expect(calcTax(items)).toBeCloseTo(9500, 6);
    expect(calcTotal(items, 0)).toBeCloseTo(59500, 6);
  });

  it('computes tax from the item base (after item discount), summed across items at different rates', () => {
    const items: CalcItem[] = [
      { unitPrice: 100, quantity: 2, discount: 0, taxRate: 0.19 }, // base 200 → tax 38
      { unitPrice: 1000, quantity: 1, discount: 200, taxRate: 0.05 }, // base 800 → tax 40
    ];
    expect(calcTax(items)).toBeCloseTo(78, 6);
  });
});

describe('calcTotal', () => {
  it('is subtotal - globalDiscount + tax (global discount does not reduce the tax base)', () => {
    const items: CalcItem[] = [{ unitPrice: 10000, quantity: 5, discount: 0, taxRate: 0.19 }];
    // subtotal 50000, tax 9500 (from the un-discounted item base), global discount 5000
    expect(calcTotal(items, 5000)).toBeCloseTo(54500, 6);
  });

  it('matches a plain no-tax sale (most existing test data)', () => {
    const items: CalcItem[] = [{ unitPrice: 15000, quantity: 1, discount: 0, taxRate: 0 }];
    expect(calcTotal(items, 0)).toBe(15000);
  });
});

describe('paymentsBalanced / sumPayments', () => {
  it('balances when the single line covers the full total (default, non-mixed mode)', () => {
    expect(paymentsBalanced([{ amount: 60000 }], 60000)).toBe(true);
  });

  it('balances a 3-way mixed split that sums exactly to total', () => {
    const lines = [{ amount: 30000 }, { amount: 20000 }, { amount: 10000 }];
    expect(sumPayments(lines)).toBe(60000);
    expect(paymentsBalanced(lines, 60000)).toBe(true);
  });

  // Doc test (SPRINT-10 US-046): pagos que suman $59.999 sobre total $60.000
  // (fuera de tolerancia) → rechazada. A $1 COP gap must NOT balance.
  it('rejects a $1 COP gap — matches backend tolerance exactly', () => {
    expect(paymentsBalanced([{ amount: 59999 }], 60000)).toBe(false);
  });

  it('rejects overpayment the same way as underpayment', () => {
    expect(paymentsBalanced([{ amount: 60001 }], 60000)).toBe(false);
  });

  it('tolerates sub-peso floating point rounding', () => {
    // 0.1 + 0.2 !== 0.3 in IEEE754 — this must still balance.
    expect(paymentsBalanced([{ amount: 0.1 }, { amount: 0.2 }], 0.3)).toBe(true);
  });
});

describe('calcChange', () => {
  it('is cashReceived - total when cash covers the total', () => {
    expect(calcChange(60000, 50000)).toBe(10000);
  });

  it('is zero when cash received exactly matches total', () => {
    expect(calcChange(50000, 50000)).toBe(0);
  });

  it('never goes negative when cash received is short', () => {
    expect(calcChange(40000, 50000)).toBe(0);
  });
});

describe('exceedsCreditLimit', () => {
  it('is false when the customer has no credit limit configured', () => {
    expect(exceedsCreditLimit(90000, 20000, undefined)).toBe(false);
    expect(exceedsCreditLimit(90000, 20000, 0)).toBe(false);
    expect(exceedsCreditLimit(90000, 20000, null)).toBe(false);
  });

  it('is false when balance + cart total stays within the limit', () => {
    expect(exceedsCreditLimit(50000, 40000, 100000)).toBe(false);
  });

  it('is true when balance + cart total exceeds the limit', () => {
    expect(exceedsCreditLimit(80000, 30000, 100000)).toBe(true);
  });

  it('is false at exactly the limit (boundary is not an excess)', () => {
    expect(exceedsCreditLimit(70000, 30000, 100000)).toBe(false);
  });
});
