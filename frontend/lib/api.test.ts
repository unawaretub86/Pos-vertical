import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Contrato US-054 T-11 (AC-15)
describe('api base URL', () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    vi.resetModules();
    fetchMock.mockReset();
    fetchMock.mockResolvedValue({ ok: true, status: 200, json: async () => ({ data: 1 }) });
    vi.stubGlobal('fetch', fetchMock);
    vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it('uses relative /api when NEXT_PUBLIC_API_URL is undefined', async () => {
    vi.stubEnv('NEXT_PUBLIC_API_URL', undefined as unknown as string);
    const { api } = await import('./api');
    await api.get('/x');
    expect(fetchMock.mock.calls[0][0]).toBe('/api/x');
  });

  it('uses NEXT_PUBLIC_API_URL when defined', async () => {
    vi.stubEnv('NEXT_PUBLIC_API_URL', 'http://localhost:8080/api');
    const { api } = await import('./api');
    await api.get('/x');
    expect(fetchMock.mock.calls[0][0]).toBe('http://localhost:8080/api/x');
  });
});
