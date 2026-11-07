// Customer context shown in the payment panel at checkout (US-043), from
// GET /customers/{id}/pos-summary. Pure helpers, unit tested without React.
import { formatCOP } from './format';

export interface PosSummaryTopProduct {
  product_id: number;
  name: string;
  times_bought: number;
}

export interface PosSummary {
  /** Zone-less UTC string "YYYY-MM-DD HH:MM:SS" (SQLite CURRENT_TIMESTAMP). */
  last_visit_at: string | null;
  top_products: PosSummaryTopProduct[];
  balance: number;
  days_overdue: number | null;
  credit_limit: number | null;
}

const SHOP_TIME_ZONE = 'America/Bogota';

const dayFormatter = new Intl.DateTimeFormat('en-CA', {
  timeZone: SHOP_TIME_ZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
});

/** Calendar day of `d` in Bogotá, as a UTC-midnight timestamp for day arithmetic. */
function bogotaDay(d: Date): number {
  const parts = dayFormatter.formatToParts(d);
  const get = (type: string) => Number(parts.find(p => p.type === type)?.value);
  return Date.UTC(get('year'), get('month') - 1, get('day'));
}

/** Parses the API's zone-less UTC timestamp; a naive `new Date()` would read it as local time. */
export function parseUTCTimestamp(s: string): Date {
  return new Date(s.replace(' ', 'T') + 'Z');
}

/** "Última visita: hoy / hace 1 día / hace N días", counting Bogotá calendar days. */
export function lastVisitLabel(lastVisitAt: string | null, now: Date): string | null {
  if (!lastVisitAt) return null;
  const visit = parseUTCTimestamp(lastVisitAt);
  if (Number.isNaN(visit.getTime())) return null;
  const days = Math.max(0, Math.round((bogotaDay(now) - bogotaDay(visit)) / 86_400_000));
  if (days === 0) return 'Última visita: hoy';
  return `Última visita: hace ${days} ${days === 1 ? 'día' : 'días'}`;
}

/** Shown when the customer has never bought anything. */
export function newCustomerLabel(summary: PosSummary): string | null {
  if (summary.last_visit_at === null && summary.top_products.length === 0) return 'Cliente nuevo — sin historial';
  return null;
}

/** "Lo de siempre: A · B · C" in server order, or null without history. */
export function usualProductsLabel(topProducts: PosSummaryTopProduct[]): string | null {
  if (topProducts.length === 0) return null;
  return `Lo de siempre: ${topProducts.map(p => p.name).join(' · ')}`;
}

/** Aging color band, same thresholds as the Customers list: < 7 muted, 7–14 warning, > 14 danger. */
function agingColor(days: number | null): string {
  if (days === null || days < 7) return 'var(--text-muted)';
  if (days <= 14) return 'var(--warning)';
  return 'var(--danger)';
}

/** Debt badge "Debe: $X (N días)", or null when the customer owes nothing. */
export function debtBadge(balance: number, daysOverdue: number | null): { text: string; color: string } | null {
  if (balance <= 0) return null;
  const age = daysOverdue === null ? '' : ` (${daysOverdue} ${daysOverdue === 1 ? 'día' : 'días'})`;
  return { text: `Debe: ${formatCOP(balance)}${age}`, color: agingColor(daysOverdue) };
}

/** Shown when the current debt is already over the limit (0/null = no limit). */
export function limitExceededLabel(balance: number, creditLimit: number | null): string | null {
  if (!creditLimit || balance <= creditLimit) return null;
  return `Superó tope de crédito (${formatCOP(creditLimit)})`;
}

export interface UsualProductState {
  type: 'product' | 'service';
  stock: number;
  active: boolean;
}

/**
 * Whether one more unit of a "lo de siempre" product can be added, mirroring
 * addToCart's stock rule. `product` is null when its request failed.
 */
export function decideUsualAdd(
  name: string,
  product: UsualProductState | null,
  inCart: number,
): { add: true } | { add: false; toast: string } {
  if (!product || !product.active) return { add: false, toast: `${name} no está disponible — no se agregó` };
  if (product.type === 'product' && inCart + 1 > product.stock) return { add: false, toast: `${name} no tiene stock — no se agregó` };
  return { add: true };
}
