// US-106 contract section 4, T-8 (AC-8).
//
// Helper API from the contract (frontend/lib/discountAudit.ts):
//   describeDiscountAudit(entity: string, entityID: number, details: string): string
//   where details is the raw audit_log.details JSON string.
import { describe, it, expect } from 'vitest';
import { describeDiscountAudit } from './discountAudit';

describe('T-8 describeDiscountAudit', () => {
  it('describes a line discount with the product name', () => {
    expect(
      describeDiscountAudit(
        'sale_item',
        7,
        '{"product_id":5,"product_name":"Arroz","quantity":2,"unit_price":50000,"amount":2000}',
      ),
    ).toBe('Descuento de $2.000 en Arroz (venta #7)');
  });

  it('keeps the sale-level text for entity sale', () => {
    expect(describeDiscountAudit('sale', 7, '{"amount":1000.00}')).toBe('Descuento de $1.000 en venta #7');
  });

  it('falls back to the product id when product_name is missing', () => {
    expect(describeDiscountAudit('sale_item', 7, '{"product_id":5,"amount":2000}')).toBe(
      'Descuento de $2.000 en producto #5 (venta #7)',
    );
  });

  it('falls back to a generic line text when line details are malformed', () => {
    expect(() => describeDiscountAudit('sale_item', 7, 'not json')).not.toThrow();
    expect(describeDiscountAudit('sale_item', 7, 'not json')).toBe('Descuento de $0 en producto (venta #7)');
  });

  it('falls back to $0 when sale-level details are malformed', () => {
    expect(() => describeDiscountAudit('sale', 7, 'not json')).not.toThrow();
    expect(describeDiscountAudit('sale', 7, 'not json')).toBe('Descuento de $0 en venta #7');
  });
});
