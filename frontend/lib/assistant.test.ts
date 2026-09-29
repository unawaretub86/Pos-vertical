import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// US-062 contract §4, T-1..T-11. Pure logic + the network layer with a stubbed
// `fetch` (vitest runs in the `node` environment: no DOM, no jsdom).
//
// The store keeps module-level state (history survives SPA navigation, dies on
// F5), so every test loads a FRESH copy of the module with vi.resetModules() +
// dynamic import. That is also exactly how T-9 simulates F5.
//
// API expected from frontend/lib/assistant.ts (names from contract §5; the
// shapes below are what these tests pin):
//   askAssistant(query): Promise<AskResponse>
//   getState(): { loading: boolean; messages: Message[] }
//   subscribe(listener): () => void            (useSyncExternalStore contract)
//   send(text): Promise<void>
//   bindUser(userId): void
//   Message = { role: 'user' | 'assistant'; text: string; intent?: string; suggestions?: string[] }
//   parseAnswerText(text): { intro: string; items: string[]; footer: string[] }
//   activeChips(messages): string[]
//   canSeeAssistant(user): boolean
//   isHiddenPath(pathname): boolean
//   GREETING_TEXT, GREETING_CHIPS

type AssistantModule = typeof import('./assistant');

async function load(): Promise<AssistantModule> {
  vi.resetModules();
  return await import('./assistant');
}

const ERROR_TEXT = 'No pude consultar los datos. Intentá de nuevo.';

const SALES_DATA = {
  intent: 'sales',
  period: { label: 'Hoy' },
  text: 'Hoy vendiste $40.000 en 2 ventas.',
  data: { total: 40000, count: 2 },
  suggestions: [] as string[],
  tier: 'regex',
  confidence: 1,
};

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

let fetchMock: ReturnType<typeof vi.fn>;

beforeEach(() => {
  fetchMock = vi.fn();
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

// Lets every already-resolved microtask / zero-delay timer run, while a
// manually controlled fetch promise stays pending.
const flush = () => new Promise<void>((r) => setTimeout(r, 0));

describe('T-1 askAssistant', () => {
  it('POSTs the trimmed query to /ai/ask and returns the data object', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ data: SALES_DATA }));
    const { askAssistant } = await load();

    const result = await askAssistant('  cuánto vendí hoy ');

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url).endsWith('/ai/ask')).toBe(true);
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body)).toEqual({ query: 'cuánto vendí hoy' });
    expect(result.intent).toBe('sales');
    expect(result.text).toBe('Hoy vendiste $40.000 en 2 ventas.');
  });
});

describe('T-2 greeting', () => {
  it('exposes the approved greeting text and the 3 chips in fixed order', async () => {
    const { GREETING_CHIPS, GREETING_TEXT } = await load();

    expect(GREETING_CHIPS).toEqual(['cuánto vendí hoy', 'quién me debe', 'qué se está acabando']);
    expect(GREETING_CHIPS).toHaveLength(3);
    expect(GREETING_TEXT).toBe(
      'Hola, soy tu asistente. Preguntame por tus ventas, tu cartera o tu inventario, o tocá una de estas:',
    );
  });
});

describe('T-3 send: loading then answer', () => {
  it('is loading with only the user message while pending, then appends the answer', async () => {
    let resolveFetch!: (r: Response) => void;
    fetchMock.mockReturnValue(new Promise<Response>((r) => (resolveFetch = r)));
    const { send, getState } = await load();

    const pending = send('cuánto vendí hoy');
    await flush();

    let state = getState();
    expect(state.loading).toBe(true);
    expect(state.messages).toHaveLength(1);
    expect(state.messages[0].role).toBe('user');
    expect(state.messages[0].text).toBe('cuánto vendí hoy');

    resolveFetch(jsonResponse({ data: SALES_DATA }));
    await pending;

    state = getState();
    expect(state.loading).toBe(false);
    expect(state.messages).toHaveLength(2);
    expect(state.messages[0].role).toBe('user');
    expect(state.messages[0].text).toBe('cuánto vendí hoy');
    expect(state.messages[1].role).toBe('assistant');
    expect(state.messages[1].text).toBe('Hoy vendiste $40.000 en 2 ventas.');
  });
});

describe('T-4 send: failures show a friendly message', () => {
  const scenarios: { name: string; arm: () => void }[] = [
    {
      name: 'network failure (TypeError)',
      arm: () => fetchMock.mockRejectedValue(new TypeError('Failed to fetch')),
    },
    {
      name: 'HTTP 403 forbidden',
      arm: () => fetchMock.mockResolvedValue(jsonResponse({ error: 'forbidden' }, 403)),
    },
    {
      name: 'HTTP 500',
      arm: () =>
        fetchMock.mockResolvedValue(jsonResponse({ error: 'error procesando la consulta' }, 500)),
    },
    {
      name: 'non-JSON body',
      arm: () =>
        fetchMock.mockResolvedValue(
          new Response('<html>Bad gateway</html>', {
            status: 500,
            headers: { 'Content-Type': 'text/html' },
          }),
        ),
    },
  ];

  for (const s of scenarios) {
    it(`${s.name} -> fixed Spanish message, loading false, no raw error leaked`, async () => {
      s.arm();
      const { send, getState } = await load();

      await send('cuánto vendí hoy');

      const state = getState();
      expect(state.loading).toBe(false);
      expect(state.messages).toHaveLength(2);
      expect(state.messages[state.messages.length - 1].text).toBe(ERROR_TEXT);
      const all = state.messages.map((m) => m.text).join('\n');
      expect(all).not.toContain('forbidden');
      expect(all).not.toContain('HTTP 500');
      expect(all).not.toContain('error procesando la consulta');
      expect(all).not.toContain('Failed to fetch');
    });
  }
});

describe('T-5 send: empty input and double submit', () => {
  it('whitespace-only text never reaches the backend nor the history', async () => {
    const { send, getState } = await load();

    await send('   ');

    expect(fetchMock).toHaveBeenCalledTimes(0);
    expect(getState().messages).toEqual([]);
  });

  it('a second send while the first is pending is ignored (1 fetch call)', async () => {
    let resolveFetch!: (r: Response) => void;
    fetchMock.mockReturnValue(new Promise<Response>((r) => (resolveFetch = r)));
    const { send, getState } = await load();

    const first = send('a');
    await flush();
    const second = send('a');
    await flush();

    expect(fetchMock).toHaveBeenCalledTimes(1);

    resolveFetch(jsonResponse({ data: SALES_DATA }));
    await Promise.all([first, second]);

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(getState().messages).toHaveLength(2);
  });
});

describe('T-6 parseAnswerText', () => {
  it('numbered debtors list: intro, 5 items without the "N. " prefix, "y N más" footer', async () => {
    const { parseAnswerText } = await load();
    const text =
      'Te deben en total $250.000:\n' +
      '1. Dora — $100.000\n' +
      '2. Eli — $80.000\n' +
      '3. Fer — $70.000\n' +
      '4. Gus — $0\n' +
      '5. Ivo — $0\n' +
      'y 1 más';

    const parsed = parseAnswerText(text);

    expect(parsed.intro).toBe('Te deben en total $250.000:');
    expect(parsed.items).toEqual([
      'Dora — $100.000',
      'Eli — $80.000',
      'Fer — $70.000',
      'Gus — $0',
      'Ivo — $0',
    ]);
    expect(parsed.footer).toEqual(['y 1 más']);
  });

  it('dash list (payment mix): 2 items and the "Ojo" line in the footer', async () => {
    const { parseAnswerText } = await load();
    const ojo = 'Ojo: "transferencia" agrupa Nequi, Daviplata y otras transferencias.';
    const text = 'Hoy te entró:\n- Efectivo: $30.000\n- Tarjeta: $10.000\n' + ojo;

    const parsed = parseAnswerText(text);

    expect(parsed.intro).toBe('Hoy te entró:');
    expect(parsed.items).toEqual(['Efectivo: $30.000', 'Tarjeta: $10.000']);
    expect(parsed.footer).toEqual([ojo]);
  });

  it('plain sentence: intro only, no items, no footer', async () => {
    const { parseAnswerText } = await load();

    const parsed = parseAnswerText('Hoy vendiste $40.000 en 2 ventas.');

    expect(parsed.intro).toBe('Hoy vendiste $40.000 en 2 ventas.');
    expect(parsed.items).toEqual([]);
    expect(parsed.footer).toEqual([]);
  });

  it('top products: the trailing "Lo que más vendés es …" line is footer, not an item', async () => {
    const { parseAnswerText } = await load();
    const contrast =
      'Lo que más vendés es Prod A (90 unidades), pero lo que más plata te deja es Prod B (38% de margen).';
    const text =
      'Hoy lo que más vendiste fue:\n' +
      '1. Prod A — 90 unidades ($900.000)\n' +
      '2. Prod B — 24 unidades ($240.000)\n' +
      contrast;

    const parsed = parseAnswerText(text);

    expect(parsed.intro).toBe('Hoy lo que más vendiste fue:');
    expect(parsed.items).toEqual([
      'Prod A — 90 unidades ($900.000)',
      'Prod B — 24 unidades ($240.000)',
    ]);
    expect(parsed.footer).toEqual([contrast]);
  });
});

describe('T-7 canSeeAssistant', () => {
  it('admin sees it; cashier and no user do not', async () => {
    const { canSeeAssistant } = await load();

    expect(canSeeAssistant({ id: 1, username: 'don.carlos', role: 'admin' })).toBe(true);
    expect(canSeeAssistant({ id: 2, username: 'caja', role: 'cashier' })).toBe(false);
    expect(canSeeAssistant(null)).toBe(false);
  });
});

describe('T-8 isHiddenPath', () => {
  it('hides only on exactly /pos (not on /portfolio, /purchases, /, /reports)', async () => {
    const { isHiddenPath } = await load();

    expect(isHiddenPath('/pos')).toBe(true);
    expect(isHiddenPath('/portfolio')).toBe(false);
    expect(isHiddenPath('/purchases')).toBe(false);
    expect(isHiddenPath('/')).toBe(false);
    expect(isHiddenPath('/reports')).toBe(false);
  });
});

describe('T-9 history lifetime', () => {
  it('survives unsubscribe/resubscribe, dies on module reload (F5), never touches storage', async () => {
    const setItem = vi.fn();
    const storage = { getItem: vi.fn(() => null), setItem, removeItem: vi.fn(), clear: vi.fn() };
    vi.stubGlobal('localStorage', storage);
    vi.stubGlobal('sessionStorage', storage);
    fetchMock.mockResolvedValue(jsonResponse({ data: SALES_DATA }));

    const first = await load();
    await first.send('cuánto vendí hoy');
    expect(first.getState().messages).toHaveLength(2);

    // Widget unmounts (page navigation) and the next page mounts a new one.
    const unsubscribe = first.subscribe(() => {});
    unsubscribe();
    const unsubscribeAgain = first.subscribe(() => {});
    expect(first.getState().messages).toHaveLength(2);
    unsubscribeAgain();

    // F5 == the whole module graph is evaluated again.
    const reloaded = await load();
    expect(reloaded.getState().messages).toHaveLength(0);

    expect(setItem).toHaveBeenCalledTimes(0);
  });
});

describe('T-10 bindUser', () => {
  it('keeps the history for the same user id and clears it when the id changes', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ data: SALES_DATA }));
    const { bindUser, send, getState } = await load();

    bindUser(1);
    await send('cuánto vendí hoy');
    expect(getState().messages).toHaveLength(2);

    bindUser(1);
    expect(getState().messages).toHaveLength(2);

    bindUser(2);
    expect(getState().messages).toHaveLength(0);
  });
});

describe('T-11 activeChips', () => {
  it('returns the suggestions of the last assistant message', async () => {
    const { activeChips } = await load();

    const chips = activeChips([
      { role: 'user', text: 'cuanto vendi el juebes' },
      {
        role: 'assistant',
        text: '¿De qué fecha?',
        intent: 'sales',
        suggestions: ['hoy', 'ayer', 'esta semana', 'este mes'],
      },
    ]);

    expect(chips).toEqual(['hoy', 'ayer', 'esta semana', 'este mes']);
  });

  it('returns [] while the last entry is the user message (request in flight)', async () => {
    const { activeChips } = await load();

    const chips = activeChips([
      {
        role: 'assistant',
        text: '¿De qué fecha?',
        intent: 'sales',
        suggestions: ['hoy', 'ayer', 'esta semana', 'este mes'],
      },
      { role: 'user', text: 'cuánto vendí hoy' },
    ]);

    expect(chips).toEqual([]);
  });

  it('unknown with empty suggestions falls back to the 3 greeting chips', async () => {
    const { activeChips, GREETING_CHIPS } = await load();

    const chips = activeChips([
      { role: 'user', text: 'asdf' },
      { role: 'assistant', text: 'No te entendí.', intent: 'unknown', suggestions: [] },
    ]);

    expect(chips).toEqual(GREETING_CHIPS);
    expect(chips).toHaveLength(3);
  });

  it('never returns chips of an older answer (only the latest is tappable)', async () => {
    const { activeChips } = await load();

    const chips = activeChips([
      { role: 'user', text: 'cuanto vendi el juebes' },
      {
        role: 'assistant',
        text: '¿De qué fecha?',
        intent: 'sales',
        suggestions: ['hoy', 'ayer', 'esta semana', 'este mes'],
      },
      { role: 'user', text: 'asdf' },
      {
        role: 'assistant',
        text: 'No te entendí.',
        intent: 'unknown',
        suggestions: ['cuánto vendí hoy', 'quién me debe', 'qué se está acabando'],
      },
    ]);

    expect(chips).toEqual(['cuánto vendí hoy', 'quién me debe', 'qué se está acabando']);
    expect(chips).not.toContain('hoy');
  });
});
