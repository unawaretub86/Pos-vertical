// US-086 contract section 4, T-1..T-7 (AC-1, AC-2, AC-4, AC-7).
//
// Helper API assumed (the contract leaves the names to the implementer):
//   resolveRouteParam(param: string | string[] | undefined, pathname: string, prefix: string): string | null
//     Returns `param` when it is a non-empty string other than '_' (pnpm dev).
//     Otherwise returns the first path segment after `/<prefix>/`, URI-decoded,
//     ignoring a trailing '/'; null when that segment is missing, empty or '_'.
//   parseCustomerId(segment: string | null): number | null
//     Positive integer id, or null (no request, `Cliente no encontrado`).
import { describe, it, expect } from 'vitest';
import { resolveRouteParam, parseCustomerId } from './routeParam';

describe('resolveRouteParam', () => {
  it('T-1 placeholder param reads the customer id from the path', () => {
    expect(resolveRouteParam('_', '/customers/1', 'customers')).toBe('1');
  });

  it('T-2 placeholder param ignores a trailing slash', () => {
    expect(resolveRouteParam('_', '/customers/42/', 'customers')).toBe('42');
  });

  it('T-3 a real param wins on the dev server', () => {
    expect(resolveRouteParam('7', '/customers/7', 'customers')).toBe('7');
  });

  it('T-4 undefined param reads the catalog token from the path, URI-decoded', () => {
    expect(resolveRouteParam(undefined, '/catalogo/tok086abc', 'catalogo')).toBe('tok086abc');
    expect(resolveRouteParam(undefined, '/catalogo/a%20b', 'catalogo')).toBe('a b');
  });

  it('T-5 placeholder param reads the order id from the mesas path', () => {
    expect(resolveRouteParam('_', '/mesas/12', 'mesas')).toBe('12');
  });

  it('T-6 missing, empty or placeholder segment, or a different prefix, gives null', () => {
    expect(resolveRouteParam('_', '/customers/_', 'customers')).toBeNull();
    expect(resolveRouteParam('_', '/customers', 'customers')).toBeNull();
    expect(resolveRouteParam('_', '/customers/', 'customers')).toBeNull();
    expect(resolveRouteParam('_', '/mesas/12', 'customers')).toBeNull();
  });

  it('T-8 malformed percent-encoding gives null without throwing', () => {
    expect(() => resolveRouteParam('_', '/catalogo/%E0', 'catalogo')).not.toThrow();
    expect(resolveRouteParam('_', '/catalogo/%E0', 'catalogo')).toBeNull();
    expect(() => resolveRouteParam(undefined, '/catalogo/%E0', 'catalogo')).not.toThrow();
    expect(resolveRouteParam(undefined, '/catalogo/%E0', 'catalogo')).toBeNull();
  });
});

describe('parseCustomerId', () => {
  it('T-7 accepts only a positive integer segment', () => {
    expect(parseCustomerId('1')).toBe(1);
    expect(parseCustomerId('abc')).toBeNull();
    expect(parseCustomerId('0')).toBeNull();
    expect(parseCustomerId('-3')).toBeNull();
    expect(parseCustomerId('1.5')).toBeNull();
    expect(parseCustomerId(null)).toBeNull();
  });
});
