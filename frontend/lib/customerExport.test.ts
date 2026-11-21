// R-236 contract section 4, T-1..T-8 and T-10 (AC-1..AC-12).
// Helper API from the contract (section 5):
//   buildCustomersCSV(rows): string
//   customersExportFileName(now: Date): string
//   runCustomerExport({ get, download, now, setBusy }): Promise<'ok' | 'empty' | 'error'>  (never rejects)
import { describe, it, expect } from 'vitest';
import {
  buildCustomersCSV,
  customersExportFileName,
  runCustomerExport,
} from './customerExport';

type Row = Parameters<typeof buildCustomersCSV>[0][number];
type ExportDeps = Parameters<typeof runCustomerExport>[0];

const HEADER = 'Nombre;Teléfono;Tope de crédito;Saldo;Deuda más antigua';

// Builds a row shaped like the admin GET /customers response.
function row(
  id: number,
  name: string,
  phone: string,
  credit_limit: number | null,
  balance: number,
  oldest_debt_at: string | null,
): Row {
  return {
    id,
    name,
    phone,
    address: '',
    notes: '',
    credit_limit,
    created_at: '2026-01-01T00:00:00Z',
    balance,
    oldest_debt_at,
  } as unknown as Row;
}

// Fixture rows from the contract, section 4.
const A = row(1, 'Gus', '', null, -20000, '2026-11-16T02:27:36Z');
const B = row(2, 'Ana Pérez', '3001234567', 200000, 150000, '2026-09-25T05:00:00Z');
const C = row(3, 'Beto', '', null, 1234567, '2026-11-18T02:30:00Z');
const D = row(4, 'Fer', '', 0, 0, '2026-11-15T02:27:36Z');
const E = row(5, 'Álvaro; "el Mono"', '+57 300 123 4567', 50000, 30000, '2026-11-10T15:00:00Z');

function lines(csv: string): string[] {
  return csv.replace(/^﻿/, '').split('\r\n');
}

describe('T-1 buildCustomersCSV header (AC-1)', () => {
  it('an empty list gives exactly the BOM and the header, with no trailing CRLF', () => {
    const csv = buildCustomersCSV([]);
    expect(csv).toBe('﻿' + HEADER);
    expect(csv.startsWith('﻿')).toBe(true);
    expect(csv.includes(',')).toBe(false);
    expect(csv.endsWith('\r\n')).toBe(false);
  });
});

describe('T-2 buildCustomersCSV rows (AC-2, AC-3, AC-4, AC-5, AC-7)', () => {
  const csv = buildCustomersCSV([A, B, C, D, E]);
  const out = lines(csv);

  it('writes the header and one line per customer, sorted by localeCompare', () => {
    expect(out).toEqual([
      HEADER,
      '"Álvaro; ""el Mono""";+57 300 123 4567;50000;30000;10/11/2026',
      'Ana Pérez;3001234567;200000;150000;25/09/2026',
      'Beto;;;1234567;17/11/2026',
      'Fer;;0;0;',
      'Gus;;;-20000;',
    ]);
  });

  it('has no TOTAL row and no trailing CRLF', () => {
    expect(out.some((l) => l.startsWith('TOTAL'))).toBe(false);
    expect(csv.endsWith('\r\n')).toBe(false);
  });

  it('writes the oldest debt as the Bogota day, not the UTC day', () => {
    const beto = out.find((l) => l.startsWith('Beto;'));
    expect(beto?.endsWith(';17/11/2026')).toBe(true);
    expect(csv.includes('18/11/2026')).toBe(false);
  });
});

describe('T-3 buildCustomersCSV money rounding (AC-3)', () => {
  it('rounds balances with Math.round and writes plain digits', () => {
    const out = lines(
      buildCustomersCSV([
        row(10, 'Uno', '', 100000, 15000.4, '2026-11-10T15:00:00Z'),
        row(11, 'Dos', '', 1234567, 14999.6, '2026-11-10T15:00:00Z'),
      ]),
    );
    const uno = out[2].split(';');
    const dos = out[1].split(';');
    expect(uno[0]).toBe('Uno');
    expect(uno[3]).toBe('15000');
    expect(dos[0]).toBe('Dos');
    expect(dos[3]).toBe('15000');
    for (const cells of [uno, dos]) {
      for (const money of [cells[2], cells[3]]) {
        expect(money.includes('$')).toBe(false);
        expect(money.includes('.')).toBe(false);
      }
    }
    expect(dos[2]).toBe('1234567');
    expect(uno[2]).toBe('100000');
  });
});

describe('T-4 buildCustomersCSV text quoting (AC-6)', () => {
  it('quotes and doubles inner double quotes', () => {
    const out = lines(buildCustomersCSV([row(20, 'Doña "Tina"', '', null, 0, null)]));
    expect(out[1]).toBe('"Doña ""Tina""";;;0;');
  });

  it('quotes a name with a line feed', () => {
    const out = lines(buildCustomersCSV([row(21, 'Luz\nMarina', '', null, 0, null)]));
    expect(out[1]).toBe('"Luz\nMarina";;;0;');
  });

  it('writes a plain name unquoted', () => {
    const out = lines(buildCustomersCSV([row(22, 'Ana Pérez', '', null, 0, null)]));
    expect(out[1]).toBe('Ana Pérez;;;0;');
  });
});

describe('T-5 customersExportFileName (AC-9)', () => {
  it('uses the Bogota date at 21:30 Bogota (already 18/11 in UTC)', () => {
    expect(customersExportFileName(new Date('2026-11-18T02:30:00Z'))).toBe('clientes-2026-11-17.csv');
  });

  it('uses the Bogota date at 10:00 Bogota', () => {
    expect(customersExportFileName(new Date('2026-11-17T15:00:00Z'))).toBe('clientes-2026-11-17.csv');
  });
});

// Fake dependencies for runCustomerExport that record every call in order.
function fakeDeps(result: () => Promise<unknown>) {
  const log: string[] = [];
  const getCalls: unknown[][] = [];
  const downloads: unknown[][] = [];
  const busy: boolean[] = [];
  const deps = {
    get: (...args: unknown[]) => {
      getCalls.push(args);
      log.push('get');
      return result().finally(() => log.push('get-settled'));
    },
    download: (...args: unknown[]) => {
      downloads.push(args);
      log.push('download');
    },
    now: new Date('2026-11-17T15:00:00Z'),
    setBusy: (v: boolean) => {
      busy.push(v);
      log.push(`busy:${v}`);
    },
  } as unknown as ExportDeps;
  return { deps, log, getCalls, downloads, busy };
}

describe('T-6 runCustomerExport success (AC-8, AC-9)', () => {
  it('requests /customers with no query and downloads the file', async () => {
    const f = fakeDeps(() => Promise.resolve({ customers: [B] }));
    const res = await runCustomerExport(f.deps);
    expect(res).toBe('ok');
    expect(f.getCalls).toEqual([['/customers']]);
    expect(f.downloads).toEqual([['clientes-2026-11-17.csv', buildCustomersCSV([B])]]);
  });
});

describe('T-7 runCustomerExport empty (AC-10)', () => {
  it('null customers downloads nothing and returns empty', async () => {
    const f = fakeDeps(() => Promise.resolve({ customers: null }));
    expect(await runCustomerExport(f.deps)).toBe('empty');
    expect(f.downloads).toEqual([]);
  });

  it('an empty list downloads nothing and returns empty', async () => {
    const f = fakeDeps(() => Promise.resolve({ customers: [] }));
    expect(await runCustomerExport(f.deps)).toBe('empty');
    expect(f.downloads).toEqual([]);
  });
});

describe('T-8 runCustomerExport failure (AC-11)', () => {
  // The page maps 'error' to the toast "No se pudo exportar los clientes" (covered by M-1).
  it('a rejected request resolves to error, never rejects, and downloads nothing', async () => {
    const f = fakeDeps(() => Promise.reject(new Error('red')));
    await expect(runCustomerExport(f.deps)).resolves.toBe('error');
    expect(f.downloads).toEqual([]);
  });
});

describe('T-10 runCustomerExport busy state (AC-12)', () => {
  const cases: Array<[string, () => Promise<unknown>, string]> = [
    ['ok', () => Promise.resolve({ customers: [B] }), 'ok'],
    ['empty', () => Promise.resolve({ customers: [] }), 'empty'],
    ['error', () => Promise.reject(new Error('red')), 'error'],
  ];

  for (const [label, result, expected] of cases) {
    it(`sets busy true before get and false after get settles (${label})`, async () => {
      const f = fakeDeps(result);
      expect(await runCustomerExport(f.deps)).toBe(expected);
      expect(f.busy).toEqual([true, false]);
      const busyOn = f.log.indexOf('busy:true');
      const busyOff = f.log.indexOf('busy:false');
      expect(busyOn).toBeGreaterThanOrEqual(0);
      expect(busyOn).toBeLessThan(f.log.indexOf('get'));
      expect(busyOff).toBeGreaterThan(f.log.indexOf('get-settled'));
    });
  }
});
