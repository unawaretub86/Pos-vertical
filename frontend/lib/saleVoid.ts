import { api } from './api';
import { formatCOP } from './format';

// Admin void of a completed sale from the dashboard (US-095).

export interface VoidPayment {
  method: string;
  amount: number;
}

export interface VoidableSale {
  id: number;
  total: number;
  register_id?: number | null;
  payments?: VoidPayment[];
}

export interface VoidSummary {
  title: string;
  total: string;
  lines: string[];
  customer: string | null;
}

const PAYMENT_LABELS: Record<string, string> = {
  cash: 'Efectivo',
  transfer: 'Transferencia',
  card: 'Tarjeta',
  credit: 'Fiado',
};

const MSG_CLOSED_REGISTER = 'Esta venta es de una caja ya cerrada; su cierre no cambia.';
const MSG_VOID_FALLBACK = 'No se pudo anular la venta';
const MSG_LOAD_FALLBACK = 'No se pudo cargar la venta';

// Server error codes that are not user text (contract US-095 D-2).
const VOID_ERROR_TEXT: Record<string, string> = {
  sale_has_accepted_invoice:
    'Esta venta tiene un documento electrónico ante la DIAN; para anularla se necesita una nota crédito.',
  forbidden: 'Solo un administrador puede anular ventas.',
};

export function canVoidSale(role: string | null | undefined, status: string): boolean {
  return role === 'admin' && status === 'completed';
}

export function voidSummary(
  sale: { id: number; total: number; payments: VoidPayment[] },
  customerName: string | null,
): VoidSummary {
  return {
    title: `¿Anular la venta #${sale.id}?`,
    total: `Total: ${formatCOP(sale.total)}`,
    lines: sale.payments.map(p => `${PAYMENT_LABELS[p.method] ?? p.method} ${formatCOP(p.amount)}`),
    customer: customerName ? `Cliente: ${customerName}` : null,
  };
}

// Warning for a sale whose register is not the one open now: voiding it does
// not touch the stored close, and any cash handed back is recorded nowhere
// unless the admin enters it by hand (R-112).
export function closedRegisterNotice(
  saleRegisterId: number | null,
  openRegisterId: number | null,
  payments: VoidPayment[],
): string[] {
  if (saleRegisterId !== null && saleRegisterId === openRegisterId) return [];
  const notice = [MSG_CLOSED_REGISTER];
  const cash = payments.filter(p => p.method === 'cash').reduce((sum, p) => sum + p.amount, 0);
  if (cash > 0) {
    notice.push(
      openRegisterId !== null
        ? `Si devuelves ${formatCOP(cash)} en efectivo, regístralo como salida en la caja abierta.`
        : `Si devuelves ${formatCOP(cash)} en efectivo, no quedará registrado en ninguna caja.`,
    );
  }
  return notice;
}

export function voidErrorMessage(raw: string): string {
  if (!raw.trim()) return MSG_VOID_FALLBACK;
  return VOID_ERROR_TEXT[raw] ?? raw;
}

export function saleLoadErrorMessage(raw: string): string {
  return raw.trim() ? raw : MSG_LOAD_FALLBACK;
}

export function voidSale(id: number): Promise<{ id: number; status: string }> {
  return api.delete<{ id: number; status: string }>(`/sales/${id}/void`);
}
