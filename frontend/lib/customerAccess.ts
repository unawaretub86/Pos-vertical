// Who may open the Customers screens (/customers and /customers/{id}). They
// show every balance and the receivables, which stay with the owner (US-105).

import type { AuthUser } from './auth';

export function canSeeCustomers(user: AuthUser | null): boolean {
  return user !== null && user.role === 'admin';
}
