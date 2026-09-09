const TOKEN_KEY = 'pos_token';
const USER_KEY = 'pos_user';
const FORCE_PW_KEY = 'pos_force_pw';

export interface AuthUser {
  id: number;
  username: string;
  role: 'admin' | 'cashier';
}

export function getToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function getUser(): AuthUser | null {
  if (typeof window === 'undefined') return null;
  try {
    const raw = localStorage.getItem(USER_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

export function setAuth(token: string, user: AuthUser, forcePasswordChange = false): void {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
  if (forcePasswordChange) {
    localStorage.setItem(FORCE_PW_KEY, '1');
  } else {
    localStorage.removeItem(FORCE_PW_KEY);
  }
}

export function clearAuth(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
  localStorage.removeItem(FORCE_PW_KEY);
}

export function isAuthenticated(): boolean {
  return !!getToken();
}

export function mustChangePassword(): boolean {
  if (typeof window === 'undefined') return false;
  return localStorage.getItem(FORCE_PW_KEY) === '1';
}

export function clearForcePassword(): void {
  if (typeof window !== 'undefined') localStorage.removeItem(FORCE_PW_KEY);
}
