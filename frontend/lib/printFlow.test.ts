// US-080 contract section 4, T-17 (AC-11, AC-12, AC-13).
//
// Helper API assumed from the contract (it names only shouldAutoPrint and "a call wrapper"):
//   shouldAutoPrint(method: string): boolean
//   handleTicketPrint(method, saleId, deps): Promise<{ outcome: 'browser' | 'sent' | 'failed'; keepTicketVisible: boolean }>
//   deps = { post(path, body), toast(message, type), browserPrint() }
import { describe, it, expect, vi } from 'vitest';
import { shouldAutoPrint, handleTicketPrint } from './printFlow';

function makeDeps(post = vi.fn().mockResolvedValue({ printed: true })) {
  return { post, toast: vi.fn(), browserPrint: vi.fn() };
}

describe('shouldAutoPrint', () => {
  it('is false for browser and true for both ESC/POS methods', () => {
    expect(shouldAutoPrint('browser')).toBe(false);
    expect(shouldAutoPrint('escpos_network')).toBe(true);
    expect(shouldAutoPrint('escpos_windows')).toBe(true);
  });

  it('is false for an unknown or empty method', () => {
    expect(shouldAutoPrint('')).toBe(false);
    expect(shouldAutoPrint('laser')).toBe(false);
  });
});

describe('handleTicketPrint', () => {
  it('browser method: no API call, falls back to window.print and shows no toast', async () => {
    const deps = makeDeps();
    const res = await handleTicketPrint('browser', 7, deps);
    expect(deps.post).not.toHaveBeenCalled();
    expect(deps.browserPrint).toHaveBeenCalledTimes(1);
    expect(deps.toast).not.toHaveBeenCalled();
    expect(res.outcome).toBe('browser');
  });

  it('escpos_network: exactly one POST to the print endpoint, success toast, ticket stays visible', async () => {
    const deps = makeDeps();
    const res = await handleTicketPrint('escpos_network', 7, deps);
    expect(deps.post).toHaveBeenCalledTimes(1);
    expect(deps.post.mock.calls[0][0]).toBe('/printer/print-ticket/7');
    expect(deps.browserPrint).not.toHaveBeenCalled();
    expect(deps.toast).toHaveBeenCalledTimes(1);
    expect(deps.toast).toHaveBeenCalledWith('Ticket enviado a la impresora', 'success');
    expect(res.outcome).toBe('sent');
    expect(res.keepTicketVisible).toBe(true);
  });

  it('API rejects: error toast carries the server message, ticket stays visible, no browser dialog', async () => {
    const deps = makeDeps(vi.fn().mockRejectedValue(new Error('No se pudo conectar con la impresora')));
    const res = await handleTicketPrint('escpos_network', 7, deps);
    expect(deps.post).toHaveBeenCalledTimes(1);
    expect(deps.toast).toHaveBeenCalledTimes(1);
    expect(deps.toast).toHaveBeenCalledWith('No se pudo conectar con la impresora', 'error');
    expect(deps.browserPrint).not.toHaveBeenCalled();
    expect(res.outcome).toBe('failed');
    expect(res.keepTicketVisible).toBe(true);
  });

  it('escpos_windows button click: one POST, no window.print', async () => {
    const deps = makeDeps();
    await handleTicketPrint('escpos_windows', 12, deps);
    expect(deps.post).toHaveBeenCalledTimes(1);
    expect(deps.post.mock.calls[0][0]).toBe('/printer/print-ticket/12');
    expect(deps.browserPrint).not.toHaveBeenCalled();
  });
});
