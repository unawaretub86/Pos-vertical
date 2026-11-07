// Credit (fiado) confirmation rule at checkout (US-043). Pure and
// framework-free so it can be unit tested without React/DOM. The backend
// never rejects a sale for the limit; this only decides whether `Cobrar`
// asks "¿Fiar de todas formas?" first.
import { formatCOP } from './format';
import { exceedsCreditLimit } from './saleCalc';

export type CreditReason = 'limit' | 'overdue';

export interface CreditCheckInput {
  balance: number;
  creditLimit: number | null;
  daysOverdue: number | null;
  /** Sum of the credit payment lines: what this sale adds to the debt. */
  creditAmount: number;
}

/** Days of debt age above which a credit sale asks for confirmation. */
export const OVERDUE_CONFIRM_DAYS = 30;

/** Sum of the payment lines paid on credit (IVA and discounts already applied). */
export function creditAmount(lines: { method: string; amount: number }[]): number {
  return lines.reduce((sum, l) => (l.method === 'credito' ? sum + l.amount : sum), 0);
}

/**
 * Reasons to confirm a credit sale, limit first; empty when none applies.
 * A credit limit of 0 or null means no limit.
 */
export function needsCreditConfirmation(input: CreditCheckInput): CreditReason[] {
  if (input.creditAmount <= 0) return [];
  const reasons: CreditReason[] = [];
  if (exceedsCreditLimit(input.balance, input.creditAmount, input.creditLimit)) reasons.push('limit');
  if (input.balance > 0 && input.daysOverdue !== null && input.daysOverdue > OVERDUE_CONFIRM_DAYS) reasons.push('overdue');
  return reasons;
}

/** Dialog message naming each reason, in the order given, separated by one space. */
export function creditConfirmationMessage(reasons: CreditReason[], input: CreditCheckInput): string {
  return reasons
    .map(r =>
      r === 'limit'
        ? `Con esta venta la deuda queda en ${formatCOP(input.balance + input.creditAmount)} y el tope es ${formatCOP(input.creditLimit ?? 0)}.`
        : `Tiene deuda de hace ${input.daysOverdue} días.`,
    )
    .join(' ');
}
