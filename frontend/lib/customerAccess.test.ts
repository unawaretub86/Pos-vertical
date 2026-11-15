// US-105 contract section 4, T-8 (AC-6, AC-7).
// Helper API from the contract:
//   canSeeCustomers(user: AuthUser | null): boolean
// Used by the guards of /customers and /customers/{id}: only the admin sees them.
import { describe, it, expect } from 'vitest';
import { canSeeCustomers } from './customerAccess';

describe('T-8 canSeeCustomers', () => {
  it('no user does not see the Customers screens', () => {
    expect(canSeeCustomers(null)).toBe(false);
  });

  it('the cashier does not see the Customers screens', () => {
    expect(canSeeCustomers({ id: 2, username: 'caja', role: 'cashier' })).toBe(false);
  });

  it('the admin sees the Customers screens', () => {
    expect(canSeeCustomers({ id: 1, username: 'don.carlos', role: 'admin' })).toBe(true);
  });
});
