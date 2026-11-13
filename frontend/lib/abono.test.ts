// US-082 contract section 4, T-13 (AC-15, AC-16).
//
// Helper API assumed from the contract (only showsNoRegisterNotice is named):
//   type AbonoMethod = 'cash' | 'transfer' | 'card'
//   ABONO_METHOD_OPTIONS: { value: AbonoMethod; label: string }[]   // selector order
//   DEFAULT_ABONO_METHOD: AbonoMethod
//   showsNoRegisterNotice(method: AbonoMethod, registerId: number | null): boolean
import { describe, it, expect } from 'vitest';
import {
  ABONO_METHOD_OPTIONS,
  DEFAULT_ABONO_METHOD,
  showsNoRegisterNotice,
  quickAbonoAmounts,
  abonoFieldValue,
  abonoAmountToSend,
  abonoPreview,
  customerBalanceLabel,
} from './abono';

describe('ABONO_METHOD_OPTIONS', () => {
  it('lists Efectivo, Transferencia, Tarjeta mapped to cash, transfer, card', () => {
    expect(ABONO_METHOD_OPTIONS.map((o) => [o.label, o.value])).toEqual([
      ['Efectivo', 'cash'],
      ['Transferencia', 'transfer'],
      ['Tarjeta', 'card'],
    ]);
  });

  it('defaults to cash', () => {
    expect(DEFAULT_ABONO_METHOD).toBe('cash');
  });
});

describe('showsNoRegisterNotice', () => {
  it('cash abono with no register shows the notice', () => {
    expect(showsNoRegisterNotice('cash', null)).toBe(true);
  });

  it('cash abono linked to a register does not show it', () => {
    expect(showsNoRegisterNotice('cash', 3)).toBe(false);
  });

  it('transfer abono with no register does not show it', () => {
    expect(showsNoRegisterNotice('transfer', null)).toBe(false);
  });

  it('card abono with no register does not show it', () => {
    expect(showsNoRegisterNotice('card', null)).toBe(false);
  });
});

// US-084 contract section 4, T-1..T-8 and T-10 (AC-4, AC-5, AC-7, AC-8, AC-9, AC-15, AC-18).
//
// Helper API named by the contract:
//   type QuickAmount = { label: string; amount: number }
//   quickAbonoAmounts(balance: number): QuickAmount[]
//   abonoFieldValue(amount: number): number
//   abonoAmountToSend(balance: number, typed: number): number
//   abonoPreview(balance: number, typed: number): { valid: boolean; line: string | null; error: string | null }
//   customerBalanceLabel(balance: number): { text: string; color: string }

describe('quickAbonoAmounts', () => {
  it('T-1 balance 450,000 shows all four in order, Mitad 225,000', () => {
    expect(quickAbonoAmounts(450000)).toEqual([
      { label: 'Total', amount: 450000 },
      { label: 'Mitad', amount: 225000 },
      { label: '$50.000', amount: 50000 },
      { label: '$20.000', amount: 20000 },
    ]);
  });

  it('T-2 balance 30,000 omits $50.000', () => {
    expect(quickAbonoAmounts(30000)).toEqual([
      { label: 'Total', amount: 30000 },
      { label: 'Mitad', amount: 15000 },
      { label: '$20.000', amount: 20000 },
    ]);
  });

  it('T-3 balance 15,000 shows only Total and Mitad 7,500', () => {
    expect(quickAbonoAmounts(15000)).toEqual([
      { label: 'Total', amount: 15000 },
      { label: 'Mitad', amount: 7500 },
    ]);
  });

  it('T-3 balance 50,000 shows four entries with Total and $50.000 both 50,000', () => {
    expect(quickAbonoAmounts(50000)).toEqual([
      { label: 'Total', amount: 50000 },
      { label: 'Mitad', amount: 25000 },
      { label: '$50.000', amount: 50000 },
      { label: '$20.000', amount: 20000 },
    ]);
  });

  it('T-3 balance 45,001 floors Mitad to 22,500', () => {
    const mitad = quickAbonoAmounts(45001).find((q) => q.label === 'Mitad');
    expect(mitad).toEqual({ label: 'Mitad', amount: 22500 });
  });

  it('T-4 balance 11,901.19 keeps the exact Total and floors Mitad to 5,950', () => {
    const quick = quickAbonoAmounts(11901.19);
    expect(quick).toEqual([
      { label: 'Total', amount: 11901.19 },
      { label: 'Mitad', amount: 5950 },
    ]);
    expect(quick[0].amount).toBe(11901.19);
  });

  it('T-4 balance 1 shows Total only', () => {
    expect(quickAbonoAmounts(1)).toEqual([{ label: 'Total', amount: 1 }]);
  });

  it('T-4 residue 0.19 shows Total only with the exact amount (Q-7)', () => {
    expect(quickAbonoAmounts(0.19)).toEqual([{ label: 'Total', amount: 0.19 }]);
  });

  it('T-4 balance 0 and negative balance show no quick amounts', () => {
    expect(quickAbonoAmounts(0)).toEqual([]);
    expect(quickAbonoAmounts(-50000)).toEqual([]);
  });
});

describe('abonoPreview', () => {
  it('T-5 balance 450,000 and amount 50,000 shows the new balance 400,000', () => {
    expect(abonoPreview(450000, 50000)).toEqual({
      valid: true,
      line: 'Saldo anterior: $450.000 → Nuevo saldo: $400.000',
      error: null,
    });
  });

  it('T-5 amount equal to the balance ends with Nuevo saldo: $0', () => {
    const p = abonoPreview(450000, 450000);
    expect(p.valid).toBe(true);
    expect(p.error).toBeNull();
    expect(p.line).toBe('Saldo anterior: $450.000 → Nuevo saldo: $0');
    expect(p.line?.endsWith('Nuevo saldo: $0')).toBe(true);
  });

  it('T-5 residue 0.19 with field value 1 is valid and settles to $0 (Q-7)', () => {
    expect(abonoPreview(0.19, 1)).toEqual({
      valid: true,
      line: 'Saldo anterior: $0 → Nuevo saldo: $0',
      error: null,
    });
  });

  it('T-6 amount above the balance is invalid with the over-balance message and no line', () => {
    expect(abonoPreview(30000, 50000)).toEqual({
      valid: false,
      line: null,
      error: 'El monto no puede superar $30.000',
    });
  });

  it('T-6 empty field (0) is invalid with no line and no message', () => {
    expect(abonoPreview(30000, 0)).toEqual({ valid: false, line: null, error: null });
  });
});

describe('abonoAmountToSend', () => {
  it('T-7 field value of the balance maps back to the exact balance', () => {
    expect(abonoAmountToSend(11901.19, 11901)).toBe(11901.19);
    expect(abonoAmountToSend(11901.6, 11902)).toBe(11901.6);
    expect(abonoAmountToSend(0.19, 1)).toBe(0.19);
  });

  it('T-7 any other typed amount is sent as typed', () => {
    expect(abonoAmountToSend(450000, 50000)).toBe(50000);
    expect(abonoAmountToSend(11901.19, 11902)).toBe(11902);
  });

  it('T-7 empty field (0) is never mapped to the balance', () => {
    expect(abonoAmountToSend(0.19, 0)).toBe(0);
  });

  it('T-7 preview uses the mapped amount: 11,901.6 with field 11,902 settles to $0', () => {
    const p = abonoPreview(11901.6, 11902);
    expect(p.valid).toBe(true);
    expect(p.error).toBeNull();
    expect(p.line).toBe('Saldo anterior: $11.902 → Nuevo saldo: $0');
  });

  it('T-7 preview: 11,901.19 with field 11,902 is above the balance and invalid', () => {
    const p = abonoPreview(11901.19, 11902);
    expect(p.valid).toBe(false);
    expect(p.line).toBeNull();
  });

  it('T-7 preview: residue 0.19 with empty field is invalid with no line and no message', () => {
    expect(abonoPreview(0.19, 0)).toEqual({ valid: false, line: null, error: null });
  });
});

describe('customerBalanceLabel', () => {
  it('T-8 negative balance shows Saldo a favor in success color', () => {
    expect(customerBalanceLabel(-50000)).toEqual({
      text: 'Saldo a favor: $50.000',
      color: 'var(--success)',
    });
  });

  it('T-8 zero balance shows Al día in success color', () => {
    expect(customerBalanceLabel(0)).toEqual({ text: 'Al día', color: 'var(--success)' });
  });

  it('T-8 positive balance shows the amount in danger color', () => {
    expect(customerBalanceLabel(400000)).toEqual({ text: '$400.000', color: 'var(--danger)' });
  });
});

describe('abonoFieldValue', () => {
  it('T-10 rounds to whole pesos', () => {
    expect(abonoFieldValue(225000)).toBe(225000);
    expect(abonoFieldValue(11901.19)).toBe(11901);
    expect(abonoFieldValue(11901.6)).toBe(11902);
  });

  it('T-10 a positive amount below one peso gives at least 1 (Q-8)', () => {
    expect(abonoFieldValue(0.19)).toBe(1);
    expect(abonoFieldValue(0.5)).toBe(1);
  });

  it('T-10 zero gives 0', () => {
    expect(abonoFieldValue(0)).toBe(0);
  });
});

// US-102 (R-133) T-20: a $10.001 sale at 19% on credit now carries a whole
// credit line of 11901, so the abono of its displayed amount settles it.
describe('whole-peso fiado balance (US-102 T-20)', () => {
  it('T-20 abonoFieldValue(11901) is 11901', () => {
    expect(abonoFieldValue(11901)).toBe(11901);
  });

  it('T-20 abonoAmountToSend(11901, 11901) is 11901 and the preview ends at $0', () => {
    expect(abonoAmountToSend(11901, 11901)).toBe(11901);
    expect(abonoPreview(11901, 11901)).toEqual({
      valid: true,
      line: 'Saldo anterior: $11.901 → Nuevo saldo: $0',
      error: null,
    });
  });

  it('T-20 a settled balance of 0 reads Al día', () => {
    expect(customerBalanceLabel(0)).toEqual({ text: 'Al día', color: 'var(--success)' });
  });
});
