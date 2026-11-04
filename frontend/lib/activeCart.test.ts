import { describe, it, expect } from 'vitest';
import { readActiveCart, writeActiveCart, ACTIVE_CART_KEY, ACTIVE_CART_MAX_AGE_MS } from './activeCart';

// R-73: the live POS cart survives a session expiry (hard navigation to /login).

function memoryStorage(initial: Record<string, string> = {}) {
  const data = { ...initial };
  return {
    data,
    getItem: (k: string) => (k in data ? data[k] : null),
    setItem: (k: string, v: string) => { data[k] = v; },
    removeItem: (k: string) => { delete data[k]; },
  };
}

const NOW = 1_700_000_000_000;
const item = { product: { id: 7, name: 'Arroz' }, quantity: 2, unitPrice: 3000 };

describe('activeCart', () => {
  it('round-trips items, customer and global discount', () => {
    const s = memoryStorage();
    writeActiveCart(s, { items: [item], customer: { id: 1, name: 'Ana' }, globalDiscountInput: '500' }, NOW);
    const got = readActiveCart<typeof item, { id: number; name: string }>(s, NOW + 60_000);
    expect(got?.items).toEqual([item]);
    expect(got?.customer).toEqual({ id: 1, name: 'Ana' });
    expect(got?.globalDiscountInput).toBe('500');
  });

  it('removes the stored cart when it is empty', () => {
    const s = memoryStorage();
    writeActiveCart(s, { items: [item], customer: null, globalDiscountInput: '' }, NOW);
    expect(s.data[ACTIVE_CART_KEY]).toBeDefined();
    writeActiveCart(s, { items: [], customer: null, globalDiscountInput: '' }, NOW);
    expect(s.data[ACTIVE_CART_KEY]).toBeUndefined();
  });

  it('drops a cart older than the max age and cleans the key', () => {
    const s = memoryStorage();
    writeActiveCart(s, { items: [item], customer: null, globalDiscountInput: '' }, NOW);
    expect(readActiveCart(s, NOW + ACTIVE_CART_MAX_AGE_MS - 1)).not.toBeNull();
    expect(readActiveCart(s, NOW + ACTIVE_CART_MAX_AGE_MS + 1)).toBeNull();
    expect(s.data[ACTIVE_CART_KEY]).toBeUndefined();
  });

  it('returns null for missing, corrupt or malformed data', () => {
    expect(readActiveCart(memoryStorage(), NOW)).toBeNull();
    expect(readActiveCart(memoryStorage({ [ACTIVE_CART_KEY]: '{not json' }), NOW)).toBeNull();
    expect(readActiveCart(memoryStorage({ [ACTIVE_CART_KEY]: JSON.stringify({ items: 'x', savedAt: NOW }) }), NOW)).toBeNull();
    expect(readActiveCart(memoryStorage({ [ACTIVE_CART_KEY]: JSON.stringify({ items: [item] }) }), NOW)).toBeNull();
  });

  it('does not throw when storage is unavailable', () => {
    const broken = {
      getItem: () => { throw new Error('denied'); },
      setItem: () => { throw new Error('denied'); },
      removeItem: () => { throw new Error('denied'); },
    };
    expect(() => writeActiveCart(broken, { items: [item], customer: null, globalDiscountInput: '' }, NOW)).not.toThrow();
    expect(readActiveCart(broken, NOW)).toBeNull();
  });
});
