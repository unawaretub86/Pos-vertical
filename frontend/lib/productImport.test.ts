import { describe, it, expect } from 'vitest';
import {
  buildTemplateCSV,
  bytesToBase64,
  buildMappingPayload,
  canCommit,
  formatImportError,
  moreErrorsText,
  describeProductImportAudit,
  canImportProducts,
  PRODUCT_IMPORT_ACTION_LABELS,
} from './productImport';

// Contract US-094 §4, F-1..F-6 (product catalog import).
//
// Function names come from contract §5 (frontend/lib/productImport.ts).
// Assumed, not fixed by the contract:
//   - describeProductImportAudit(details: string): string, where details is the
//     raw audit_log.details JSON string (F-5 shows a single argument).
//   - The audit label map is exported as PRODUCT_IMPORT_ACTION_LABELS, following
//     the OPENING_BALANCE_ACTION_LABELS precedent the audit page spreads into
//     its ACTION_LABELS (frontend/app/audit/page.tsx:42).

const TEMPLATE_HEADER =
  'Nombre;Código de barras;Categoría;Unidad;Costo;Precio de venta;Stock;Stock mínimo;IVA;Código interno';
const TEMPLATE_ROW_1 = 'Arroz Diana 500 g;7702001001234;Granos;unidad;1.800;2.500;24;6;0;ARZ-500';
const TEMPLATE_ROW_2 = 'Queso campesino;;Lácteos;kg;14.000;18.000;3,5;1;0;';

describe('F-1 buildTemplateCSV', () => {
  it('has a BOM, CRLF lines, the AC-28 header and exactly the two example rows', () => {
    const csv = buildTemplateCSV();
    expect(csv.startsWith('﻿')).toBe(true);
    const body = csv.slice(1);
    expect(body).toContain('\r\n');
    // Every line break is CRLF: no bare \n.
    expect(body.replace(/\r\n/g, '').includes('\n')).toBe(false);
    const lines = body.split('\r\n').filter((l) => l !== '');
    expect(lines).toHaveLength(3);
    expect(lines[0]).toBe(TEMPLATE_HEADER);
    expect(lines[1]).toBe(TEMPLATE_ROW_1);
    expect(lines[2]).toBe(TEMPLATE_ROW_2);
  });
});

describe('F-2 bytesToBase64', () => {
  it('encodes raw bytes, not UTF-8 text: Caf + 0xE9 → Q2Fm6Q==', () => {
    expect(bytesToBase64(new Uint8Array([0x43, 0x61, 0x66, 0xe9]))).toBe('Q2Fm6Q==');
  });

  it('round-trips 70,000 random bytes without a stack overflow', () => {
    const input = new Uint8Array(70_000);
    for (let i = 0; i < input.length; i++) input[i] = Math.floor(Math.random() * 256);
    const b64 = bytesToBase64(input);
    const decoded = Uint8Array.from(Buffer.from(b64, 'base64'));
    expect(decoded.length).toBe(70_000);
    expect(Buffer.compare(Buffer.from(decoded), Buffer.from(input))).toBe(0);
  });
});

describe('F-3 buildMappingPayload', () => {
  it('omits fields set to (no importar), the empty value', () => {
    expect(buildMappingPayload({ name: 'PRODUCTO', sale_price: 'Precio', barcode: '' })).toEqual({
      name: 'PRODUCTO',
      sale_price: 'Precio',
    });
  });
});

describe('F-4 canCommit, formatImportError, moreErrorsText', () => {
  it('commits only a valid report with at least one valid row', () => {
    expect(canCommit({ valid: true, valid_rows: 3 })).toBe(true);
    expect(canCommit({ valid: false, valid_rows: 8 })).toBe(false);
    expect(canCommit({ valid: true, valid_rows: 0 })).toBe(false);
  });

  it('formats a row error as Fila · Columna: mensaje', () => {
    expect(formatImportError({ row: 7, column: 'Precio de venta', message: 'valor inválido…' })).toBe(
      'Fila 7 · Precio de venta: valor inválido…',
    );
  });

  it('tells how many errors are not shown', () => {
    expect(moreErrorsText(300, 200)).toBe('y 100 errores más');
    expect(moreErrorsText(2, 2)).toBe('');
  });
});

describe('F-5 audit description', () => {
  it('describes a product_import row', () => {
    expect(describeProductImportAudit('{"products_created":3,"file_name":"inventario.csv"}')).toBe(
      '3 productos importados de «inventario.csv»',
    );
  });

  it('labels the product_import action', () => {
    expect(PRODUCT_IMPORT_ACTION_LABELS.product_import).toBe('Importación de productos');
  });
});

describe('F-6 canImportProducts', () => {
  it('is true only for an admin', () => {
    expect(canImportProducts({ id: 1, username: 'admin', role: 'admin' })).toBe(true);
    expect(canImportProducts({ id: 2, username: 'caja1', role: 'cashier' })).toBe(false);
    expect(canImportProducts(null)).toBe(false);
  });
});
