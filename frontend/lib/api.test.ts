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

// Contrato US-056 T-18 (AC-24)
describe('api.patch', () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    vi.resetModules();
    fetchMock.mockReset();
    fetchMock.mockResolvedValue({ ok: true, status: 200, json: async () => ({ data: 1 }) });
    vi.stubGlobal('fetch', fetchMock);
    vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('T-18: sends PATCH with JSON body', async () => {
    const { api } = await import('./api');
    await api.patch('/x', { a: 1 });
    const init = fetchMock.mock.calls[0][1];
    expect(init.method).toBe('PATCH');
    expect(init.body).toBe('{"a":1}');
  });
});

// Contract US-077 T-1..T-10 (R-49: forced redirect on expired session)
describe('api 401 handling (US-077)', () => {
  const fetchMock = vi.fn();
  let store: Map<string, string>;
  let assign: ReturnType<typeof vi.fn>;

  const seed = () => {
    store = new Map<string, string>([
      ['pos_token', 'tok'],
      ['pos_user', '{"id":1,"username":"ana","role":"admin"}'],
      ['pos_force_pw', '1'],
    ]);
    vi.stubGlobal('localStorage', {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => void store.set(k, v),
      removeItem: (k: string) => void store.delete(k),
    });
  };

  const stubWindow = (pathname: string) => {
    assign = vi.fn();
    vi.stubGlobal('window', { location: { pathname, assign } });
  };

  const respond = (status: number, body: unknown) =>
    fetchMock.mockResolvedValue({
      ok: status >= 200 && status < 300,
      status,
      json: async () => body,
    });

  const expectSessionIntact = () => {
    expect(store.get('pos_token')).toBe('tok');
    expect(store.get('pos_user')).toBe('{"id":1,"username":"ana","role":"admin"}');
    expect(store.get('pos_force_pw')).toBe('1');
  };

  beforeEach(() => {
    vi.resetModules();
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
    seed();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('T-1: 401 on authenticated request clears session and navigates to /login', async () => {
    stubWindow('/inventory');
    respond(401, { error: 'unauthorized' });
    const { api } = await import('./api');
    await api.get('/products').catch(() => {});
    expect(assign).toHaveBeenCalledTimes(1);
    expect(assign).toHaveBeenCalledWith('/login');
    expect(store.has('pos_token')).toBe(false);
    expect(store.has('pos_user')).toBe(false);
    expect(store.has('pos_force_pw')).toBe(false);
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer tok');
  });

  it('T-2: 401 rejects with the body error', async () => {
    stubWindow('/inventory');
    respond(401, { error: 'unauthorized' });
    const { api } = await import('./api');
    await expect(api.get('/products')).rejects.toThrow('unauthorized');
  });

  it('T-3: 401 with non-JSON body still redirects and rejects with HTTP 401', async () => {
    stubWindow('/pos');
    fetchMock.mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => {
        throw new SyntaxError('x');
      },
    });
    const { api } = await import('./api');
    await expect(api.get('/products')).rejects.toThrow('HTTP 401');
    expect(assign).toHaveBeenCalledTimes(1);
    expect(assign).toHaveBeenCalledWith('/login');
    expect(store.has('pos_token')).toBe(false);
  });

  it('T-4: wrong password on /auth/login with stale session does not clear or navigate', async () => {
    stubWindow('/inventory');
    respond(401, { error: 'credenciales inválidas' });
    const { api } = await import('./api');
    await expect(api.post('/auth/login', { username: 'ana', password: 'bad' })).rejects.toThrow(
      'credenciales inválidas'
    );
    expect(assign).not.toHaveBeenCalled();
    expect(store.get('pos_token')).toBe('tok');
  });

  it('T-5: wrong password on /login page does not clear or navigate', async () => {
    stubWindow('/login');
    respond(401, { error: 'credenciales inválidas' });
    const { api } = await import('./api');
    await expect(api.post('/auth/login', { username: 'ana', password: 'bad' })).rejects.toThrow(
      'credenciales inválidas'
    );
    expect(assign).not.toHaveBeenCalled();
    expectSessionIntact();
  });

  it.each(['/login', '/login/'])('T-6: 401 while on %s does not loop', async (pathname) => {
    stubWindow(pathname);
    respond(401, { error: 'unauthorized' });
    const { api } = await import('./api');
    await expect(api.get('/setup/status')).rejects.toThrow('unauthorized');
    expect(assign).toHaveBeenCalledTimes(0);
    expectSessionIntact();
  });

  it.each([
    [400, { error: 'request inválido' }, 'request inválido'],
    [403, { error: 'forbidden' }, 'forbidden'],
    [404, { error: 'not found' }, 'not found'],
    [500, { error: 'boom' }, 'boom'],
  ])('T-7: status %i rejects with its message and keeps the session', async (status, body, msg) => {
    stubWindow('/pos');
    respond(status, body);
    const { api } = await import('./api');
    await expect(api.get('/x')).rejects.toThrow(msg);
    expect(assign).toHaveBeenCalledTimes(0);
    expect(store.size).toBe(3);
  });

  it('T-7: status 200 resolves data without navigation', async () => {
    stubWindow('/pos');
    respond(200, { data: 42 });
    const { api } = await import('./api');
    await expect(api.get('/x')).resolves.toBe(42);
    expect(assign).toHaveBeenCalledTimes(0);
    expect(store.size).toBe(3);
  });

  it('T-8: shouldRedirectOnUnauthorized truth table', async () => {
    const mod = (await import('./api')) as unknown as {
      shouldRedirectOnUnauthorized?: (a: {
        status: number;
        requestPath: string;
        pathname: string;
      }) => boolean;
    };
    const fn = mod.shouldRedirectOnUnauthorized;
    expect(typeof fn).toBe('function');
    const f = fn!;
    expect(f({ status: 401, requestPath: '/products', pathname: '/inventory' })).toBe(true);
    expect(f({ status: 401, requestPath: '/auth/login', pathname: '/inventory' })).toBe(false);
    expect(f({ status: 401, requestPath: '/products', pathname: '/login' })).toBe(false);
    expect(f({ status: 401, requestPath: '/products', pathname: '/login/' })).toBe(false);
    expect(f({ status: 403, requestPath: '/products', pathname: '/pos' })).toBe(false);
    expect(f({ status: 200, requestPath: '/x', pathname: '/pos' })).toBe(false);
  });

  it('T-9: parallel 401s navigate exactly once', async () => {
    stubWindow('/reports');
    respond(401, { error: 'unauthorized' });
    const { api } = await import('./api');
    const results = await Promise.allSettled([
      api.get('/reports/sales'),
      api.get('/reports/portfolio'),
      api.get('/reports/registers'),
    ]);
    expect(results.map((r) => r.status)).toEqual(['rejected', 'rejected', 'rejected']);
    expect(assign).toHaveBeenCalledTimes(1);
  });

  it('T-10: 401 without window rejects with the API error, not ReferenceError', async () => {
    // window intentionally not stubbed (node env)
    respond(401, { error: 'unauthorized' });
    const { api } = await import('./api');
    await expect(api.get('/products')).rejects.toThrow('unauthorized');
    expectSessionIntact();
  });
});
