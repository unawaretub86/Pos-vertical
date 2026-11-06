// US-080: decides how a ticket is printed. ESC/POS methods go through the
// server; "browser" keeps the window.print() flow.

export type PrintMethod = 'browser' | 'escpos_network' | 'escpos_windows';

export interface PrintFlowDeps {
  post: (path: string, body: unknown) => Promise<unknown>;
  toast: (message: string, type?: 'success' | 'error') => void;
  browserPrint: () => void;
}

export interface PrintOutcome {
  outcome: 'browser' | 'sent' | 'failed';
  keepTicketVisible: boolean;
}

export function shouldAutoPrint(method: string): boolean {
  return method === 'escpos_network' || method === 'escpos_windows';
}

export async function handleTicketPrint(
  method: string,
  saleId: number,
  deps: PrintFlowDeps,
): Promise<PrintOutcome> {
  if (!shouldAutoPrint(method)) {
    deps.browserPrint();
    return { outcome: 'browser', keepTicketVisible: true };
  }
  try {
    await deps.post(`/printer/print-ticket/${saleId}`, {});
    deps.toast('Ticket enviado a la impresora', 'success');
    return { outcome: 'sent', keepTicketVisible: true };
  } catch (err) {
    deps.toast(err instanceof Error ? err.message : 'No se pudo imprimir el ticket', 'error');
    return { outcome: 'failed', keepTicketVisible: true };
  }
}
