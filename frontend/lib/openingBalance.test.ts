// US-092 contract section 4, T-14 (AC-14, AC-15) and T-15 (AC-16).
//
// Helper API assumed from the contract (frontend/lib/openingBalance.ts):
//   parseAmountInput(raw: string): number | null
//   isValidDebtDate(date: string, today: string): boolean     // both YYYY-MM-DD
//   openingBalanceLabel(o: { amount: number; debt_date: string }): string
// T-15 names no helper; assumed (the audit page imports it):
//   describeOpeningBalanceAudit(action: string, entityId: number, details: string): string
//   where details is the raw audit_log.details JSON string.
import { describe, it, expect } from 'vitest';
import {
  parseAmountInput,
  isValidDebtDate,
  openingBalanceLabel,
  describeOpeningBalanceAudit,
} from './openingBalance';

describe('T-14 parseAmountInput', () => {
  it('reads "150.000" as 150000', () => {
    expect(parseAmountInput('150.000')).toBe(150000);
  });

  it('returns null for an empty field', () => {
    expect(parseAmountInput('')).toBeNull();
  });

  it('returns null for "0"', () => {
    expect(parseAmountInput('0')).toBeNull();
  });
});

describe('T-14 isValidDebtDate', () => {
  const today = '2026-11-09';

  it('accepts a past date', () => {
    expect(isValidDebtDate('2026-09-25', today)).toBe(true);
  });

  it('accepts today', () => {
    expect(isValidDebtDate('2026-11-09', today)).toBe(true);
  });

  it('rejects tomorrow', () => {
    expect(isValidDebtDate('2026-11-10', today)).toBe(false);
  });

  it('rejects a date that does not exist', () => {
    expect(isValidDebtDate('2026-02-30', today)).toBe(false);
  });
});

describe('T-14 openingBalanceLabel', () => {
  it('formats amount in COP and the date as dd/mm/yyyy', () => {
    expect(openingBalanceLabel({ amount: 150000, debt_date: '2026-09-25' })).toBe('$150.000 desde 25/09/2026');
  });
});

describe('T-15 describeOpeningBalanceAudit', () => {
  it('describes a create row', () => {
    expect(
      describeOpeningBalanceAudit('opening_balance_create', 7, '{"amount":150000,"debt_date":"2026-09-25"}'),
    ).toBe('Cliente #7: $150.000 desde 25/09/2026');
  });

  it('describes an update row', () => {
    expect(
      describeOpeningBalanceAudit(
        'opening_balance_update',
        7,
        '{"from_amount":150000,"to_amount":120000,"from_date":"2026-09-25","to_date":"2026-09-20"}',
      ),
    ).toBe('Cliente #7: de $150.000 (25/09/2026) a $120.000 (20/09/2026)');
  });
});
