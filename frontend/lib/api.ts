import { clearAuth, getToken } from './auth';

const BASE = process.env.NEXT_PUBLIC_API_URL ?? '/api';

// Navigate to /login at most once per page load, even with parallel 401s.
let redirecting = false;

// A 401 means a dead session, except on the login form (wrong password) and
// while already on /login (avoids a reload loop).
export function shouldRedirectOnUnauthorized({
  status,
  requestPath,
  pathname,
}: {
  status: number;
  requestPath: string;
  pathname: string;
}): boolean {
  if (status !== 401) return false;
  if (requestPath === '/auth/login') return false;
  return pathname !== '/login' && pathname !== '/login/';
}

interface ApiResponse<T> {
  data?: T;
  error?: string;
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(`${BASE}${path}`, { ...options, headers });

  // Expired session: clear it and go to /login before parsing the body,
  // which may not be JSON.
  if (
    typeof window !== 'undefined' &&
    shouldRedirectOnUnauthorized({
      status: res.status,
      requestPath: path,
      pathname: window.location.pathname,
    }) &&
    !redirecting
  ) {
    redirecting = true;
    clearAuth();
    window.location.assign('/login');
  }

  let json: ApiResponse<T> = {};
  try {
    json = await res.json();
  } catch (err) {
    // Only a 401 may have a non-JSON body; any other status keeps failing loudly.
    if (res.status !== 401) throw err;
  }

  if (!res.ok) {
    throw new Error(json.error ?? `HTTP ${res.status}`);
  }
  return json.data as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  patch: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
};

export interface SetupStatus {
  completed: boolean;
}

export interface OnboardingPayload {
  full_name: string;
  username: string;
  password: string;
  business_name: string;
  nit: string;
}

export interface OnboardingResponse {
  token: string;
  user: { id: number; username: string; role: 'admin' | 'cashier' };
}

export function getSetupStatus(): Promise<SetupStatus> {
  return api.get<SetupStatus>('/setup/status');
}

export function submitOnboarding(payload: OnboardingPayload): Promise<OnboardingResponse> {
  return api.post<OnboardingResponse>('/onboarding', payload);
}
