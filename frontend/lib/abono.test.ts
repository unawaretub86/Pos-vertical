// US-082 contract section 4, T-13 (AC-15, AC-16).
//
// Helper API assumed from the contract (only showsNoRegisterNotice is named):
//   type AbonoMethod = 'cash' | 'transfer' | 'card'
//   ABONO_METHOD_OPTIONS: { value: AbonoMethod; label: string }[]   // selector order
//   DEFAULT_ABONO_METHOD: AbonoMethod
//   showsNoRegisterNotice(method: AbonoMethod, registerId: number | null): boolean
import { describe, it, expect } from 'vitest';
import { ABONO_METHOD_OPTIONS, DEFAULT_ABONO_METHOD, showsNoRegisterNotice } from './abono';

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
