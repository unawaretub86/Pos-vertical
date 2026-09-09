import { describe, it, expect } from 'vitest';
import { sumDenominations } from './denominations';

describe('sumDenominations', () => {
  it('is zero when every denomination has count 0', () => {
    expect(sumDenominations({ 100000: 0, 50000: 0, 1000: 0 })).toBe(0);
  });

  it('sums value * count across mixed denominations', () => {
    expect(sumDenominations({ 100000: 2, 50000: 1, 1000: 3 })).toBe(253000);
  });

  it('matches a non-round total ($123.450) with several zero-count rows', () => {
    expect(sumDenominations({
      100000: 1, 50000: 0, 20000: 1, 10000: 0, 5000: 0,
      2000: 1, 1000: 1, 500: 0, 200: 2, 100: 0, 50: 1,
    })).toBe(123450);
  });
});
