// Contract US-085 §4, T-1..T-8: aging chips (debt-age buckets) on the Customers list.
import { describe, it, expect } from 'vitest';
import { daysOverdue, agingBucket, bucketTotals, filterByBucket, toggleBucket, bucketLabel } from './aging';
import { formatCOP } from './format';

// Fixed clock: 07/11/2026 08:00 Bogotá.
const now = Date.parse('2026-11-07T13:00:00Z');
const DAY = 86_400_000;

function ago(ms: number): string {
  return new Date(now - ms).toISOString();
}

function cust(name: string, balance: number, oldest: string | null) {
  return { name, balance, oldest_debt_at: oldest };
}

const ana = cust('Ana', 75000, ago(5 * DAY));
const beto = cust('Beto', 300000, ago(20 * DAY));
const caro = cust('Caro', 330000, ago(28 * DAY));
const fito = cust('Fito', 0, ago(40 * DAY));
const gus = cust('Gus', -20000, ago(45 * DAY));
const hugo = cust('Hugo', 0.19, ago(3 * DAY));

const t1Customers = [ana, beto, caro];
const t2Customers = [ana, beto, caro, fito, gus, hugo];

describe('bucketTotals', () => {
  it('T-1: Ana 75,000 @5d, Beto 300,000 @20d, Caro 330,000 @28d -> 75,000/1, 630,000/2, 0/0', () => {
    const t = bucketTotals(t1Customers, now);
    expect(t['0-14']).toEqual({ total: 75000, count: 1 });
    expect(t['15-30']).toEqual({ total: 630000, count: 2 });
    expect(t['>30']).toEqual({ total: 0, count: 0 });
    expect(formatCOP(t['0-14'].total)).toBe('$75.000');
    expect(formatCOP(t['15-30'].total)).toBe('$630.000');
    expect(formatCOP(t['>30'].total)).toBe('$0');
  });

  it('T-2: balance 0 @40d and -20,000 @45d are not counted; 0.19 @3d counts -> first bucket 75,000.19/2', () => {
    const t = bucketTotals(t2Customers, now);
    expect(t['>30']).toEqual({ total: 0, count: 0 });
    expect(t['15-30']).toEqual({ total: 630000, count: 2 });
    expect(t['0-14'].count).toBe(2);
    expect(t['0-14'].total).toBeCloseTo(75000.19, 6);
  });

  it('T-3: $10,000 each at 0, 14, 15d-1s, 15, 30, 31 days -> 30,000/3, 20,000/2, 10,000/1; null date -> first bucket', () => {
    const customers = [
      cust('D0', 10000, ago(0)),
      cust('D14', 10000, ago(14 * DAY)),
      cust('D15m1s', 10000, ago(15 * DAY - 1000)),
      cust('D15', 10000, ago(15 * DAY)),
      cust('D30', 10000, ago(30 * DAY)),
      cust('D31', 10000, ago(31 * DAY)),
    ];
    const t = bucketTotals(customers, now);
    expect(t['0-14']).toEqual({ total: 30000, count: 3 });
    expect(t['15-30']).toEqual({ total: 20000, count: 2 });
    expect(t['>30']).toEqual({ total: 10000, count: 1 });

    const withNull = bucketTotals([cust('NoDate', 5000, null)], now);
    expect(withNull['0-14']).toEqual({ total: 5000, count: 1 });
    expect(withNull['15-30']).toEqual({ total: 0, count: 0 });
    expect(withNull['>30']).toEqual({ total: 0, count: 0 });
  });

  it('T-3: agingBucket boundaries 0, 14 -> 0-14; 15, 30 -> 15-30; 31 -> >30', () => {
    expect(agingBucket(0)).toBe('0-14');
    expect(agingBucket(14)).toBe('0-14');
    expect(agingBucket(15)).toBe('15-30');
    expect(agingBucket(30)).toBe('15-30');
    expect(agingBucket(31)).toBe('>30');
  });
});

describe('daysOverdue', () => {
  it('T-4: 1h ago -> 0; 2026-11-07T01:00:00Z -> 0; 14d -> 14; 31d -> 31; null -> 0', () => {
    expect(daysOverdue(ago(3_600_000), now)).toBe(0);
    expect(daysOverdue('2026-11-07T01:00:00Z', now)).toBe(0);
    expect(daysOverdue(ago(14 * DAY), now)).toBe(14);
    expect(daysOverdue(ago(31 * DAY), now)).toBe(31);
    expect(daysOverdue(null, now)).toBe(0);
  });
});

describe('filterByBucket', () => {
  it('T-5: over T-2 customers: 15-30 -> [Beto, Caro]; >30 -> []; null -> all 6; 0-14 -> [Ana, Hugo]', () => {
    expect(filterByBucket(t2Customers, '15-30', now).map((c) => c.name)).toEqual(['Beto', 'Caro']);
    expect(filterByBucket(t2Customers, '>30', now)).toEqual([]);
    expect(filterByBucket(t2Customers, null, now)).toEqual(t2Customers);
    expect(filterByBucket(t2Customers, '0-14', now).map((c) => c.name)).toEqual(['Ana', 'Hugo']);
  });
});

describe('toggleBucket', () => {
  it('T-6: select, clear, switch; inactive zero-count chip ignored; active zero-count chip still clears', () => {
    expect(toggleBucket(null, '15-30', 2)).toBe('15-30');
    expect(toggleBucket('15-30', '15-30', 2)).toBeNull();
    expect(toggleBucket('15-30', '>30', 1)).toBe('>30');
    expect(toggleBucket(null, '>30', 0)).toBeNull();
    expect(toggleBucket('15-30', '>30', 0)).toBe('15-30');
    expect(toggleBucket('15-30', '15-30', 0)).toBeNull();
  });
});

describe('refresh after abono', () => {
  it('T-7: Dora 450,000 + Beto 80,000 @20d -> 530,000/2; Dora 400,000 -> 480,000/2; Beto 0 -> 400,000/1 and filter -> [Dora]', () => {
    const d20 = ago(20 * DAY);
    let list = [cust('Dora', 450000, d20), cust('Beto', 80000, d20)];
    expect(bucketTotals(list, now)['15-30']).toEqual({ total: 530000, count: 2 });

    list = [cust('Dora', 400000, d20), cust('Beto', 80000, d20)];
    expect(bucketTotals(list, now)['15-30']).toEqual({ total: 480000, count: 2 });

    list = [cust('Dora', 400000, d20), cust('Beto', 0, d20)];
    expect(bucketTotals(list, now)['15-30']).toEqual({ total: 400000, count: 1 });
    expect(filterByBucket(list, '15-30', now).map((c) => c.name)).toEqual(['Dora']);
  });
});

describe('bucketLabel', () => {
  it('T-8: chip labels with formatCOP amount and count (en dash U+2013 in 15–30)', () => {
    expect(bucketLabel('15-30', 630000, 2)).toBe('15–30 días: $630.000 (2)');
    expect(bucketLabel('0-14', 75000, 1)).toBe('Hasta 14 días: $75.000 (1)');
    expect(bucketLabel('>30', 0, 0)).toBe('Más de 30 días: $0 (0)');
  });
});
