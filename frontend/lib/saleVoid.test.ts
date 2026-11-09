import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  canVoidSale,
  voidSummary,
  closedRegisterNotice,
  voidErrorMessage,
  saleLoadErrorMessage,
} from './saleVoid';

// Contract US-095 §4, T-1..T-5 and T-9 (admin void button for a sale).

const DIAN_TEXT =
  'Esta venta tiene un documento electrónico ante la DIAN; para anularla se necesita una nota crédito.';
const NET_TEXT = 'No se pudo conectar con el servidor. Revise la red e intente de nuevo.';
const BUSY_TEXT = 'El sistema está ocupado, reintenta en un momento';
const CLOSED_TEXT = 'Esta venta es de una caja ya cerrada; su cierre no cambia.';

describe('canVoidSale', () => {
  it('T-1: only an admin on a completed sale', () => {
    expect(canVoidSale('admin', 'completed')).toBe(true);
    expect(canVoidSale('admin', 'voided')).toBe(false);
    expect(canVoidSale('cashier', 'completed')).toBe(false);
    expect(canVoidSale(null, 'completed')).toBe(false);
    expect(canVoidSale(undefined, 'completed')).toBe(false);
  });
});

describe('voidSummary', () => {
  it('T-2: mixed cash + credit sale with a customer', () => {
    const s = voidSummary(
      {
        id: 3,
        total: 12000,
        payments: [
          { method: 'cash', amount: 7000 },
          { method: 'credit', amount: 5000 },
        ],
      },
      'Dora',
    );
    expect(s).toEqual({
      title: '¿Anular la venta #3?',
      total: 'Total: $12.000',
      lines: ['Efectivo $7.000', 'Fiado $5.000'],
      customer: 'Cliente: Dora',
    });
  });

  it('T-2: cash sale with no customer', () => {
    const s = voidSummary({ id: 1, total: 30000, payments: [{ method: 'cash', amount: 30000 }] }, null);
    expect(s.title).toBe('¿Anular la venta #1?');
    expect(s.total).toBe('Total: $30.000');
    expect(s.lines).toEqual(['Efectivo $30.000']);
    expect(s.customer).toBeNull();
  });

  it('T-2: transfer and card labels', () => {
    const s = voidSummary(
      {
        id: 4,
        total: 15000,
        payments: [
          { method: 'transfer', amount: 10000 },
          { method: 'card', amount: 5000 },
        ],
      },
      null,
    );
    expect(s.lines).toEqual(['Transferencia $10.000', 'Tarjeta $5.000']);
  });

  it('T-2: unknown method falls back to the raw key', () => {
    const s = voidSummary({ id: 5, total: 1000, payments: [{ method: 'bono', amount: 1000 }] }, null);
    expect(s.lines).toEqual(['bono $1.000']);
  });

  it('T-2: no line ever says mixed', () => {
    const sales = [
      { id: 3, total: 12000, payments: [{ method: 'cash', amount: 7000 }, { method: 'credit', amount: 5000 }] },
      { id: 4, total: 15000, payments: [{ method: 'transfer', amount: 10000 }, { method: 'card', amount: 5000 }] },
    ];
    for (const sale of sales) {
      for (const line of voidSummary(sale, 'Dora').lines) {
        expect(line).not.toContain('mixed');
      }
    }
  });
});

describe('closedRegisterNotice', () => {
  it('T-3: closed register with a cash portion and a register open', () => {
    expect(
      closedRegisterNotice(1, 2, [
        { method: 'cash', amount: 7000 },
        { method: 'credit', amount: 5000 },
      ]),
    ).toEqual([CLOSED_TEXT, 'Si devuelves $7.000 en efectivo, regístralo como salida en la caja abierta.']);
  });

  it('T-3: closed register with no register open', () => {
    expect(closedRegisterNotice(1, null, [{ method: 'cash', amount: 7000 }])).toEqual([
      CLOSED_TEXT,
      'Si devuelves $7.000 en efectivo, no quedará registrado en ninguna caja.',
    ]);
  });

  it('T-3: closed register, credit only → only the first sentence', () => {
    expect(closedRegisterNotice(1, 2, [{ method: 'credit', amount: 50000 }])).toEqual([CLOSED_TEXT]);
  });

  it('T-3: sale of the open register → nothing', () => {
    expect(closedRegisterNotice(2, 2, [{ method: 'cash', amount: 7000 }])).toEqual([]);
  });

  it('T-3: sale with no register id → both sentences', () => {
    expect(closedRegisterNotice(null, 2, [{ method: 'cash', amount: 30000 }])).toEqual([
      CLOSED_TEXT,
      'Si devuelves $30.000 en efectivo, regístralo como salida en la caja abierta.',
    ]);
  });
});

describe('voidErrorMessage', () => {
  it('T-4: maps server codes to readable text', () => {
    expect(voidErrorMessage('sale_has_accepted_invoice')).toBe(DIAN_TEXT);
    expect(voidErrorMessage('forbidden')).toBe('Solo un administrador puede anular ventas.');
    expect(voidErrorMessage('la venta ya fue anulada')).toBe('la venta ya fue anulada');
    expect(voidErrorMessage('venta no encontrada')).toBe('venta no encontrada');
    expect(voidErrorMessage('')).toBe('No se pudo anular la venta');
  });
});

describe('saleLoadErrorMessage', () => {
  it('T-9: passes the api message through, falls back when blank', () => {
    expect(saleLoadErrorMessage('venta no encontrada')).toBe('venta no encontrada');
    expect(saleLoadErrorMessage(NET_TEXT)).toBe(NET_TEXT);
    expect(saleLoadErrorMessage(BUSY_TEXT)).toBe(BUSY_TEXT);
    expect(saleLoadErrorMessage('')).toBe('No se pudo cargar la venta');
    expect(saleLoadErrorMessage('  ')).toBe('No se pudo cargar la venta');
  });
});

describe('voidSale', () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    vi.resetModules();
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
    vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  const urls = () => fetchMock.mock.calls.map((c) => String(c[0]));

  it('T-5: 200 → one DELETE with no body', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ data: { id: 3, status: 'voided' } }),
    });
    const { voidSale } = await import('./saleVoid');
    const res = await voidSale(3);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toBe('/api/sales/3/void');
    const init = fetchMock.mock.calls[0][1];
    expect(init.method).toBe('DELETE');
    expect(init.body).toBeUndefined();
    expect(res.status).toBe('voided');
  });

  it('T-5: 409 electronic document → rejects with the code, no credit-note call', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 409,
      json: async () => ({
        error: 'sale_has_accepted_invoice',
        message: 'la venta tiene un documento electrónico aceptado; se requiere nota crédito para anular',
        document_id: 'doc-1',
        cude: 'CUDE123',
        requires_credit_note: true,
      }),
    });
    const { voidSale } = await import('./saleVoid');
    await expect(voidSale(3)).rejects.toThrow(new Error('sale_has_accepted_invoice'));
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(urls().some((u) => u.includes('void-with-credit-note'))).toBe(false);
  });

  it('T-5: 403 → rejects with forbidden', async () => {
    fetchMock.mockResolvedValue({ ok: false, status: 403, json: async () => ({ error: 'forbidden' }) });
    const { voidSale } = await import('./saleVoid');
    await expect(voidSale(3)).rejects.toThrow(new Error('forbidden'));
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('T-5: network error → neutral network text after one call', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));
    const { voidSale } = await import('./saleVoid');
    await expect(voidSale(3)).rejects.toThrow(new Error(NET_TEXT));
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
