// Pure helpers for the product catalog import (US-094): the CSV template,
// base64 of the picked file, the mapping payload, report texts and the
// audit page labels. The backend does the parsing and validation.
import type { AuthUser } from './auth';

/** Importable fields in the backend order (AC-6), with their Spanish labels. */
export const IMPORT_FIELDS: { key: ImportField; label: string; required: boolean }[] = [
  { key: 'name', label: 'Nombre', required: true },
  { key: 'sale_price', label: 'Precio de venta', required: true },
  { key: 'barcode', label: 'Código de barras', required: false },
  { key: 'category', label: 'Categoría', required: false },
  { key: 'unit', label: 'Unidad', required: false },
  { key: 'cost', label: 'Costo', required: false },
  { key: 'stock', label: 'Stock', required: false },
  { key: 'min_stock', label: 'Stock mínimo', required: false },
  { key: 'tax_rate', label: 'IVA', required: false },
  { key: 'code', label: 'Código interno', required: false },
];

export type ImportField =
  | 'name' | 'sale_price' | 'barcode' | 'category' | 'unit'
  | 'cost' | 'stock' | 'min_stock' | 'tax_rate' | 'code';

/** {field: header}; an empty header means "(no importar)". */
export type ImportMapping = Partial<Record<ImportField, string>>;

export interface ImportRowError {
  row: number;
  field?: string;
  column: string;
  message: string;
}

/** Dry-run answer of POST /products/import (AC-19). */
export interface ImportReport {
  file_name: string;
  encoding: string;
  delimiter: string;
  headers: string[];
  mapping: ImportMapping;
  ignored_headers: string[];
  total_rows: number;
  valid_rows: number;
  error_count: number;
  errors: ImportRowError[];
  new_categories: string[];
  valid: boolean;
  preview: Record<string, unknown>[];
}

/** Commit answer of POST /products/import (AC-21). */
export interface ImportResult {
  products_created: number;
  categories_created: number;
  stock_entries_created: number;
  first_product_id: number;
  last_product_id: number;
}

export const TEMPLATE_FILE_NAME = 'plantilla-productos.csv';

const TEMPLATE_LINES = [
  'Nombre;Código de barras;Categoría;Unidad;Costo;Precio de venta;Stock;Stock mínimo;IVA;Código interno',
  'Arroz Diana 500 g;7702001001234;Granos;unidad;1.800;2.500;24;6;0;ARZ-500',
  'Queso campesino;;Lácteos;kg;14.000;18.000;3,5;1;0;',
];

/** The downloadable template: UTF-8 BOM, `;` delimiter, CRLF (AC-28). */
export function buildTemplateCSV(): string {
  return '﻿' + TEMPLATE_LINES.join('\r\n');
}

/**
 * Base64 of raw bytes (not of UTF-8 text, so a Windows-1252 file survives).
 * Built in chunks: spreading a large array into String.fromCharCode
 * overflows the call stack.
 */
export function bytesToBase64(bytes: Uint8Array): string {
  const CHUNK = 0x8000;
  let binary = '';
  for (let i = 0; i < bytes.length; i += CHUNK) {
    binary += String.fromCharCode.apply(null, Array.from(bytes.subarray(i, i + CHUNK)));
  }
  return btoa(binary);
}

/** Drops the fields set to "(no importar)" (empty value). */
export function buildMappingPayload(mapping: Record<string, string>): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [field, header] of Object.entries(mapping)) {
    if (header) out[field] = header;
  }
  return out;
}

/** The commit button is enabled only for a valid report with rows. */
export function canCommit(report: { valid: boolean; valid_rows: number }): boolean {
  return report.valid && report.valid_rows > 0;
}

/** "Fila 7 · Precio de venta: valor inválido…" */
export function formatImportError(e: { row: number; column: string; message: string }): string {
  return e.column ? `Fila ${e.row} · ${e.column}: ${e.message}` : `Fila ${e.row}: ${e.message}`;
}

/** "y 100 errores más" when the report lists fewer errors than it found. */
export function moreErrorsText(errorCount: number, shown: number): string {
  const rest = errorCount - shown;
  return rest > 0 ? `y ${rest} errores más` : '';
}

/** Only an admin may import a catalog (AC-24). */
export function canImportProducts(user: AuthUser | null): boolean {
  return user?.role === 'admin';
}

/** Audit page label of the import action. */
export const PRODUCT_IMPORT_ACTION_LABELS: Record<string, string> = {
  product_import: 'Importación de productos',
};

/** "3 productos importados de «inventario.csv»", from the raw details JSON. */
export function describeProductImportAudit(details: string): string {
  let parsed: Record<string, unknown> = {};
  try { parsed = JSON.parse(details || '{}'); } catch { /* malformed details: describe with defaults */ }
  return `${Number(parsed.products_created ?? 0)} productos importados de «${String(parsed.file_name ?? '')}»`;
}

/**
 * Header cells of the first non-blank line of the picked file, for the manual
 * mapping when the backend recognized no required column (it then answers 400
 * with no report). Mirrors the backend reading: UTF-8 (BOM stripped) or else
 * Windows-1252; delimiter `;`, `,` or tab, whichever appears most outside quotes.
 */
export function readHeaderLine(bytes: Uint8Array): string[] {
  let text: string;
  try {
    text = new TextDecoder('utf-8', { fatal: true }).decode(bytes);
  } catch {
    text = new TextDecoder('windows-1252').decode(bytes);
  }
  if (text.startsWith('\uFEFF')) text = text.slice(1);
  const line = text.split(/\r?\n/).find(l => l.trim() !== '') ?? '';

  const counts: Record<string, number> = { ';': 0, ',': 0, '\t': 0 };
  let inQuote = false;
  for (const c of line) {
    if (c === '"') inQuote = !inQuote;
    else if (!inQuote && c in counts) counts[c]++;
  }
  let delim = ';';
  for (const d of [',', '\t']) if (counts[d] > counts[delim]) delim = d;

  const cells: string[] = [];
  let cur = '';
  inQuote = false;
  for (let i = 0; i < line.length; i++) {
    const c = line[i];
    if (c === '"') {
      if (inQuote && line[i + 1] === '"') { cur += '"'; i++; } else inQuote = !inQuote;
    } else if (c === delim && !inQuote) {
      cells.push(cur.trim());
      cur = '';
    } else {
      cur += c;
    }
  }
  cells.push(cur.trim());
  return cells;
}
