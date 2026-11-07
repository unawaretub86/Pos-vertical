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

const MSG_BUSY = 'El sistema está ocupado, reintenta en un momento';
const MSG_NET_SALE =
  'No se pudo conectar con el servidor. Revise la red y el historial de ventas antes de cobrar de nuevo.';
const MSG_NET_NEUTRAL = 'No se pudo conectar con el servidor. Revise la red e intente de nuevo.';

// Waits before the 2nd and 3rd attempt after a 503. The server answers 503
// only when SQLITE_BUSY kept a write from committing, so any method is safe
// to retry.
const BUSY_RETRY_DELAYS_MS = [500, 1000];

// A network error gives no answer, so a charge may already be committed: the
// two charge endpoints tell the cashier to check the sales history first.
export function networkErrorMessage(method: string, path: string): string {
  const p = path.split('?')[0];
  const isCharge = p === '/sales' || /^\/orders\/[^/]+\/checkout$/.test(p);
  return method.toUpperCase() === 'POST' && isCharge ? MSG_NET_SALE : MSG_NET_NEUTRAL;
}

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

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

  let res: Response;
  for (let attempt = 0; ; attempt++) {
    try {
      res = await fetch(`${BASE}${path}`, { ...options, headers });
    } catch {
      // Never retried: the request may have reached the server.
      throw new Error(networkErrorMessage(options.method ?? 'GET', path));
    }
    if (res.status !== 503 || attempt >= BUSY_RETRY_DELAYS_MS.length) break;
    await sleep(BUSY_RETRY_DELAYS_MS[attempt]);
  }

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
    // Only a 401 may have a non-JSON body; a 503 falls back to the busy text;
    // any other status keeps failing loudly.
    if (res.status === 503) throw new Error(MSG_BUSY);
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
