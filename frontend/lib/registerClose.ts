// US-104: register close with a faltante (shortage) after a confirmation.
// Pure helpers for the close modal, the close ticket and the register histories.
import { formatCOP, parseInputCOP } from './format';

export type CloseKind = 'faltante' | 'exact' | 'sobrante';

export interface CloseCheck {
  counted: number;
  difference: number;
  kind: CloseKind;
  canSubmit: boolean;
  needsConfirm: boolean;
}

/**
 * closeCheck evaluates the counted amount (as typed, e.g. "125.000") against
 * the expected cash. An empty field cannot be submitted; a shortage can, but
 * only after a confirmation.
 */
export function closeCheck(counted: string, expected: number): CloseCheck {
  const empty = counted === '';
  const countedNum = parseInputCOP(counted);
  const difference = empty ? 0 : countedNum - expected;
  const kind: CloseKind = difference < 0 ? 'faltante' : difference > 0 ? 'sobrante' : 'exact';
  return {
    counted: countedNum,
    difference,
    kind,
    canSubmit: !empty,
    needsConfirm: !empty && difference < 0,
  };
}

/** Warning shown in the close modal while the counted amount is below expected. */
export function faltanteWarning(missing: number): string {
  return `Hay un faltante de ${formatCOP(Math.abs(missing))}. Vuelve a contar; si de verdad falta, puedes cerrar y el faltante quedará registrado.`;
}

/** Message of the confirmation dialog before closing with a shortage. */
export function faltanteConfirmMessage(counted: number, expected: number): string {
  return `Contaste ${formatCOP(counted)} y se esperaban ${formatCOP(expected)}. La caja se cerrará con un faltante de ${formatCOP(expected - counted)} y quedará así en el historial y en el ticket de cierre.`;
}

export type DifferenceTone = 'danger' | 'success' | 'muted';

export interface DifferenceDisplay {
  label: string;
  amount: string;
  tone: DifferenceTone;
}

/**
 * differenceDisplay renders a stored register difference: a shortage is
 * "Faltante" with the absolute amount; zero or a surplus keeps "Diferencia"
 * ("Dif." in the short form) with a leading "+"; null (open register) is "—".
 */
export function differenceDisplay(diff: number | null, opts: { short?: boolean } = {}): DifferenceDisplay {
  const neutral = opts.short ? 'Dif.' : 'Diferencia';
  if (diff == null) return { label: neutral, amount: '—', tone: 'muted' };
  if (diff < 0) return { label: 'Faltante', amount: formatCOP(Math.abs(diff)), tone: 'danger' };
  return { label: neutral, amount: '+' + formatCOP(diff), tone: 'success' };
}

/** CSS color variable for a difference tone. */
export function differenceColor(tone: DifferenceTone): string {
  if (tone === 'danger') return 'var(--danger)';
  if (tone === 'success') return 'var(--success)';
  return 'var(--text-muted)';
}

export interface CloseBreakdownSummary {
  register: { opening_amount: number };
  cash_sales: number;
  cash_abonos: number;
  cash_in: number;
  cash_out: number;
  expected: number;
}

export interface BreakdownRow {
  label: string;
  value: number;
}

/** Close modal breakdown; the signed values add up to the expected cash. */
export function closeBreakdown(summary: CloseBreakdownSummary): BreakdownRow[] {
  return [
    { label: 'Base apertura', value: summary.register.opening_amount },
    { label: 'Ventas efectivo', value: summary.cash_sales },
    { label: 'Abonos en efectivo', value: summary.cash_abonos },
    { label: 'Entradas', value: summary.cash_in },
    { label: 'Salidas manuales', value: -summary.cash_out },
  ];
}

/**
 * singleFlight wraps an async function so that calls made while one is pending
 * are ignored (they resolve to undefined). The guard is released when the
 * pending call settles; a rejection still reaches its caller.
 */
export function singleFlight<A extends unknown[], R>(
  fn: (...args: A) => Promise<R>,
): (...args: A) => Promise<R | undefined> {
  let pending = false;
  return async (...args: A) => {
    if (pending) return undefined;
    pending = true;
    try {
      return await fn(...args);
    } finally {
      pending = false;
    }
  };
}

const CLOSE_ERROR_FALLBACK = 'No se pudo cerrar la caja';

/** Text of the error toast for a failed close. */
export function closeErrorMessage(err: unknown): string {
  if (err instanceof Error && err.message) return err.message;
  return CLOSE_ERROR_FALLBACK;
}
