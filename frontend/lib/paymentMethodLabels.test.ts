// US-107 T-5 (AC-8): the payment panel labels its methods from one explicit
// map, and the credit method reads "Fiado" (R-185).
import { describe, it, expect } from 'vitest';
import { PAYMENT_METHOD_LABELS } from '@/components/SalePaymentPanel';

describe('PAYMENT_METHOD_LABELS (R-185)', () => {
  it('maps every method key to its Spanish label, credito -> Fiado', () => {
    expect(PAYMENT_METHOD_LABELS).toEqual({
      efectivo: 'Efectivo',
      transferencia: 'Transferencia',
      tarjeta: 'Tarjeta',
      credito: 'Fiado',
    });
  });

  it('never labels a method by capitalizing its key (no "Credito")', () => {
    expect(Object.values(PAYMENT_METHOD_LABELS)).not.toContain('Credito');
  });
});
