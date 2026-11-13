// US-104 contract section 4: T-1, T-2, T-3, T-9, T-10, T-11.
// Worked example: opening $100.000 + cash sale $30.000 → expected $130.000;
// counted $125.000 → faltante (shortage) $5.000.
import { describe, it, expect, vi } from 'vitest';
import {
  closeCheck,
  faltanteWarning,
  faltanteConfirmMessage,
  differenceDisplay,
  closeBreakdown,
  singleFlight,
  closeErrorMessage,
} from './registerClose';

describe('closeCheck (T-1: AC-1, AC-8, AC-9)', () => {
  it('counted 125.000 vs 130.000 → faltante, submit enabled, needs confirm', () => {
    const c = closeCheck('125.000', 130000);
    expect(c.difference).toBe(-5000);
    expect(c.kind).toBe('faltante');
    expect(c.canSubmit).toBe(true);
    expect(c.needsConfirm).toBe(true);
  });

  it('counted 130.000 vs 130.000 → exact, no confirm', () => {
    const c = closeCheck('130.000', 130000);
    expect(c.difference).toBe(0);
    expect(c.kind).toBe('exact');
    expect(c.canSubmit).toBe(true);
    expect(c.needsConfirm).toBe(false);
  });

  it('counted 131.000 vs 130.000 → sobrante 1.000, no confirm', () => {
    const c = closeCheck('131.000', 130000);
    expect(c.difference).toBe(1000);
    expect(c.kind).toBe('sobrante');
    expect(c.canSubmit).toBe(true);
    expect(c.needsConfirm).toBe(false);
  });

  it('empty counted → submit disabled, no confirm', () => {
    const c = closeCheck('', 130000);
    expect(c.canSubmit).toBe(false);
    expect(c.needsConfirm).toBe(false);
  });

  it('counted 0 vs 130.000 → faltante 130.000, needs confirm', () => {
    const c = closeCheck('0', 130000);
    expect(c.difference).toBe(-130000);
    expect(c.kind).toBe('faltante');
    expect(c.canSubmit).toBe(true);
    expect(c.needsConfirm).toBe(true);
  });
});

describe('warning and confirmation texts (T-2: AC-1, AC-2)', () => {
  it('warning for a faltante of 5.000', () => {
    expect(faltanteWarning(5000)).toBe(
      'Hay un faltante de $5.000. Vuelve a contar; si de verdad falta, puedes cerrar y el faltante quedará registrado.',
    );
  });

  it('confirm message for counted 125.000 / expected 130.000', () => {
    expect(faltanteConfirmMessage(125000, 130000)).toBe(
      'Contaste $125.000 y se esperaban $130.000. La caja se cerrará con un faltante de $5.000 y quedará así en el historial y en el ticket de cierre.',
    );
  });

  it('large amounts render with dots', () => {
    expect(faltanteWarning(1234567)).toContain('$1.234.567');
    const msg = faltanteConfirmMessage(1000000, 2234567);
    expect(msg).toContain('Contaste $1.000.000');
    expect(msg).toContain('se esperaban $2.234.567');
    expect(msg).toContain('faltante de $1.234.567');
  });
});

describe('differenceDisplay (T-3: AC-5, AC-6, AC-7)', () => {
  const cases: [number | null, { short?: boolean } | undefined, string, string, string][] = [
    [-5000, undefined, 'Faltante', '$5.000', 'danger'],
    [-1234567, undefined, 'Faltante', '$1.234.567', 'danger'],
    [0, undefined, 'Diferencia', '+$0', 'success'],
    [1000, undefined, 'Diferencia', '+$1.000', 'success'],
    [null, undefined, 'Diferencia', '—', 'muted'],
    [-5000, { short: true }, 'Faltante', '$5.000', 'danger'],
    [1000, { short: true }, 'Dif.', '+$1.000', 'success'],
    [null, { short: true }, 'Dif.', '—', 'muted'],
  ];

  it.each(cases)('differenceDisplay(%s, %o) → %s %s %s', (diff, opts, label, amount, tone) => {
    const d = differenceDisplay(diff, opts);
    expect(d.label).toBe(label);
    expect(d.amount).toBe(amount);
    expect(d.tone).toBe(tone);
    expect(d.amount).not.toContain('$-');
  });
});

describe('closeBreakdown (T-9: AC-12)', () => {
  it('lists the five lines in order and they add up to expected', () => {
    const rows = closeBreakdown({
      register: { opening_amount: 100000 },
      cash_sales: 30000,
      cash_abonos: 20000,
      cash_in: 15000,
      cash_out: 8000,
      expected: 157000,
    });
    expect(rows.map(r => r.label)).toEqual([
      'Base apertura',
      'Ventas efectivo',
      'Abonos en efectivo',
      'Entradas',
      'Salidas manuales',
    ]);
    expect(rows.map(r => r.value)).toEqual([100000, 30000, 20000, 15000, -8000]);
    expect(rows.reduce((s, r) => s + r.value, 0)).toBe(157000);
  });
});

describe('singleFlight (T-10: AC-13)', () => {
  it('ignores calls while one is pending and releases after it resolves', async () => {
    let resolve!: () => void;
    const fn = vi.fn(() => new Promise<void>(r => { resolve = r; }));
    const guarded = singleFlight(fn);

    const p1 = guarded();
    const p2 = guarded();
    expect(fn).toHaveBeenCalledTimes(1);
    resolve();
    await p1;
    await p2;

    const p3 = guarded();
    expect(fn).toHaveBeenCalledTimes(2);
    resolve();
    await p3;
  });

  it('releases the guard after a rejection and passes the rejection to the caller', async () => {
    let calls = 0;
    const fn = vi.fn(() => {
      calls++;
      return calls === 1 ? Promise.reject(new Error('la caja ya está cerrada')) : Promise.resolve();
    });
    const guarded = singleFlight(fn);

    await expect(guarded()).rejects.toThrow('la caja ya está cerrada');
    await guarded();
    expect(fn).toHaveBeenCalledTimes(2);
  });

  it('passes the arguments through', async () => {
    const fn = vi.fn((n: number) => Promise.resolve(n));
    const guarded = singleFlight(fn);
    await guarded(125000);
    expect(fn).toHaveBeenCalledWith(125000);
  });
});

describe('closeErrorMessage (T-11: AC-14)', () => {
  it('uses the server message', () => {
    expect(closeErrorMessage(new Error('la caja ya está cerrada'))).toBe('la caja ya está cerrada');
    expect(closeErrorMessage(new Error('error cerrando caja'))).toBe('error cerrando caja');
  });

  it('uses the network message', () => {
    const net = 'No se pudo conectar con el servidor. Revise la red e intente de nuevo.';
    expect(closeErrorMessage(new Error(net))).toBe(net);
  });

  it('falls back on an empty message or a non-Error value', () => {
    expect(closeErrorMessage(new Error(''))).toBe('No se pudo cerrar la caja');
    expect(closeErrorMessage('boom')).toBe('No se pudo cerrar la caja');
    expect(closeErrorMessage(undefined)).toBe('No se pudo cerrar la caja');
  });
});
