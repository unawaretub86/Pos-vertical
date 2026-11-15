// Audit page text for `discount` rows (US-106): entity "sale" is the
// sale-level discount, entity "sale_item" is a per-line discount whose
// entity_id is the sale id.
import { formatCOP } from './format';

/**
 * Describes a discount audit row from its raw details JSON:
 * sale → "Descuento de $1.000 en venta #7";
 * sale_item → "Descuento de $2.000 en Arroz (venta #7)", falling back to
 * "producto #5" without a name and "producto" without a product id.
 */
export function describeDiscountAudit(entity: string, entityID: number, details: string): string {
  let parsed: Record<string, unknown> = {};
  try { parsed = JSON.parse(details || '{}'); } catch { /* malformed details: describe with defaults */ }
  if (typeof parsed !== 'object' || parsed === null) parsed = {};
  const amount = formatCOP(Number(parsed.amount ?? 0) || 0);

  if (entity === 'sale_item') {
    let product = 'producto';
    if (typeof parsed.product_name === 'string' && parsed.product_name !== '') {
      product = parsed.product_name;
    } else if (typeof parsed.product_id === 'number') {
      product = `producto #${parsed.product_id}`;
    }
    return `Descuento de ${amount} en ${product} (venta #${entityID})`;
  }
  return `Descuento de ${amount} en venta #${entityID}`;
}
