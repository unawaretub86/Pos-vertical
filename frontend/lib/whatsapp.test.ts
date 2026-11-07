// US-083 contract section 4, T-1..T-6 (AC-1, AC-2, AC-3, AC-4).
//
// Helper API proposed by the contract (frontend/lib/whatsapp.ts):
//   buildCollectionMessage(customerName: string, balance: number, businessName: string): string
//   buildCollectionWhatsAppUrl(phone: string, customerName: string, balance: number, businessName: string): string
import { describe, it, expect } from 'vitest';
import { buildCollectionMessage, buildCollectionWhatsAppUrl } from './whatsapp';

const WITH_NAME =
  'Hola Ana, le escribimos de Tienda Don Carlos. Le recordamos que tiene un saldo pendiente de $45.000. Gracias.';
const WITHOUT_NAME = 'Hola Ana, le recordamos que tiene un saldo pendiente de $45.000. Gracias.';

// Returns the raw (still encoded) value after `?text=`.
function rawText(url: string): string {
  const idx = url.indexOf('?text=');
  expect(idx).toBeGreaterThan(-1);
  return url.slice(idx + '?text='.length);
}

describe('buildCollectionMessage', () => {
  it('T-1 names the business after the greeting', () => {
    expect(buildCollectionMessage('Ana', 45000, 'Tienda Don Carlos')).toBe(WITH_NAME);
  });

  it('T-2 empty or whitespace-only business name drops the business sentence', () => {
    for (const biz of ['', '   ']) {
      const msg = buildCollectionMessage('Ana', 45000, biz);
      expect(msg).toBe(WITHOUT_NAME);
      expect(msg).not.toContain('el negocio');
      expect(msg).not.toContain('  ');
    }
  });

  it('T-3 trims leading and trailing spaces of the business name', () => {
    expect(buildCollectionMessage('Ana', 45000, '  Tienda Don Carlos  ')).toBe(WITH_NAME);
  });
});

describe('buildCollectionWhatsAppUrl', () => {
  it('T-4 prefixes 57 to the phone digits and encodes the message with name and balance', () => {
    const url = buildCollectionWhatsAppUrl('300 123 4567', 'Ana', 1234567, 'Tienda Don Carlos');
    expect(url.startsWith('https://wa.me/573001234567?text=')).toBe(true);
    const decoded = decodeURIComponent(rawText(url));
    expect(decoded).toContain('Tienda Don Carlos');
    expect(decoded).toContain('$1.234.567');
    expect(decoded).toBe(
      'Hola Ana, le escribimos de Tienda Don Carlos. Le recordamos que tiene un saldo pendiente de $1.234.567. Gracias.',
    );
  });

  it('T-5 does not double the 57 prefix and percent-encodes & in the business name', () => {
    const url = buildCollectionWhatsAppUrl('+57 300-123-4567', 'Ana', 45000, 'Tienda Don Carlos & Hijos');
    expect(url.startsWith('https://wa.me/573001234567?text=')).toBe(true);
    const raw = rawText(url);
    expect(raw).toContain('%26');
    expect(raw).not.toContain('&');
    expect(decodeURIComponent(raw)).toContain('Tienda Don Carlos & Hijos');
  });

  it('T-6 empty business name yields the no-name message in the URL', () => {
    const url = buildCollectionWhatsAppUrl('3001234567', 'Ana', 45000, '');
    expect(url.startsWith('https://wa.me/573001234567?text=')).toBe(true);
    expect(decodeURIComponent(rawText(url))).toBe(WITHOUT_NAME);
  });
});
