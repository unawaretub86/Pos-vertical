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

// Contract US-081 F-1..F-5 (AC-8..AC-11): retry only on 503; never on other
// statuses or on network errors; Spanish network error text by method and path.
describe('api 503 retry and network errors (US-081)', () => {
  const fetchMock = vi.fn();
  const BUSY = 'El sistema está ocupado, reintenta en un momento';
  const NET_SALE =
    'No se pudo conectar con el servidor. Revise la red y el historial de ventas antes de cobrar de nuevo.';
  const NET_NEUTRAL = 'No se pudo conectar con el servidor. Revise la red e intente de nuevo.';

  const res = (status: number, body: unknown) => ({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  });
  const nonJSON = (status: number) => ({
    ok: status >= 200 && status < 300,
    status,
    json: async () => {
      throw new SyntaxError('Unexpected token < in JSON');
    },
  });

  beforeEach(() => {
    vi.resetModules();
    vi.useFakeTimers();
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
    vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('F-1: POST /sales retries on 503, 503 then resolves on 201 with 500 ms and 1000 ms waits', async () => {
    fetchMock
      .mockResolvedValueOnce(res(503, { error: BUSY }))
      .mockResolvedValueOnce(res(503, { error: BUSY }))
      .mockResolvedValueOnce(res(201, { data: { id: 7 } }));
    const { api } = await import('./api');
    const body = { items: [{ product_id: 1, quantity: 1 }], total: 50000 };
    const p = api.post<{ id: number }>('/sales', body);
    p.catch(() => {}); // keeps an early assertion failure from also reporting an unhandled rejection

    await vi.advanceTimersByTimeAsync(0);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(499);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(999);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(1);
    expect(fetchMock).toHaveBeenCalledTimes(3);

    await expect(p).resolves.toEqual({ id: 7 });
    const [url0, init0] = fetchMock.mock.calls[0];
    for (const [url, init] of fetchMock.mock.calls) {
      expect(url).toBe(url0);
      expect(init.method).toBe('POST');
      expect(init.body).toBe(JSON.stringify(body));
      expect(init.headers).toEqual(init0.headers);
    }
  });

  it('F-2: three 503s with a JSON error reject with that message after exactly 3 calls', async () => {
    fetchMock.mockResolvedValue(res(503, { error: BUSY }));
    const { api } = await import('./api');
    const p = api.post('/sales', { total: 50000 }).catch((e: unknown) => e);
    await vi.advanceTimersByTimeAsync(5000);
    const err = await p;
    expect(err).toBeInstanceOf(Error);
    expect((err as Error).message).toBe(BUSY);
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('F-2: three 503s with a non-JSON body reject with the busy fallback, not a SyntaxError', async () => {
    fetchMock.mockImplementation(async () => nonJSON(503));
    const { api } = await import('./api');
    const p = api.post('/sales', { total: 50000 }).catch((e: unknown) => e);
    await vi.advanceTimersByTimeAsync(5000);
    const err = await p;
    expect(err).toBeInstanceOf(Error);
    expect(err).not.toBeInstanceOf(SyntaxError);
    expect((err as Error).message).toBe(BUSY);
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('F-3: GET /products retries once on 503 and resolves on 200', async () => {
    fetchMock
      .mockResolvedValueOnce(res(503, { error: BUSY }))
      .mockResolvedValueOnce(res(200, { data: [{ id: 1 }] }));
    const { api } = await import('./api');
    const p = api.get('/products');
    p.catch(() => {}); // see F-1
    await vi.advanceTimersByTimeAsync(500);
    await expect(p).resolves.toEqual([{ id: 1 }]);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it.each([
    [400, 'stock insuficiente: hay 0 unidades de Arroz, intentas vender 1'],
    [409, 'La mesa tiene un pedido abierto'],
    [500, 'boom'],
  ])('F-4: status %i on api.post is not retried and rejects with the body error', async (status, msg) => {
    fetchMock.mockResolvedValue(res(status, { error: msg }));
    const { api } = await import('./api');
    const p = api.post('/sales', { total: 50000 }).catch((e: unknown) => e);
    await vi.advanceTimersByTimeAsync(5000);
    const err = await p;
    expect((err as Error).message).toBe(msg);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('F-4: 401 is not retried and still redirects to /login', async () => {
    const assign = vi.fn();
    vi.stubGlobal('window', { location: { pathname: '/pos', assign } });
    fetchMock.mockResolvedValue(res(401, { error: 'unauthorized' }));
    const { api } = await import('./api');
    const p = api.post('/sales', { total: 50000 }).catch((e: unknown) => e);
    await vi.advanceTimersByTimeAsync(5000);
    const err = await p;
    expect((err as Error).message).toBe('unauthorized');
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(assign).toHaveBeenCalledTimes(1);
    expect(assign).toHaveBeenCalledWith('/login');
  });

  it.each([
    ['a', 'post', '/sales', NET_SALE],
    ['b', 'post', '/orders/12/checkout', NET_SALE],
    ['c', 'get', '/products', NET_NEUTRAL],
    ['d', 'post', '/customers', NET_NEUTRAL],
    ['e', 'get', '/sales', NET_NEUTRAL],
  ] as const)(
    'F-5(%s): network error on %s %s is not retried and rejects with the Spanish message',
    async (_case, method, path, msg) => {
      fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));
      const { api } = await import('./api');
      const call = method === 'get' ? api.get(path) : api.post(path, { total: 50000 });
      const p = call.catch((e: unknown) => e);
      await vi.advanceTimersByTimeAsync(5000);
      const err = await p;
      expect(err).toBeInstanceOf(Error);
      expect((err as Error).message).toBe(msg);
      expect(fetchMock).toHaveBeenCalledTimes(1);
    }
  );

  it('F-5: networkErrorMessage chooses the sale text only for POST /sales and POST /orders/{id}/checkout', async () => {
    const mod = (await import('./api')) as unknown as {
      networkErrorMessage?: (method: string, path: string) => string;
    };
    expect(typeof mod.networkErrorMessage).toBe('function');
    const f = mod.networkErrorMessage!;
    expect(f('POST', '/sales')).toBe(NET_SALE);
    expect(f('POST', '/sales?print=1')).toBe(NET_SALE);
    expect(f('POST', '/orders/12/checkout')).toBe(NET_SALE);
    expect(f('POST', '/orders/12/items')).toBe(NET_NEUTRAL);
    expect(f('POST', '/orders/12/checkout/x')).toBe(NET_NEUTRAL);
    expect(f('POST', '/sales/5/print-ticket')).toBe(NET_NEUTRAL);
    expect(f('GET', '/sales')).toBe(NET_NEUTRAL);
    expect(f('DELETE', '/sales/5/void')).toBe(NET_NEUTRAL);
    expect(f('POST', '/customers')).toBe(NET_NEUTRAL);
  });
});
