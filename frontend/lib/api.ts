import { getToken } from './auth';

const BASE = process.env.NEXT_PUBLIC_API_URL ?? '/api';

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
  const json: ApiResponse<T> = await res.json();

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
