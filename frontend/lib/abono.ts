// Pure helpers for the abono (partial payment toward a fiado balance) form
// (US-082). Framework-free so they can be unit tested without React/DOM.

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
