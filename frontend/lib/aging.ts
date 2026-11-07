// Debt-age buckets for the aging chips on the Customers list (US-085).
// Pure helpers: every function takes `now` explicitly so it can be tested.
import { formatCOP } from './format';

export type AgingBucket = '0-14' | '15-30' | '>30';

export const AGING_BUCKETS: AgingBucket[] = ['0-14', '15-30', '>30'];

interface AgingCustomer {
  balance: number;
  oldest_debt_at: string | null;
}

const DAY_MS = 86400000;

const BUCKET_NAMES: Record<AgingBucket, string> = {
  '0-14': 'Hasta 14 días',
  '15-30': '15–30 días',
  '>30': 'Más de 30 días',
};

// Whole elapsed 24-hour periods since `iso`; a null date counts as 0.
export function daysOverdue(iso: string | null, now: number): number {
  if (!iso) return 0;
  return Math.floor((now - new Date(iso).getTime()) / DAY_MS);
}

export function agingBucket(days: number): AgingBucket {
  if (days > 30) return '>30';
  if (days >= 15) return '15-30';
  return '0-14';
}

// Sums balances and counts customers per bucket; balances <= 0 are not counted.
export function bucketTotals<T extends AgingCustomer>(
  customers: T[],
  now: number,
): Record<AgingBucket, { total: number; count: number }> {
  const totals: Record<AgingBucket, { total: number; count: number }> = {
    '0-14': { total: 0, count: 0 },
    '15-30': { total: 0, count: 0 },
    '>30': { total: 0, count: 0 },
  };
  for (const c of customers) {
    if (c.balance <= 0) continue;
    const t = totals[agingBucket(daysOverdue(c.oldest_debt_at, now))];
    t.total += c.balance;
    t.count += 1;
  }
  return totals;
}

// Keeps the customers with debt in `bucket`, in input order; a null bucket returns the list unchanged.
export function filterByBucket<T extends AgingCustomer>(
  customers: T[],
  bucket: AgingBucket | null,
  now: number,
): T[] {
  if (bucket === null) return customers;
  return customers.filter(
    (c) => c.balance > 0 && agingBucket(daysOverdue(c.oldest_debt_at, now)) === bucket,
  );
}

// Single selection: tapping the active chip clears it (even at count 0);
// an inactive chip with count 0 is not selectable.
export function toggleBucket(
  current: AgingBucket | null,
  clicked: AgingBucket,
  clickedCount: number,
): AgingBucket | null {
  if (current === clicked) return null;
  if (clickedCount <= 0) return current;
  return clicked;
}

export function bucketLabel(bucket: AgingBucket, total: number, count: number): string {
  return `${BUCKET_NAMES[bucket]}: ${formatCOP(total)} (${count})`;
}
