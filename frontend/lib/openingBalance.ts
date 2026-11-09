// Pure helpers for the opening fiado balance (deuda anterior, US-092): amount
// parsing, debt date validation and the labels shown in the customer detail
// and the audit page.
import { formatCOP } from './format';

/** Largest opening balance the backend accepts (whole pesos). */
export const MAX_OPENING_BALANCE = 100_000_000;

export interface OpeningBalance {
  id: number;
  customer_id: number;
  amount: number;
  debt_date: string; // YYYY-MM-DD, server-local
  note: string;
  created_by: number;
  created_at: string;
  updated_at?: string;
}

/**
 * Reads the amount typed in the "Monto" field (digits with thousands dots,
 * e.g. "150.000") as whole pesos. Empty or 0 is null.
 */
export function parseAmountInput(raw: string): number | null {
  const digits = raw.replace(/\D/g, '');
  if (!digits) return null;
  const n = parseInt(digits, 10);
  return n > 0 ? n : null;
}

/** Today's local calendar date as YYYY-MM-DD (default and max of the date input). */
export function todayISODate(now: Date = new Date()): string {
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, '0');
  const d = String(now.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

/** True when date is a real YYYY-MM-DD calendar date not after today. */
export function isValidDebtDate(date: string, today: string): boolean {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  if (!m) return false;
  const y = Number(m[1]);
  const mo = Number(m[2]);
  const d = Number(m[3]);
  const probe = new Date(Date.UTC(y, mo - 1, d));
  if (probe.getUTCFullYear() !== y || probe.getUTCMonth() !== mo - 1 || probe.getUTCDate() !== d) {
    return false;
  }
  // Same-layout ISO dates compare correctly as strings.
  return date <= today;
}

/** "2026-09-25" → "25/09/2026". Anything else is returned unchanged. */
export function formatDebtDate(date: string): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  return m ? `${m[3]}/${m[2]}/${m[1]}` : date;
}

/** "$150.000 desde 25/09/2026" */
export function openingBalanceLabel(o: { amount: number; debt_date: string }): string {
  return `${formatCOP(o.amount)} desde ${formatDebtDate(o.debt_date)}`;
}

/** Audit page labels of the two opening balance actions. */
export const OPENING_BALANCE_ACTION_LABELS: Record<string, string> = {
  opening_balance_create: 'Deuda anterior registrada',
  opening_balance_update: 'Deuda anterior corregida',
};

/**
 * Describes an opening balance audit row from its raw details JSON:
 * create → "Cliente #7: $150.000 desde 25/09/2026";
 * update → "Cliente #7: de $150.000 (25/09/2026) a $120.000 (20/09/2026)".
 */
export function describeOpeningBalanceAudit(action: string, entityId: number, details: string): string {
  let parsed: Record<string, unknown> = {};
  try { parsed = JSON.parse(details || '{}'); } catch { /* malformed details: describe with defaults */ }
  const amount = (k: string) => formatCOP(Number(parsed[k] ?? 0));
  const date = (k: string) => formatDebtDate(String(parsed[k] ?? ''));

  if (action === 'opening_balance_update') {
    return `Cliente #${entityId}: de ${amount('from_amount')} (${date('from_date')}) a ${amount('to_amount')} (${date('to_date')})`;
  }
  return `Cliente #${entityId}: ${amount('amount')} desde ${date('debt_date')}`;
}
