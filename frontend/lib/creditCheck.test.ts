// Contract US-043 §4, T-1..T-8: credit confirmation rule at checkout.
import { describe, it, expect } from 'vitest';
import { needsCreditConfirmation, creditAmount, creditConfirmationMessage } from './creditCheck';
import { calcTotal, exceedsCreditLimit } from './saleCalc';

describe('needsCreditConfirmation', () => {
  it('T-1: limit 500,000, balance 450,000, credit 60,000, 26 days -> limit', () => {
    expect(needsCreditConfirmation({ balance: 450000, creditLimit: 500000, daysOverdue: 26, creditAmount: 60000 })).toEqual(['limit']);
  });

  it('T-2: limit 500,000, balance 450,000, credit 40,000, 10 days -> none', () => {
    expect(needsCreditConfirmation({ balance: 450000, creditLimit: 500000, daysOverdue: 10, creditAmount: 40000 })).toEqual([]);
  });

  it('T-3: balance + credit equal to the limit (500,000) is not exceeding -> none', () => {
    expect(needsCreditConfirmation({ balance: 450000, creditLimit: 500000, daysOverdue: 10, creditAmount: 50000 })).toEqual([]);
  });

  it('T-4: no limit (null and 0), balance 80,000, 31 days -> overdue; 30 days -> none', () => {
    for (const creditLimit of [null, 0]) {
      expect(needsCreditConfirmation({ balance: 80000, creditLimit, daysOverdue: 31, creditAmount: 20000 })).toEqual(['overdue']);
      expect(needsCreditConfirmation({ balance: 80000, creditLimit, daysOverdue: 30, creditAmount: 20000 })).toEqual([]);
    }
  });

  it('T-5: limit 500,000, balance 480,000, 45 days, credit 30,000 -> limit and overdue, in that order', () => {
    expect(needsCreditConfirmation({ balance: 480000, creditLimit: 500000, daysOverdue: 45, creditAmount: 30000 })).toEqual(['limit', 'overdue']);
  });

  it('T-6: credit amount 0 (cash sale), balance 480,000, 45 days -> none', () => {
    expect(needsCreditConfirmation({ balance: 480000, creditLimit: 500000, daysOverdue: 45, creditAmount: 0 })).toEqual([]);
  });
});

describe('creditAmount (T-7)', () => {
  it('T-7: /pos sale 50,000 + 19% IVA = 59,500 all on credit -> credit amount 59,500 -> limit', () => {
    const total = calcTotal([{ unitPrice: 50000, quantity: 1, discount: 0, taxRate: 0.19 }], 0);
    const amount = creditAmount([{ method: 'credito', amount: total }]);
    expect(amount).toBeCloseTo(59500, 6);
    expect(needsCreditConfirmation({ balance: 450000, creditLimit: 500000, daysOverdue: 26, creditAmount: amount })).toEqual(['limit']);
    // The old check used the pre-tax cartTotal: 450,000 + 50,000 = 500,000, no warning.
    expect(exceedsCreditLimit(450000, 50000, 500000)).toBe(false);
  });

  it('T-7: mixed 60,000 = 40,000 cash + 20,000 credit -> credit amount 20,000 -> none', () => {
    const amount = creditAmount([
      { method: 'efectivo', amount: 40000 },
      { method: 'credito', amount: 20000 },
    ]);
    expect(amount).toBe(20000);
    expect(needsCreditConfirmation({ balance: 450000, creditLimit: 500000, daysOverdue: 26, creditAmount: amount })).toEqual([]);
  });
});

describe('creditConfirmationMessage (T-8)', () => {
  it('T-8: limit reason names the resulting debt and the limit', () => {
    const input = { balance: 450000, creditLimit: 500000, daysOverdue: 26, creditAmount: 60000 };
    expect(creditConfirmationMessage(['limit'], input)).toBe('Con esta venta la deuda queda en $510.000 y el tope es $500.000.');
  });

  it('T-8: overdue reason names the debt age', () => {
    const input = { balance: 80000, creditLimit: null, daysOverdue: 45, creditAmount: 20000 };
    expect(creditConfirmationMessage(['overdue'], input)).toBe('Tiene deuda de hace 45 días.');
  });

  it('T-8: both reasons -> limit sentence first, one space, overdue sentence', () => {
    const input = { balance: 480000, creditLimit: 500000, daysOverdue: 45, creditAmount: 30000 };
    expect(creditConfirmationMessage(['limit', 'overdue'], input)).toBe(
      'Con esta venta la deuda queda en $510.000 y el tope es $500.000. Tiene deuda de hace 45 días.',
    );
  });
});
