// US-107 T-6 (AC-10): formatCOP puts the sign before the currency symbol and
// prints anything that rounds to zero as $0 (R-191).
import { describe, it, expect } from 'vitest';
import { formatCOP } from './format';

describe('formatCOP (R-191)', () => {
  it('negative amounts: -8000 -> -$8.000, -1234567 -> -$1.234.567', () => {
    expect(formatCOP(-8000)).toBe('-$8.000');
    expect(formatCOP(-1234567)).toBe('-$1.234.567');
  });

  it('positive amounts unchanged: 8000 -> $8.000, 1234567 -> $1.234.567, 999 -> $999', () => {
    expect(formatCOP(8000)).toBe('$8.000');
    expect(formatCOP(1234567)).toBe('$1.234.567');
    expect(formatCOP(999)).toBe('$999');
  });

  it('anything that rounds to zero prints $0: 0, -0, -0.4', () => {
    expect(formatCOP(0)).toBe('$0');
    expect(formatCOP(-0)).toBe('$0');
    expect(formatCOP(-0.4)).toBe('$0');
  });

  it('rounding unchanged: 1500.5 -> $1.501', () => {
    expect(formatCOP(1500.5)).toBe('$1.501');
  });
});
