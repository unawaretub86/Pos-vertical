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

  // US-102 (R-133, AC-10): tolerance 0. Money is whole pesos, so any gap,
  // even a sub-peso one, is unbalanced (was: tolerated below 0.5).
  it('T-19 requires exact equality: a sub-peso gap is not balanced', () => {
    expect(paymentsBalanced([{ amount: 11901 }], 11901.5)).toBe(false);
    expect(paymentsBalanced([{ amount: 11901 }], 11901.19)).toBe(false);
    expect(paymentsBalanced([{ amount: 11901.19 }], 11901)).toBe(false);
    expect(paymentsBalanced([{ amount: 11901 }], 11901)).toBe(true);
  });

  it('T-19 three whole payments that add up to the total balance exactly', () => {
    expect(paymentsBalanced([{ amount: 5000 }, { amount: 6000 }, { amount: 901 }], 11901)).toBe(true);
  });
});

// US-102 (R-133) T-19 / T-22: the whole-peso rule of contract section 3, pinned
// 1:1 with internal/service/sale_money_test.go (T-1 .. T-5, T-22). Half away
// from zero, per line: lineBase = round(qty x unit), lineTax = round(base x rate).
describe('whole-peso rule (shared cases with the Go tests)', () => {
  const line = (unitPrice: number, quantity: number, taxRate: number, discount = 0): CalcItem => ({
    unitPrice,
    quantity,
    discount,
    taxRate,
  });

  const cases: {
    name: string;
    items: CalcItem[];
    globalDiscount: number;
    subtotal: number;
    tax: number;
    total: number;
  }[] = [
    { name: 'T-1 $10.001 at 19%', items: [line(10001, 1, 0.19)], globalDiscount: 0, subtotal: 10001, tax: 1900, total: 11901 },
    { name: 'T-2 3 x $1.001 at 19% (per line)', items: [line(1001, 1, 0.19), line(1001, 1, 0.19), line(1001, 1, 0.19)], globalDiscount: 0, subtotal: 3003, tax: 570, total: 3573 },
    { name: 'T-3 $50 at 5% (2,5 rounds up)', items: [line(50, 1, 0.05)], globalDiscount: 0, subtotal: 50, tax: 3, total: 53 },
    { name: 'T-3 $1.050 at 19% (199,5 rounds up)', items: [line(1050, 1, 0.19)], globalDiscount: 0, subtotal: 1050, tax: 200, total: 1250 },
    { name: 'T-4 0,5 kg at $9.999', items: [line(9999, 0.5, 0)], globalDiscount: 0, subtotal: 5000, tax: 0, total: 5000 },
    { name: 'T-4 0,1 kg at $9.999', items: [line(9999, 0.1, 0)], globalDiscount: 0, subtotal: 1000, tax: 0, total: 1000 },
    { name: 'T-4 0,125 kg at $9.999', items: [line(9999, 0.125, 0)], globalDiscount: 0, subtotal: 1250, tax: 0, total: 1250 },
    { name: 'T-5 item discount $1 on $10.001 at 19%', items: [line(10001, 1, 0.19, 1)], globalDiscount: 0, subtotal: 10000, tax: 1900, total: 11900 },
    { name: 'T-5 global discount $500 does not lower the tax base', items: [line(10001, 1, 0.19)], globalDiscount: 500, subtotal: 10001, tax: 1900, total: 11401 },
    { name: 'T-22 $1.001 at 8%', items: [line(1001, 1, 0.08)], globalDiscount: 0, subtotal: 1001, tax: 80, total: 1081 },
    { name: 'T-22 $1.050 at 8%', items: [line(1050, 1, 0.08)], globalDiscount: 0, subtotal: 1050, tax: 84, total: 1134 },
    { name: 'T-22 $1.001 at 2,5%', items: [line(1001, 1, 0.025)], globalDiscount: 0, subtotal: 1001, tax: 25, total: 1026 },
  ];

  for (const c of cases) {
    it(`${c.name}: subtotal ${c.subtotal}, tax ${c.tax}, total ${c.total}`, () => {
      expect(calcSubtotal(c.items)).toBe(c.subtotal);
      expect(calcTax(c.items)).toBe(c.tax);
      expect(calcTotal(c.items, c.globalDiscount)).toBe(c.total);
    });
  }

  it('T-19 1,005 kg at $1.000 is 1005 (the float 1.005 * 1000 is 1004.9999999999999)', () => {
    expect(calcSubtotal([line(1000, 1.005, 0)])).toBe(1005);
  });

  it('T-19 a quantity that float-multiplies just under a half still rounds half up', () => {
    // 1,005 kg x $10 = 10.05 -> 10 (not a half); 0,5 kg x $1 = 0.5 -> 1.
    expect(calcSubtotal([line(10, 1.005, 0)])).toBe(10);
    expect(calcSubtotal([line(1, 0.5, 0)])).toBe(1);
  });

  it('T-19 per-line sum differs from per-sale rounding: 570, not 571', () => {
    const items = [line(1001, 1, 0.19), line(1001, 1, 0.19), line(1001, 1, 0.19)];
    expect(calcTax(items)).toBe(570);
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
