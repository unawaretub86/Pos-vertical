// US-104 contract section 4, T-4 (AC-5): the close ticket labels a negative
// difference as Faltante with the absolute amount; a surplus keeps Diferencia +$X.
// Rendered with react-dom/server under vitest's node environment.
import { describe, it, expect } from 'vitest';
import { renderToString } from 'react-dom/server';
import { createElement } from 'react';
import CloseTicketPreview, { type CloseTicketData } from '@/components/CloseTicketPreview';

function ticket(actual: number, difference: number): CloseTicketData {
  return {
    business: { name: 'Tienda Dora', address: '', nit: '', ticket_header: '', ticket_footer: '' },
    cashier: 'cajero1',
    opened_at: '2026-11-13T08:00:00-05:00',
    closed_at: '2026-11-13T18:00:00-05:00',
    opening_amount: 100000,
    by_payment_method: [{ method: 'cash', total: 30000 }],
    cash_abonos: 0,
    cash_in: 0,
    cash_out: 0,
    expected: 130000,
    actual_close: actual,
    difference,
  };
}

describe('CloseTicketPreview difference line (T-4)', () => {
  it('shortage of 5.000 → Faltante $5.000, never $-5.000 nor Diferencia', () => {
    const html = renderToString(createElement(CloseTicketPreview, { ticket: ticket(125000, -5000) }));
    expect(html).toContain('Faltante');
    expect(html).toContain('$5.000');
    expect(html).not.toContain('$-5.000');
    expect(html).not.toContain('Diferencia');
  });

  it('surplus of 1.000 → Diferencia +$1.000', () => {
    const html = renderToString(createElement(CloseTicketPreview, { ticket: ticket(131000, 1000) }));
    expect(html).toContain('Diferencia');
    expect(html).toContain('+$1.000');
    expect(html).not.toContain('Faltante');
  });
});
