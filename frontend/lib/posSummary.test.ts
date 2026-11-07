// Contract US-043 §4, T-9..T-13: POS summary panel helpers.
import { describe, it, expect } from 'vitest';
import {
  debtBadge,
  lastVisitLabel,
  newCustomerLabel,
  usualProductsLabel,
  limitExceededLabel,
  decideUsualAdd,
  type PosSummary,
} from './posSummary';

describe('debtBadge (T-9)', () => {
  it('T-9: badge text', () => {
    expect(debtBadge(450000, 26)?.text).toBe('Debe: $450.000 (26 días)');
    expect(debtBadge(80000, 1)?.text).toBe('Debe: $80.000 (1 día)');
    expect(debtBadge(80000, null)?.text).toBe('Debe: $80.000');
  });

  it('T-9: no badge when balance <= 0', () => {
    expect(debtBadge(0, null)).toBeNull();
    expect(debtBadge(0, 26)).toBeNull();
    expect(debtBadge(-20000, null)).toBeNull();
  });

  it('T-9: color band 6 muted, 7 and 14 warning, 15 danger', () => {
    expect(debtBadge(80000, 6)?.color).toBe('var(--text-muted)');
    expect(debtBadge(80000, 7)?.color).toBe('var(--warning)');
    expect(debtBadge(80000, 14)?.color).toBe('var(--warning)');
    expect(debtBadge(80000, 15)?.color).toBe('var(--danger)');
  });
});

describe('lastVisitLabel (T-10)', () => {
  // 2026-11-07 08:00 in Bogotá (UTC-5).
  const now = new Date('2026-11-07T13:00:00Z');

  it('T-10: last_visit_at is read as UTC and counted in Bogotá calendar days', () => {
    // 2026-11-07 01:00 UTC = 06/11 20:00 Bogotá -> yesterday, not today.
    expect(lastVisitLabel('2026-11-07 01:00:00', now)).toBe('Última visita: hace 1 día');
    // 2026-11-07 12:00 UTC = 07/11 07:00 Bogotá.
    expect(lastVisitLabel('2026-11-07 12:00:00', now)).toBe('Última visita: hoy');
    // 2026-11-04 05:15:55 UTC = 04/11 00:15 Bogotá.
    expect(lastVisitLabel('2026-11-04 05:15:55', now)).toBe('Última visita: hace 3 días');
  });

  it('T-10: Bogotá midnight edge — 05:00:00 UTC is today, 04:59:59 UTC is yesterday', () => {
    expect(lastVisitLabel('2026-11-07 05:00:00', now)).toBe('Última visita: hoy');
    expect(lastVisitLabel('2026-11-07 04:59:59', now)).toBe('Última visita: hace 1 día');
  });

  it('T-10: uses Bogotá days, not UTC days (now 07/11 22:00 Bogotá = 08/11 03:00 UTC)', () => {
    const lateEvening = new Date('2026-11-08T03:00:00Z');
    // 2026-11-07 14:00 UTC = 07/11 09:00 Bogotá -> same Bogotá day.
    expect(lastVisitLabel('2026-11-07 14:00:00', lateEvening)).toBe('Última visita: hoy');
  });

  it('T-10: null -> null', () => {
    expect(lastVisitLabel(null, now)).toBeNull();
  });
});

describe('newCustomerLabel / usualProductsLabel (T-11)', () => {
  const empty: PosSummary = { last_visit_at: null, top_products: [], balance: 0, days_overdue: null, credit_limit: null };

  it('T-11: no history -> "Cliente nuevo — sin historial" and no "Lo de siempre"', () => {
    expect(newCustomerLabel(empty)).toBe('Cliente nuevo — sin historial');
    expect(usualProductsLabel(empty.top_products)).toBeNull();
  });

  it('T-11: two products in server order joined with " · "', () => {
    const summary: PosSummary = {
      ...empty,
      last_visit_at: '2026-11-04 05:15:55',
      top_products: [
        { product_id: 1, name: 'Arroz', times_bought: 3 },
        { product_id: 2, name: 'Aceite', times_bought: 2 },
      ],
    };
    expect(usualProductsLabel(summary.top_products)).toBe('Lo de siempre: Arroz · Aceite');
    expect(newCustomerLabel(summary)).toBeNull();
  });
});

describe('limitExceededLabel (T-12)', () => {
  it('T-12: limit 150,000, balance 180,000 -> exceeded text', () => {
    expect(limitExceededLabel(180000, 150000)).toBe('Superó tope de crédito ($150.000)');
  });

  it('T-12: balance equal to the limit -> none', () => {
    expect(limitExceededLabel(150000, 150000)).toBeNull();
  });

  it('T-12: no limit (null) -> none', () => {
    expect(limitExceededLabel(180000, null)).toBeNull();
  });
});

describe('decideUsualAdd (T-13)', () => {
  it('T-13: product with stock 0 -> skipped with stock toast', () => {
    expect(decideUsualAdd('Arroz', { type: 'product', stock: 0, active: true }, 0)).toEqual({
      add: false,
      toast: 'Arroz no tiene stock — no se agregó',
    });
  });

  it('T-13: stock 5 with 4 already in cart -> added', () => {
    expect(decideUsualAdd('Leche', { type: 'product', stock: 5, active: true }, 4)).toEqual({ add: true });
  });

  it('T-13: stock 5 with 5 already in cart -> skipped with stock toast', () => {
    expect(decideUsualAdd('Leche', { type: 'product', stock: 5, active: true }, 5)).toEqual({
      add: false,
      toast: 'Leche no tiene stock — no se agregó',
    });
  });

  it('T-13: inactive product -> skipped with unavailable toast', () => {
    expect(decideUsualAdd('Aceite', { type: 'product', stock: 10, active: false }, 0)).toEqual({
      add: false,
      toast: 'Aceite no está disponible — no se agregó',
    });
  });

  it('T-13: failed product request (null) -> skipped with unavailable toast', () => {
    expect(decideUsualAdd('Arroz', null, 0)).toEqual({
      add: false,
      toast: 'Arroz no está disponible — no se agregó',
    });
  });

  it('T-13: service-type product with stock 0 -> added', () => {
    expect(decideUsualAdd('Domicilio', { type: 'service', stock: 0, active: true }, 0)).toEqual({ add: true });
  });
});
