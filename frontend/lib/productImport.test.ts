import { describe, it, expect } from 'vitest';
import {
  buildTemplateCSV,
  bytesToBase64,
  buildMappingPayload,
  canCommit,
  formatImportError,
  moreErrorsText,
  describeProductImportAudit,
  readyText,
  commitLabel,
  importedToast,
  canImportProducts,
  PRODUCT_IMPORT_ACTION_LABELS,
  STARTER_CATALOG_URL,
  STARTER_CATALOG_FILE_NAME,
  STARTER_CATALOG_WARNING,
  loadStarterCatalog,
  shouldShowStarterWarning,
} from './productImport';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

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
    expect(csv.startsWith('\uFEFF')).toBe(true);
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

// R-157 (US-097) singular/plural wording. Helper names readyText, commitLabel and
// importedToast are chosen here: contract §4 leaves them to the implementer ("e.g.").
// Assumed signatures: readyText(n: number), importedToast(n: number),
// commitLabel(report: {valid, valid_rows} | null | undefined).
describe('R-157 T-1 count wording helpers', () => {
  it('readyText: 1 is singular, 3 is plural', () => {
    expect(readyText(1)).toBe('1 producto listo para importar');
    expect(readyText(3)).toBe('3 productos listos para importar');
  });

  it('commitLabel: 1 is singular, 3 is plural', () => {
    expect(commitLabel({ valid: true, valid_rows: 1 })).toBe('Importar 1 producto');
    expect(commitLabel({ valid: true, valid_rows: 3 })).toBe('Importar 3 productos');
  });

  it('commitLabel: plain Importar with no report, an invalid report or 0 valid rows', () => {
    expect(commitLabel(null)).toBe('Importar');
    expect(commitLabel(undefined)).toBe('Importar');
    expect(commitLabel({ valid: false, valid_rows: 8 })).toBe('Importar');
    expect(commitLabel({ valid: true, valid_rows: 0 })).toBe('Importar');
  });

  it('importedToast: 1 is singular, 3 is plural', () => {
    expect(importedToast(1)).toBe('Se importó 1 producto');
    expect(importedToast(3)).toBe('Se importaron 3 productos');
  });
});

describe('R-157 T-2 moreErrorsText singular', () => {
  it('one hidden error reads y 1 error más; 100 and 0 unchanged', () => {
    expect(moreErrorsText(3, 2)).toBe('y 1 error más');
    expect(moreErrorsText(300, 200)).toBe('y 100 errores más');
    expect(moreErrorsText(2, 2)).toBe('');
  });
});

describe('R-157 T-3 audit description singular', () => {
  it('1 product reads singular; 3 and 0 plural', () => {
    expect(describeProductImportAudit('{"products_created":1,"file_name":"inventario.csv"}')).toBe(
      '1 producto importado de «inventario.csv»',
    );
    expect(describeProductImportAudit('{"products_created":3,"file_name":"inventario.csv"}')).toBe(
      '3 productos importados de «inventario.csv»',
    );
    expect(describeProductImportAudit('{"products_created":0,"file_name":"inventario.csv"}')).toBe(
      '0 productos importados de «inventario.csv»',
    );
  });
});

// Contract R-173 (US-103) §4, T-6..T-8 (starter catalog).
// Assumed, not fixed by the contract: loadStarterCatalog(fetchFn) resolves to a
// Uint8Array and the fetch stub exposes { ok, status, arrayBuffer() }.

function fetchStub(ok: boolean, status: number, bytes: number[] = []) {
  return async () => ({
    ok,
    status,
    arrayBuffer: async () => new Uint8Array(bytes).buffer,
  });
}

describe('T-6 starter catalog loading', () => {
  it('exposes the URL and the file name', () => {
    expect(STARTER_CATALOG_URL).toBe('/catalogo-base-tienda.csv');
    expect(STARTER_CATALOG_FILE_NAME).toBe('catalogo-base-tienda.csv');
  });

  it('returns the same 4 bytes the fetch delivered', async () => {
    const out = await loadStarterCatalog(fetchStub(true, 200, [0xef, 0xbb, 0xbf, 0x4e]) as never);
    expect(Array.from(out)).toEqual([0xef, 0xbb, 0xbf, 0x4e]);
  });

  it('throws the Spanish error on a 404', async () => {
    await expect(loadStarterCatalog(fetchStub(false, 404) as never)).rejects.toThrow(
      'No se pudo cargar el catálogo base',
    );
  });
});

describe('T-7 template header matches the starter file', () => {
  it('first line of buildTemplateCSV equals the first line of the starter CSV', () => {
    const file = readFileSync(resolve(__dirname, '../public/catalogo-base-tienda.csv'), 'utf8');
    const starterFirst = file.replace(/^\uFEFF/, '').split('\r\n')[0];
    const templateFirst = buildTemplateCSV().replace(/^\uFEFF/, '').split('\r\n')[0];
    expect(starterFirst).toBe(templateFirst);
    expect(starterFirst).toBe(TEMPLATE_HEADER);
  });
});

describe('T-8 starter placeholder-price warning', () => {
  it('has the exact warning text', () => {
    expect(STARTER_CATALOG_WARNING).toBe(
      'Todos los precios son $1.000 de referencia; ajústelos antes de vender.',
    );
  });

  it('shows only for the starter file name', () => {
    expect(shouldShowStarterWarning('catalogo-base-tienda.csv')).toBe(true);
    expect(shouldShowStarterWarning('mis-productos.csv')).toBe(false);
    expect(shouldShowStarterWarning('')).toBe(false);
  });
});
