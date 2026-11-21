// Customer CSV export from `Clientes` (R-236). Built in the browser from the
// admin GET /customers response, in a layout Spanish (Colombia) Excel opens in
// columns: UTF-8 with BOM, `;` delimiter, CRLF. This header is the layout the
// customer import (R-219) and the free fiado spreadsheet must read.

/** A customer row as the admin GET /customers returns it (fields the export uses). */
export interface CustomerExportRow {
  name: string;
  phone: string;
  credit_limit: number | null;
  balance: number;
  oldest_debt_at: string | null;
}

export interface CustomerExportResponse {
  customers: CustomerExportRow[] | null;
}

export type CustomerExportResult = 'ok' | 'empty' | 'error';

export interface CustomerExportDeps {
  get: (path: string) => Promise<CustomerExportResponse>;
  download: (fileName: string, csv: string) => void;
  now: Date;
  setBusy: (busy: boolean) => void;
}

export const CUSTOMERS_CSV_HEADER = ['Nombre', 'Teléfono', 'Tope de crédito', 'Saldo', 'Deuda más antigua'];

const BOM = '﻿';
const DELIMITER = ';';
const SHOP_TIME_ZONE = 'America/Bogota';

const dayFormatter = new Intl.DateTimeFormat('en-CA', {
  timeZone: SHOP_TIME_ZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
});

/** Calendar day of `d` in Bogotá as its year, month and day strings. */
function bogotaDayParts(d: Date): { year: string; month: string; day: string } {
  const parts = dayFormatter.formatToParts(d);
  const get = (type: string) => parts.find(p => p.type === type)?.value ?? '';
  return { year: get('year'), month: get('month'), day: get('day') };
}

/** Quotes a text cell RFC 4180 style when it holds the delimiter, a quote or a line break. */
function textCell(s: string): string {
  if (/[;"\r\n]/.test(s)) return `"${s.replace(/"/g, '""')}"`;
  return s;
}

/** Whole pesos as plain digits, rounded as formatCOP does. */
function moneyCell(n: number | null): string {
  if (n === null || n === undefined) return '';
  return String(Math.round(n));
}

/** dd/mm/yyyy in Bogotá, only while the customer still owes. */
function oldestDebtCell(balance: number, oldestDebtAt: string | null): string {
  if (balance <= 0 || !oldestDebtAt) return '';
  const d = new Date(oldestDebtAt);
  if (Number.isNaN(d.getTime())) return '';
  const { year, month, day } = bogotaDayParts(d);
  return `${day}/${month}/${year}`;
}

/** Builds the customer CSV, sorted by name like the screen's default sort, with no total row. */
export function buildCustomersCSV(rows: CustomerExportRow[]): string {
  const sorted = [...rows].sort((a, b) => a.name.localeCompare(b.name));
  const lines = sorted.map(c =>
    [
      textCell(c.name),
      textCell(c.phone ?? ''),
      moneyCell(c.credit_limit),
      moneyCell(c.balance),
      oldestDebtCell(c.balance, c.oldest_debt_at),
    ].join(DELIMITER),
  );
  return BOM + [CUSTOMERS_CSV_HEADER.join(DELIMITER), ...lines].join('\r\n');
}

/** `clientes-YYYY-MM-DD.csv` with today's date in Bogotá. */
export function customersExportFileName(now: Date): string {
  const { year, month, day } = bogotaDayParts(now);
  return `clientes-${year}-${month}-${day}.csv`;
}

/**
 * Fetches every active customer (no search, no chip filter) and downloads the
 * CSV. Never rejects: the page maps the result to a toast. `setBusy` brackets
 * the request so the button stays disabled while it is in flight.
 */
export async function runCustomerExport({ get, download, now, setBusy }: CustomerExportDeps): Promise<CustomerExportResult> {
  setBusy(true);
  let customers: CustomerExportRow[];
  try {
    const res = await get('/customers');
    customers = res?.customers ?? [];
  } catch {
    setBusy(false);
    return 'error';
  }
  setBusy(false);
  if (customers.length === 0) return 'empty';
  try {
    download(customersExportFileName(now), buildCustomersCSV(customers));
  } catch {
    return 'error';
  }
  return 'ok';
}
