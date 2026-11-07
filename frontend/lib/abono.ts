// Pure helpers for the abono (partial payment toward a fiado balance) form
// (US-082). Framework-free so they can be unit tested without React/DOM.

import { formatCOP } from './format';

export type AbonoMethod = 'cash' | 'transfer' | 'card';

/** Payment methods offered in the abono form, in selector order. */
export const ABONO_METHOD_OPTIONS: { value: AbonoMethod; label: string }[] = [
  { value: 'cash', label: 'Efectivo' },
  { value: 'transfer', label: 'Transferencia' },
  { value: 'card', label: 'Tarjeta' },
];

export const DEFAULT_ABONO_METHOD: AbonoMethod = 'cash';

/**
 * True when a saved abono is cash but no register received it, so it will not
 * appear in any register close and the user must be told.
 */
export function showsNoRegisterNotice(method: AbonoMethod, registerId: number | null): boolean {
  return method === 'cash' && registerId === null;
}

// ─── Abono modal (US-084) ─────────────────────────────────────────────────────

export type QuickAmount = { label: string; amount: number };

/**
 * Quick-amount buttons for a balance, in display order. `Total` carries the
 * exact balance (cents included) and shows whenever the balance is positive;
 * `Mitad` is floor(balance / 2); the fixed amounts show only when 1 ≤ amount ≤ balance.
 */
export function quickAbonoAmounts(balance: number): QuickAmount[] {
  if (!(balance > 0)) return [];
  const result: QuickAmount[] = [{ label: 'Total', amount: balance }];
  const candidates: QuickAmount[] = [
    { label: 'Mitad', amount: Math.floor(balance / 2) },
    { label: formatCOP(50000), amount: 50000 },
    { label: formatCOP(20000), amount: 20000 },
  ];
  for (const q of candidates) {
    if (q.amount >= 1 && q.amount <= balance) result.push(q);
  }
  return result;
}

/** Whole-peso value a quick amount writes into the digits-only field (at least 1 when positive). */
export function abonoFieldValue(amount: number): number {
  return amount > 0 ? Math.max(1, Math.round(amount)) : 0;
}

/**
 * Amount actually sent: when the typed value equals the balance's field value
 * (what `Total` fills), the exact balance is sent so the debt reaches 0.
 */
export function abonoAmountToSend(balance: number, typed: number): number {
  return typed > 0 && typed === abonoFieldValue(balance) ? balance : typed;
}

/** Live preview and validation for the abono modal. */
export function abonoPreview(
  balance: number,
  typed: number,
): { valid: boolean; line: string | null; error: string | null } {
  const toSend = abonoAmountToSend(balance, typed);
  if (!(toSend > 0)) return { valid: false, line: null, error: null };
  if (toSend > balance) {
    return { valid: false, line: null, error: `El monto no puede superar ${formatCOP(balance)}` };
  }
  const next = toSend === balance ? 0 : balance - toSend;
  return {
    valid: true,
    line: `Saldo anterior: ${formatCOP(balance)} → Nuevo saldo: ${formatCOP(next)}`,
    error: null,
  };
}

/** Balance text and color for the Customers list and the detail card. */
export function customerBalanceLabel(balance: number): { text: string; color: string } {
  if (balance > 0) return { text: formatCOP(balance), color: 'var(--danger)' };
  if (balance < 0) return { text: `Saldo a favor: ${formatCOP(-balance)}`, color: 'var(--success)' };
  return { text: 'Al día', color: 'var(--success)' };
}
