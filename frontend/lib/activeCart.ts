// Persistence of the live POS cart (R-73). When the 24 h session token expires,
// api.ts sends the cashier to /login with a hard navigation and the in-memory
// cart is lost. The cart is saved on every change and restored on the next
// visit to /pos. Pure and storage-injected so it can be unit tested without a DOM.

export const ACTIVE_CART_KEY = 'pos.activeCart';
// A cart older than this is dropped: its prices and stock are probably stale.
export const ACTIVE_CART_MAX_AGE_MS = 12 * 60 * 60 * 1000;

interface StorageLike {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

export interface ActiveCart<TItem, TCustomer> {
  items: TItem[];
  customer: TCustomer | null;
  globalDiscountInput: string;
}

export function writeActiveCart<TItem, TCustomer>(
  storage: StorageLike,
  cart: ActiveCart<TItem, TCustomer>,
  now: number = Date.now(),
): void {
  try {
    if (cart.items.length === 0) {
      storage.removeItem(ACTIVE_CART_KEY);
      return;
    }
    storage.setItem(ACTIVE_CART_KEY, JSON.stringify({ ...cart, savedAt: now }));
  } catch {
    // storage unavailable: the cart just won't survive a reload
  }
}

export function readActiveCart<TItem, TCustomer>(
  storage: StorageLike,
  now: number = Date.now(),
): ActiveCart<TItem, TCustomer> | null {
  try {
    const raw = storage.getItem(ACTIVE_CART_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    if (!parsed || !Array.isArray(parsed.items) || parsed.items.length === 0 || typeof parsed.savedAt !== 'number') {
      return null;
    }
    if (now - parsed.savedAt > ACTIVE_CART_MAX_AGE_MS) {
      storage.removeItem(ACTIVE_CART_KEY);
      return null;
    }
    return {
      items: parsed.items,
      customer: parsed.customer ?? null,
      globalDiscountInput: typeof parsed.globalDiscountInput === 'string' ? parsed.globalDiscountInput : '',
    };
  } catch {
    return null;
  }
}
